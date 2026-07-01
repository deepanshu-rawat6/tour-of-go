package sync_deep_dive

// pool.go — sync.Pool: reduce GC pressure by reusing short-lived allocations
//
// sync.Pool is a set of temporary objects that may be saved and reused to
// avoid unnecessary allocations and the resulting GC pressure.
//
// Key properties:
//   • Objects in the pool MAY be GC'd at any time (between GC cycles).
//     The pool is NOT a cache — never store state you can't recreate cheaply.
//   • Any object from Get() should be treated as freshly initialised;
//     it may be a brand-new object OR a recycled one from a previous Put.
//   • Pool is safe for concurrent use.
//
// Who uses sync.Pool in the stdlib?
//   • fmt          — reuses the pp (print parser) struct to format strings.
//   • encoding/json — reuses encode/decode state structs.
//   • net/http      — reuses bufio.Reader/Writer wrapping TCP connections.
//
// ANTI-PATTERN — do NOT use sync.Pool for:
//   • Database/HTTP connection pools — connections have lifecycle (open/close),
//     errors, and idle timeouts. Use database/sql (built-in pool) or a
//     dedicated library. If the GC collects a pool object, you lose the
//     connection permanently without draining it properly.
//   • Objects that require cleanup finalizers — the GC can collect them
//     without running any cleanup code.

import (
	"bytes"
	"fmt"
	"sync"
)

// -----------------------------------------------------------------------------
// bytes.Buffer pool — the canonical sync.Pool use case
//
// Problem: fmt.Sprintf, JSON encoding, HTTP body building, etc. all require
// a temporary []byte buffer that lives for a single request then gets GC'd.
// Under high QPS this creates millions of short-lived allocations per second,
// pressuring the GC and causing latency spikes ("GC storms").
//
// Solution: pool bytes.Buffer objects, Reset them before each use.
// -----------------------------------------------------------------------------

// bufPool is a package-level pool of *bytes.Buffer objects.
// The New function is called by Get() when the pool is empty — it provides
// the zero value for a newly allocated object.
var bufPool = sync.Pool{
	New: func() any {
		// Allocate a buffer with a reasonable initial capacity to avoid
		// small initial growth allocations on first use.
		return bytes.NewBuffer(make([]byte, 0, 256))
	},
}

// formatJSON is a toy function that uses the pool to build a JSON string
// without allocating a fresh buffer every call.
func formatJSON(key, value string) string {
	// GET: retrieve a buffer from the pool (or allocate a new one via New).
	// The type assertion is required because Pool stores interface{} (any).
	buf := bufPool.Get().(*bytes.Buffer)

	// Reset is MANDATORY before use — the buffer may contain data from a
	// previous call. We can't trust its contents.
	buf.Reset()

	fmt.Fprintf(buf, `{%q:%q}`, key, value)
	result := buf.String()

	// PUT: return the buffer to the pool for reuse.
	// After this call, we must NOT read or write buf — ownership transfers
	// back to the pool, and another goroutine may Get() it at any moment.
	bufPool.Put(buf)

	return result
}

// -----------------------------------------------------------------------------
// Demonstrating GC behaviour
// -----------------------------------------------------------------------------

// objectWithID tracks allocation count so we can see pool reuse.
type objectWithID struct {
	id   int
	data [128]byte // non-trivial size so we'd actually care about reuse
}

var (
	allocCount int
	objPool    = sync.Pool{
		New: func() any {
			allocCount++ // in real code use atomic.AddInt64
			return &objectWithID{id: allocCount}
		},
	}
)

// -----------------------------------------------------------------------------
// poolExample
// -----------------------------------------------------------------------------

func poolExample() {
	fmt.Println("--- sync.Pool ---")

	// --- bytes.Buffer pool ---
	fmt.Println("  [pool] bytes.Buffer pool:")
	results := make([]string, 5)
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			results[n] = formatJSON(fmt.Sprintf("key%d", n), fmt.Sprintf("val%d", n))
		}(i)
	}
	wg.Wait()

	for _, r := range results {
		fmt.Printf("    %s\n", r)
	}

	// --- Get/Put and GC eviction ---
	fmt.Println()
	fmt.Println("  [pool] object pool with GC eviction demonstration:")

	// First Get — pool is empty, so New() is called.
	obj1 := objPool.Get().(*objectWithID)
	fmt.Printf("    got obj id=%d (allocCount=%d)\n", obj1.id, allocCount)

	// Return to pool.
	objPool.Put(obj1)

	// Second Get — pool should return obj1 (same ID).
	obj2 := objPool.Get().(*objectWithID)
	fmt.Printf("    got obj id=%d (reused=%v)\n", obj2.id, obj2.id == obj1.id)
	objPool.Put(obj2)

	// Note: if runtime.GC() ran between the two Gets, obj1 would have been
	// evicted and a NEW object (id=2) would be returned. This is by design —
	// the pool is a performance hint to the GC, not a hard cache.
	//
	// In production, you can call runtime.GC() in tests to verify that your
	// code never depends on pool objects surviving across GC boundaries.

	// --- Anti-pattern: don't pool connections ---
	fmt.Println()
	fmt.Println("  [pool] ANTI-PATTERN: connection pooling with sync.Pool")
	fmt.Println("    Don't do: connPool.Put(conn) // conn may be GC'd without Close()")
	fmt.Println("    Do:       use database/sql's built-in pool, or pgxpool, or")
	fmt.Println("              any library that properly tracks connection lifecycle.")

	// --- sizeof comment ---
	fmt.Println()
	fmt.Printf("  [pool] objectWithID size: %d bytes (worth pooling at high QPS)\n",
		func() int {
			var o objectWithID
			// unsafe.Sizeof would give the exact answer; here we use a field count proxy.
			_ = o
			return 128 + 8 // data + id (approximate, depends on alignment)
		}(),
	)
}
