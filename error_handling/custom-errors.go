package error_handling

import (
	"errors"
	"fmt"
)

// Sentinel errors — package-level variables for comparison with errors.Is
var (
	ErrNotFound   = errors.New("not found")
	ErrPermission = errors.New("permission denied")
)

// ValidationError is a custom error type carrying extra context
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed on %q: %s", e.Field, e.Message)
}

func findUser(id int) error {
	if id <= 0 {
		return &ValidationError{Field: "id", Message: "must be positive"}
	}
	if id > 100 {
		return ErrNotFound
	}
	return nil
}

func customErrorsExample() {
	fmt.Println("Custom errors and sentinels:")

	// Sentinel error check
	err := findUser(999)
	if errors.Is(err, ErrNotFound) {
		fmt.Println("  Sentinel match: user not found")
	}

	// Custom error type — use errors.As to extract the struct
	err = findUser(-1)
	var ve *ValidationError
	if errors.As(err, &ve) {
		fmt.Printf("  ValidationError: field=%q msg=%q\n", ve.Field, ve.Message)
	}

	// Success case
	err = findUser(42)
	fmt.Println("  findUser(42):", err)
}
