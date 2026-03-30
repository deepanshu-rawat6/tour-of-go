package methods

import "fmt"

type Point struct {
	X, Y int
}

// String implements the fmt.Stringer interface.
// fmt.Println automatically calls String() if it exists.
func (p Point) String() string {
	return fmt.Sprintf("(%d, %d)", p.X, p.Y)
}

type Color int

const (
	Red Color = iota
	Green
	Blue
)

// String on a named type — makes enum-like types print nicely.
func (c Color) String() string {
	switch c {
	case Red:
		return "Red"
	case Green:
		return "Green"
	case Blue:
		return "Blue"
	default:
		return "Unknown"
	}
}

func stringerExample() {
	fmt.Println("fmt.Stringer interface:")

	p := Point{X: 3, Y: 7}
	fmt.Println("  Point:", p)           // calls p.String() automatically
	fmt.Printf("  Formatted: %v\n", p)  // also calls String()
	fmt.Printf("  Explicit:  %s\n", p)

	fmt.Println("\n  Colors (named type with String()):")
	for _, c := range []Color{Red, Green, Blue} {
		fmt.Printf("    %d = %s\n", c, c)
	}
}
