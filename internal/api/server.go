package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"pwni-file-sync/internal/config"

	"github.com/rs/zerolog"
)

type Server struct {
	httpServer *http.Server
	logger     zerolog.Logger
}

func NewServer(cfg *config.ServerConfig, handler http.Handler, log zerolog.Logger) *Server {
	host := cfg.Host
	if host == "" {
		host = "0.0.0.0"
	}

	port := cfg.Port
	if port <= 0 {
		port = 8080
	}

	readTimeout := time.Duration(cfg.ReadTimeoutSec) * time.Second
	if readTimeout <= 0 {
		readTimeout = 30 * time.Second
	}

	writeTimeout := time.Duration(cfg.WriteTimeoutSec) * time.Second
	if writeTimeout <= 0 {
		writeTimeout = 60 * time.Second
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  120 * time.Second,
	}

	return &Server{
		httpServer: srv,
		logger:     log,
	}
}

func (s *Server) Start() error {
	s.logger.Info().Str("addr", s.httpServer.Addr).Msg("HTTP File Server listening for requests")
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server failed: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("Shutting down HTTP File Server...")
	return s.httpServer.Shutdown(ctx)
}
