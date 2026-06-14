package methods

import "fmt"

type Rectangle struct {
	Width, Height float64
}

// Area uses a value receiver — it gets a copy of the Rectangle.
// Use value receivers when the method doesn't need to modify the struct.
func (r Rectangle) Area() float64 {
	return r.Width * r.Height
}

func (r Rectangle) Perimeter() float64 {
	return 2 * (r.Width + r.Height)
}

func valueReceiversExample() {
	fmt.Println("Value Receivers:")
	r := Rectangle{Width: 5, Height: 3}
	fmt.Printf("  Rectangle{%.0f, %.0f}\n", r.Width, r.Height)
	fmt.Printf("  Area:      %.1f\n", r.Area())
	fmt.Printf("  Perimeter: %.1f\n", r.Perimeter())

	// Value receiver: original is unchanged even if method modified a copy
	fmt.Println("\n  Value receivers work on copies — original is never mutated")
}
