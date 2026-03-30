package error_handling

import (
	"errors"
	"fmt"
)

func divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

func basicErrorExample() {
	fmt.Println("Basic error handling:")

	result, err := divide(10, 2)
	if err != nil {
		fmt.Println("  Error:", err)
	} else {
		fmt.Println("  10 / 2 =", result)
	}

	_, err = divide(5, 0)
	if err != nil {
		fmt.Println("  Error:", err)
	}
}
