package flow_control_statements

import "fmt"

// Sqrt /*

/*

 	As a way to play with functions and loops, let's implement a square root function: given a number x, we want to find
	the number z for which z² is most nearly x.

	Computers typically compute the square root of x using a loop.
	Starting with some guess z, we can adjust z based on how close z² is to x, producing a better guess:

*/

func Sqrt(x float64) float64 {
	c := 1
	z := 1.0

	for c <= 10 {
		z -= (z*z - x) / (2 * z)
		fmt.Printf("%v Iteration: %v\n", c, z)
		c += 1
	}

	return z
}

// sqrt using binary search
func sqrtUsingBinarySearch(x int) int {
	i := 0
	j := x
	num := 0

	for i <= j {
		num = i + (j-i)/2

		if num*num == x {
			return num
		} else if num*num > x {
			j = num - 1
		} else {
			i = num + 1
		}
	}

	return i - 1
}

func exerciseLoopsAndFunctions() {
	fmt.Println("Exercise of loops and functions:")

	fmt.Println(sqrtUsingBinarySearch(101))

	fmt.Println(Sqrt(float64(2)))
}
