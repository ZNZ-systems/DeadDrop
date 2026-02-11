package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const csrfCookieName = "csrf_token"
const csrfHeaderName = "X-CSRF-Token"

// CSRFWithOptions provides double-submit cookie CSRF protection.
// It generates a token if not present, sets it as a cookie,
// and validates it on non-safe methods.
func CSRFWithOptions(secureCookies bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get or generate CSRF token
			cookie, err := r.Cookie(csrfCookieName)
			var token string
			if err != nil || cookie.Value == "" {
				token = generateCSRFToken()
				http.SetCookie(w, &http.Cookie{
					Name:     csrfCookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: false, // JS needs to read it for HTMX
					Secure:   secureCookies,
					SameSite: http.SameSiteLaxMode,
				})
			} else {
				token = cookie.Value
			}

			// For non-safe methods, validate the token
			if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
				submitted := r.Header.Get(csrfHeaderName)
				if submitted == "" {
					submitted = r.FormValue("csrf_token")
				}
				if submitted != token {
					http.Error(w, "CSRF token mismatch", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CSRF is the default CSRF middleware (non-secure cookies for backwards compatibility).
func CSRF(next http.Handler) http.Handler {
	return CSRFWithOptions(false)(next)
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
