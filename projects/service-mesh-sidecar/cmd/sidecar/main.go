// Command sidecar is a TCP proxy with rate limiting, circuit breaking, and health checks.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"tour_of_go/projects/service-mesh-sidecar/internal/config"
	"tour_of_go/projects/service-mesh-sidecar/internal/health"
	"tour_of_go/projects/service-mesh-sidecar/internal/metrics"
	"tour_of_go/projects/service-mesh-sidecar/internal/middleware"
	"tour_of_go/projects/service-mesh-sidecar/internal/proxy"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, _ := config.Load(os.Getenv("CONFIG_PATH"))

	m := metrics.New()
	rl := middleware.NewTokenBucket(cfg.RateLimit.Rate, cfg.RateLimit.Burst)
	cb := middleware.NewCircuitBreaker(cfg.CircuitBreaker.Threshold, cfg.CircuitBreaker.RetryAfter)
	hc := health.NewChecker(cfg.Proxy.UpstreamAddr, cfg.Health.Interval, cfg.Health.Timeout)
	p := proxy.NewProxy(cfg.Proxy.ListenAddr, cfg.Proxy.UpstreamAddr, rl, cb, m)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc.Start(ctx)

	// Metrics HTTP server
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", metrics.Handler())
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			if hc.IsHealthy() {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok"}`))
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"status":"upstream_unhealthy"}`))
			}
		})
		slog.Info("metrics server starting", "addr", cfg.Proxy.MetricsAddr)
		http.ListenAndServe(cfg.Proxy.MetricsAddr, mux)
	}()

	// TCP proxy
	go func() {
		if err := p.Start(ctx); err != nil {
			slog.Error("proxy error", "error", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig
	slog.Info("shutting down sidecar")
	cancel()
}
