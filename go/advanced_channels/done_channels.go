// done_channels.go — the "done channel" cancellation pattern.
//
// HISTORY:
// Before context.Context was added in Go 1.7, the idiomatic way to signal
// cancellation across goroutines was a "done" channel: a channel of struct{}
// that is closed when work should stop.
//
// Closing a channel broadcasts to ALL receivers simultaneously (unlike sending
// one value, which wakes only ONE receiver). That broadcast property is the key.
//
// KEY INSIGHT:
//   - send on done: only one goroutine wakes up
//   - close(done): ALL goroutines waiting on done wake up immediately
//
// Today, context.Context is preferred, but understanding done channels helps
// you read older code and understand what context.WithCancel does under the hood.
package advanced_channels

import (
	"context"
	"fmt"
	"time"
)

// doneChannelExample runs all sub-demos of the done channel pattern.
func doneChannelExample() {
	fmt.Println("1. Basic done channel (goroutine leak demo + fix)")
	leakAndFix()

	fmt.Println("\n2. Select with done: work OR timeout")
	selectWithDone()

	fmt.Println("\n3. close(done) broadcasts to multiple goroutines")
	broadcastClose()

	fmt.Println("\n4. Modern equivalent: context.WithCancel")
	contextEquivalent()
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. Goroutine leak and the done-channel fix
// ─────────────────────────────────────────────────────────────────────────────

// leakingGenerator starts a goroutine that sends integers forever.
// PROBLEM: if the caller stops reading, the goroutine is stuck trying to send
// on a channel nobody reads from — this is a goroutine LEAK.
func leakingGenerator() <-chan int {
	ch := make(chan int)
	go func() {
		// This goroutine will run forever if nobody reads from ch.
		// It leaks if the caller abandons ch without draining it.
		for i := 0; ; i++ {
			ch <- i // blocks if nobody is reading
		}
	}()
	return ch
}

// safeGenerator accepts a done channel.
// When done is closed, the goroutine exits cleanly — no leak.
func safeGenerator(done <-chan struct{}) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch) // signal to callers that we are done producing
		for i := 0; ; i++ {
			select {
			case <-done:
				// done was closed → time to stop; defer will close ch
				fmt.Println("  safeGenerator: received done signal, exiting")
				return
			case ch <- i:
				// successfully sent value i
			}
		}
	}()
	return ch
}

func leakAndFix() {
	// ── Leaking version (we only read 3 values, then abandon the channel) ──
	// In a real program this goroutine would live until the process exits.
	leakyCh := leakingGenerator()
	for i := 0; i < 3; i++ {
		fmt.Printf("  leaky: got %d\n", <-leakyCh)
	}
	// leakyCh is now abandoned — the goroutine inside leakingGenerator is
	// permanently blocked trying to send. That's the leak.
	fmt.Println("  leaky: caller stopped reading (goroutine leaked!)")

	// ── Fixed version using done channel ──
	done := make(chan struct{}) // zero-size struct uses no memory
	safeCh := safeGenerator(done)

	for i := 0; i < 3; i++ {
		fmt.Printf("  safe:  got %d\n", <-safeCh)
	}

	// close(done) broadcasts to the goroutine inside safeGenerator
	close(done)

	// Drain any last values that the goroutine sent before noticing done.
	// After ch is closed by the goroutine, this loop ends automatically.
	for v := range safeCh {
		_ = v
	}
	fmt.Println("  safe:  goroutine cleaned up, no leak")
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. select with done: do work OR timeout, whichever comes first
// ─────────────────────────────────────────────────────────────────────────────

// doWork simulates a task that takes some time.
// It sends its result on resultCh, or exits early if done is closed.
func doWork(done <-chan struct{}, resultCh chan<- string, workDuration time.Duration) {
	go func() {
		select {
		case <-time.After(workDuration):
			// Work completed before done was signalled
			resultCh <- fmt.Sprintf("work done after %v", workDuration)
		case <-done:
			// Cancelled before work finished
			resultCh <- "work cancelled"
		}
	}()
}

func selectWithDone() {
	resultCh := make(chan string, 1)
	timeout := 50 * time.Millisecond

	// Case A: work finishes before timeout
	done := make(chan struct{})
	doWork(done, resultCh, 10*time.Millisecond) // fast work
	select {
	case res := <-resultCh:
		fmt.Printf("  case A: %s\n", res)
	case <-time.After(timeout):
		fmt.Println("  case A: timed out")
	}
	close(done) // clean up even when work finishes normally

	// Case B: timeout fires before work finishes
	done2 := make(chan struct{})
	doWork(done2, resultCh, 200*time.Millisecond) // slow work
	select {
	case res := <-resultCh:
		fmt.Printf("  case B: %s\n", res)
	case <-time.After(timeout):
		fmt.Printf("  case B: timed out after %v, cancelling\n", timeout)
		close(done2) // signal the goroutine to stop
		// Wait for the goroutine to acknowledge cancellation
		fmt.Printf("  case B: ack: %s\n", <-resultCh)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. close(done) broadcasts to ALL goroutines simultaneously
// ─────────────────────────────────────────────────────────────────────────────

func broadcastClose() {
	done := make(chan struct{})
	fanCount := 3
	finished := make(chan string, fanCount)

	for i := 0; i < fanCount; i++ {
		i := i // capture loop variable
		go func() {
			select {
			case <-done:
				// Every goroutine receives this close simultaneously.
				// If we had sent a value instead of closing, only ONE would
				// wake up. close() is the broadcast primitive.
				finished <- fmt.Sprintf("goroutine %d stopped", i)
			}
		}()
	}

	time.Sleep(10 * time.Millisecond)

	// One close() wakes ALL goroutines at once.
	// Compare: sending N individual values would require a loop.
	close(done)

	for i := 0; i < fanCount; i++ {
		fmt.Printf("  broadcast: %s\n", <-finished)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. Modern equivalent: context.WithCancel
// ─────────────────────────────────────────────────────────────────────────────

// contextEquivalent shows how context.WithCancel is essentially a done channel
// with additional features (deadlines, values, error reporting).
func contextEquivalent() {
	// context.WithCancel returns a context whose Done() channel is closed
	// when cancel() is called — exactly like our done channel pattern.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // always call cancel to release resources

	results := make(chan int, 5)

	go func() {
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				// ctx.Done() returns a <-chan struct{} that is closed on cancel()
				// ctx.Err() tells us WHY: context.Canceled or context.DeadlineExceeded
				fmt.Printf("  context goroutine stopped: %v\n", ctx.Err())
				close(results)
				return
			case results <- i:
				// sent value successfully
			}
		}
	}()

	// Read 4 values then cancel
	for v := range results {
		fmt.Printf("  context: got %d\n", v)
		if v >= 3 {
			cancel() // equivalent to close(done)
		}
	}

	// SUMMARY:
	// done channel   → close(done)          → ctx.Done() channel is closed
	// no error info  → manual tracking      → ctx.Err() returns the reason
	// no deadline    → time.After workaround → context.WithTimeout / WithDeadline
	// no values      → separate params       → context.WithValue
}
