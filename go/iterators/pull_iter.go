//go:build go1.23

// pull_iter.go — iter.Pull: converting push iterators to pull iterators.
//
// PUSH vs PULL:
//
//   PUSH iterator (iter.Seq):
//     The iterator drives the loop. You give it a yield function and it calls
//     yield for each element. Simple to write, but you can only advance it
//     by "running" the loop.
//
//   PULL iterator (returned by iter.Pull):
//     The consumer drives the loop. You call next() to get the next value.
//     Easier to use in coroutine-style code (e.g., advancing two iterators
//     in lock-step for a zip operation).
//
// iter.Pull[V](seq iter.Seq[V]) (next func() (V, bool), stop func())
//
// CRITICAL: Always call stop() when you're done — even if you've read all values.
// stop() releases the goroutine that iter.Pull uses internally to simulate
// coroutine suspension.
package iterators

import (
	"fmt"
	"iter"
)

// pullIterExample demonstrates iter.Pull and its use cases.
func pullIterExample() {
	fmt.Println("1. Basic iter.Pull usage")
	basicPull()

	fmt.Println("\n2. When to use pull: zip two iterators in lock-step")
	zipExample()

	fmt.Println("\n3. Push vs Pull: when each is appropriate")
	pushVsPull()
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. Basic iter.Pull usage
// ─────────────────────────────────────────────────────────────────────────────

func basicPull() {
	// Create a push iterator (Seq)
	seq := upTo(5) // from range_func.go: yields 0,1,2,3,4

	// Convert it to a pull iterator.
	// iter.Pull starts an internal goroutine that pauses between yields.
	// next() resumes that goroutine one step.
	// stop() terminates the goroutine (MUST be called to avoid goroutine leak).
	next, stop := iter.Pull(seq)
	defer stop() // ← always defer stop() immediately after Pull

	fmt.Print("  pull values: ")
	for {
		v, ok := next() // advance one step; ok is false when exhausted
		if !ok {
			break
		}
		fmt.Printf("%d ", v)
	}
	fmt.Println()

	// After the loop ends, stop() is called by the defer.
	// Even if we break early, stop() cleans up the internal goroutine.
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. The killer use case: zip two iterators in lock-step
// ─────────────────────────────────────────────────────────────────────────────

// zip combines two sequences element-by-element, yielding (a, b) pairs.
// This is IMPOSSIBLE to write correctly with push iterators alone — you'd
// need two loops nested, which doesn't work for independent iterators.
// Pull iterators let us advance both sequences one step at a time.
func zip[A, B any](seqA iter.Seq[A], seqB iter.Seq[B]) iter.Seq2[A, B] {
	return func(yield func(A, B) bool) {
		// Convert both push iterators to pull iterators
		nextA, stopA := iter.Pull(seqA)
		nextB, stopB := iter.Pull(seqB)

		// CRITICAL: stop both when we're done, regardless of how we exit.
		defer stopA()
		defer stopB()

		for {
			a, okA := nextA() // advance A one step
			b, okB := nextB() // advance B one step

			if !okA || !okB {
				// One or both sequences exhausted — zip stops here.
				// (zip stops at the shorter sequence)
				return
			}

			if !yield(a, b) {
				return // consumer broke out early
			}
		}
	}
}

func zipExample() {
	names := sliceValues([]string{"Alice", "Bob", "Carol", "Dave"}) // 4 items
	scores := sliceValues([]int{95, 87, 92})                        // 3 items — shorter

	fmt.Println("  zipped (stops at shorter):")
	for name, score := range zip(names, scores) {
		fmt.Printf("    %s: %d\n", name, score)
	}
	// Output: Alice:95, Bob:87, Carol:92 — Dave is dropped because scores ran out
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. Push vs Pull: when to use each
// ─────────────────────────────────────────────────────────────────────────────

func pushVsPull() {
	fmt.Println("  Push iterator advantages:")
	fmt.Println("    - Simpler to write (just call yield in a loop)")
	fmt.Println("    - No goroutine overhead")
	fmt.Println("    - Works naturally with for-range")
	fmt.Println("    - Better for simple transformations (filter, map, take)")
	fmt.Println()

	fmt.Println("  Pull iterator advantages:")
	fmt.Println("    - Advance two iterators in lock-step (zip, merge-sort)")
	fmt.Println("    - Interleave iteration with other logic (state machines)")
	fmt.Println("    - Test iterators step-by-step")
	fmt.Println()

	// Demonstrate: using pull to test step-by-step
	fmt.Println("  Step-by-step testing with pull:")
	seq := upTo(3)
	next, stop := iter.Pull(seq)
	defer stop()

	v1, _ := next()
	v2, _ := next()
	v3, _ := next()
	_, ok := next() // should be exhausted

	fmt.Printf("    step1=%d, step2=%d, step3=%d, exhausted=%v\n", v1, v2, v3, !ok)

	// Show iter.Pull2 for key-value iterators
	fmt.Println("\n  iter.Pull2 for Seq2 (key-value):")

	// Demonstrate iter.Pull2
	kvPairs := func(yield func(int, string) bool) {
		data := []string{"alpha", "beta", "gamma"}
		for i, v := range data {
			if !yield(i, v) {
				return
			}
		}
	}

	nextKV, stopKV := iter.Pull2(kvPairs)
	defer stopKV()

	for {
		k, v, ok := nextKV()
		if !ok {
			break
		}
		fmt.Printf("    [%d]=%s\n", k, v)
	}
}
