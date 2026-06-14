package ctx_examples

import (
	"context"
	"fmt"
	"time"
)

func cancellationExample() {
	fmt.Println("context.WithCancel:")

	ctx, cancel := context.WithCancel(context.Background())

	// Worker that respects cancellation
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				fmt.Println("  worker: context cancelled, stopping")
				return
			default:
				fmt.Println("  worker: doing work...")
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	time.Sleep(12 * time.Millisecond)
	cancel() // signal the worker to stop
	<-done   // wait for worker to finish
	fmt.Println("  main: worker stopped cleanly")
}
