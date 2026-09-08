package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"os"
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
		port = 8443 // Default to HTTPS port if not specified
	}

	readTimeout := time.Duration(cfg.ReadTimeoutSec) * time.Second
	if readTimeout <= 0 {
		readTimeout = 30 * time.Second
	}

	// Disable WriteTimeout to allow long-lived SSE connections
	var writeTimeout time.Duration = 0

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

func (s *Server) generateSelfSignedCert(certPath, keyPath string) error {
	if _, err := os.Stat(certPath); err == nil {
		if _, err := os.Stat(keyPath); err == nil {
			return nil // Certs already exist
		}
	}

	s.logger.Info().Msg("Generating self-signed SSL certificates for HTTP/2 support...")

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"OmniBlob"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour), // 1 year

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return err
	}

	keyOut, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	defer keyOut.Close()
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return err
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return err
	}

	s.logger.Info().Msg("Self-signed certificates generated successfully (server.crt, server.key)")
	return nil
}

func (s *Server) Start() error {
	certPath := "server.crt"
	keyPath := "server.key"

	if err := s.generateSelfSignedCert(certPath, keyPath); err != nil {
		return fmt.Errorf("failed to generate certs: %w", err)
	}

	s.logger.Info().Str("addr", s.httpServer.Addr).Msg("HTTPS Server listening for requests (HTTP/2 Enabled)")
	if err := s.httpServer.ListenAndServeTLS(certPath, keyPath); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("https server failed: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("Shutting down HTTPS Server...")
	return s.httpServer.Shutdown(ctx)
}
