package packages

import "fmt"

var i, j int = 1, 2

func variablesWithInitializersExample() {

	fmt.Println("Variables with Initializers:")

	var c, python, java = true, false, "no!"

	fmt.Println(i, c, python, java, j)
}
