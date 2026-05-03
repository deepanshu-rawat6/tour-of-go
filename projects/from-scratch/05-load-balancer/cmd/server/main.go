// Command server starts the load balancer on :8084.
// Set STRATEGY=rr|lc (default: rr). Set BACKENDS=url1,url2,url3.
package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"tour_of_go/projects/from-scratch/05-load-balancer/internal/balancer"
	"tour_of_go/projects/from-scratch/05-load-balancer/internal/proxy"
)

func main() {
	rawBackends := envOr("BACKENDS", "http://localhost:9010,http://localhost:9011,http://localhost:9012")
	var bs []*balancer.Backend
	for _, u := range strings.Split(rawBackends, ",") {
		b, err := balancer.NewBackend(strings.TrimSpace(u))
		if err != nil {
			log.Fatalf("invalid backend %s: %v", u, err)
		}
		bs = append(bs, b)
	}

	balancer.StartHealthChecker(bs, 5*time.Second)

	var b balancer.Balancer
	switch envOr("STRATEGY", "rr") {
	case "lc":
		b = balancer.NewLeastConn(bs)
		log.Println("strategy: least-connections")
	default:
		b = balancer.NewRoundRobin(bs)
		log.Println("strategy: round-robin")
	}

	log.Printf("load balancer on :8084 → %s", rawBackends)
	log.Fatal(http.ListenAndServe(":8084", proxy.New(b)))
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
