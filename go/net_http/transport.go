package net_http

// transport.go — http.Transport: connection pooling, timeouts, and retries
//
// THE TWO LAYERS OF TIMEOUT:
//
//   http.Client.Timeout            — End-to-end deadline for one request+response
//   http.Transport settings        — Low-level connection & TLS behaviour
//
// TIMEOUT COMPARISON:
//
//   client.Timeout       → includes: DNS, connect, TLS, request send, response headers, body read
//   DialContext timeout  → how long to wait to open a TCP connection
//   TLSHandshakeTimeout  → how long to wait for the TLS handshake
//   ResponseHeaderTimeout→ how long to wait for the response headers AFTER sending the request
//   IdleConnTimeout      → how long an idle keep-alive connection lives in the pool
//
// CONNECTION POOLING:
//   Go's http.Transport maintains a pool of idle TCP connections per host.
//   This avoids the 3-way TCP handshake (+ TLS) on every request.
//   Pool size is controlled by MaxIdleConns and MaxIdleConnsPerHost.

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// ── Transport configuration ───────────────────────────────────────────────────

// newProductionTransport creates an http.Transport tuned for a production
// backend service that calls one upstream host with high concurrency.
func newProductionTransport() *http.Transport {
	return &http.Transport{
		// ── Connection pool settings ─────────────────────────────────────────

		// MaxIdleConns caps the total idle connections across ALL hosts.
		// Default: 100.  Set higher if your service fans out to many hosts.
		MaxIdleConns: 200,

		// MaxIdleConnsPerHost caps idle connections to a SINGLE host.
		// Default: 2 (surprisingly low!).  For a service hitting one backend
		// with 50 goroutines, keep this at or above your concurrency.
		MaxIdleConnsPerHost: 50,

		// MaxConnsPerHost caps *all* connections (idle + active) to one host.
		// 0 = unlimited.  Setting a limit prevents accidental DDoS of upstream.
		MaxConnsPerHost: 100,

		// IdleConnTimeout is how long a pooled connection can stay idle before
		// it is closed and removed from the pool.
		IdleConnTimeout: 90 * time.Second,

		// ── TLS & dial timeouts ──────────────────────────────────────────────

		// TLSHandshakeTimeout limits how long the TLS handshake may take.
		// If the upstream server is slow to complete TLS, this fires first.
		TLSHandshakeTimeout: 10 * time.Second,

		// ResponseHeaderTimeout starts after the full request has been sent
		// and measures how long to wait for the response's first byte (headers).
		// Useful to detect stalled upstream servers.
		ResponseHeaderTimeout: 30 * time.Second,

		// ExpectContinueTimeout is relevant for large POST bodies with the
		// "Expect: 100-continue" header.  Usually fine at 1s.
		ExpectContinueTimeout: 1 * time.Second,

		// DisableKeepAlives = false (default) means connections ARE reused.
		// Only set true if you're making one-shot requests and don't want pooling.
		DisableKeepAlives: false,
	}
}

// newClientWithTimeout builds an http.Client with a transport and a
// request-level timeout.
//
// IMPORTANT: Always set client.Timeout.  Without it, a slow server can hold
// your goroutine open forever.  client.Timeout is the TOTAL budget for one
// complete request (including body read).
func newClientWithTimeout(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: newProductionTransport(),
		// This timeout fires if the entire request-response cycle takes longer
		// than `timeout`.  It cancels the underlying context automatically.
		Timeout: timeout,
	}
}

// ── Retry with exponential backoff ───────────────────────────────────────────

