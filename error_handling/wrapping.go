package error_handling

import (
	"errors"
	"fmt"
)

// dbQuery simulates a low-level database error
func dbQuery(id int) error {
	if id == 0 {
		return ErrNotFound
	}
	return nil
}

// getUser wraps the db error with context using %w
func getUser(id int) error {
	if err := dbQuery(id); err != nil {
		// %w wraps the error — preserves the chain for errors.Is / errors.As
		return fmt.Errorf("getUser(%d): %w", id, err)
	}
	return nil
}

// loadProfile wraps getUser, adding another layer
func loadProfile(id int) error {
	if err := getUser(id); err != nil {
		return fmt.Errorf("loadProfile: %w", err)
	}
	return nil
}

func wrappingExample() {
	fmt.Println("Error wrapping with %w:")

	err := loadProfile(0)
	fmt.Println("  Full error chain:", err)

	// errors.Is unwraps the entire chain to find ErrNotFound
	fmt.Println("  errors.Is(ErrNotFound):", errors.Is(err, ErrNotFound))

	// Unwrap manually to see the chain
	fmt.Println("\n  Unwrapping chain:")
	for e := err; e != nil; e = errors.Unwrap(e) {
		fmt.Printf("    → %v\n", e)
	}
}
