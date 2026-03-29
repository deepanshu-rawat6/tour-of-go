package more_types

import (
	"fmt"
	"math"
)

/*
Functions are values too. They can be passed around just like other values.

Function values may be used as function arguments and return values.
*/

func compute(fn func(float64, float64) float64) float64 {
	return fn(3, 4)
}

func functionValues() {
	hypot := func(x, y float64) float64 {
		return math.Sqrt(x*x + y*y)
	}
	// 13
	fmt.Println(hypot(5, 12))

	// hypot(x, y float 64) float 64
	fmt.Println(compute(hypot))

	// math.Pow(x, y float 64) float 64
	fmt.Println(compute(math.Pow))
}

func functionValuesExample() {
	fmt.Println("Function Values in Go:")
	functionValues()
}
