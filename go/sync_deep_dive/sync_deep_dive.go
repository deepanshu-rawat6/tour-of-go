// Package sync_deep_dive explores the advanced synchronization primitives in the
// Go standard library beyond the basic sync.Mutex covered in the concurrency topic.
//
// Topics covered:
//   - sync.Once        — guaranteed one-time initialization (singletons, config)
//   - sync.Cond        — condition variables for goroutine signaling
//   - sync.RWMutex     — multiple-reader / single-writer locking
//   - sync.Pool        — reduce GC pressure by reusing allocations
//   - errgroup pattern — fan-out with error collection (stdlib only)
package sync_deep_dive

import (
	"fmt"
	"os"
)

// Run executes all sync deep-dive examples in order.
func Run() {
	fmt.Println("=== sync Deep Dive ===")
	fmt.Println()

	onceExample()
	fmt.Println()

	condExample()
	fmt.Println()

	rwmutexExample()
	fmt.Println()

	poolExample()
	fmt.Println()

	errgroupExample()
	fmt.Println()
}

// RunExample dispatches to a single named example.
// Usage: go run . sync_deep_dive once
func RunExample(name string) {
	fmt.Printf("=== sync Deep Dive: %s ===\n\n", name)

	switch name {
	case "once":
		onceExample()
	case "cond":
		condExample()
	case "rwmutex":
		rwmutexExample()
	case "pool":
		poolExample()
	case "errgroup":
		errgroupExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  once      - sync.Once for singleton init, OnceValue, OnceFunc (Go 1.21+)")
		fmt.Println("  cond      - sync.Cond Wait/Signal/Broadcast, producer/consumer queue")
		fmt.Println("  rwmutex   - sync.RWMutex read-heavy cache, pitfalls, vs sync.Map")
		fmt.Println("  pool      - sync.Pool bytes.Buffer pool, GC behaviour, anti-patterns")
		fmt.Println("  errgroup  - WaitGroup + error channel fan-out, context cancellation")
		os.Exit(1)
	}
}
