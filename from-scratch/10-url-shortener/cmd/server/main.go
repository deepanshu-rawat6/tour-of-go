// Command server starts the URL shortener on :8087.
// Integrates:
//   - 04-rate-limiter/ratelimit: token bucket middleware (10 req/s, burst 10)
//   - 07-distributed-cache: RESP cache client (CACHE_ADDR, default :6380)
//   - 06-message-queue: MQ client for analytics events (MQ_ADDR, default :9001)
//   - 09-task-scheduler/taskscheduler: scheduled analytics flush
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"tour_of_go/projects/from-scratch/04-rate-limiter/ratelimit"
	"tour_of_go/projects/from-scratch/09-task-scheduler/taskscheduler"
	"tour_of_go/projects/from-scratch/10-url-shortener/internal/cache"
	"tour_of_go/projects/from-scratch/10-url-shortener/internal/handler"
	"tour_of_go/projects/from-scratch/10-url-shortener/internal/mq"
)

func main() {
	cacheAddr := envOr("CACHE_ADDR", ":6380")
	mqAddr := envOr("MQ_ADDR", ":9001")

	// ── Cache client (07-distributed-cache RESP server) ───────────────────────
	cacheClient, err := cache.Dial(cacheAddr)
	if err != nil {
		log.Printf("cache unavailable at %s — using in-memory fallback", cacheAddr)
	} else {
		log.Printf("connected to cache at %s", cacheAddr)
	}

	// ── MQ client (06-message-queue TCP server) ───────────────────────────────
	mqClient, err2 := mq.Dial(mqAddr)
	if err2 != nil {
		log.Printf("MQ unavailable at %s — analytics disabled", mqAddr)
		mqClient = nil
	} else {
		log.Printf("connected to MQ at %s", mqAddr)
	}

	// ── Task scheduler (09-task-scheduler) ───────────────────────────────────
	sched := taskscheduler.New()
	sched.Add("analytics-log", "log analytics every minute", "* * * * *", func() {
		log.Println("[scheduler] analytics flush tick")
	})
	go taskscheduler.Start(sched, context.Background())

	// ── Rate limiter (04-rate-limiter) ────────────────────────────────────────
	rl := ratelimit.NewTokenBucket(10, 10)

	// ── HTTP handler ──────────────────────────────────────────────────────────
	var c handler.Cache
	if cacheClient != nil {
		c = cacheClient
	} else {
		c = newInMemCache()
	}

	h := handler.New(c, mqClient, 86400)
	mux := http.NewServeMux()
	h.Register(mux)

	addr := envOr("ADDR", ":8087")
	log.Printf("url-shortener on %s (rate-limit: 10 req/s, burst 10)", addr)
	log.Fatal(http.ListenAndServe(addr, ratelimit.Middleware(rl)(mux)))
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type inMemCache struct{ data map[string]string }

func newInMemCache() *inMemCache { return &inMemCache{data: make(map[string]string)} }
func (m *inMemCache) Set(k, v string, _ int) error { m.data[k] = v; return nil }
func (m *inMemCache) Get(k string) (string, error) {
	v, ok := m.data[k]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return v, nil
}
func (m *inMemCache) Del(k string) error { delete(m.data, k); return nil }
