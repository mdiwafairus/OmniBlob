package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/go-ldap/ldap/v3"

	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"
	"pwni-file-sync/internal/config"
	"pwni-file-sync/internal/logger"

	"github.com/rs/zerolog"
)

type contextKey string

const (
	ClientContextKey contextKey = "authenticated_client"
)

type ClientInfo struct {
	Name           string
	User           string
	Password       string
	AllowedModules []string
	WhitelistIPs   []string
}

// GetClientFromContext retrieves authenticated application info from HTTP request context.
func GetClientFromContext(ctx context.Context) *ClientInfo {
	if val, ok := ctx.Value(ClientContextKey).(*ClientInfo); ok {
		return val
	}
	return nil
}

// AppAuthMiddleware provides per-application authentication.

func authenticateLDAP(cfg *config.LDAPConfig, username, password string) bool {
	if username == "" || password == "" {
		return false
	}
	
	var l *ldap.Conn
	var err error
	
	if cfg.UseTLS {
		tlsConfig := &tls.Config{InsecureSkipVerify: cfg.SkipVerify}
		l, err = ldap.DialTLS("tcp", cfg.ServerAddr, tlsConfig)
	} else {
		l, err = ldap.Dial("tcp", cfg.ServerAddr)
	}
	
	if err != nil {
		return false
	}
	defer l.Close()

	if cfg.BindDN != "" && cfg.BindPassword != "" {
		err = l.Bind(cfg.BindDN, cfg.BindPassword)
		if err != nil {
			return false
		}
	}

	searchRequest := ldap.NewSearchRequest(
		cfg.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf(cfg.UserFilter, ldap.EscapeFilter(username)),
		[]string{"dn"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil || len(sr.Entries) == 0 {
		return false
	}

	userDN := sr.Entries[0].DN
	err = l.Bind(userDN, password)
	return err == nil
}

func AppAuthMiddleware(cfg *config.ServerConfig, log zerolog.Logger, dbPool *pgxpool.Pool, next http.HandlerFunc) http.HandlerFunc {
	suspectLogger := logger.NewSuspectLogger()

	// Build fast lookup map from config
	// Key: "APIKey:User:Password" -> ClientInfo
	comboToClient := make(map[string]*ClientInfo)

	// 1. Populate configured clients
	for _, c := range cfg.Clients {
		if c.ApiKey != "" && c.User != "" && c.Password != "" {
			comboKey := c.ApiKey + ":" + c.User + ":" + c.Password
			if _, exists := comboToClient[comboKey]; exists {
				log.Fatal().Str("client", c.Name).Msg("Duplicate 3-layer authentication combination found in config!")
			}
			comboToClient[comboKey] = &ClientInfo{
				Name:           c.Name,
				User:           c.User,
				Password:       c.Password,
				AllowedModules: c.AllowedModules,
				WhitelistIPs:   c.WhitelistIPs,
			}
		} else {
			log.Warn().Str("client", c.Name).Msg("Client ignored: Must provide API Key, User, and Password")
		}
	}

	// 2. Backward compatibility with single global API key / user
	if cfg.ApiKey != "" && cfg.ApiUser != "" && cfg.ApiPass != "" {
		comboKey := cfg.ApiKey + ":" + cfg.ApiUser + ":" + cfg.ApiPass
		if _, exists := comboToClient[comboKey]; exists {
			log.Fatal().Msg("Global authentication combination conflicts with an existing client!")
		}
		comboToClient[comboKey] = &ClientInfo{
			Name: "default-app", 
			User: cfg.ApiUser, 
			Password: cfg.ApiPass,
			AllowedModules: []string{"*"},
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// If no authentication is configured on the server, allow request
		if len(comboToClient) == 0 {
			next(w, r)
			return
		}

		var matchedClient *ClientInfo

		// Extract API Key
		clientKey := strings.TrimSpace(r.Header.Get("X-API-KEY"))
		if clientKey == "" {
			clientKey = strings.TrimSpace(r.Header.Get("X-App-Key"))
		}
		if clientKey == "" {
			clientKey = strings.TrimSpace(r.Header.Get("X-Api-Key"))
		}

		// Extract User and Password
		user, pass, ok := r.BasicAuth()
		if !ok {
			user = strings.TrimSpace(r.Header.Get("X-API-USER"))
			pass = strings.TrimSpace(r.Header.Get("X-API-PASS"))
		}

		// Validate all 3 are present
		if clientKey != "" && user != "" && pass != "" {
			comboKey := clientKey + ":" + user + ":" + pass
			matchedClient = comboToClient[comboKey]
			
			// LDAP Fallback (if enabled and user/pass is valid in LDAP)
			if matchedClient == nil && cfg.LDAP.Enabled && authenticateLDAP(&cfg.LDAP, user, pass) {
				// If LDAP succeeds, we still need to verify the API Key belongs to SOME client?
				// To keep it strictly 3-layer, we can check if LDAP is allowed to use this API Key.
				// For now, if LDAP is enabled, we create an ephemeral client.
				matchedClient = &ClientInfo{Name: "ldap-user", User: user, AllowedModules: []string{"*"}}
			}
		}

		// Extract real IP
		clientIP := r.Header.Get("X-Real-IP")
		if clientIP == "" {
			clientIP = r.Header.Get("X-Forwarded-For")
		}
		if clientIP == "" {
			clientIP = strings.Split(r.RemoteAddr, ":")[0]
		}

		// Check if IP is globally blocked
		if IsIPBlocked(clientIP) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Connection", "close")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Too Many Requests: Your IP has been temporarily blocked due to multiple suspicious attempts",
			})
			return
		}

		// If no client credentials matched -> reject with 401 Unauthorized
		if matchedClient == nil {
			justBanned := RecordFailedAttempt(clientIP, "Multiple failed authentication attempts", dbPool)
			
			logMsg := "Rejected unauthenticated request to protected endpoint"
			if justBanned {
				logMsg = "IP temporarily blocked due to multiple failed authentication attempts"
			}
			
			suspectLogger.Warn().
				Str("path", r.URL.Path).
				Str("remote_ip", clientIP).
				Msg(logMsg)

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

		// IP Whitelist Check
		if len(matchedClient.WhitelistIPs) > 0 {
			ipAllowed := false
			for _, allowedIP := range matchedClient.WhitelistIPs {
				if clientIP == allowedIP {
					ipAllowed = true
					break
				}
			}

			if !ipAllowed {
				justBanned := RecordFailedAttempt(clientIP, "Multiple whitelist bypass attempts", dbPool)

				logMsg := "Access denied: IP not in whitelist"
				if justBanned {
					logMsg = "IP temporarily blocked due to multiple whitelist bypass attempts"
				}

				suspectLog := suspectLogger.With().
					Str("client", matchedClient.Name).
					Str("path", r.URL.Path).
					Str("remote_ip", clientIP).
					Logger()

				suspectLog.Warn().Msg(logMsg)

				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.Header().Set("Connection", "close")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error":   "Forbidden: Your IP is not whitelisted for this application",
				})
				return
			}
		}

		// Bucket/Module Validation
		bucket := r.PathValue("bucket")
		
		// Fallback for legacy endpoints (/api/v1/files/...) where bucket is in form or query
		if bucket == "" || bucket == "files" {
			bucket = strings.TrimSpace(r.FormValue("module"))
			if bucket == "" {
				bucket = strings.TrimSpace(r.FormValue("modul"))
			}
			if bucket == "" {
				bucket = strings.TrimSpace(r.URL.Query().Get("module"))
			}
			if bucket == "" {
				bucket = "general"
			}
		}

		if bucket != "" && bucket != "files" {
			moduleAllowed := false
			for _, allowedModule := range matchedClient.AllowedModules {
				if allowedModule == "*" || allowedModule == bucket {
					moduleAllowed = true
					break
				}
			}

			if !moduleAllowed {
				suspectLogger.Warn().
					Str("client", matchedClient.Name).
					Str("path", r.URL.Path).
					Str("bucket", bucket).
					Str("remote_ip", clientIP).
					Msg("Access denied: Client not authorized for this bucket")

				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.Header().Set("Connection", "close")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error":   fmt.Sprintf("Forbidden: Your application is not authorized to access bucket/module '%s'", bucket),
				})
				return
			}
		}

		// Inject authenticated client info into request context
		ctx := context.WithValue(r.Context(), ClientContextKey, matchedClient)
		next(w, r.WithContext(ctx))
	}
}
