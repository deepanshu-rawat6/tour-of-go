package net_http

// client.go — Making HTTP requests: GET, POST, headers, body safety, error handling
//
// GOLDEN RULES for http.Client:
//   1. Always defer resp.Body.Close() — even if you don't read the body.
//      Not closing leaks the TCP connection out of the pool.
//   2. Always drain the body before closing — io.Copy(io.Discard, resp.Body).
//      Without draining, the TCP connection cannot be reused.
//   3. Wrap large response bodies in io.LimitReader to cap memory usage.
//   4. Check resp.StatusCode — a non-nil error means network failure;
//      a 4xx/5xx comes back as a successful *http.Response.
//   5. Use json.NewEncoder + io.Pipe for streaming POST bodies (no buffer copy).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// ── Request/Response types ────────────────────────────────────────────────────

// CreateOrderRequest is the payload sent in a POST body.
type CreateOrderRequest struct {
	Product  string  `json:"product"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

// CreateOrderResponse is returned by the server.
type CreateOrderResponse struct {
	OrderID string  `json:"order_id"`
	Total   float64 `json:"total"`
	Status  string  `json:"status"`
}

// APIError is returned by the server on 4xx/5xx.
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

// ── GET with custom headers ───────────────────────────────────────────────────

// getWithHeaders demonstrates how to add custom headers to a GET request.
// Common use cases: Authorization, X-Request-ID, Accept, User-Agent.
func getWithHeaders(client *http.Client, baseURL string) {
	fmt.Println("  [1] GET with custom headers:")

	// http.NewRequest creates the request without sending it.
	// This gives us a chance to modify headers before calling client.Do.
	req, err := http.NewRequest(http.MethodGet, baseURL+"/orders/42", nil)
	if err != nil {
		fmt.Printf("    build request error: %v\n", err)
		return
	}

	// Set headers individually — each Set() call replaces any existing value.
	req.Header.Set("Authorization", "Bearer my-api-token")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Request-ID", "req-abc-123") // for distributed tracing
	req.Header.Set("User-Agent", "my-service/1.0")

	resp, err := client.Do(req)
	if err != nil {
		// err here means a NETWORK failure (DNS, TCP, timeout).
		// A 404 or 500 response does NOT trigger an error — check StatusCode.
		fmt.Printf("    network error: %v\n", err)
		return
	}
	// RULE: always close the body, even for error status codes.
	// defer runs even if we return early due to a status check.
	defer resp.Body.Close()

	// io.LimitReader caps how many bytes we'll read.
	// Without this, a buggy or malicious server could return a 1 GB body
	// and exhaust all memory.  1 MB is a common cap for API responses.
	const maxBodyBytes = 1 << 20 // 1 MiB
	limited := io.LimitReader(resp.Body, maxBodyBytes)

	body, _ := io.ReadAll(limited)
	fmt.Printf("    status: %d\n    headers echoed: %s\n    body: %s\n\n",
		resp.StatusCode,
		resp.Header.Get("X-Received-Headers"),
		strings.TrimSpace(string(body)))
}

// ── POST with streaming JSON body via io.Pipe ─────────────────────────────────

// postWithPipe sends a JSON POST body without buffering the entire payload
// in a []byte first.
//
// HOW io.Pipe WORKS:
//   io.Pipe() returns a (pr *PipeReader, pw *PipeWriter) pair.
//   Writes to pw are synchronised with reads from pr — zero-copy.
//   The HTTP client reads from pr as the goroutine writes to pw.
//   This is useful for very large request bodies (streaming upload).
func postWithPipe(client *http.Client, baseURL string) {
	fmt.Println("  [2] POST with json.NewEncoder + io.Pipe (streaming body):")

	pr, pw := io.Pipe()

	// A goroutine writes the JSON-encoded body to pw.
	// The HTTP client goroutine reads from pr at the same time.
	// When the goroutine calls pw.Close(), the reader sees EOF.
	go func() {
		enc := json.NewEncoder(pw)
		order := CreateOrderRequest{
			Product:  "Keyboard",
			Quantity: 2,
			Price:    49.99,
		}
		if err := enc.Encode(order); err != nil {
			// CloseWithError propagates the error to the reader.
			pw.CloseWithError(err)
			return
		}
		pw.Close() // signal EOF to the reader (HTTP client)
	}()

	req, err := http.NewRequest(http.MethodPost, baseURL+"/orders", pr)
	if err != nil {
		fmt.Printf("    build request error: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer my-api-token")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("    network error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Parse the response based on status code.
	if err := parseResponse(resp); err != nil {
		fmt.Printf("    error: %v\n\n", err)
		return
	}

	// Decode success body.
	var result CreateOrderResponse
	limited := io.LimitReader(resp.Body, 1<<20)
	if err := json.NewDecoder(limited).Decode(&result); err != nil {
		fmt.Printf("    decode error: %v\n\n", err)
		return
	}
	fmt.Printf("    created order: id=%q total=%.2f status=%q\n\n",
		result.OrderID, result.Total, result.Status)
}

// ── POST with bytes.Buffer (simpler, fine for small bodies) ──────────────────

// postWithBuffer is the simpler alternative to io.Pipe for small JSON bodies.
// It encodes the whole payload into a buffer then sends it.
func postWithBuffer(client *http.Client, baseURL string) {
	fmt.Println("  [3] POST with bytes.Buffer (simpler for small payloads):")

	var buf bytes.Buffer
	order := CreateOrderRequest{Product: "Mouse", Quantity: 1, Price: 29.99}
	if err := json.NewEncoder(&buf).Encode(order); err != nil {
		fmt.Printf("    encode error: %v\n", err)
		return
	}

	// bytes.NewReader wraps the buffer's bytes as an io.Reader.
	// Unlike bytes.Buffer, NewReader is seek-able (useful for retries).
	req, err := http.NewRequest(http.MethodPost, baseURL+"/orders", bytes.NewReader(buf.Bytes()))
	if err != nil {
		fmt.Printf("    build request error: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("    network error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if err := parseResponse(resp); err != nil {
		fmt.Printf("    error: %v\n\n", err)
		return
	}

	var result CreateOrderResponse
	_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result)
	fmt.Printf("    created order: id=%q total=%.2f status=%q\n\n",
		result.OrderID, result.Total, result.Status)
}

// ── Status code checking and error body decoding ──────────────────────────────

// parseResponse checks the status code and tries to decode an error body on
// 4xx/5xx.  This separates the "was the request successful?" concern from
// the "decode the happy path" concern.
//
// IMPORTANT: parseResponse does NOT close the body.  On success (2xx) the
// caller still needs to read + close the body.  On error we drain+close here.
func parseResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil // success — let the caller read the body
	}

	// On error: read the error body (capped), decode it, then drain+close.
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, 64*1024) // 64 KiB max for error bodies
	var apiErr APIError
	if err := json.NewDecoder(limited).Decode(&apiErr); err != nil {
		// Server returned a non-JSON error body (e.g. plain text nginx error).
		return fmt.Errorf("HTTP %d: could not parse error body: %w", resp.StatusCode, err)
	}
	apiErr.Code = resp.StatusCode
	return apiErr
}

// ── Demonstration ─────────────────────────────────────────────────────────────

func clientExample() {
	fmt.Println("--- http.Client: GET, POST, body safety, error handling ---")

	// Local test server that simulates an orders API.
	mux := http.NewServeMux()

	// GET /orders/:id — echoes received headers, returns a fake order.
	mux.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, APIError{Message: "use GET"})
			return
		}
		// Echo back the headers we received (for demonstration).
		received := []string{
			"Authorization: " + r.Header.Get("Authorization"),
			"X-Request-ID: " + r.Header.Get("X-Request-ID"),
			"User-Agent: " + r.Header.Get("User-Agent"),
		}
		w.Header().Set("X-Received-Headers", strings.Join(received, " | "))
		writeJSON(w, http.StatusOK, map[string]any{
			"order_id": "ord-42",
			"product":  "Widget",
			"qty":      1,
		})
	})

	// POST /orders — reads the JSON body, returns a created order.
	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, APIError{Message: "use POST"})
			return
		}
		defer r.Body.Close()

		var order CreateOrderRequest
		limited := io.LimitReader(r.Body, 1<<20)
		if err := json.NewDecoder(limited).Decode(&order); err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Message: "invalid JSON: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusCreated, CreateOrderResponse{
			OrderID: "ord-" + order.Product,
			Total:   order.Price * float64(order.Quantity),
			Status:  "created",
		})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	fmt.Printf("  Test server: %s\n\n", ts.URL)

	// Use a short timeout so the example runs quickly.
	client := &http.Client{Timeout: 5 * time.Second}

	getWithHeaders(client, ts.URL)
	postWithPipe(client, ts.URL)
	postWithBuffer(client, ts.URL)

	fmt.Println("  Safety checklist:")
	fmt.Println("    ✓ defer resp.Body.Close()           — always, even on error status")
	fmt.Println("    ✓ io.LimitReader(resp.Body, max)    — cap memory usage")
	fmt.Println("    ✓ check resp.StatusCode explicitly  — non-2xx is not an error in Go")
	fmt.Println("    ✓ drain body before closing on 4xx  — conn reuse requires full read")
	fmt.Println("    ✓ io.Pipe for large request bodies  — avoid buffering in RAM")
}
