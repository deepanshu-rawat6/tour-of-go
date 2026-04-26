package metrics

import (
	"context"
	"log/slog"
	"time"

	"github.com/tour-of-go/xdp-firewall/internal/core"
)

// Poller reads per-CPU eBPF counters on a ticker and updates Prometheus metrics.
type Poller struct {
	reader   core.CounterReader
	engine   interface{ ListRules() []string }
	interval time.Duration
}

func NewPoller(reader core.CounterReader, engine interface{ ListRules() []string }, interval time.Duration) *Poller {
	return &Poller{reader: reader, engine: engine, interval: interval}
}

// Run starts the polling loop. Blocks until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// Track previous totals to compute deltas (Prometheus counters are cumulative).
	var prevPackets, prevDrops uint64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c, err := p.reader.ReadCounters()
			if err != nil {
				slog.Error("reading BPF counters", "error", err)
				continue
			}
			// Add delta since last poll.
			if c.PacketsTotal >= prevPackets {
				PacketsTotal.Add(float64(c.PacketsTotal - prevPackets))
			}
			if c.DropsTotal >= prevDrops {
				DropsTotal.Add(float64(c.DropsTotal - prevDrops))
			}
			prevPackets = c.PacketsTotal
			prevDrops = c.DropsTotal

			BlacklistSize.Set(float64(len(p.engine.ListRules())))
		}
	}
}
