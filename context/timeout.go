package ctx_examples

import (
	"context"
	"fmt"
	"time"
)

// slowOperation simulates a slow external call (e.g., DB query, HTTP request)
func slowOperation(ctx context.Context) error {
	select {
	case <-time.After(50 * time.Millisecond): // takes 50ms
		return nil
	case <-ctx.Done():
		return ctx.Err() // context.DeadlineExceeded or context.Canceled
	}
}

func timeoutExample() {
	fmt.Println("context.WithTimeout:")

	// Case 1: timeout is generous — operation succeeds
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := slowOperation(ctx)
	fmt.Println("  100ms timeout, 50ms op:", err) // nil

	// Case 2: timeout is tight — operation is cancelled
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel2()
	err = slowOperation(ctx2)
	fmt.Println("  10ms timeout, 50ms op:", err) // context deadline exceeded

	// Always defer cancel() — even if the deadline fires, it releases resources
	fmt.Println("\n  Rule: always defer cancel() immediately after WithTimeout/WithCancel")
}
