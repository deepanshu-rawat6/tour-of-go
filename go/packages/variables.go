package packages

import "fmt"

/*

The var statement declares a list of variables; as in function argument lists, the type is last.

A var statement can be at package or function level. We see both in this example.

*/

/*

 Inside a function, the := short assignment statement can be used in place of a var declaration with implicit type.

Outside a function, every statement begins with a keyword (var, func, and so on) and so the := construct is not available.

*/

var c, python, java bool

func variablesExample() {
	var i int

	fmt.Println("Variables:")

	fmt.Println(i, c, java, python)

	fmt.Println("Shor Variables declaration:")

	// short var declaration
	k := 3

	fmt.Println(i, j, k, c, java, python)
}
