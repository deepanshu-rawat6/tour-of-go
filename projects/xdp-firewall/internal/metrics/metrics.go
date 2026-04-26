package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	PacketsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "xdp_packets_total",
		Help: "Total packets processed by the XDP program.",
	})
	DropsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "xdp_drops_total",
		Help: "Total packets dropped by the XDP blacklist.",
	})
	BlacklistSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "xdp_blacklist_size",
		Help: "Number of CIDR rules currently in the blacklist.",
	})
)

func Register() {
	prometheus.MustRegister(PacketsTotal, DropsTotal, BlacklistSize)
}
