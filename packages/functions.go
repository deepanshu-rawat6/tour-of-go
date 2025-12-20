package packages

import "fmt"

//func add(x int, y int) int {
//	return x + y
//}

// So, if you have variables of same types then we can use:
func add(x, y int) int {
	return x + y
}

func functionsExample() {
	fmt.Println("Functions:")

	x := 42
	y := 32

	fmt.Printf("The addition of %v & %v : %v \n", x, y, add(x, y))
}
