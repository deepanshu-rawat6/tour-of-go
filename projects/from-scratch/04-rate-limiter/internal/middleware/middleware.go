// Package middleware provides HTTP rate limiting middleware.
package middleware

import (
	"net/http"
	"tour_of_go/projects/from-scratch/04-rate-limiter/internal/limiter"
)

// RateLimit returns middleware that rejects requests with 429 when the limiter denies.
func RateLimit(l limiter.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.Allow() {
				http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
