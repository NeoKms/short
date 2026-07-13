package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vladislav/short/internal/cache"
	"github.com/vladislav/short/internal/config"
	"github.com/vladislav/short/internal/httpapi"
	"github.com/vladislav/short/internal/link"
	"github.com/vladislav/short/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	postgres, err := storage.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer postgres.Close()
	if err = postgres.Migrate(ctx); err != nil {
		return err
	}

	redisCache, err := cache.Open(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		return err
	}
	defer redisCache.Close()

	service := link.NewService(postgres, redisCache, cfg.CacheTTL, cfg.DefaultLinkTTL)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.New(service, cfg.PublicBaseURL, postgres, redisCache, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server started", "address", cfg.HTTPAddr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err = <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
