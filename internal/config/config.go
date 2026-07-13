package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr        string
	PublicBaseURL   string
	DatabaseURL     string
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	CacheTTL        time.Duration
	DefaultLinkTTL  time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	redisDB, err := strconv.Atoi(getenv("REDIS_DB", "0"))
	if err != nil {
		return Config{}, fmt.Errorf("parse REDIS_DB: %w", err)
	}

	cacheTTL, err := time.ParseDuration(getenv("CACHE_TTL", "24h"))
	if err != nil {
		return Config{}, fmt.Errorf("parse CACHE_TTL: %w", err)
	}

	defaultLinkTTL, err := time.ParseDuration(getenv("DEFAULT_LINK_TTL", "0"))
	if err != nil {
		return Config{}, fmt.Errorf("parse DEFAULT_LINK_TTL: %w", err)
	}

	shutdownTimeout, err := time.ParseDuration(getenv("SHUTDOWN_TIMEOUT", "10s"))
	if err != nil {
		return Config{}, fmt.Errorf("parse SHUTDOWN_TIMEOUT: %w", err)
	}

	cfg := Config{
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		PublicBaseURL:   strings.TrimRight(getenv("PUBLIC_BASE_URL", "http://localhost:8080"), "/"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		RedisAddr:       getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   os.Getenv("REDIS_PASSWORD"),
		RedisDB:         redisDB,
		CacheTTL:        cacheTTL,
		DefaultLinkTTL:  defaultLinkTTL,
		ShutdownTimeout: shutdownTimeout,
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.PublicBaseURL == "" {
		return Config{}, fmt.Errorf("PUBLIC_BASE_URL is required")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
