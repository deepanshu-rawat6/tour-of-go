package sync_deep_dive

// errgroup.go — building errgroup semantics from stdlib primitives
//
// golang.org/x/sync/errgroup is the production-grade solution for fan-out with
// error collection. We deliberately avoid that external dependency here so this
// package stays stdlib-only. Everything built below mirrors its behaviour.
//
// Production usage (reference — not compiled here):
//
//   import "golang.org/x/sync/errgroup"
//
//   g, ctx := errgroup.WithContext(context.Background())
//   for _, url := range urls {
//       url := url // capture loop variable (pre-Go 1.22)
//       g.Go(func() error {
//           return fetch(ctx, url)
//       })
//   }
//   if err := g.Wait(); err != nil {
//       log.Fatal(err)
//   }
//
// Our stdlib implementation below reproduces the same API surface.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// -----------------------------------------------------------------------------
// stdErrGroup — WaitGroup + error channel, context-aware
// -----------------------------------------------------------------------------

// stdErrGroup fans out goroutines and collects the FIRST error.
// All other goroutines continue running to completion (or until they check ctx).
// This matches golang.org/x/sync/errgroup's default behaviour.
type stdErrGroup struct {
	wg      sync.WaitGroup
	errOnce sync.Once    // ensure only the first error is stored
	err     error        // first non-nil error
	ctx     context.Context
	cancel  context.CancelFunc
}

// newErrGroup creates an errgroup with a cancellable context.
// When the first goroutine returns an error, the context is cancelled,
// signalling all other goroutines to stop gracefully.
func newErrGroup(parent context.Context) *stdErrGroup {
	ctx, cancel := context.WithCancel(parent)
	return &stdErrGroup{ctx: ctx, cancel: cancel}
}

// Go starts a goroutine running fn. If fn returns a non-nil error, it is
// recorded (first error wins) and the group's context is cancelled.
func (g *stdErrGroup) Go(fn func() error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := fn(); err != nil {
			// errOnce ensures only the first error is stored even if multiple
			// goroutines return errors concurrently.
			g.errOnce.Do(func() {
				g.err = err
				// Cancel context so other workers can detect cancellation via
				// ctx.Done() and return early.
				if g.cancel != nil {
					g.cancel()
				}
			})
		}
	}()
}

// Wait blocks until all goroutines launched by Go have completed.
// It returns the first non-nil error (or nil if all succeeded).
// The context cancel function is called when Wait returns.
func (g *stdErrGroup) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel() // release context resources
	}
	return g.err
}

// -----------------------------------------------------------------------------
// Simulated "HTTP fetch" workers
// -----------------------------------------------------------------------------

// fetchResult simulates what a parallel HTTP call might return.
type fetchResult struct {
	url    string
	status int
}

// simulateFetch pretends to HTTP GET a URL. It respects context cancellation.
// A negative delay parameter causes the fetch to return an error (for demo).
func simulateFetch(ctx context.Context, url string, delayMs int) (fetchResult, error) {
	delay := time.Duration(delayMs) * time.Millisecond

	select {
	case <-ctx.Done():
		// Context was cancelled (probably by another goroutine's error).
		return fetchResult{}, fmt.Errorf("fetch %s: %w", url, ctx.Err())
	case <-time.After(delay):
		// Simulate an error for one of the URLs.
		if delayMs < 0 {
			return fetchResult{}, fmt.Errorf("fetch %s: HTTP 500", url)
		}
		return fetchResult{url: url, status: 200}, nil
	}
}

// -----------------------------------------------------------------------------
// errgroupExample
// -----------------------------------------------------------------------------

func errgroupExample() {
	fmt.Println("--- errgroup (stdlib: WaitGroup + error channel) ---")

	// -------------------------------------------------------------------------
	// Demo 1: All succeed — collect results concurrently.
	// -------------------------------------------------------------------------
	fmt.Println("  [errgroup] demo 1: all workers succeed")

	urls := []struct {
		url     string
		delayMs int
	}{
		{"https://api.example.com/users", 30},
		{"https://api.example.com/posts", 20},
		{"https://api.example.com/comments", 40},
		{"https://api.example.com/todos", 10},
	}

	// Use a mutex to protect the results slice since multiple goroutines write.
	var mu sync.Mutex
	results := make([]fetchResult, 0, len(urls))

	g := newErrGroup(context.Background())
	for _, u := range urls {
		u := u // capture — unnecessary in Go 1.22+ but shown for compatibility
		g.Go(func() error {
			res, err := simulateFetch(g.ctx, u.url, u.delayMs)
			if err != nil {
				return err
			}
			mu.Lock()
			results = append(results, res)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Printf("  [errgroup] unexpected error: %v\n", err)
	} else {
		fmt.Printf("  [errgroup] all %d fetches succeeded\n", len(results))
		for _, r := range results {
			fmt.Printf("    %s → %d\n", r.url, r.status)
		}
	}

	// -------------------------------------------------------------------------
	// Demo 2: One worker fails — context cancels remaining workers.
	// -------------------------------------------------------------------------
	fmt.Println()
	fmt.Println("  [errgroup] demo 2: one worker fails → context cancels others")

	type job struct {
		url     string
		delayMs int
	}
	jobs := []job{
		{"https://svc/a", 20},
		{"https://svc/b", -1},  // will fail immediately
		{"https://svc/c", 50},  // will be cancelled before finishing
		{"https://svc/d", 100}, // will be cancelled before finishing
	}

	var cancelled atomic.Int64
	g2 := newErrGroup(context.Background())

	for _, j := range jobs {
		j := j
		g2.Go(func() error {
			res, err := simulateFetch(g2.ctx, j.url, j.delayMs)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					cancelled.Add(1)
					return err // propagate so Wait() returns it (first only stored)
				}
				fmt.Printf("    worker %s FAILED: %v\n", j.url, err)
				return err
			}
			fmt.Printf("    worker %s OK: %d\n", res.url, res.status)
			return nil
		})
	}

	err := g2.Wait()
	fmt.Printf("  [errgroup] first error: %v\n", err)
	fmt.Printf("  [errgroup] workers cancelled by ctx: %d\n", cancelled.Load())

	// -------------------------------------------------------------------------
	// Aside: golang.org/x/sync/errgroup additions
	// -------------------------------------------------------------------------
	fmt.Println()
	fmt.Println("  [errgroup] golang.org/x/sync/errgroup extras:")
	fmt.Println("    g.SetLimit(n)    — cap concurrent goroutines to n")
	fmt.Println("    g.TryGo(fn)      — launch only if under limit, else drop")
	fmt.Println("    Both useful for bounding parallelism (e.g. max 10 HTTP calls)")
}
