package concurrency

import (
	"fmt"
	"os"
)

// Run executes all concurrency examples
func Run() {
	fmt.Println("=== Concurrency ===")
	fmt.Println()

	goroutinesExample()
	fmt.Println()

	channelsExample()
	fmt.Println()

	selectExample()
	fmt.Println()

	mutexExample()
	fmt.Println()

	workerPoolExample()
	fmt.Println()
}

// RunExample runs a specific concurrency example by name
func RunExample(name string) {
	fmt.Printf("=== Concurrency: %s ===\n\n", name)

	switch name {
	case "goroutines":
		goroutinesExample()
	case "channels":
		channelsExample()
	case "select":
		selectExample()
	case "mutex":
		mutexExample()
	case "worker-pool":
		workerPoolExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  goroutines")
		fmt.Println("  channels")
		fmt.Println("  select")
		fmt.Println("  mutex")
		fmt.Println("  worker-pool")
		os.Exit(1)
	}
}
