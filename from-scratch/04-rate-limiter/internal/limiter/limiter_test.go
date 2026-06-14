package limiter_test

import (
	"testing"
	"time"

	"tour_of_go/projects/from-scratch/04-rate-limiter/internal/limiter"
)

func countAllowed(l limiter.Limiter, n int) int {
	allowed := 0
	for i := 0; i < n; i++ {
		if l.Allow() {
			allowed++
		}
	}
	return allowed
}

func TestTokenBucket_BurstThenReject(t *testing.T) {
	l := limiter.NewTokenBucket(1, 3) // burst=3, rate=1/s
	got := countAllowed(l, 5)
	if got != 3 {
		t.Fatalf("want 3 (burst), got %d", got)
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	l := limiter.NewTokenBucket(10, 1) // rate=10/s, burst=1
	l.Allow()                          // consume burst
	time.Sleep(150 * time.Millisecond) // wait for ~1.5 tokens
	if !l.Allow() {
		t.Fatal("expected allow after refill")
	}
}

func TestFixedWindow_LimitAndReset(t *testing.T) {
	l := limiter.NewFixedWindow(3, 100*time.Millisecond)
	got := countAllowed(l, 5)
	if got != 3 {
		t.Fatalf("want 3, got %d", got)
	}
	time.Sleep(110 * time.Millisecond)
	if !l.Allow() {
		t.Fatal("expected allow after window reset")
	}
}

func TestSlidingWindow_LimitAndSlide(t *testing.T) {
	l := limiter.NewSlidingWindow(3, 100*time.Millisecond)
	got := countAllowed(l, 5)
	if got != 3 {
		t.Fatalf("want 3, got %d", got)
	}
	time.Sleep(110 * time.Millisecond)
	if !l.Allow() {
		t.Fatal("expected allow after window slides")
	}
}

func TestLeakyBucket_CapacityLimit(t *testing.T) {
	l := limiter.NewLeakyBucket(3, time.Hour) // drain rate=1/hour (effectively never drains in test)
	got := countAllowed(l, 5)
	if got != 3 {
		t.Fatalf("want 3 (capacity), got %d", got)
	}
}
