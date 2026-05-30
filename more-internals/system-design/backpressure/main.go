package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

var (
	inflight atomic.Int64
	maxLoad  int64 = 10
	shed     atomic.Int64
	served   atomic.Int64
)

func loadSheddingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := inflight.Add(1)
		defer inflight.Add(-1)

		if current > maxLoad {
			shed.Add(1)
			w.Header().Set("Retry-After", "1")
			http.Error(w, "overloaded", http.StatusServiceUnavailable)
			return
		}
		served.Add(1)
		next.ServeHTTP(w, r)
	})
}

func workHandler(w http.ResponseWriter, _ *http.Request) {
	time.Sleep(100 * time.Millisecond) // simulate work
	fmt.Fprint(w, "ok")
}

func statsHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintf(w, `{"inflight":%d,"served":%d,"shed":%d,"max":%d}`,
		inflight.Load(), served.Load(), shed.Load(), maxLoad)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/work", workHandler)
	mux.HandleFunc("/stats", statsHandler)

	handler := loadSheddingMiddleware(mux)

	fmt.Println("Backpressure demo on :8080")
	fmt.Println("  GET /work  — simulated work (100ms)")
	fmt.Println("  GET /stats — current stats")
	fmt.Printf("  Max inflight: %d (excess gets 503)\n", maxLoad)
	fmt.Println("\nTest: hey -n 100 -c 20 http://localhost:8080/work")

	http.ListenAndServe(":8080", handler)
}
