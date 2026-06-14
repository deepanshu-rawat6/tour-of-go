package more_types

import (
	"fmt"
)

/*
Go functions may be closures. A closure is a function value that references variables from outside its body.
The function may access and assign to the referenced variables; in this sense the function is "bound" to the variables.

For example, the adder function returns a closure. Each closure is bound to its own sum variable.
*/

func adder() func(int) int {
	sum := 0
	return func(x int) int {
		sum += x
		return sum
	}
}

func functionClosures() {
	pos, neg := adder(), adder()
	for i := 0; i < 10; i++ {
		fmt.Println(
			pos(i),
			neg(-2*i),
		)
	}
}

// fibonacci is a function that returns
// a function that returns an int.
func fibonacci() func() int {
	f0, f1 := 0, 1
	return func() int {
		ret := f0
		f0, f1 = f1, f0+f1
		return ret
	}
}

func fibonacciExercise() {
	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}
}

func functionClosuresExample() {
	fmt.Println("Function Closures in Go:")
	functionClosures()

	fmt.Println("Finonacci Example:")
	fibonacciExercise()
}
