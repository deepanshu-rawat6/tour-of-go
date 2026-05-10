package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"tour_of_go/projects/from-scratch/11-api-gateway/internal/auth"
	ratelimit "tour_of_go/projects/from-scratch/04-rate-limiter/ratelimit"
)

type contextKey string

const (
	claimsKey    contextKey = "claims"
	requestIDKey contextKey = "requestID"
)

// ClaimsFromContext retrieves Claims injected by Auth middleware.
func ClaimsFromContext(ctx context.Context) (auth.Claims, bool) {
	c, ok := ctx.Value(claimsKey).(auth.Claims)
	return c, ok
}

// RequestIDFromContext retrieves the request ID injected by RequestID middleware.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// RequestID checks for X-Request-ID header; generates a UUID if absent.
// Injects the ID into context and echoes it in the response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Auth validates the Bearer JWT and injects Claims into context.
func Auth(a *auth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hdr := r.Header.Get("Authorization")
			if !strings.HasPrefix(hdr, "Bearer ") {
				http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
				return
			}
			claims, err := a.Validate(strings.TrimPrefix(hdr, "Bearer "))
			if err != nil {
				http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole short-circuits with 403 if the authenticated user lacks the required role.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok || claims.Role != role {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit wraps the imported 04-rate-limiter middleware.
func RateLimit(l ratelimit.Limiter) func(http.Handler) http.Handler {
	return ratelimit.Middleware(l)
}
