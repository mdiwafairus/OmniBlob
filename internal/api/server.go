package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"pwni-file-sync/internal/config"
	"pwni-file-sync/internal/logger"

	"github.com/rs/zerolog"
)

type Server struct {
	httpServer  *http.Server
	logger      zerolog.Logger
	securityCfg *config.SecurityConfig
	auditLogger *logger.AuditLogger
}

func NewServer(cfg *config.ServerConfig, secCfg *config.SecurityConfig, auditLogger *logger.AuditLogger, handler http.Handler, log zerolog.Logger) *Server {
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
		httpServer:  srv,
		logger:      log,
		securityCfg: secCfg,
		auditLogger: auditLogger,
	}
}

func (s *Server) Start() error {
	s.logger.Info().Str("addr", s.httpServer.Addr).Msg("HTTP File Server listening for requests")
	
	// Create raw TCP listener
	
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.httpServer.Addr, err)
	}

	// Wrap with our Whitelisted Listener if in direct_socket mode
	if s.securityCfg.Mode == "direct_socket" {
		s.logger.Info().Strs("allowed_ips", s.securityCfg.AllowedSocketIPs).Msg("Security Mode: direct_socket (Layer 4 TCP Drop enabled)")
		ln = NewWhitelistedListener(ln, s.securityCfg, s.auditLogger)
	} else if s.securityCfg.Mode == "behind_proxy" {
		s.logger.Info().Msg("Security Mode: behind_proxy (Trusting Proxy X-Forwarded-For)")
	}

	if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server failed: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("Shutting down HTTP File Server...")
	return s.httpServer.Shutdown(ctx)
}
