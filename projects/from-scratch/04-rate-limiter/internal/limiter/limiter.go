// Package limiter provides four rate limiting algorithms as a reusable library.
package limiter

import (
	"sync"
	"time"
)

// Limiter is the common interface for all rate limiting algorithms.
type Limiter interface {
	Allow() bool
}

// ── Token Bucket ──────────────────────────────────────────────────────────────

// TokenBucket allows bursts up to Burst, refilling at Rate tokens/second.
type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	rate       float64 // tokens per second
	burst      float64
	lastRefill time.Time
}

func NewTokenBucket(rate, burst float64) *TokenBucket {
	return &TokenBucket{tokens: burst, rate: rate, burst: burst, lastRefill: time.Now()}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	tb.tokens = min64(tb.burst, tb.tokens+now.Sub(tb.lastRefill).Seconds()*tb.rate)
	tb.lastRefill = now
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// ── Leaky Bucket ──────────────────────────────────────────────────────────────

// LeakyBucket shapes traffic to a constant rate using a buffered channel.
type LeakyBucket struct {
	queue chan struct{}
}

func NewLeakyBucket(capacity int, rate time.Duration) *LeakyBucket {
	lb := &LeakyBucket{queue: make(chan struct{}, capacity)}
	go func() {
		ticker := time.NewTicker(rate)
		defer ticker.Stop()
		for range ticker.C {
			select {
			case <-lb.queue:
			default:
			}
		}
	}()
	return lb
}

func (lb *LeakyBucket) Allow() bool {
	select {
	case lb.queue <- struct{}{}:
		return true
	default:
		return false
	}
}

// ── Fixed Window ──────────────────────────────────────────────────────────────

// FixedWindow resets the counter at the start of each window.
type FixedWindow struct {
	mu     sync.Mutex
	counts map[int64]int
	limit  int
	window time.Duration
}

func NewFixedWindow(limit int, window time.Duration) *FixedWindow {
	return &FixedWindow{counts: make(map[int64]int), limit: limit, window: window}
}

func (fw *FixedWindow) Allow() bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	key := time.Now().UnixNano() / int64(fw.window)
	if fw.counts[key] >= fw.limit {
		return false
	}
	fw.counts[key]++
	return true
}

// ── Sliding Window Log ────────────────────────────────────────────────────────

// SlidingWindow tracks per-request timestamps for exact rate limiting.
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
	n := 0
	for _, t := range sw.logs {
		if t.After(boundary) {
			sw.logs[n] = t
			n++
		}
	}
	sw.logs = sw.logs[:n]
	if len(sw.logs) >= sw.limit {
		return false
	}
	sw.logs = append(sw.logs, now)
	return true
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
