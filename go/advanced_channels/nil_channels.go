// nil_channels.go — nil channels and the "disable a select case" trick.
//
// RULE: A nil channel blocks forever on both send and receive.
//
// In a select statement, a case that would block forever is simply never chosen.
// That means: setting a channel variable to nil REMOVES that case from
// consideration — it can never fire.
//
// This gives you dynamic control over which cases in a select are active,
// without restructuring your entire select block.
//
// CLASSIC INTERVIEW TRICK:
//   "How do you merge two channels and stop reading from one when it's exhausted?"
//   Answer: nil the channel after the source closes.
package advanced_channels

import "fmt"

// nilChannelExample runs all nil-channel sub-demos.
func nilChannelExample() {
	fmt.Println("1. A nil channel blocks forever")
	nilBlocks()

	fmt.Println("\n2. Nil channel disables a select case")
	nilDisablesCase()

	fmt.Println("\n3. Merging two event streams, ignoring exhausted ones")
	mergeStreams()

	fmt.Println("\n4. Dynamic enable/disable of work items")
	dynamicEnableDisable()
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. A nil channel blocks forever on both send AND receive
// ─────────────────────────────────────────────────────────────────────────────

func nilBlocks() {
	// var ch chan int  →  ch is nil
	var ch chan int

	// Attempting to send or receive from a nil channel blocks forever.
	// If this were outside a select, the program would deadlock.
	// In a select, the nil case is simply never chosen.
	select {
	case v := <-ch:
		// This case can NEVER fire because ch is nil.
		fmt.Println("  received:", v) // unreachable
	default:
		// The default case fires because the nil receive would block.
		fmt.Println("  nil receive would block → default chosen")
	}

	select {
	case ch <- 42:
		// This case can NEVER fire because ch is nil.
		fmt.Println("  sent 42") // unreachable
	default:
		fmt.Println("  nil send would block → default chosen")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. Nil channel disables a select case
// ─────────────────────────────────────────────────────────────────────────────

func nilDisablesCase() {
	chA := make(chan string, 1)
	chB := make(chan string, 1)

	chA <- "from A"
	chB <- "from B"

	// Both channels have a value ready. We read from A, then DISABLE A by
	// setting the variable to nil. The next select iteration can only choose B.
	var a, b chan string = chA, chB

	for i := 0; i < 3; i++ {
		select {
		case v, ok := <-a:
			if !ok || v == "" {
				// Channel closed or nil — disable it
				a = nil
				fmt.Println("  a exhausted → setting a = nil")
				continue
			}
			fmt.Printf("  round %d: got %q from a\n", i, v)
			a = nil // disable a after first read for demo

		case v, ok := <-b:
			if !ok || v == "" {
				b = nil
				fmt.Println("  b exhausted → setting b = nil")
				continue
			}
			fmt.Printf("  round %d: got %q from b\n", i, v)

		default:
			// Both a and b are nil → default fires
			fmt.Printf("  round %d: both channels nil, nothing to read\n", i)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. Merge two event streams; stop reading from one when it's exhausted
// ─────────────────────────────────────────────────────────────────────────────

// produce sends n values on a channel then closes it.
func produce(n int, label string) <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)
		for i := 1; i <= n; i++ {
			ch <- fmt.Sprintf("%s:%d", label, i)
		}
	}()
	return ch
}

// mergeTwo merges two channels into one, stopping each source when closed.
// The nil trick avoids the need to track two separate boolean flags.
func mergeTwo(c1, c2 <-chan string) <-chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		// We copy c1 and c2 into local variables so we can nil them.
		// (We cannot assign to a function parameter declared as <-chan.)
		ch1 := c1
		ch2 := c2

		for ch1 != nil || ch2 != nil {
			// As long as at least one channel is still active, keep looping.
			select {
			case v, ok := <-ch1:
				if !ok {
					// ch1 was closed. Nil it so this case is never selected again.
					// Without this, a closed channel returns zero values forever,
					// causing an infinite loop.
					ch1 = nil
					fmt.Println("  stream1 exhausted, disabling")
					continue
				}
				out <- v

			case v, ok := <-ch2:
				if !ok {
					ch2 = nil
					fmt.Println("  stream2 exhausted, disabling")
					continue
				}
				out <- v
			}
		}
		// Both channels are now nil → the for condition is false → we exit.
	}()
	return out
}

func mergeStreams() {
	// stream1 sends 2 values, stream2 sends 3 values
	merged := mergeTwo(produce(2, "s1"), produce(3, "s2"))
	for v := range merged {
		fmt.Printf("  merged: %s\n", v)
	}
	fmt.Println("  both streams done")
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. Dynamic enable/disable of work items
// ─────────────────────────────────────────────────────────────────────────────

// dynamicEnableDisable shows a scheduler-like pattern where you enable or
// disable processing of a work channel based on runtime state.
func dynamicEnableDisable() {
	work := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		work <- i
	}
	close(work)

	// pauseCh controls whether we process work.
	// When it's nil, the work case is disabled (paused).
	// When it's set to 'work', the work case is active (running).
	var activeWork <-chan int = work

	pause := false
	processed := 0

	for {
		select {
		case v, ok := <-activeWork:
			if !ok {
				fmt.Printf("  work channel closed, processed %d items\n", processed)
				return
			}
			fmt.Printf("  processing item %d\n", v)
			processed++

			// After item 2, simulate a pause
			if processed == 2 && !pause {
				fmt.Println("  pausing work channel (setting to nil)")
				activeWork = nil // disables the work case
				pause = true

				// Re-enable after a simulated condition
				// (In real code this might be triggered by another goroutine)
				fmt.Println("  re-enabling work channel")
				activeWork = work
			}

		default:
			// Nothing to do — in production you'd block here or sleep
			if activeWork == nil {
				fmt.Println("  work is paused, nothing to do")
				activeWork = work // re-enable for demo
			}
		}
	}
}
