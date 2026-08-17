package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

func NewRouter(h *Handler, log zerolog.Logger) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", h.HealthCheck)

	// API Routes
	mux.HandleFunc("/api/v1/files/upload", h.Upload)
	mux.HandleFunc("/api/v1/files/view", h.ServeFileByRef)
	mux.HandleFunc("/api/v1/files/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/files/upload" {
			h.Upload(w, r)
			return
		}
		if r.URL.Path == "/api/v1/files/view" {
			h.ServeFileByRef(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/files/") {
			h.ServeFileByID(w, r)
			return
		}
		http.NotFound(w, r)
	})

	// Wrap with Middlewares
	var handler http.Handler = mux
	handler = corsMiddleware(handler)
	handler = loggingMiddleware(handler, log)
	handler = recoveryMiddleware(handler, log)

	return handler
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Range")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, ETag, X-Storage-Path")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler, log zerolog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		if r.URL.Path != "/health" { // keep healthcheck logs minimal
			log.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", wrapped.statusCode).
				Dur("duration_ms", duration).
				Str("remote_ip", r.RemoteAddr).
				Msg("HTTP Request")
		}
	})
}

func recoveryMiddleware(next http.Handler, log zerolog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error().Interface("panic", rec).Str("path", r.URL.Path).Msg("Unhandled HTTP Panic recovered")
				http.Error(w, `{"success":false,"error":"Internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
