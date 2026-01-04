package more_types

import "fmt"

// Vertex
// A struct is a collection of fields.
// Struct fields are accessed using a dot.
//
// Struct fields can be accessed through a struct pointer.
// To access the field X of a struct when we have the struct pointer p we could write (*p).X.
// However, that notation is cumbersome, so the language permits us instead to write just p.X, without the explicit dereference.
// /*
type Vertex struct {
	X, Y int
}

/*

Struct Literals

 A struct literal denotes a newly allocated struct value by listing the values of its fields.

 You can list just a subset of fields by using the Name: syntax. (And the order of named fields is irrelevant.)

 The special prefix & returns a pointer to the struct value.

*/

var (
	v1 = Vertex{1, 2}  // has type Vertex
	v2 = Vertex{X: 1}  // Y: 0 is implicit
	v3 = Vertex{}      // X: 0 and Y: 0
	px = &Vertex{1, 2} // has type *Vertex
)

func structExample() {
	fmt.Println("Struct Example:")

	fmt.Println(Vertex{1, 2})

	fmt.Println("Struct Fields:")
	v := Vertex{1, 2}
	v.X = 4
	fmt.Println(v.X)

	// Here Vertex{4, 2} --> Vertex{1e9, 2}
	fmt.Println("Pointers to Struct:")
	p := &v
	p.X = 1e9
	fmt.Println(v)
	(*p).X = 100
	fmt.Println(v)

	fmt.Println("Struct Literals:")
	fmt.Println(v1, px, v2, v3)
}
