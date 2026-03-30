package interfaces

import "fmt"

func typeAssertionsExample() {
	fmt.Println("Type Assertions:")

	var s Shape = Circle{Radius: 3}

	// Single-value assertion — panics if wrong type
	c := s.(Circle)
	fmt.Printf("  Asserted Circle radius: %.1f\n", c.Radius)

	// Comma-ok pattern — safe, no panic
	if r, ok := s.(Rect); ok {
		fmt.Println("  It's a Rect:", r)
	} else {
		fmt.Println("  Not a Rect (safe assertion with ok)")
	}

	fmt.Println("\nType Switch:")
	describe := func(i interface{}) {
		switch v := i.(type) {
		case int:
			fmt.Printf("  int: %d\n", v)
		case string:
			fmt.Printf("  string: %q\n", v)
		case bool:
			fmt.Printf("  bool: %v\n", v)
		case Circle:
			fmt.Printf("  Circle with radius %.1f\n", v.Radius)
		default:
			fmt.Printf("  unknown type: %T\n", v)
		}
	}

	describe(42)
	describe("hello")
	describe(true)
	describe(Circle{Radius: 2.5})
	describe(3.14)
}
