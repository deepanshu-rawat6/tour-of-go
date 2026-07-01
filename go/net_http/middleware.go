package net_http

// middleware.go — Middleware chain using the decorator pattern
//
// WHAT IS MIDDLEWARE?
//
// In Go's net/http, middleware is any function with the signature:
//
//   func(http.Handler) http.Handler
//
// It takes the *next* handler in the chain, wraps it in new behaviour,
// and returns a new http.Handler.  When the new handler's ServeHTTP is
// called, it can:
//   • Run logic BEFORE the next handler (e.g. log start time)
//   • Choose whether to CALL the next handler at all (e.g. auth check)
//   • Run logic AFTER the next handler returns (e.g. log duration)
//   • Intercept panics (recovery middleware)
//
// HOW WRAPPING WORKS (Decorator Pattern):
//
//   outer handler = Recovery(Auth(Logging(business handler)))
//
// Call flow for an incoming request:
//   Recovery.ServeHTTP → Logging.ServeHTTP → Auth.ServeHTTP → business handler
//   (panic catch)         (log start)         (check token)     (real work)
//                                                                 ↓
//   Recovery.ServeHTTP ← Logging.ServeHTTP ← (returns)      ← returns
//   (no panic, skip)      (log duration)

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// Middleware is the canonical type alias for Go middleware.
// Using a named type (rather than writing func(http.Handler)http.Handler inline)
// makes code more readable and lets us define the Chain helper below.
type Middleware func(http.Handler) http.Handler

// ── LoggingMiddleware ─────────────────────────────────────────────────────────

// LoggingMiddleware logs the HTTP method, path, and duration of every request.
// It wraps `next` in a closure that records the start time before calling next,
// then logs the elapsed duration after next returns.
func LoggingMiddleware(next http.Handler) http.Handler {
	// http.HandlerFunc converts our closure into an http.Handler.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now() // captured BEFORE next.ServeHTTP

		// ► Call the next handler in the chain.
		//   Until this line, we're in the "before" phase.
		next.ServeHTTP(w, r)

		// ◄ We're now in the "after" phase — next has returned.
		elapsed := time.Since(start)
		fmt.Printf("  [LOG] %s %s  %v\n", r.Method, r.URL.Path, elapsed)
	})
}

// ── AuthMiddleware ────────────────────────────────────────────────────────────

// AuthMiddleware checks for a valid Bearer token in the Authorization header.
// If the token is missing or wrong, it rejects the request with 401 and does
// NOT call the next handler — short-circuiting the chain.
func AuthMiddleware(next http.Handler) http.Handler {
	const validToken = "secret-token-123"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// Authorization: Bearer <token>
		// strings.TrimPrefix strips the "Bearer " prefix; if the header is
		// absent or malformed, trimmed will equal authHeader (no change).
		token := strings.TrimPrefix(authHeader, "Bearer ")

		if token != validToken {
			// Reject: write 401 and return WITHOUT calling next.
			// Returning here is what "short-circuits" the middleware chain.
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "missing or invalid Authorization header",
			})
			return // ← do NOT call next.ServeHTTP
		}

		// Token is valid — let the request continue down the chain.
		next.ServeHTTP(w, r)
	})
}

// ── RecoveryMiddleware ────────────────────────────────────────────────────────

// RecoveryMiddleware catches panics in downstream handlers and returns a clean
// 500 response instead of crashing the entire server process.
//
// Without recovery, a panic in any goroutine serving a request will bring down
// the whole server.  This is critical for production services.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// defer + recover: if next.ServeHTTP panics, recover() catches the panic
		// value and lets us handle it gracefully instead of crashing.
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Printf("  [RECOVERY] panic caught: %v\n", rec)
				// At this point the ResponseWriter may have already had headers
				// written by the panicking handler.  We can only write a new
				// status if WriteHeader hasn't been called yet.
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "internal server error",
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// ── Chain helper ──────────────────────────────────────────────────────────────

// Chain applies a list of middlewares to a handler in order.
// The FIRST middleware in the slice is the OUTERMOST wrapper:
//
//   Chain(h, A, B, C) = A(B(C(h)))
//
// So the call order for a request is:  A → B → C → h → C → B → A
//
// This is the standard "onion model": first middleware = outermost layer.
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	// We iterate in reverse so that middlewares[0] ends up outermost.
	// After the loop: h = middlewares[0](middlewares[1](middlewares[2](original h)))
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// ── Business handlers used in demonstration ───────────────────────────────────

// protectedHandler requires auth and returns a success message.
func protectedHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "you are authenticated!",
		"user":    "alice",
	})
}

// panicHandler intentionally panics to demonstrate RecoveryMiddleware.
func panicHandler(w http.ResponseWriter, r *http.Request) {
	panic("something went terribly wrong")
}

// ── Demonstration ─────────────────────────────────────────────────────────────

func middlewareExample() {
	fmt.Println("--- Middleware chain (decorator pattern) ---")

	// Build the middleware-wrapped handler.
	//
	// Call order when a request arrives:
	//   RecoveryMiddleware → LoggingMiddleware → AuthMiddleware → handler
	//
	// The outermost middleware (Recovery) runs first and last.
	// It wraps everything so panics from ANY inner layer are caught.
	protected := Chain(
		http.HandlerFunc(protectedHandler),
		RecoveryMiddleware, // outermost — catches panics from everything below
		LoggingMiddleware,  // logs start time before, duration after
		AuthMiddleware,     // checks token; rejects if invalid
	)

	panicRoute := Chain(
		http.HandlerFunc(panicHandler),
		RecoveryMiddleware,
		LoggingMiddleware,
	)

	mux := http.NewServeMux()
	mux.Handle("/protected", protected)
	mux.Handle("/panic", panicRoute)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	fmt.Printf("  Test server: %s\n\n", ts.URL)

	// ── Scenario 1: missing auth token ───────────────────────────────────────
	fmt.Println("  — Request without auth token:")
	resp, body := doRequest("GET", ts.URL+"/protected", "", nil)
	fmt.Printf("    status: %d\n    body:   %s\n\n", resp.StatusCode, body)

	// ── Scenario 2: valid auth token ─────────────────────────────────────────
	fmt.Println("  — Request with valid token:")
	req, _ := http.NewRequest("GET", ts.URL+"/protected", nil)
	req.Header.Set("Authorization", "Bearer secret-token-123")
	resp2, err2 := http.DefaultClient.Do(req)
	if err2 == nil {
		defer resp2.Body.Close()
		b, _ := io.ReadAll(resp2.Body)
		fmt.Printf("    status: %d\n    body:   %s\n\n", resp2.StatusCode, strings.TrimSpace(string(b)))
	}

	// ── Scenario 3: panic recovery ───────────────────────────────────────────
	fmt.Println("  — Request to panicking handler:")
	resp, body = doRequest("GET", ts.URL+"/panic", "", nil)
	fmt.Printf("    status: %d\n    body:   %s\n\n", resp.StatusCode, body)

	fmt.Println("  Middleware call stack:")
	fmt.Println("    RecoveryMiddleware  ← outermost; catches panics from inner layers")
	fmt.Println("    LoggingMiddleware   ← records time before, logs duration after")
	fmt.Println("    AuthMiddleware      ← short-circuits if token invalid")
	fmt.Println("    handler             ← runs only when all middleware passes")
}
