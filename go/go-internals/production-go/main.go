package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

// --- Atomic counter demo ---

var requestCount atomic.Int64

func helloHandler(w http.ResponseWriter, r *http.Request) {
	count := requestCount.Add(1)
	fmt.Fprintf(w, "request #%d\n", count)
}

// --- Atomic pointer config reload ---

type Config struct {
	RateLimit int
	Timeout   time.Duration
}

var currentConfig atomic.Pointer[Config]

func init() {
	currentConfig.Store(&Config{RateLimit: 100, Timeout: 5 * time.Second})
}

// --- Lock-free max tracker (CAS) ---

var maxLatency atomic.Int64

func recordLatency(ms int64) {
	for {
		old := maxLatency.Load()
		if ms <= old {
			return
		}
		if maxLatency.CompareAndSwap(old, ms) {
			return
		}
	}
}

func main() {
	// Demo: atomic operations
	fmt.Println("=== Production Go Patterns ===")
	fmt.Println("\n--- Atomic Counter ---")
	for i := 0; i < 5; i++ {
		requestCount.Add(1)
	}
	fmt.Printf("Request count: %d\n", requestCount.Load())

	fmt.Println("\n--- Atomic Pointer (Config Reload) ---")
	cfg := currentConfig.Load()
	fmt.Printf("Current config: RateLimit=%d, Timeout=%s\n", cfg.RateLimit, cfg.Timeout)
	currentConfig.Store(&Config{RateLimit: 200, Timeout: 10 * time.Second})
	cfg = currentConfig.Load()
	fmt.Printf("After reload:   RateLimit=%d, Timeout=%s\n", cfg.RateLimit, cfg.Timeout)

	fmt.Println("\n--- CAS Max Tracker ---")
	for _, lat := range []int64{10, 50, 30, 80, 20, 90, 45} {
		recordLatency(lat)
	}
	fmt.Printf("Max latency observed: %dms\n", maxLatency.Load())

	// Demo: graceful shutdown with HTTP server
	fmt.Println("\n--- Graceful Shutdown Demo ---")
	fmt.Println("Starting server on :8080 (pprof on /debug/pprof/)")
	fmt.Println("Press Ctrl+C to trigger graceful shutdown")

	mux := http.NewServeMux()
	mux.HandleFunc("/", helloHandler)
	mux.Handle("/debug/pprof/", http.DefaultServeMux)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		<-gCtx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		fmt.Println("\nShutting down HTTP server...")
		return srv.Shutdown(shutCtx)
	})

	g.Go(func() error {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Println("Clean shutdown complete.")
}
