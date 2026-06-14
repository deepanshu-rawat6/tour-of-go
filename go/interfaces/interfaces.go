package interfaces

import (
	"fmt"
	"os"
)

// Run executes all interfaces examples
func Run() {
	fmt.Println("=== Interfaces ===")
	fmt.Println()

	basicInterfaceExample()
	fmt.Println()

	typeAssertionsExample()
	fmt.Println()

	emptyInterfaceExample()
	fmt.Println()

	interfaceEmbeddingExample()
	fmt.Println()
}

// RunExample runs a specific interfaces example by name
func RunExample(name string) {
	fmt.Printf("=== Interfaces: %s ===\n\n", name)

	switch name {
	case "basic":
		basicInterfaceExample()
	case "type-assertions":
		typeAssertionsExample()
	case "empty-interface":
		emptyInterfaceExample()
	case "embedding":
		interfaceEmbeddingExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  basic")
		fmt.Println("  type-assertions")
		fmt.Println("  empty-interface")
		fmt.Println("  embedding")
		os.Exit(1)
	}
}
