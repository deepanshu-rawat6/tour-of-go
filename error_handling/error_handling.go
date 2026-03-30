package error_handling

import (
	"fmt"
	"os"
)

// Run executes all error handling examples
func Run() {
	fmt.Println("=== Error Handling ===")
	fmt.Println()

	basicErrorExample()
	fmt.Println()

	customErrorsExample()
	fmt.Println()

	wrappingExample()
	fmt.Println()

	panicRecoverExample()
	fmt.Println()
}

// RunExample runs a specific error handling example by name
func RunExample(name string) {
	fmt.Printf("=== Error Handling: %s ===\n\n", name)

	switch name {
	case "basic":
		basicErrorExample()
	case "custom-errors":
		customErrorsExample()
	case "wrapping":
		wrappingExample()
	case "panic-recover":
		panicRecoverExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  basic")
		fmt.Println("  custom-errors")
		fmt.Println("  wrapping")
		fmt.Println("  panic-recover")
		os.Exit(1)
	}
}
