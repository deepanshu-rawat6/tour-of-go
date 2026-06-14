package generics

import "fmt"

// genericMin returns the smaller of two ordered values.
// The type parameter T is constrained to types that support < operator.
func genericMin[T int | float64 | string](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Map applies a transform function to every element of a slice.
// Two type parameters: T (input) and U (output).
func Map[T, U any](s []T, f func(T) U) []U {
	result := make([]U, len(s))
	for i, v := range s {
		result[i] = f(v)
	}
	return result
}

func basicGenericsExample() {
	fmt.Println("Generic min function:")
	fmt.Println("  min(3, 5)       =", genericMin(3, 5))
	fmt.Println("  min(3.14, 2.71) =", genericMin(3.14, 2.71))
	fmt.Println("  min(\"go\", \"rust\") =", genericMin("go", "rust"))

	fmt.Println("\nGeneric Map function:")
	nums := []int{1, 2, 3, 4, 5}
	doubled := Map(nums, func(n int) int { return n * 2 })
	fmt.Println("  doubled:", doubled)

	strs := Map(nums, func(n int) string { return fmt.Sprintf("item-%d", n) })
	fmt.Println("  as strings:", strs)
}
