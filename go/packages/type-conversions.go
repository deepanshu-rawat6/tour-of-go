package packages

import (
	"fmt"
	"math"
)

/*
Unlike in C, in Go assignment between items of different type requires an explicit conversion.
Try removing the float64 or uint conversions in the example and see what happens.
*/

//var i int = 42
//var f float64 = float64(i)
//var u uint = uint(f)

// OR

//i := 42
//f := float64(i)
//u := uint(f)

func typeConversions() {
	fmt.Println("Type Conversions:")
	var x, y int = 3, 4
	var f float64 = math.Sqrt(float64(x*x + y*y))
	var z uint = uint(f)
	fmt.Println(x, y, z)
}
