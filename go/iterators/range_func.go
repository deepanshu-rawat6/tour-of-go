//go:build go1.23

// range_func.go — range-over-func: iterating with a push iterator.
//
// WHAT IS A PUSH ITERATOR?
// A push iterator is a function that drives the loop body.
// It "pushes" values into the caller by calling a yield function.
//
// The yield function returns false when the caller wants to stop
// (e.g., break out of the loop). The iterator MUST respect that signal.
//
// MENTAL MODEL:
//   - The iterator function IS the loop driver (it calls yield repeatedly).
//   - yield IS the loop body (it runs once per item).
//   - yield returns false → break was called → iterator should return.
//
// Three function shapes work with range in Go 1.23:
//
//	func(yield func() bool)              — no values (loop variable)
//	func(yield func(V) bool)             — one value:  for v := range f
//	func(yield func(K, V) bool)          — two values: for k, v := range f
package iterators

import (
	"fmt"
	"iter"
)

// rangeFuncExample demonstrates range-over-func.
func rangeFuncExample() {
	fmt.Println("1. Simple integer iterator")
	simpleIterator()

	fmt.Println("\n2. Range over a linked list")
	linkedListIterator()

	fmt.Println("\n3. break, continue, return all work correctly")
	breakContinueReturn()

	fmt.Println("\n4. Mental model: iterator as loop driver")
	mentalModel()
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. Simple integer iterator
// ─────────────────────────────────────────────────────────────────────────────

// upTo returns an iterator that yields integers 0, 1, ..., n-1.
//
// The type signature func(yield func(int) bool) is the push iterator shape.
// Go 1.23 lets you range over any function with this signature.
func upTo(n int) iter.Seq[int] {
	// iter.Seq[int] is defined as: type Seq[V any] func(yield func(V) bool)
	// It's just a named function type — nothing magic about it.
	return func(yield func(int) bool) {
		for i := 0; i < n; i++ {
			if !yield(i) {
				// yield returned false → the caller broke out of the loop.
				// We MUST return here; if we keep calling yield after it
				// returns false, the runtime will panic.
				return
			}
		}
	}
}

func simpleIterator() {
	for v := range upTo(5) {
		fmt.Printf("  %d", v)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. Range over a custom collection: a linked list
// ─────────────────────────────────────────────────────────────────────────────

// node is a singly-linked list node.
type node[T any] struct {
	val  T
	next *node[T]
}

// List is a simple linked list with an iterator method.
type List[T any] struct {
	head *node[T]
}

// Push adds a value to the end of the list.
func (l *List[T]) Push(v T) {
	n := &node[T]{val: v}
	if l.head == nil {
		l.head = n
		return
	}
	cur := l.head
	for cur.next != nil {
		cur = cur.next
	}
	cur.next = n
}

// All returns an iterator over all values in the list.
// Because this method has the right shape, you can write:
//
//	for v := range myList.All() { ... }
func (l *List[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for cur := l.head; cur != nil; cur = cur.next {
			if !yield(cur.val) {
				return // caller broke out
			}
		}
	}
}

func linkedListIterator() {
	lst := &List[string]{}
	for _, s := range []string{"alpha", "beta", "gamma", "delta"} {
		lst.Push(s)
	}

	fmt.Print("  linked list: ")
	for v := range lst.All() {
		fmt.Printf("%s ", v)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. break, continue, return — all work normally inside range-over-func
// ─────────────────────────────────────────────────────────────────────────────

func breakContinueReturn() {
	// break: yield returns false, iterator returns early
	fmt.Print("  break at 3: ")
	for v := range upTo(10) {
		if v == 3 {
			break // yield returns false; upTo returns early
		}
		fmt.Printf("%d ", v)
	}
	fmt.Println()

	// continue: the loop body skips to the next iteration; yield is called again
	fmt.Print("  skip evens: ")
	for v := range upTo(8) {
		if v%2 == 0 {
			continue // skip even numbers
		}
		fmt.Printf("%d ", v)
	}
	fmt.Println()

	// return from an enclosing function: yield returns false, iterator returns
	result := firstOver(upTo(100), 5)
	fmt.Printf("  first value > 5: %d\n", result)
}

// firstOver returns the first value from seq that is greater than threshold.
// It uses return inside the for-range, which correctly signals the iterator.
func firstOver(seq iter.Seq[int], threshold int) int {
	for v := range seq {
		if v > threshold {
			return v // yield returns false → upTo exits its goroutine
		}
	}
	return -1
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. Mental model: the iterator is the loop driver
// ─────────────────────────────────────────────────────────────────────────────

// explicitYield manually calls the iterator to show what the compiler does
// under the hood when you write "for v := range upTo(5)".
func mentalModel() {
	// This is (roughly) what the compiler desugars "for v := range upTo(5)"
	// into:
	//
	//   upTo(5)(func(v int) bool {
	//       fmt.Printf("  %d ", v)
	//       return true // continue
	//   })
	//
	// The iterator calls your function (the loop body) for each value.
	// Returning false from the loop body is like a break.

	fmt.Print("  manual call: ")
	upTo(5)(func(v int) bool {
		fmt.Printf("%d ", v)
		return v < 3 // stop after 3 (equivalent to 'break' when v == 4)
	})
	fmt.Println()
}
