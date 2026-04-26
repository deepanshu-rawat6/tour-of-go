package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	EventsReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "k8s_events_received_total",
		Help: "Total K8s events received from informers.",
	})
	EventsDropped = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "k8s_events_dropped_total",
		Help: "Total events dropped by the filter.",
	})
	EventsDeduplicated = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "k8s_events_deduplicated_total",
		Help: "Total events suppressed by the dedup engine.",
	})
	EventsForwarded = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "k8s_events_forwarded_total",
		Help: "Total events forwarded to storage and alerters.",
	})
	AlertsSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "k8s_alerts_sent_total",
		Help: "Total alerts sent.",
	})
)

func Register() {
	prometheus.MustRegister(
		EventsReceived, EventsDropped,
		EventsDeduplicated, EventsForwarded, AlertsSent,
	)
}
