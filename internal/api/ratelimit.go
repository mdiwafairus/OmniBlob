package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"pwni-file-sync/internal/config"
)

type clientRater struct {
	tokens int
	last   time.Time
}

var (
	rateMu       sync.Mutex
	clientLimits = make(map[string]*clientRater)
	
// Default limit if not set in config: 100 requests per minute per IP
	defaultRateLimit = 100
	rateWindow       = 1 * time.Minute
)

func rateLimitMiddleware(cfg *config.ServerConfig) func(http.Handler) http.Handler {
	limit := cfg.RateLimitRPM
	if limit <= 0 {
		limit = defaultRateLimit
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := r.Header.Get("X-Real-IP")
			if clientIP == "" {
				clientIP = r.Header.Get("X-Forwarded-For")
			}
			if clientIP == "" {
				clientIP = strings.Split(r.RemoteAddr, ":")[0]
			}

			rateMu.Lock()
			rater, exists := clientLimits[clientIP]
			now := time.Now()

			if !exists {
				rater = &clientRater{tokens: limit - 1, last: now}
				clientLimits[clientIP] = rater
			} else {
				// Replenish tokens if window has passed
				elapsed := now.Sub(rater.last)
				if elapsed > rateWindow {
					rater.tokens = limit
				}

				if rater.tokens > 0 {
					rater.tokens--
					rater.last = now
				} else {
					rateMu.Unlock()
					
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.Header().Set("Connection", "close")
					w.Header().Set("Retry-After", "60")
					w.WriteHeader(http.StatusTooManyRequests)
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"error":   "Too Many Requests: Rate limit exceeded (DDoS Protection)",
					})
					return
				}
			}
			rateMu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

