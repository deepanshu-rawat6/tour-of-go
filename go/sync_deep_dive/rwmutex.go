package sync_deep_dive

// rwmutex.go — sync.RWMutex: multiple concurrent readers, exclusive writer
//
// When to prefer RWMutex over Mutex:
//   • Read operations dominate (reads >> writes).
//   • Read path is NOT trivially fast (e.g. involves I/O or a heavy lookup).
//
// Benchmark context (approximate, hardware-dependent):
//   • sync.Mutex:   ~20 ns per Lock/Unlock pair (one goroutine, no contention)
//   • sync.RWMutex: ~25 ns per RLock/RUnlock pair
//   But with 8 readers and 1 writer concurrently:
//   • Mutex   throughput: ~1x
//   • RWMutex throughput: ~7x  (reads parallelise)
//
// sync.Map vs sync.RWMutex:
//   sync.Map is optimised for two specific access patterns:
//     1. Keys are written once, then read many times (stable key set).
//     2. Multiple goroutines read/write disjoint key sets (no key contention).
//   For general-purpose maps with mixed access, a map + RWMutex is often
//   simpler and comparably fast. sync.Map avoids per-lock overhead but has
//   more complex internals (dirty/read split).
//
// PITFALL — RLock does NOT nest safely:
//   If goroutine G holds RLock and calls RLock again:
//     • On its own, a second RLock from the SAME goroutine usually succeeds
//       because RWMutex counts readers.
//     • BUT if a writer is waiting between the two RLocks, the second RLock
//       blocks (writers get priority over new readers). Since G also holds
//       the first RLock which blocks the writer from proceeding, we have
//       a deadlock: G waits for the writer, writer waits for G.
//   Rule: never hold RLock while calling any function that might take RLock
//   or Lock on the same mutex.

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// -----------------------------------------------------------------------------
// Read-heavy cache
// -----------------------------------------------------------------------------

// safeCache is a string→string cache backed by RWMutex.
// Multiple goroutines can read concurrently; writes are exclusive.
type safeCache struct {
	mu    sync.RWMutex
	store map[string]string
	// hits/miss use atomic operations so they can be incremented under RLock
	// without a data race. Never use plain ints here — incrementing under RLock
	// allows concurrent writes to the counter from multiple reader goroutines.
	hits int64
	miss int64
}

func newSafeCache() *safeCache {
	return &safeCache{store: make(map[string]string)}
}

// Get reads a value. Multiple goroutines can be in Get() simultaneously
// because they all call RLock, which allows concurrent reads.
func (c *safeCache) Get(key string) (string, bool) {
	// RLock: acquire read lock — blocks only if a writer holds the write lock.
	c.mu.RLock()
	defer c.mu.RUnlock() // RUnlock must balance every RLock

	v, ok := c.store[key]
	// Use atomic increment because multiple goroutines concurrently hold RLock
	// and would race on plain integer increments. atomic.AddInt64 is safe here.
	if ok {
		atomic.AddInt64(&c.hits, 1)
	} else {
		atomic.AddInt64(&c.miss, 1)
	}
	return v, ok
}

// Set writes a value. Exclusive — no reads or other writes can proceed.
func (c *safeCache) Set(key, value string) {
	// Lock: acquire write lock — blocks until ALL readers and writers finish.
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store[key] = value
}

// Delete removes a key under write lock.
func (c *safeCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, key)
}

// Snapshot returns a copy of the entire map under read lock.
// Returning a copy avoids holding the lock while the caller iterates —
// which would prevent any concurrent writes for the entire iteration.
func (c *safeCache) Snapshot() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make(map[string]string, len(c.store))
	for k, v := range c.store {
		out[k] = v
	}
	return out
}

// -----------------------------------------------------------------------------
// rwmutexExample
// -----------------------------------------------------------------------------

func rwmutexExample() {
	fmt.Println("--- sync.RWMutex ---")

	cache := newSafeCache()

	// Seed some data.
	cache.Set("user:1", "Alice")
	cache.Set("user:2", "Bob")
	cache.Set("user:3", "Carol")

	var wg sync.WaitGroup

	// Launch 10 concurrent readers.
	// All can proceed in parallel because they only take RLock.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("user:%d", (id%3)+1)
			if v, ok := cache.Get(key); ok {
				fmt.Printf("  [rwmutex] reader %2d got %s=%s\n", id, key, v)
			}
		}(i)
	}

	// Launch 2 concurrent writers.
	// Each writer blocks ALL readers and other writers for its critical section.
	for i := 4; i <= 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("user:%d", id)
			val := fmt.Sprintf("User%d", id)
			cache.Set(key, val)
			fmt.Printf("  [rwmutex] writer set %s=%s\n", key, val)
		}(i)
	}

	wg.Wait()

	snap := cache.Snapshot()
	fmt.Printf("  [rwmutex] snapshot has %d keys\n", len(snap))

	// --- Pitfall: nested RLock deadlock scenario (explained, not run) ---
	fmt.Println()
	fmt.Println("  [rwmutex] PITFALL — nested RLock deadlock:")
	fmt.Println("    var mu sync.RWMutex")
	fmt.Println("    mu.RLock()              // G holds read lock")
	fmt.Println("    go func() { mu.Lock() } // writer queues, blocks")
	fmt.Println("    mu.RLock()              // G tries second RLock ← DEADLOCK")
	fmt.Println("    // writer is waiting for G's first RLock to release,")
	fmt.Println("    // but new readers are blocked behind the queued writer.")
	fmt.Println("    // G is waiting for its second RLock → circular wait.")

	// --- Demonstrate that Lock() blocks while readers hold RLock ---
	fmt.Println()
	var mu2 sync.RWMutex
	done := make(chan struct{})

	mu2.RLock()
	fmt.Println("  [rwmutex] RLock held by main goroutine")

	go func() {
		// This will block until we release the RLock below.
		mu2.Lock()
		fmt.Println("  [rwmutex] writer acquired exclusive Lock")
		time.Sleep(5 * time.Millisecond)
		mu2.Unlock()
		close(done)
	}()

	time.Sleep(20 * time.Millisecond) // let writer goroutine start and queue
	fmt.Println("  [rwmutex] releasing RLock — writer should unblock")
	mu2.RUnlock()

	<-done
	fmt.Println("  [rwmutex] writer finished")
}
