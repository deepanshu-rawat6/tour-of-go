package flow_control_statements

import "fmt"

func forLoop() {
	fmt.Println("For Loop:")

	sum := 0
	for i := 0; i < 10; i++ {
		sum += i
	}
	fmt.Printf("The sum of numbers from 0 to 9 : %v\n", sum)

	fmt.Println()

	fmt.Println("Initial and post conditions are optional:")
	//	Another variation
	// The initial and the post conditions are optional
	k := 1
	for k < 1000 {
		k += k
	}

	fmt.Println(k)

	fmt.Println()

	fmt.Println("For loop acting like while loop:")
	//	For loop acting as C's while loop
	j := 1
	for j < 1000 {
		j += j
	}

	fmt.Println(j)

	fmt.Println()

	fmt.Println("Infinite for loop(commented out):")
	// Infinite loop
	//for {
	//
	//}

}
