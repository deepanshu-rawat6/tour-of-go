package scheduler

import (
	"context"
	"log/slog"
	"time"

	"tour_of_go/projects/realtime-dashboard/internal/ws"
)

// Poller polls the distributed-scheduler API and broadcasts updates to the WebSocket hub.
type Poller struct {
	client   *Client
	hub      *ws.Hub
	interval time.Duration
	log      *slog.Logger
}

func NewPoller(client *Client, hub *ws.Hub, interval time.Duration) *Poller {
	return &Poller{client: client, hub: hub, interval: interval, log: slog.Default()}
}

// Start begins polling. Stops when ctx is cancelled.
func (p *Poller) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.poll(ctx)
			}
		}
	}()
}

func (p *Poller) poll(ctx context.Context) {
	keys, err := p.client.GetConcurrencyKeys(ctx)
	if err != nil {
		p.log.Debug("scheduler poll failed", "error", err)
		return
	}
	p.hub.Broadcast(ws.Message{Type: "concurrency_update", Data: keys})
}
