package packages

import (
	"fmt"
	"math"
	"math/rand"
)

// basicExample demonstrates importing and using packages
func basicExample() {
	fmt.Println("Basic Packages & Imports:")
	fmt.Println("My favourite number is", rand.Intn(10))
	fmt.Printf("Now you have %g problems.\n", math.Sqrt(36))
}
