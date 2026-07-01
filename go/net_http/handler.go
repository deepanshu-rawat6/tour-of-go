package net_http

// handler.go — http.Handler, http.HandlerFunc, ServeMux, and the request lifecycle
//
// The http.Handler interface is the heart of Go's HTTP server:
//
//   type Handler interface {
//       ServeHTTP(ResponseWriter, *Request)
//   }
//
// Anything that implements ServeHTTP can serve HTTP.
// http.HandlerFunc is a func type that itself implements ServeHTTP — it lets
// you turn any compatible func into an http.Handler without defining a struct.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// ── Custom http.Handler implemented as a struct ───────────────────────────────

// GreetHandler is a struct-based handler.  Struct handlers are useful when
// the handler needs injected dependencies (DB, logger, config, etc.).
type GreetHandler struct {
	// Prefix is injected at construction time — no global state needed.
	Prefix string
}

// ServeHTTP satisfies the http.Handler interface.
// Go's HTTP server calls this for every incoming request matched to this handler.
//
// Request lifecycle visible here:
//  1. r.Method  — "GET", "POST", etc.
//  2. r.URL     — parsed URL including path, query, fragment
//  3. r.Header  — request headers as map[string][]string
//  4. r.Body    — io.ReadCloser; MUST be read and/or closed by the handler
func (h GreetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract the last path segment as the "name".
	// r.URL.Path for "/greet/Alice" → split gives ["", "greet", "Alice"]
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	name := "World"
	if len(parts) >= 2 && parts[1] != "" {
		name = parts[1]
	}

	// Write a JSON response the idiomatic way:
	//   1. Set Content-Type BEFORE WriteHeader — headers must come first.
	//   2. WriteHeader sets the status code; calling w.Write implies 200 if
	//      WriteHeader was never called.
	writeJSON(w, http.StatusOK, map[string]string{
		"greeting": fmt.Sprintf("%s, %s!", h.Prefix, name),
		"method":   r.Method,
		"path":     r.URL.Path,
	})
}

// ── http.HandlerFunc — func → Handler adapter ─────────────────────────────────

// http.HandlerFunc is defined in the stdlib as:
//
//   type HandlerFunc func(ResponseWriter, *Request)
//   func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) { f(w, r) }
//
// This means any func(ResponseWriter, *Request) can be cast to http.HandlerFunc
// and used wherever an http.Handler is expected.

// healthHandler is a plain function that we'll adapt via http.HandlerFunc.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// echoHandler reads the request body and echoes it back.
// Shows: reading Body, setting response headers, method inspection.
func echoHandler(w http.ResponseWriter, r *http.Request) {
	// Always close the body — even if you don't read it.
	// This lets the HTTP/1.1 connection be reused.
	defer r.Body.Close()

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "only POST is supported",
		})
		return
	}

	// io.ReadAll reads everything from the body. For production use
	// io.LimitReader to cap the size (see client.go).
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Echo: set a custom response header then mirror the body.
	w.Header().Set("X-Echo-Length", fmt.Sprintf("%d", len(body)))
	writeJSON(w, http.StatusOK, map[string]string{
		"echoed": string(body),
		"length": fmt.Sprintf("%d bytes", len(body)),
	})
}

// ── http.ServeMux — request router ───────────────────────────────────────────
//
// ServeMux matches incoming request URLs to registered patterns.
// Patterns can be:
//   - "/exact/path"        — exact match only
//   - "/prefix/"           — matches /prefix/ and everything under it (trailing slash)
//   - "GET /path"          — (Go 1.22+) method + path pattern
//
// The longest matching pattern wins.

func buildMux() *http.ServeMux {
	mux := http.NewServeMux()

	// Register the struct-based handler under a prefix pattern.
	// The trailing "/" means "match /greet/ and any deeper path".
	mux.Handle("/greet/", GreetHandler{Prefix: "Hello"})

	// Convert a plain func to an http.Handler using http.HandlerFunc.
	mux.Handle("/health", http.HandlerFunc(healthHandler))

	// http.HandleFunc is a shorthand: mux.HandleFunc(p, f) is equivalent to
	// mux.Handle(p, http.HandlerFunc(f)).
	mux.HandleFunc("/echo", echoHandler)

	// A catch-all route — "/" matches everything not matched by a longer pattern.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "route not found",
			"path":  r.URL.Path,
		})
	})

	return mux
}

// ── Helper ────────────────────────────────────────────────────────────────────

// writeJSON is a small helper used throughout this package.
// It sets Content-Type, writes the status code, and encodes the payload.
// Order matters: Header() calls must happen before WriteHeader(), which must
// happen before any Write() — headers are sent with the first write.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ── Demonstration ─────────────────────────────────────────────────────────────

func handlerExample() {
	fmt.Println("--- http.Handler, HandlerFunc, ServeMux ---")

	mux := buildMux()

	// httptest.NewServer spins up a real HTTP server on a random localhost port
	// and returns a *httptest.Server.  No firewall rules, no port conflicts.
	// Calling Close() shuts the server down immediately.
	ts := httptest.NewServer(mux)
	defer ts.Close() // always clean up

	fmt.Printf("  Test server listening at: %s\n\n", ts.URL)

	// ── Scenario 1: struct-based handler with path param ─────────────────────
	resp, body := doRequest("GET", ts.URL+"/greet/Alice", "", nil)
	fmt.Printf("  GET /greet/Alice → %d\n  body: %s\n", resp.StatusCode, body)

	// ── Scenario 2: HandlerFunc — health check ───────────────────────────────
	resp, body = doRequest("GET", ts.URL+"/health", "", nil)
	fmt.Printf("  GET /health     → %d\n  body: %s\n", resp.StatusCode, body)

	// ── Scenario 3: HandlerFunc — POST with body ─────────────────────────────
	resp, body = doRequest("POST", ts.URL+"/echo", "application/json", strings.NewReader(`{"msg":"hello"}`))
	echoLen := resp.Header.Get("X-Echo-Length")
	fmt.Printf("  POST /echo      → %d  X-Echo-Length: %s\n  body: %s\n", resp.StatusCode, echoLen, body)

	// ── Scenario 4: wrong method ─────────────────────────────────────────────
	resp, body = doRequest("GET", ts.URL+"/echo", "", nil)
	fmt.Printf("  GET /echo       → %d (method not allowed)\n  body: %s\n", resp.StatusCode, body)

	// ── Scenario 5: catch-all 404 ────────────────────────────────────────────
	resp, body = doRequest("GET", ts.URL+"/unknown/route", "", nil)
	fmt.Printf("  GET /unknown    → %d (not found)\n  body: %s\n", resp.StatusCode, body)

	fmt.Println("  Request lifecycle summary:")
	fmt.Println("    1. Server accepts TCP connection")
	fmt.Println("    2. http package parses request line + headers")
	fmt.Println("    3. ServeMux.ServeHTTP finds the longest matching pattern")
	fmt.Println("    4. The matched Handler.ServeHTTP is called")
	fmt.Println("    5. Handler writes headers then body via ResponseWriter")
	fmt.Println("    6. Connection kept-alive for the next request (HTTP/1.1)")
}

// doRequest is a test helper that makes an HTTP request and returns the
// response + body string.  It uses the default http.Client (no custom transport).
func doRequest(method, url, contentType string, body io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return &http.Response{StatusCode: 0}, err.Error()
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &http.Response{StatusCode: 0}, err.Error()
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	return resp, strings.TrimSpace(string(b))
}
