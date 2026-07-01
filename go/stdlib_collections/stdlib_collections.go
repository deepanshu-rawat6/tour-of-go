// Package stdlib_collections demonstrates the Go 1.21+ standard library
// packages: slices, maps, and cmp.
//
// These packages were added in Go 1.21 and provide generic, type-safe
// utilities that replace hand-rolled helper functions. They use Go generics
// under the hood (constraints from golang.org/x/exp were upstreamed).
//
// Run:
//
//	go run . stdlib_collections          # run all examples
//	go run . stdlib_collections slices   # run just the slices example
//	go run . stdlib_collections maps     # run just the maps example
//	go run . stdlib_collections cmp      # run just the cmp example
package stdlib_collections

import (
	"fmt"
	"os"
)

// Run executes all stdlib_collections examples in order.
func Run() {
	fmt.Println("=== stdlib_collections ===")
	fmt.Println()

	slicesExample()
	fmt.Println()

	mapsExample()
	fmt.Println()

	cmpExample()
	fmt.Println()
}

// RunExample runs a specific stdlib_collections example by name.
//
// Supported names: "slices", "maps", "cmp"
func RunExample(name string) {
	fmt.Printf("=== stdlib_collections: %s ===\n\n", name)

	switch name {
	case "slices":
		slicesExample()
	case "maps":
		mapsExample()
	case "cmp":
		cmpExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  slices  - slices.Contains, Sort, BinarySearch, Compact, Insert, Delete …")
		fmt.Println("  maps    - maps.Keys, Values, Clone, Copy, Equal, Delete …")
		fmt.Println("  cmp     - cmp.Compare, Equal, Or; multi-key sorts")
		os.Exit(1)
	}
}