// doWithRetry executes an HTTP request and retries on transient failures.
//
// Retry strategy:
//   • Retry only on 5xx responses or network errors (not on 4xx — those are
//     the caller's fault and retrying won't help).
//   • Exponential backoff: wait = base * 2^attempt with a cap.
//   • Each attempt passes the same context, so the caller's deadline is
//     respected across all retries combined.
//
// Parameters:
//   client     — the http.Client to use (with its own transport/timeout)
//   req        — the request to send; NOTE: Body is consumed on the first
//                attempt, so for retrying POST bodies use a body factory
//                (not shown here for brevity — use io.NopCloser + bytes.Buffer)
//   maxRetries — maximum number of additional attempts after the first
//   baseDelay  — starting wait time (doubled each attempt)
func doWithRetry(client *http.Client, req *http.Request, maxRetries int, baseDelay time.Duration) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: base * 2^(attempt-1)
			// Cap at 30 seconds to avoid very long waits.
			backoff := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt-1)))
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			fmt.Printf("    retry #%d after %v backoff\n", attempt, backoff)

			// Honor context cancellation during the wait.
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(backoff):
				// continue after waiting
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			// Network-level error (DNS failure, refused connection, timeout).
			// These are retryable.
			lastErr = fmt.Errorf("attempt %d network error: %w", attempt+1, err)
			fmt.Printf("    attempt %d: network error: %v\n", attempt+1, err)
			continue
		}

		// 5xx = server-side transient error (overload, restart, etc.) → retry.
		// 4xx = client error (bad request, unauthorized) → do NOT retry.
		// 2xx/3xx = success.
		if resp.StatusCode >= 500 {
			// Drain and close the body to allow connection reuse.
			io.Copy(io.Discard, resp.Body) //nolint:errcheck
			resp.Body.Close()
			lastErr = fmt.Errorf("attempt %d: server error %d", attempt+1, resp.StatusCode)
			fmt.Printf("    attempt %d: server returned %d, will retry\n", attempt+1, resp.StatusCode)
			continue
		}

		// Success (or non-retryable error code).
		return resp, nil
	}

	return nil, fmt.Errorf("all %d attempts failed: %w", maxRetries+1, lastErr)
}

// ── Demonstration ─────────────────────────────────────────────────────────────

func transportExample() {
	fmt.Println("--- http.Transport: connection pool, timeouts, retries ---")

	// We'll simulate two behaviours with a single test server:
	//   GET /ok       → 200 OK immediately
	//   GET /slow     → sleeps 100ms (but client timeout is 2s, so it still succeeds)
	//   GET /retry    → returns 503 twice, then 200 on the third try

	retryCount := 0 // track how many times /retry has been hit

	mux := http.NewServeMux()

	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"route": "ok", "pooled": "true"})
	})

	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond) // simulate slow backend
		writeJSON(w, http.StatusOK, map[string]string{"route": "slow"})
	})

	mux.HandleFunc("/retry", func(w http.ResponseWriter, r *http.Request) {
		retryCount++
		if retryCount < 3 {
			// First two calls: simulate a transient server error.
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "service temporarily unavailable",
			})
			return
		}
		// Third call: success.
		writeJSON(w, http.StatusOK, map[string]string{
			"result":   "ok after retry",
			"attempts": fmt.Sprintf("%d", retryCount),
		})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	fmt.Printf("  Test server: %s\n\n", ts.URL)

	// ── 1. Basic request using the custom transport ───────────────────────────
	client := newClientWithTimeout(5 * time.Second)

	fmt.Println("  [1] Request with production transport (connection pool tuned):")
	resp, err := client.Get(ts.URL + "/ok")
	if err != nil {
		fmt.Printf("    error: %v\n", err)
	} else {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("    status: %d  body: %s\n\n", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	// ── 2. Context-aware request ─────────────────────────────────────────────
	fmt.Println("  [2] Context-aware request (per-request deadline):")
	// req.WithContext returns a shallow copy of req with a new context.
	// The context deadline fires independently of client.Timeout.
	// Whichever fires first cancels the request.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/slow", nil)
	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("    context cancelled or timeout: %v\n\n", err)
	} else {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("    status: %d  body: %s\n\n", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	// ── 3. Retry with exponential backoff ────────────────────────────────────
	fmt.Println("  [3] Retry with exponential backoff (server returns 503 twice):")
	retryReq, _ := http.NewRequestWithContext(
		context.Background(),
		"GET",
		ts.URL+"/retry",
		nil,
	)
	resp, err = doWithRetry(client, retryReq, 3, 10*time.Millisecond)
	if err != nil {
		fmt.Printf("    final error: %v\n\n", err)
	} else {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("    final status: %d  body: %s\n\n", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	fmt.Println("  Transport tuning cheatsheet:")
	fmt.Println("    MaxIdleConnsPerHost  → match to your per-host concurrency")
	fmt.Println("    IdleConnTimeout      → ~90s (match your upstream keepalive)")
	fmt.Println("    TLSHandshakeTimeout  → 10s is a safe default")
	fmt.Println("    client.Timeout       → total budget including body read")
	fmt.Println("    context deadline     → per-request budget, composes with client.Timeout")
	fmt.Println("    Retry on 5xx only    → never retry 4xx (caller's bug)")
}
