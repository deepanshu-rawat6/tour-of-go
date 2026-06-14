package generics

import (
	"fmt"
	"os"
)

// Run executes all generics examples
func Run() {
	fmt.Println("=== Generics ===")
	fmt.Println()

	basicGenericsExample()
	fmt.Println()

	typeConstraintsExample()
	fmt.Println()

	genericTypesExample()
	fmt.Println()
}

// RunExample runs a specific generics example by name
func RunExample(name string) {
	fmt.Printf("=== Generics: %s ===\n\n", name)

	switch name {
	case "basic":
		basicGenericsExample()
	case "type-constraints":
		typeConstraintsExample()
	case "generic-types":
		genericTypesExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  basic")
		fmt.Println("  type-constraints")
		fmt.Println("  generic-types")
		os.Exit(1)
	}
}
