package methods

import (
	"fmt"
	"os"
)

// Run executes all methods examples
func Run() {
	fmt.Println("=== Methods ===")
	fmt.Println()

	valueReceiversExample()
	fmt.Println()

	pointerReceiversExample()
	fmt.Println()

	stringerExample()
	fmt.Println()
}

// RunExample runs a specific methods example by name
func RunExample(name string) {
	fmt.Printf("=== Methods: %s ===\n\n", name)

	switch name {
	case "value-receivers":
		valueReceiversExample()
	case "pointer-receivers":
		pointerReceiversExample()
	case "stringer":
		stringerExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  value-receivers")
		fmt.Println("  pointer-receivers")
		fmt.Println("  stringer")
		os.Exit(1)
	}
}
