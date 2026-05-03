// Package balancer provides backend selection strategies for a load balancer.
package balancer

import (
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// Backend represents an upstream server.
type Backend struct {
	URL         *url.URL
	ActiveConns atomic.Int64
	healthy     atomic.Bool
}

func NewBackend(rawURL string) (*Backend, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	b := &Backend{URL: u}
	b.healthy.Store(true)
	return b, nil
}

func (b *Backend) IsHealthy() bool  { return b.healthy.Load() }
func (b *Backend) SetHealthy(v bool) { b.healthy.Store(v) }

// Balancer selects a healthy backend.
type Balancer interface {
	Next() *Backend
}

// ── Round Robin ───────────────────────────────────────────────────────────────

type RoundRobin struct {
	backends []*Backend
	idx      atomic.Uint64
}

func NewRoundRobin(backends []*Backend) *RoundRobin { return &RoundRobin{backends: backends} }

func (rr *RoundRobin) Next() *Backend {
	n := uint64(len(rr.backends))
	for i := uint64(0); i < n; i++ {
		idx := rr.idx.Add(1) % n
		b := rr.backends[idx]
		if b.IsHealthy() {
			return b
		}
	}
	return nil
}

// ── Least Connections ─────────────────────────────────────────────────────────

type LeastConn struct {
	mu       sync.Mutex
	backends []*Backend
}

func NewLeastConn(backends []*Backend) *LeastConn { return &LeastConn{backends: backends} }

func (lc *LeastConn) Next() *Backend {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	var best *Backend
	for _, b := range lc.backends {
		if !b.IsHealthy() {
			continue
		}
		if best == nil || b.ActiveConns.Load() < best.ActiveConns.Load() {
			best = b
		}
	}
	return best
}

// ── Health Checker ────────────────────────────────────────────────────────────

// StartHealthChecker polls each backend's /health endpoint every interval.
func StartHealthChecker(backends []*Backend, interval time.Duration) {
	client := &http.Client{Timeout: 2 * time.Second}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			for _, b := range backends {
				resp, err := client.Get(b.URL.String() + "/health")
				if err != nil || resp.StatusCode >= 500 {
					b.SetHealthy(false)
				} else {
					b.SetHealthy(true)
				}
				if resp != nil {
					resp.Body.Close()
				}
			}
		}
	}()
}
