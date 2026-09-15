package api

import (
	"net"
	"strings"

	"pwni-file-sync/internal/config"
	"pwni-file-sync/internal/logger"
)

type whitelistedListener struct {
	net.Listener
	cfg         *config.SecurityConfig
	auditLogger *logger.AuditLogger
}

func NewWhitelistedListener(inner net.Listener, cfg *config.SecurityConfig, auditLogger *logger.AuditLogger) net.Listener {
	return &whitelistedListener{
		Listener:    inner,
		cfg:         cfg,
		auditLogger: auditLogger,
	}
}

func (l *whitelistedListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	if l.cfg.Mode == "direct_socket" {
		remoteAddr := conn.RemoteAddr().String()
		ip := strings.Split(remoteAddr, ":")[0]

		if !isIPAllowed(ip, l.cfg.AllowedSocketIPs) {
			if l.auditLogger != nil {
				l.auditLogger.LogConnection(
					remoteAddr,
					"N/A (Dropped at TCP Layer)",
					"DENIED",
					"denied: direct socket access attempted from untrusted IP",
				)
			}
			conn.Close()
			// Continue accepting next connection to avoid server stopping
			return l.Accept()
		}

		// Lolos L4 Whitelist
		// Note: We don't log success here to avoid spamming the log for every valid request at L4.
		// L7 middleware will handle successful L7 logging if needed.
	}

	return conn, nil
}

// isIPAllowed checks if an IP matches any CIDR or literal IP in the allowed list
func isIPAllowed(ip string, allowed []string) bool {
	if len(allowed) == 0 {
		return true // If empty, assume open (or could be strict deny, depending on policy)
	}
	
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, rule := range allowed {
		if strings.Contains(rule, "/") {
			// CIDR
			_, subnet, err := net.ParseCIDR(rule)
			if err == nil && subnet.Contains(parsedIP) {
				return true
			}
		} else {
			// Literal IP
			if ip == rule {
				return true
			}
		}
	}
	return false
}
