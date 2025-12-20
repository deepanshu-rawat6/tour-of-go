package packages

import "fmt"

func typeInference() {
	fmt.Println("Type Inference:")
	v := 42
	fmt.Printf("v is of type %T\n", v)
}
