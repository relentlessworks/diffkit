package auth

import (
	"net/http"
	"strings"
)

// Middleware wraps an http.Handler with authentication.
type Middleware func(http.Handler) http.Handler

// APIKeyMiddleware returns middleware that checks for a bearer token
// matching the provided API key. If apiKey is empty, no auth is required.
func APIKeyMiddleware(apiKey string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("error: missing or invalid auth token | hint: send Authorization: Bearer <api-key> header\n"))
				return
			}
			token := strings.TrimPrefix(auth, "Bearer ")
			if token != apiKey {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("error: invalid api key | hint: check your API key and try again\n"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
