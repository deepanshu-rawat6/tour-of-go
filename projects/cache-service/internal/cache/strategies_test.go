package cache_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"tour_of_go/projects/cache-service/internal/cache"
	"tour_of_go/projects/cache-service/internal/store"
)

var ctx = context.Background()

func TestCacheAside_MissPopulatesCache(t *testing.T) {
	mem := store.NewMemory()
	mem.Set(ctx, "k", "v")
	lru := cache.NewLRU(10)
	defer lru.Close()
	ca := cache.NewCacheAside(lru, mem, time.Minute)

	// First call: cache miss → fetches from store
	v, err := ca.Get(ctx, "k")
	if err != nil || v != "v" {
		t.Fatalf("want v, got %q err=%v", v, err)
	}
	// Second call: cache hit → store not called again
	callsBefore := mem.Calls
	ca.Get(ctx, "k")
	if mem.Calls != callsBefore {
		t.Fatal("expected cache hit on second call, but store was called again")
	}
}

func TestCacheAside_MissOnUnknownKey(t *testing.T) {
	lru := cache.NewLRU(10)
	defer lru.Close()
	ca := cache.NewCacheAside(lru, store.NewMemory(), time.Minute)
	_, err := ca.Get(ctx, "missing")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestWriteThrough_WriteAndRead(t *testing.T) {
	mem := store.NewMemory()
	lru := cache.NewLRU(10)
	defer lru.Close()
	wt := cache.NewWriteThrough(lru, mem, time.Minute)

	if err := wt.Set(ctx, "k", "v"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	// Read from cache (no store call)
	v, err := wt.Get(ctx, "k")
	if err != nil || v != "v" {
		t.Fatalf("want v, got %q err=%v", v, err)
	}
	if mem.Calls != 0 {
		t.Fatal("expected zero store reads — write-through should populate cache")
	}
}

func TestSingleflightCache_DeduplicatesConcurrentMisses(t *testing.T) {
	mem := store.NewMemory()
	mem.Set(ctx, "k", "expensive")
	lru := cache.NewLRU(10)
	defer lru.Close()
	ca := cache.NewCacheAside(lru, mem, time.Minute)
	sf := cache.NewSingleflightCache(ca)

	// Reset call counter after the Set above
	mem.Calls = 0

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sf.Get(ctx, "k")
		}()
	}
	wg.Wait()

	// Singleflight ensures the store is called at most a handful of times
	// (ideally 1, but goroutine scheduling may allow a few before the first completes)
	if mem.Calls > 3 {
		t.Fatalf("expected singleflight to deduplicate; store called %d times", mem.Calls)
	}
}
