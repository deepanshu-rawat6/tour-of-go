package core

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestDedup_FirstEventForwarded(t *testing.T) {
	var forwarded []Event
	var mu sync.Mutex
	d := NewDedupEngine(100*time.Millisecond, func(_ context.Context, e Event) {
		mu.Lock()
		forwarded = append(forwarded, e)
		mu.Unlock()
	})

	e := Event{Namespace: "default", Pod: "pod-1", Reason: "OOMKilled", Message: "OOM", Count: 1, LastSeen: time.Now()}
	d.Process(context.Background(), e)

	mu.Lock()
	n := len(forwarded)
	mu.Unlock()
	if n != 1 {
		t.Errorf("expected 1 forwarded event, got %d", n)
	}
}

func TestDedup_SubsequentSuppressed(t *testing.T) {
	var forwarded []Event
	var mu sync.Mutex
	d := NewDedupEngine(200*time.Millisecond, func(_ context.Context, e Event) {
		mu.Lock()
		forwarded = append(forwarded, e)
		mu.Unlock()
	})

	e := Event{Namespace: "default", Pod: "pod-1", Reason: "CrashLoopBackOff", Message: "crash", Count: 1, LastSeen: time.Now()}
	d.Process(context.Background(), e)
	d.Process(context.Background(), e)
	d.Process(context.Background(), e)

	mu.Lock()
	n := len(forwarded)
	mu.Unlock()
	// Only first should be forwarded immediately
	if n != 1 {
		t.Errorf("expected 1 forwarded (subsequent suppressed), got %d", n)
	}
}

func TestDedup_SummaryOnWindowExpiry(t *testing.T) {
	var forwarded []Event
	var mu sync.Mutex
	d := NewDedupEngine(50*time.Millisecond, func(_ context.Context, e Event) {
		mu.Lock()
		forwarded = append(forwarded, e)
		mu.Unlock()
	})

	e := Event{Namespace: "default", Pod: "pod-1", Reason: "OOMKilled", Message: "OOM", Count: 1, LastSeen: time.Now()}
	d.Process(context.Background(), e)
	d.Process(context.Background(), e)
	d.Process(context.Background(), e)

	// Wait for window to expire
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	n := len(forwarded)
	last := forwarded[len(forwarded)-1]
	mu.Unlock()

	// Should have: 1 immediate + 1 summary
	if n != 2 {
		t.Errorf("expected 2 events (immediate + summary), got %d", n)
	}
	if last.Count != 3 {
		t.Errorf("expected summary count=3, got %d", last.Count)
	}
}

func TestDedup_NoSummaryIfOnlyOne(t *testing.T) {
	var forwarded []Event
	var mu sync.Mutex
	d := NewDedupEngine(50*time.Millisecond, func(_ context.Context, e Event) {
		mu.Lock()
		forwarded = append(forwarded, e)
		mu.Unlock()
	})

	e := Event{Namespace: "default", Pod: "pod-1", Reason: "OOMKilled", Message: "OOM", Count: 1, LastSeen: time.Now()}
	d.Process(context.Background(), e)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	n := len(forwarded)
	mu.Unlock()
	// Only the immediate forward — no summary since count == 1
	if n != 1 {
		t.Errorf("expected 1 event (no summary for single occurrence), got %d", n)
	}
}

func TestDedup_DifferentKeysIndependent(t *testing.T) {
	var forwarded []Event
	var mu sync.Mutex
	d := NewDedupEngine(200*time.Millisecond, func(_ context.Context, e Event) {
		mu.Lock()
		forwarded = append(forwarded, e)
		mu.Unlock()
	})

	e1 := Event{Namespace: "default", Pod: "pod-1", Reason: "OOMKilled", Message: "OOM", Count: 1, LastSeen: time.Now()}
	e2 := Event{Namespace: "default", Pod: "pod-2", Reason: "OOMKilled", Message: "OOM", Count: 1, LastSeen: time.Now()}
	d.Process(context.Background(), e1)
	d.Process(context.Background(), e2)

	mu.Lock()
	n := len(forwarded)
	mu.Unlock()
	if n != 2 {
		t.Errorf("expected 2 forwarded (different keys), got %d", n)
	}
}
