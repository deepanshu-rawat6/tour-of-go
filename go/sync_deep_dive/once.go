package sync_deep_dive

// once.go — sync.Once, sync.OnceValue, sync.OnceFunc
//
// Key insight: sync.Once guarantees a function runs EXACTLY ONCE across any
// number of concurrent callers, even if they all call Do() simultaneously.
//
// Benchmark: Once.Do after first execution is ~1 ns — it's a single atomic
// load. There is zero lock contention on the hot path.

import (
	"fmt"
	"sync"
)

// -----------------------------------------------------------------------------
// 1. Simulated database connection — the classic singleton pattern.
// -----------------------------------------------------------------------------

// dbConnection represents an expensive-to-create resource.
type dbConnection struct {
	dsn string
}

var (
	// once guards the single initialization of globalDB.
	// ANTI-PATTERN: Never copy a sync.Once after first use — copying it after
	// Do() has run resets its internal state in the copy, so the function
	// could run again in the copy. Always keep Once in a pointer or package-
	// level variable.
	once     sync.Once
	globalDB *dbConnection
)

// getDB returns the singleton database connection, initialising it on first
// call. All 100 goroutines that race here will block until the first one
// finishes, then all receive the same *dbConnection.
func getDB() *dbConnection {
	once.Do(func() {
		// This closure runs ONCE, no matter how many goroutines call getDB().
		fmt.Println("  [once] initialising DB connection (runs only once)")
		globalDB = &dbConnection{dsn: "postgres://localhost/mydb"}
	})
	return globalDB
}

// -----------------------------------------------------------------------------
// 2. sync.OnceValue (Go 1.21+) — Once that returns a value.
//    Under the hood it wraps sync.Once with an inner variable; the big
//    advantage is that the return value is part of the API contract.
// -----------------------------------------------------------------------------

// getConfig uses OnceValue to load configuration lazily and safely.
// The returned function can be called any number of times; the initialiser
// runs exactly once and the result is cached.
var getConfig = sync.OnceValue(func() map[string]string {
	fmt.Println("  [OnceValue] loading config (runs only once)")
	return map[string]string{
		"db_host": "localhost",
		"port":    "5432",
	}
})

// -----------------------------------------------------------------------------
// 3. sync.OnceFunc (Go 1.21+) — Once that wraps a void function.
//    Useful when you want to pass the once-function as a value (e.g. to a
//    constructor) rather than bind it to a package-level variable.
// -----------------------------------------------------------------------------

// initMetrics wraps a startup side-effect (registering Prometheus metrics,
// running migrations, etc.) so it happens at most once.
var initMetrics = sync.OnceFunc(func() {
	fmt.Println("  [OnceFunc] registering metrics (runs only once)")
})

// -----------------------------------------------------------------------------
// onceExample ties it all together.
// -----------------------------------------------------------------------------

func onceExample() {
	fmt.Println("--- sync.Once ---")

	// Spin up 100 goroutines that all call getDB() at "the same time".
	// Only ONE should print the initialisation message.
	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			db := getDB()
			// All 100 goroutines receive the same pointer.
			_ = db
		}()
	}
	wg.Wait()
	fmt.Printf("  [once] globalDB pointer: %p\n", globalDB)

	// --- OnceValue ---
	fmt.Println()
	cfg1 := getConfig()
	cfg2 := getConfig() // second call — no re-initialisation
	fmt.Printf("  [OnceValue] cfg1 == cfg2: %v (same map: %p == %p)\n",
		&cfg1 == &cfg2, cfg1, cfg2) // cfg1 == cfg2 values, same underlying data

	// --- OnceFunc ---
	fmt.Println()
	initMetrics() // initialiser runs
	initMetrics() // no-op: already ran
	initMetrics() // no-op again

	// --- Anti-pattern demonstration (copy) ---
	fmt.Println()
	fmt.Println("  [anti-pattern] copying a sync.Once:")
	var original sync.Once
	original.Do(func() { fmt.Println("  original ran") })

	// DO NOT do this — copy after first use is a bug.
	// copiedOnce := original   // copied.done == 1, so Do() is a no-op. Looks
	//                          // correct here, but if you copy BEFORE the first
	//                          // Do(), both will run their functions independently,
	//                          // breaking the guarantee. The race detector won't
	//                          // always catch it. Always pass *sync.Once.
	fmt.Println("  (copy example skipped — see comment in source for the bug)")
}
