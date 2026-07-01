// Package net_http demonstrates the Go standard library's net/http package
// from multiple angles: routing, middleware, transport tuning, and client usage.
//
// Each sub-file is self-contained and uses httptest.NewServer so no real
// network port needs to be open when running examples.
//
// Run all examples:
//
//	go run . net_http
//
// Run a specific example:
//
//	go run . net_http handler
//	go run . net_http middleware
//	go run . net_http transport
//	go run . net_http client
package net_http

import (
	"fmt"
	"os"
)

// Run executes all net/http examples in a logical learning order.
func Run() {
	fmt.Println("=== net_http ===")
	fmt.Println()

	// 1. Handlers & routing — how the server side works
	handlerExample()
	fmt.Println()

	// 2. Middleware — decorator chain wrapping handlers
	middlewareExample()
	fmt.Println()

	// 3. Transport — connection pool, timeouts, retries
	transportExample()
	fmt.Println()

	// 4. Client — making requests with headers, bodies, error handling
	clientExample()
	fmt.Println()
}

// RunExample runs a single net/http example by name.
func RunExample(name string) {
	fmt.Printf("=== net_http: %s ===\n\n", name)

	switch name {
	case "handler":
		handlerExample()
	case "middleware":
		middlewareExample()
	case "transport":
		transportExample()
	case "client":
		clientExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  handler    - Custom http.Handler, HandlerFunc, ServeMux routing, httptest")
		fmt.Println("  middleware - Logging, Auth, Recovery middleware chain (decorator pattern)")
		fmt.Println("  transport  - Connection pooling, timeouts, retries with backoff")
		fmt.Println("  client     - GET/POST with headers, bodies, LimitReader, error handling")
		os.Exit(1)
	}
}
