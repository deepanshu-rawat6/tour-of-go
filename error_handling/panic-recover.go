package error_handling

import "fmt"

// safeDiv demonstrates recover() — catching a panic and returning an error
func safeDiv(a, b int) (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic: %v", r)
		}
	}()
	// This panics if b == 0 (integer division by zero)
	result = a / b
	return result, nil
}

func panicRecoverExample() {
	fmt.Println("Panic and Recover:")

	// Normal case
	result, err := safeDiv(10, 2)
	fmt.Printf("  safeDiv(10, 2): result=%d, err=%v\n", result, err)

	// Panic case — recover catches it
	result, err = safeDiv(10, 0)
	fmt.Printf("  safeDiv(10, 0): result=%d, err=%v\n", result, err)

	fmt.Println("\n  Rule: use panic only for truly unrecoverable programmer errors")
	fmt.Println("  Rule: use error returns for expected failure conditions")
}
