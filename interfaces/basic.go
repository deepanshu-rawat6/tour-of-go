package interfaces

import (
	"fmt"
	"math"
)

// Shape is an interface — any type with Area() and Perimeter() satisfies it.
// Go uses IMPLICIT satisfaction: no "implements" keyword needed.
type Shape interface {
	Area() float64
	Perimeter() float64
}

type Circle struct{ Radius float64 }
type Rect struct{ W, H float64 }

func (c Circle) Area() float64      { return math.Pi * c.Radius * c.Radius }
func (c Circle) Perimeter() float64 { return 2 * math.Pi * c.Radius }
func (r Rect) Area() float64        { return r.W * r.H }
func (r Rect) Perimeter() float64   { return 2 * (r.W + r.H) }

// printShape accepts any Shape — polymorphism via interface
func printShape(s Shape) {
	fmt.Printf("  %T: area=%.2f, perimeter=%.2f\n", s, s.Area(), s.Perimeter())
}

func basicInterfaceExample() {
	fmt.Println("Implicit interface satisfaction:")
	shapes := []Shape{
		Circle{Radius: 5},
		Rect{W: 4, H: 6},
	}
	for _, s := range shapes {
		printShape(s)
	}
}
