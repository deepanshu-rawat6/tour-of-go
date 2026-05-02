// Command server starts the cache-service HTTP API.
// Set CACHE_BACKEND=redis to use Redis; default is in-memory LRU.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"tour_of_go/projects/cache-service/internal/cache"
	"tour_of_go/projects/cache-service/internal/handler"
	"tour_of_go/projects/cache-service/internal/store"
)

func main() {
	addr := envOr("ADDR", ":8081")
	backend := envOr("CACHE_BACKEND", "lru")
	redisAddr := envOr("REDIS_ADDR", "localhost:6379")
	ttl := 5 * time.Minute
	capacity := 1000

	var c handler.Cache
	switch backend {
	case "redis":
		r := cache.NewRedis(redisAddr, ttl)
		c = r
		log.Printf("cache-service using Redis backend at %s", redisAddr)
	default:
		lru := cache.NewLRU(capacity)
		mem := store.NewMemory()
		c = cache.NewSingleflightCache(cache.NewCacheAside(lru, mem, ttl))
		log.Printf("cache-service using in-memory LRU (cap=%d, ttl=%s)", capacity, ttl)
	}

	mux := http.NewServeMux()
	handler.New(c).Register(mux)

	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
