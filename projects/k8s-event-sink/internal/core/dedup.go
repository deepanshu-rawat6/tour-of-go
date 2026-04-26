package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// bucket tracks a single dedup window for one event key.
type bucket struct {
	first   Event
	count   int
	timer   *time.Timer
}

// DedupEngine implements leaky-bucket deduplication.
// First occurrence of each key is forwarded immediately.
// Subsequent occurrences within the window are suppressed.
// When the window expires, a summary event is emitted if count > 1.
type DedupEngine struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	window   time.Duration
	onEvent  func(ctx context.Context, event Event) // called for forwarded + summary events
}

func NewDedupEngine(window time.Duration, onEvent func(ctx context.Context, event Event)) *DedupEngine {
	return &DedupEngine{
		buckets: make(map[string]*bucket),
		window:  window,
		onEvent: onEvent,
	}
}

// Process handles an incoming filtered event.
func (d *DedupEngine) Process(ctx context.Context, event Event) {
	key := bucketKey(event)

	d.mu.Lock()
	b, exists := d.buckets[key]
	if !exists {
		// First occurrence — forward immediately and start window timer.
		b = &bucket{first: event, count: 1}
		b.timer = time.AfterFunc(d.window, func() {
			d.expire(ctx, key)
		})
		d.buckets[key] = b
		d.mu.Unlock()
		d.onEvent(ctx, event)
		return
	}
	// Subsequent occurrence — suppress, increment counter.
	b.count++
	b.first.LastSeen = event.LastSeen
	d.mu.Unlock()
}

// expire is called when a bucket's window timer fires.
func (d *DedupEngine) expire(ctx context.Context, key string) {
	d.mu.Lock()
	b, ok := d.buckets[key]
	if !ok {
		d.mu.Unlock()
		return
	}
	count := b.count
	summary := b.first
	delete(d.buckets, key)
	d.mu.Unlock()

	if count > 1 {
		summary.Count = count
		summary.Message = fmt.Sprintf("%s (suppressed %d similar events in window)", summary.Message, count-1)
		d.onEvent(ctx, summary)
	}
}

// Flush emits pending summaries for all open buckets (called on shutdown).
func (d *DedupEngine) Flush(ctx context.Context) {
	d.mu.Lock()
	keys := make([]string, 0, len(d.buckets))
	for k := range d.buckets {
		keys = append(keys, k)
	}
	d.mu.Unlock()
	for _, k := range keys {
		d.expire(ctx, k)
	}
}

func bucketKey(e Event) string {
	return fmt.Sprintf("%s:%s:%s", e.Namespace, e.Pod, e.Reason)
}
