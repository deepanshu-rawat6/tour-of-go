package more_types

import (
	"fmt"
	"os"
)

// Run executes all examples in the packages topic
func Run() {
	fmt.Println("=== More Types Statements ===")
	fmt.Println()

	pointersExample()
	fmt.Println()

	structExample()
	fmt.Println()

	arraysExample()
	fmt.Println()

	slicesExample()
	fmt.Println()
}

// RunExample runs a specific example by name
func RunExample(name string) {
	fmt.Printf("=== Packages & Imports: %s ===\n\n", name)

	switch name {
	case "pointers":
		pointersExample()
	case "struct":
		structExample()
	case "arrays":
		arraysExample()
	case "slices":
		slicesExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  pointers")
		fmt.Println("  struct")
		fmt.Println("  arrays")
		os.Exit(1)
	}
}
