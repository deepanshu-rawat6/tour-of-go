// Command server demonstrates all 4 rate limiting algorithms via HTTP.
// Set ALGO=token|leaky|fixed|sliding (default: token)
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"tour_of_go/projects/from-scratch/04-rate-limiter/internal/limiter"
	"tour_of_go/projects/from-scratch/04-rate-limiter/internal/middleware"
)

func main() {
	var l limiter.Limiter
	switch os.Getenv("ALGO") {
	case "leaky":
		l = limiter.NewLeakyBucket(5, 200*time.Millisecond)
		log.Println("leaky bucket (cap=5, drain=200ms)")
	case "fixed":
		l = limiter.NewFixedWindow(5, time.Second)
		log.Println("fixed window (5 req/s)")
	case "sliding":
		l = limiter.NewSlidingWindow(5, time.Second)
		log.Println("sliding window (5 req/s)")
	default:
		l = limiter.NewTokenBucket(5, 5)
		log.Println("token bucket (5 req/s, burst=5)")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok\n"))
	})

	log.Println("rate-limiter demo on :8083 — try: for i in $(seq 10); do curl -s localhost:8083/; done")
	log.Fatal(http.ListenAndServe(":8083", middleware.RateLimit(l)(mux)))
}
