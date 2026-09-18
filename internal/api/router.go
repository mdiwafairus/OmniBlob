package api

import (
	"io/fs"
	"net/http"
	"strings"
	"time"

	"pwni-file-sync/dashboard"
	"pwni-file-sync/internal/auth"
	"pwni-file-sync/internal/config"
	"pwni-file-sync/internal/logger"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
)

func NewRouter(
	h *Handler,
	dh *DashboardHandler,
	eh *ExplorerHandler,
	cfg *config.ServerConfig,
	secCfg *config.SecurityConfig,
	auditLogger *logger.AuditLogger,
	secretKey string,
	log zerolog.Logger,
	dbPool *pgxpool.Pool,
) http.Handler {
	mux := http.NewServeMux()

	// Extract the embedded dashboard files
	distFS, err := fs.Sub(dashboard.FS, "dist")

	// Serve Dashboard if built, otherwise fallback to basic status
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Do not intercept API or Health routes
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" {
			http.NotFound(w, r)
			return
		}

		// If distFS is available (embedded), serve it
		if err == nil {
			// Check if file exists in the embedded FS
			filePath := strings.TrimPrefix(r.URL.Path, "/")
			if filePath == "" {
				filePath = "index.html"
			}

			if _, statErr := fs.Stat(distFS, filePath); statErr == nil {
				// File exists, serve it
				http.FileServer(http.FS(distFS)).ServeHTTP(w, r)
				return
			}

			// Fallback to index.html for React Router / SPA
			r.URL.Path = "/"
			http.FileServer(http.FS(distFS)).ServeHTTP(w, r)
			return
		}

		// Fallback if dashboard is not embedded
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>OmniBlob Storage</title>
				<style>
					body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; padding: 40px; background-color: #f8f9fa; color: #333; }
					.container { max-width: 600px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
					h1 { color: #0056b3; }
					.status { display: inline-block; padding: 5px 10px; background: #28a745; color: white; border-radius: 4px; font-weight: bold; font-size: 14px; }
					code { background: #eee; padding: 2px 6px; border-radius: 4px; }
				</style>
			</head>
			<body>
				<div class="container">
					<h1>📦 OmniBlob Storage Node</h1>
					<p>Status: <span class="status">RUNNING</span></p>
					<p>Selamat datang! Server Object Storage & Sync Anda sedang berjalan.</p>
					<hr>
					<h3>Endpoints Tersedia:</h3>
					<ul>
						<li><code>GET /health</code> - Cek kesehatan server & database</li>
						<li><code>POST /api/v1/files/upload</code> - Upload file</li>
						<li><code>GET /api/v1/files/view?ref=...</code> - Lihat/Download file</li>
					</ul>
					<p><em>Catatan: Karena OmniBlob adalah aplikasi backend (API Server), pengelolaan file utama dilakukan melalui API Client (seperti Postman, cURL, atau Backend Utama Anda).</em></p>
				</div>
			</body>
			</html>
		`))
	})

	// Health check (Public)
	mux.HandleFunc("/health", h.HealthCheck)

	// Dashboard API Routes
	if dh != nil {
		mux.HandleFunc("/api/v1/dashboard/summary", dh.GetSummary)
		mux.HandleFunc("/api/v1/dashboard/analytics", dh.GetAnalytics)
		mux.HandleFunc("/api/v1/dashboard/quality", dh.GetQuality)
	}

	// Explorer Route
	if eh != nil {
		mux.HandleFunc("/api/v1/explorer/list", eh.ListDirectory)
	}

	// Presign URL generator (requires AppAuthMiddleware with dbPool)
	presignHandler := AppAuthMiddleware(cfg, log, dbPool, h.GeneratePresignedURL)
	mux.HandleFunc("/api/v1/presign", presignHandler)

	// API Routes (Upload is Protected with Per-App Auth OR Presigned URL)
	uploadHandler := AppAuthMiddleware(cfg, log, dbPool, h.Upload)
	bulkUploadHandler := AppAuthMiddleware(cfg, log, dbPool, h.BulkUpload)

	// Create a wrapper that checks presigned URL first, then falls back to AppAuthMiddleware
	secureUploadHandler := func(w http.ResponseWriter, r *http.Request) {
		if auth.ValidatePresignedURL(r, secretKey) {
			h.Upload(w, r)
			return
		}
		uploadHandler(w, r)
	}

	// Dynamic Bucket Routing
	mux.HandleFunc("/api/v1/{bucket}/upload", secureUploadHandler)
	mux.HandleFunc("/api/v1/{bucket}/bulk-upload", bulkUploadHandler)
	mux.HandleFunc("/api/v1/{bucket}/view", h.ServeFileByRef)
	mux.HandleFunc("/api/v1/{bucket}/", func(w http.ResponseWriter, r *http.Request) {
		bucket := r.PathValue("bucket")
		if r.URL.Path == "/api/v1/"+bucket+"/upload" {
			secureUploadHandler(w, r)
			return
		}
		if r.URL.Path == "/api/v1/"+bucket+"/bulk-upload" {
			bulkUploadHandler(w, r)
			return
		}
		if r.URL.Path == "/api/v1/"+bucket+"/view" {
			h.ServeFileByRef(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/"+bucket+"/") {
			if r.Method == http.MethodDelete {
				h.DeleteFile(w, r)
			} else {
				h.ServeFileByID(w, r)
			}
			return
		}
		http.NotFound(w, r)
	})

	// Wrap with Middlewares
	var handler http.Handler = mux
	handler = auditMiddleware(handler, secCfg, auditLogger)
	handler = rateLimitMiddleware(cfg)(handler)
	handler = corsMiddleware(handler)
	handler = loggingMiddleware(handler, log)
	handler = recoveryMiddleware(handler, log)

	return handler
}

func auditMiddleware(next http.Handler, secCfg *config.SecurityConfig, auditLogger *logger.AuditLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !secCfg.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Log if behind_proxy or in general for transparency
		xff := r.Header.Get("X-Forwarded-For")
		remoteAddr := r.RemoteAddr

		statusStr := "ignored: not in behind_proxy mode"
		if secCfg.Mode == "behind_proxy" {
			statusStr = "trusted"
		} else if xff != "" {
			statusStr = "ignored: proxy IP trusted, but XFF format invalid/tampered or direct_socket mode"
		} else {
			statusStr = "N/A"
		}

		if auditLogger != nil {
			xffLog := xff
			if xff == "" {
				xffLog = "N/A"
			} else {
				xffLog = xff + " (" + statusStr + ")"
			}

			// We only record initial connection properties here.
			// True authentication decisions might happen downstream,
			// but this gives a guaranteed L7 trail for accepted sockets.
			auditLogger.LogConnection(remoteAddr, xffLog, "L7_ACCEPTED", "Socket IP allowed, processing HTTP")
		}

		next.ServeHTTP(w, r)
	})
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