package flow_control_statements

import (
	"fmt"
	"math"
)

func sqrt(x float64) string {
	if x < 0 {
		return sqrt(-x) + "i"
	} else {
		return fmt.Sprint(math.Sqrt(x))
	}
}

// If with a short statement
//func pow(x, n, lim float64) float64 {
//	if v := math.Pow(x, n); v < lim {
//		return v
//	}
//	return lim
//}

func pow(x, n, lim float64) float64 {
	if v := math.Pow(x, n); v < lim {
		return v
	} else {
		fmt.Printf("%g >= %g\n", v, lim)
	}
	return lim
}

func ifStatement() {
	fmt.Println("If Statements:")

	fmt.Println(sqrt(2), sqrt(-4))

	fmt.Println()

	fmt.Println("If with a short statement")

	fmt.Println(pow(3, 2, 10))
	fmt.Println(pow(3, 3, 20))
}
