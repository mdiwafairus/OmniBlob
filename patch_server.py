import os

with open('internal/api/server.go', 'r') as f:
    text = f.read()

imports = """	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
"""
text = text.replace('import (\n\t"context"', f'import (\n\t"context"\n{imports}')

text = text.replace('port = 8080', 'port = 8443 // Default to HTTPS port if not specified')

old_write_timeout = """	writeTimeout := time.Duration(cfg.WriteTimeoutSec) * time.Second
	if writeTimeout <= 0 {
		writeTimeout = 60 * time.Second
	}"""
new_write_timeout = "\t// Disable WriteTimeout to allow long-lived SSE connections\n\tvar writeTimeout time.Duration = 0"
text = text.replace(old_write_timeout, new_write_timeout)

cert_func = """
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
"""

text = text.replace('func (s *Server) Start() error {', cert_func + '\nfunc (s *Server) Start() error {')

start_logic = """	certPath := "server.crt"
	keyPath := "server.key"

	if err := s.generateSelfSignedCert(certPath, keyPath); err != nil {
		return fmt.Errorf("failed to generate certs: %w", err)
	}

	s.logger.Info().Str("addr", s.httpServer.Addr).Msg("HTTPS Server listening for requests (HTTP/2 Enabled)")"""
text = text.replace('s.logger.Info().Str("addr", s.httpServer.Addr).Msg("HTTP File Server listening for requests")', start_logic)

text = text.replace('if err := s.httpServer.Serve(ln);', 'if err := s.httpServer.ServeTLS(ln, certPath, keyPath);')
text = text.replace('"Shutting down HTTP File Server..."', '"Shutting down HTTPS Server..."')

with open('internal/api/server.go', 'w') as f:
    f.write(text)
