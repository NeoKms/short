package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/vladislav/short/internal/link"
)

const maxLocationHeaderURLLength = 2048

type HealthChecker interface{ Ping(context.Context) error }

type Handler struct {
	service       *link.Service
	publicBaseURL string
	postgres      HealthChecker
	redis         HealthChecker
	logger        *slog.Logger
}

func New(service *link.Service, publicBaseURL string, postgres, redis HealthChecker, logger *slog.Logger) http.Handler {
	h := &Handler{service: service, publicBaseURL: publicBaseURL, postgres: postgres, redis: redis, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/links", h.create)
	mux.HandleFunc("GET /health/live", h.live)
	mux.HandleFunc("GET /health/ready", h.ready)
	mux.HandleFunc("GET /{code}", h.redirect)
	return h.logging(mux)
}

type createRequest struct {
	OriginalURL string     `json:"original_url"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

type createResponse struct {
	link.Link
	ShortURL string `json:"short_url"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10))
	decoder.DisallowUnknownFields()
	var request createRequest
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body must be valid JSON")
		return
	}
	created, err := h.service.Create(r.Context(), request.OriginalURL, request.ExpiresAt)
	if err != nil {
		if errors.Is(err, link.ErrInvalidURL) || errors.Is(err, link.ErrInvalidExpiry) {
			writeError(w, http.StatusBadRequest, "invalid_link", err.Error())
			return
		}
		h.logger.Error("create link", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, createResponse{Link: created, ShortURL: h.publicBaseURL + "/" + created.Code})
}

func (h *Handler) redirect(w http.ResponseWriter, r *http.Request) {
	resolved, err := h.service.Resolve(r.Context(), r.PathValue("code"))
	if errors.Is(err, link.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "short link does not exist or has expired")
		return
	}
	if err != nil {
		h.logger.Error("resolve link", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	redirect(w, r, resolved.OriginalURL)
}

func redirect(w http.ResponseWriter, r *http.Request, target string) {
	if len(target) <= maxLocationHeaderURLLength {
		http.Redirect(w, r, target, http.StatusFound)
		return
	}

	encodedTarget, _ := json.Marshal(target)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="robots" content="noindex"><title>Redirecting</title></head>
<body><p>Redirecting…</p><script>window.location.replace(%s);</script><noscript>JavaScript is required to open this link.</noscript></body>
</html>`, encodedTarget)
}

func (h *Handler) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	if err := h.postgres.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready", "postgres is unavailable")
		return
	}
	if err := h.redis.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready", "redis is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		h.logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(started))
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
