package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for the sidecar.
type Metrics struct {
	ConnectionsTotal  prometheus.Counter
	ConnectionsActive prometheus.Gauge
	RateLimited       prometheus.Counter
	CircuitBlocked    prometheus.Counter
	RequestDuration   prometheus.Histogram
}

func New() *Metrics {
	return &Metrics{
		ConnectionsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "sidecar_connections_total",
			Help: "Total TCP connections accepted",
		}),
		ConnectionsActive: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "sidecar_connections_active",
			Help: "Currently active connections",
		}),
		RateLimited: promauto.NewCounter(prometheus.CounterOpts{
			Name: "sidecar_rate_limited_total",
			Help: "Connections rejected by rate limiter",
		}),
		CircuitBlocked: promauto.NewCounter(prometheus.CounterOpts{
			Name: "sidecar_circuit_blocked_total",
			Help: "Connections blocked by circuit breaker",
		}),
		RequestDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "sidecar_request_duration_seconds",
			Help:    "Proxy request duration",
			Buckets: prometheus.DefBuckets,
		}),
	}
}

// Handler returns the Prometheus HTTP handler for /metrics.
func Handler() http.Handler {
	return promhttp.Handler()
}
