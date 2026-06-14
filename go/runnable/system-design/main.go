package main

import (
	"fmt"
	"sync"
	"time"
)

// ── Token Bucket ──────────────────────────────────────────────────────────────

type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	rate       float64 // tokens added per second
	burst      float64 // max tokens
	lastRefill time.Time
}

func NewTokenBucket(rate, burst float64) *TokenBucket {
	return &TokenBucket{tokens: burst, rate: rate, burst: burst, lastRefill: time.Now()}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens = min(tb.burst, tb.tokens+elapsed*tb.rate)
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
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

// ── Sliding Window Log ────────────────────────────────────────────────────────

type SlidingWindow struct {
	mu     sync.Mutex
	logs   []time.Time
	limit  int
	window time.Duration
}

func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
	return &SlidingWindow{limit: limit, window: window}
}

func (sw *SlidingWindow) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	boundary := now.Add(-sw.window)

	// evict old entries
	filtered := sw.logs[:0]
	for _, t := range sw.logs {
		if t.After(boundary) {
			filtered = append(filtered, t)
		}
	}
	sw.logs = filtered

	if len(sw.logs) >= sw.limit {
		return false
	}
	sw.logs = append(sw.logs, now)
	return true
}

func simulate(name string, allowFn func() bool, requests int, interval time.Duration) {
	fmt.Printf("\n%s (sending %d requests, %s apart):\n", name, requests, interval)
	allowed, rejected := 0, 0
	for i := 1; i <= requests; i++ {
		if allowFn() {
			allowed++
			fmt.Printf("  req %2d: ✓ allowed\n", i)
		} else {
			rejected++
			fmt.Printf("  req %2d: ✗ rejected\n", i)
		}
		time.Sleep(interval)
	}
	fmt.Printf("  → allowed=%d rejected=%d\n", allowed, rejected)
}

func main() {
	fmt.Println("=== Rate Limiter Simulation ===")

	// Token Bucket: 5 tokens/sec, burst of 3
	tb := NewTokenBucket(5, 3)
	simulate("Token Bucket (5 req/s, burst=3)", tb.Allow, 8, 50*time.Millisecond)

	// Sliding Window: 4 requests per 200ms window
	sw := NewSlidingWindow(4, 200*time.Millisecond)
	simulate("Sliding Window (4 req per 200ms)", sw.Allow, 8, 30*time.Millisecond)
}
