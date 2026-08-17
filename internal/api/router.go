package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"pwni-file-sync/internal/config"

	"github.com/rs/zerolog"
)

func NewRouter(h *Handler, cfg *config.ServerConfig, log zerolog.Logger) http.Handler {
	mux := http.NewServeMux()

	// Health check (Public)
	mux.HandleFunc("/health", h.HealthCheck)

	// API Routes (Upload is Protected with Auth)
	uploadHandler := authMiddleware(cfg, h.Upload)
	mux.HandleFunc("/api/v1/files/upload", uploadHandler)
	mux.HandleFunc("/api/v1/files/view", h.ServeFileByRef)
	mux.HandleFunc("/api/v1/files/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/files/upload" {
			uploadHandler(w, r)
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

// authMiddleware validates X-API-KEY header, Basic Auth (user & password), or custom headers.
func authMiddleware(cfg *config.ServerConfig, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If no auth is configured, allow all
		if cfg == nil || (cfg.ApiKey == "" && cfg.ApiUser == "" && cfg.ApiPass == "") {
			next(w, r)
			return
		}

		// 1. Check X-API-KEY header
		clientKey := strings.TrimSpace(r.Header.Get("X-API-KEY"))
		if clientKey == "" {
			clientKey = strings.TrimSpace(r.Header.Get("X-Api-Key"))
		}
		if cfg.ApiKey != "" && clientKey != "" && clientKey == cfg.ApiKey {
			next(w, r)
			return
		}

		// 2. Check HTTP Basic Auth (user:password)
		user, pass, hasBasic := r.BasicAuth()
		if hasBasic && cfg.ApiUser != "" && cfg.ApiPass != "" {
			if user == cfg.ApiUser && pass == cfg.ApiPass {
				next(w, r)
				return
			}
		}

		// 3. Check X-API-USER and X-API-PASS headers
		customUser := strings.TrimSpace(r.Header.Get("X-API-USER"))
		customPass := strings.TrimSpace(r.Header.Get("X-API-PASS"))
		if customUser != "" && customPass != "" && cfg.ApiUser != "" && cfg.ApiPass != "" {
			if customUser == cfg.ApiUser && customPass == cfg.ApiPass {
				next(w, r)
				return
			}
		}

		// 4. If authentication fails -> return 401 Unauthorized
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Connection", "close")
		w.Header().Set("WWW-Authenticate", `Basic realm="PWNI Storage Access"`)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Unauthorized: Invalid or missing API credentials (X-API-KEY or Basic Auth required)",
		})
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Range, X-API-KEY, X-API-USER, X-API-PASS")
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
