// Package middleware provides HTTP middleware following the Open/Closed Principle:
// the middleware chain is a []func(http.Handler) http.Handler — new middleware
// can be added without modifying existing handlers.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"tour_of_go/projects/secure-api/internal/domain"
	"tour_of_go/projects/secure-api/internal/ports"
)

type contextKey string

const claimsKey contextKey = "claims"

// ClaimsFromContext retrieves Claims injected by the Auth middleware.
func ClaimsFromContext(ctx context.Context) (domain.Claims, bool) {
	c, ok := ctx.Value(claimsKey).(domain.Claims)
	return c, ok
}

// Auth returns middleware that validates a Bearer JWT and injects Claims into the context.
// Depends on the ports.TokenValidator abstraction (Dependency Inversion Principle).
func Auth(v ports.TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hdr := r.Header.Get("Authorization")
			if !strings.HasPrefix(hdr, "Bearer ") {
				http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
				return
			}
			raw := strings.TrimPrefix(hdr, "Bearer ")
			claims, err := v.Validate(raw)
			if err != nil {
				http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Chain applies a slice of middleware in order (left-to-right).
func Chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
