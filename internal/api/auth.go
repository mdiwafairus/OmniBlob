package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"pwni-file-sync/internal/config"

	"github.com/rs/zerolog"
)

type contextKey string

const (
	ClientContextKey contextKey = "authenticated_client"
)

// ClientInfo holds authenticated client application metadata.
type ClientInfo struct {
	Name           string
	User           string
	AllowedModules []string
}

// GetClientFromContext retrieves authenticated application info from HTTP request context.
func GetClientFromContext(ctx context.Context) *ClientInfo {
	if val, ok := ctx.Value(ClientContextKey).(*ClientInfo); ok {
		return val
	}
	return nil
}

// AppAuthMiddleware provides per-application authentication.
func AppAuthMiddleware(cfg *config.ServerConfig, log zerolog.Logger, next http.HandlerFunc) http.HandlerFunc {
	// Build fast lookup maps from config
	keyToClient := make(map[string]*ClientInfo)
	userPassToClient := make(map[string]*ClientInfo)

	// 1. Populate configured clients
	for _, c := range cfg.Clients {
		info := &ClientInfo{
			Name:           c.Name,
			User:           c.User,
			AllowedModules: c.AllowedModules,
		}
		if c.ApiKey != "" {
			keyToClient[c.ApiKey] = info
		}
		if c.User != "" && c.Password != "" {
			userPassToClient[c.User+":"+c.Password] = info
		}
	}

	// 2. Backward compatibility with single global API key / user
	if cfg.ApiKey != "" && keyToClient[cfg.ApiKey] == nil {
		keyToClient[cfg.ApiKey] = &ClientInfo{Name: "default-app", AllowedModules: []string{"*"}}
	}
	if cfg.ApiUser != "" && cfg.ApiPass != "" && userPassToClient[cfg.ApiUser+":"+cfg.ApiPass] == nil {
		userPassToClient[cfg.ApiUser+":"+cfg.ApiPass] = &ClientInfo{Name: "default-app", User: cfg.ApiUser, AllowedModules: []string{"*"}}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// If no authentication is configured on the server, allow request
		if len(keyToClient) == 0 && len(userPassToClient) == 0 {
			next(w, r)
			return
		}

		var matchedClient *ClientInfo

		// Strategy 1: Check X-API-KEY or X-App-Key header
		clientKey := strings.TrimSpace(r.Header.Get("X-API-KEY"))
		if clientKey == "" {
			clientKey = strings.TrimSpace(r.Header.Get("X-App-Key"))
		}
		if clientKey == "" {
			clientKey = strings.TrimSpace(r.Header.Get("X-Api-Key"))
		}
		if clientKey != "" {
			matchedClient = keyToClient[clientKey]
		}

		// Strategy 2: Check HTTP Basic Auth (user:password)
		if matchedClient == nil {
			if user, pass, ok := r.BasicAuth(); ok {
				matchedClient = userPassToClient[user+":"+pass]
			}
		}

		// Strategy 3: Check Custom X-API-USER & X-API-PASS headers
		if matchedClient == nil {
			user := strings.TrimSpace(r.Header.Get("X-API-USER"))
			pass := strings.TrimSpace(r.Header.Get("X-API-PASS"))
			if user != "" && pass != "" {
				matchedClient = userPassToClient[user+":"+pass]
			}
		}

		// If no client credentials matched -> reject with 401 Unauthorized
		if matchedClient == nil {
			log.Warn().
				Str("path", r.URL.Path).
				Str("remote_ip", r.RemoteAddr).
				Msg("Rejected unauthenticated request to protected endpoint")

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Connection", "close")
			w.Header().Set("WWW-Authenticate", `Basic realm="PWNI Storage Per-App Auth"`)
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Unauthorized: Invalid application credentials or API Key",
			})
			return
		}

		// Inject authenticated client info into request context
		ctx := context.WithValue(r.Context(), ClientContextKey, matchedClient)
		next(w, r.WithContext(ctx))
	}
}
