// Package proxy implements the TCP proxy core.
//
// Design decisions:
//   - Goroutine-per-connection: cheap in Go (2KB stack vs 1MB OS thread)
//     See docs/adr/002-goroutine-per-connection.md
//   - io.Copy for bidirectional proxying: uses splice(2) on Linux (zero-copy)
//     See docs/adr/001-io-copy.md
package proxy

import (
	"context"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"tour_of_go/projects/service-mesh-sidecar/internal/middleware"
	"tour_of_go/projects/service-mesh-sidecar/internal/metrics"
)

// Proxy is the TCP proxy. It accepts connections on listenAddr and forwards
// them to upstreamAddr, applying rate limiting and circuit breaking.
type Proxy struct {
	listenAddr   string
	upstreamAddr string
	rateLimiter  *middleware.TokenBucket
	cb           *middleware.CircuitBreaker
	metrics      *metrics.Metrics
	log          *slog.Logger
}

func NewProxy(listenAddr, upstreamAddr string, rl *middleware.TokenBucket, cb *middleware.CircuitBreaker, m *metrics.Metrics) *Proxy {
	return &Proxy{
		listenAddr:   listenAddr,
		upstreamAddr: upstreamAddr,
		rateLimiter:  rl,
		cb:           cb,
		metrics:      m,
		log:          slog.Default(),
	}
}

// Start begins accepting connections. Stops when ctx is cancelled.
func (p *Proxy) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", p.listenAddr)
	if err != nil {
		return err
	}
	p.log.Info("proxy listening", "addr", p.listenAddr, "upstream", p.upstreamAddr)

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				p.log.Error("accept error", "error", err)
				continue
			}
		}
		p.metrics.ConnectionsTotal.Inc()
		p.metrics.ConnectionsActive.Inc()
		go p.handleConn(conn)
	}
}

func (p *Proxy) handleConn(clientConn net.Conn) {
	defer func() {
		clientConn.Close()
		p.metrics.ConnectionsActive.Dec()
	}()

	clientIP, _, _ := net.SplitHostPort(clientConn.RemoteAddr().String())

	// Rate limiting check
	if !p.rateLimiter.Allow(clientIP) {
		p.metrics.RateLimited.Inc()
		p.log.Debug("rate limited", "client", clientIP)
		return
	}

	// Circuit breaker check
	if err := p.cb.Allow(); err != nil {
		p.metrics.CircuitBlocked.Inc()
		p.log.Debug("circuit open, blocking connection", "client", clientIP)
		return
	}

	// Connect to upstream
	start := time.Now()
	upstreamConn, err := net.DialTimeout("tcp", p.upstreamAddr, 5*time.Second)
	if err != nil {
		p.cb.RecordFailure()
		p.log.Error("upstream connect failed", "upstream", p.upstreamAddr, "error", err)
		return
	}
	defer upstreamConn.Close()

	// Bidirectional proxy using io.Copy
	// On Linux, io.Copy uses splice(2) for zero-copy kernel-level transfer
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(upstreamConn, clientConn)
		upstreamConn.(*net.TCPConn).CloseWrite()
	}()

	go func() {
		defer wg.Done()
		io.Copy(clientConn, upstreamConn)
		clientConn.(*net.TCPConn).CloseWrite()
	}()

	wg.Wait()
	p.cb.RecordSuccess()
	p.metrics.RequestDuration.Observe(time.Since(start).Seconds())
}
