package more_types

import "fmt"

func pointers() {

	fmt.Println("Pointers in Go:")

	i, j := 42, 100

	p := &i
	fmt.Println(*p)
	*p = 21
	fmt.Println(i)

	p = &j
	*p = *p / 4
	fmt.Println(j)
}
