package middleware

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/znz-systems/deaddrop/internal/ratelimit"
)

// RateLimit returns middleware that rate-limits requests on a per-IP basis
// using the provided Limiter. When the rate limit is exceeded, it responds
// with a 429 Too Many Requests status and a JSON error body.
func RateLimit(limiter *ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				// If RemoteAddr has no port, use it as-is.
				ip = r.RemoteAddr
			}

			if !limiter.Allow(ip) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "rate limit exceeded",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
