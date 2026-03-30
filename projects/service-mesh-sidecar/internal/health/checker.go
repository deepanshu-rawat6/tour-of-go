// Package health implements periodic upstream health checking.
package health

import (
	"context"
	"log/slog"
	"net"
	"sync/atomic"
	"time"
)

// Checker periodically checks if the upstream is reachable via TCP.
// Sets a health flag that the proxy can consult.
type Checker struct {
	upstreamAddr string
	interval     time.Duration
	timeout      time.Duration
	healthy      atomic.Bool
	log          *slog.Logger
}

func NewChecker(upstreamAddr string, interval, timeout time.Duration) *Checker {
	c := &Checker{
		upstreamAddr: upstreamAddr,
		interval:     interval,
		timeout:      timeout,
		log:          slog.Default(),
	}
	c.healthy.Store(true) // assume healthy on start
	return c
}

// IsHealthy returns the current health status.
func (c *Checker) IsHealthy() bool {
	return c.healthy.Load()
}

// Start begins periodic health checks. Stops when ctx is cancelled.
func (c *Checker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.check()
			}
		}
	}()
}

func (c *Checker) check() {
	conn, err := net.DialTimeout("tcp", c.upstreamAddr, c.timeout)
	if err != nil {
		if c.healthy.Load() {
			c.log.Warn("upstream unhealthy", "addr", c.upstreamAddr, "error", err)
		}
		c.healthy.Store(false)
		return
	}
	conn.Close()
	if !c.healthy.Load() {
		c.log.Info("upstream recovered", "addr", c.upstreamAddr)
	}
	c.healthy.Store(true)
}
