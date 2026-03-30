package generics

import "fmt"

// Number is a custom type constraint — a union of numeric types.
type Number interface {
	int | int32 | int64 | float32 | float64
}

// Sum adds all elements of a slice. Works for any Number type.
func Sum[T Number](nums []T) T {
	var total T
	for _, n := range nums {
		total += n
	}
	return total
}

// Ordered is a constraint for types that support comparison operators.
type Ordered interface {
	int | float64 | string
}

// Contains reports whether target exists in the slice.
// Uses the comparable constraint so == works.
func Contains[T comparable](s []T, target T) bool {
	for _, v := range s {
		if v == target {
			return true
		}
	}
	return false
}

func typeConstraintsExample() {
	fmt.Println("Custom Number constraint:")
	ints := []int{1, 2, 3, 4, 5}
	fmt.Println("  Sum(ints):", Sum(ints))

	floats := []float64{1.1, 2.2, 3.3}
	fmt.Println("  Sum(floats):", Sum(floats))

	fmt.Println("\ncomparable constraint:")
	fmt.Println("  Contains([1,2,3], 2):", Contains([]int{1, 2, 3}, 2))
	fmt.Println("  Contains([\"a\",\"b\"], \"c\"):", Contains([]string{"a", "b"}, "c"))
}
