package interfaces

import "fmt"

// printAny accepts any value — the empty interface (any) has no method requirements
func printAny(v any) {
	fmt.Printf("  value=%v, type=%T\n", v, v)
}

func emptyInterfaceExample() {
	fmt.Println("Empty interface (any):")
	printAny(42)
	printAny("hello")
	printAny([]int{1, 2, 3})
	printAny(nil)

	// Storing mixed types in a slice — only possible with any
	fmt.Println("\n  Mixed-type slice via []any:")
	mixed := []any{1, "two", 3.0, true}
	for _, v := range mixed {
		fmt.Printf("    %T: %v\n", v, v)
	}

	fmt.Println("\n  Note: avoid any in hot paths — it causes heap allocation (boxing)")
}
