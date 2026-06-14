// Package ratelimit exposes the rate limiting algorithms for use by other modules.
package ratelimit

import (
	"net/http"
	"time"

	"tour_of_go/projects/from-scratch/04-rate-limiter/internal/limiter"
	"tour_of_go/projects/from-scratch/04-rate-limiter/internal/middleware"
)

// Limiter is the rate limiting interface.
type Limiter = limiter.Limiter

// NewTokenBucket creates a token bucket limiter (rate tokens/sec, burst capacity).
func NewTokenBucket(rate, burst float64) Limiter {
	return limiter.NewTokenBucket(rate, burst)
}

// NewSlidingWindow creates a sliding window limiter.
func NewSlidingWindow(limit int, window time.Duration) Limiter {
	return limiter.NewSlidingWindow(limit, window)
}

// Middleware returns HTTP rate limiting middleware.
func Middleware(l Limiter) func(http.Handler) http.Handler {
	return middleware.RateLimit(l)
}
