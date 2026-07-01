// Package advanced_channels demonstrates advanced channel patterns in Go.
//
// This package goes beyond the basics covered in the concurrency package and
// explores patterns that appear constantly in production Go codebases:
//
//   - Done channels: the pre-context.Context cancellation pattern
//   - Nil channels: selectively disabling select cases at runtime
//   - Directional channels: self-documenting, contract-enforcing APIs
//   - Fan-out / Fan-in: distributing work and merging results
//
// Run all examples:
//
//	go run . advanced_channels
//
// Run a specific example:
//
//	go run . advanced_channels done
//	go run . advanced_channels nil
//	go run . advanced_channels directional
//	go run . advanced_channels fan-out
//	go run . advanced_channels fan-in
package advanced_channels

import (
	"fmt"
	"os"
)

// Run executes all advanced channel examples in sequence.
func Run() {
	fmt.Println("=== Advanced Channels ===")
	fmt.Println()

	fmt.Println("--- Done Channel Pattern ---")
	doneChannelExample()
	fmt.Println()

	fmt.Println("--- Nil Channel Pattern ---")
	nilChannelExample()
	fmt.Println()

	fmt.Println("--- Directional Channels ---")
	directionalExample()
	fmt.Println()

	fmt.Println("--- Fan-Out Pattern ---")
	fanOutExample()
	fmt.Println()

	fmt.Println("--- Fan-In Pattern ---")
	fanInExample()
	fmt.Println()
}

// RunExample runs a specific advanced channel example by name.
func RunExample(name string) {
	fmt.Printf("=== Advanced Channels: %s ===\n\n", name)

	switch name {
	case "done":
		doneChannelExample()
	case "nil":
		nilChannelExample()
	case "directional":
		directionalExample()
	case "fan-out":
		fanOutExample()
	case "fan-in":
		fanInExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  done        - Done channel cancellation pattern")
		fmt.Println("  nil         - Nil channel to disable select cases")
		fmt.Println("  directional - Send-only and receive-only channels")
		fmt.Println("  fan-out     - Distribute work across N goroutines")
		fmt.Println("  fan-in      - Merge N channels into one")
		os.Exit(1)
	}
}
