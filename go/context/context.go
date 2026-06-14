package ctx_examples

import (
	"fmt"
	"os"
)

// Run executes all context examples
func Run() {
	fmt.Println("=== Context ===")
	fmt.Println()

	cancellationExample()
	fmt.Println()

	timeoutExample()
	fmt.Println()

	valuesExample()
	fmt.Println()
}

// RunExample runs a specific context example by name
func RunExample(name string) {
	fmt.Printf("=== Context: %s ===\n\n", name)

	switch name {
	case "cancellation":
		cancellationExample()
	case "timeout":
		timeoutExample()
	case "values":
		valuesExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  cancellation")
		fmt.Println("  timeout")
		fmt.Println("  values")
		os.Exit(1)
	}
}
