// Package middleware implements rate limiting and circuit breaking.
package middleware

import (
	"sync"
	"time"
)

// TokenBucket implements per-client-IP rate limiting.
// Ported from more-internals/runnable/system-design/ with per-key support.
type TokenBucket struct {
	mu         sync.Mutex
	buckets    map[string]*bucket
	rate       float64 // tokens per second
	burst      float64
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

func NewTokenBucket(rate, burst float64) *TokenBucket {
	return &TokenBucket{
		buckets: make(map[string]*bucket),
		rate:    rate,
		burst:   burst,
	}
}

// Allow returns true if the client identified by key is within rate limits.
func (tb *TokenBucket) Allow(key string) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	b, ok := tb.buckets[key]
	if !ok {
		b = &bucket{tokens: tb.burst, lastRefill: time.Now()}
		tb.buckets[key] = b
	}

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens = min(tb.burst, b.tokens+elapsed*tb.rate)
	b.lastRefill = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
