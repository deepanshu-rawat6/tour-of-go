package packages

import (
	"fmt"
	"math"
)

// exportedNamesExample demonstrates exported vs unexported names
func exportedNamesExample() {
	fmt.Println("Exported Names:")
	// Pi ~ (22/7)
	fmt.Printf("math.Pi = %v (exported, starts with capital)\n", math.Pi)
	// exponential
	fmt.Printf("math.E = %v (exported, starst with capital)\n", math.E)
	// golden ratio
	fmt.Printf("math.Phi = %v (exported, starst with capital)\n", math.Phi)
	//fmt.Println(math.pi) // Would fail - unexported name
	fmt.Println("In Go, a name is exported if it begins with a capital letter")
}
