package more_types

import (
	"fmt"
	"os"
)

// Run executes all examples in the packages topic
func Run() {
	fmt.Println("=== More Types Statements ===")
	fmt.Println()

	pointers()
	fmt.Println()
}

// RunExample runs a specific example by name
func RunExample(name string) {
	fmt.Printf("=== Packages & Imports: %s ===\n\n", name)

	switch name {
	case "pointers":
		pointers()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  pointers")
		os.Exit(1)
	}
}
