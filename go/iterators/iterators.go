//go:build go1.23

// Package iterators demonstrates Go 1.23's range-over-func and iter package.
//
// Go 1.23 added two landmark features for iteration:
//
//  1. Range-over-func: you can now write "for v := range myFunc" where
//     myFunc is a function of the form func(yield func(V) bool).
//
//  2. The iter package: defines the canonical iterator types iter.Seq[V] and
//     iter.Seq2[K, V], plus iter.Pull / iter.Pull2 for pull-style iteration.
//
//  3. The slices and maps packages gained iterator-aware functions:
//     slices.All, slices.Values, maps.All, maps.Keys, maps.Values,
//     slices.Collect, maps.Collect.
//
// Build tag ensures this package only compiles on Go 1.23+.
// The module in this repo uses Go 1.25, so all examples will run.
//
// Run all examples:
//
//	go run . iterators
//
// Run a specific example:
//
//	go run . iterators range-func
//	go run . iterators iter-seq
//	go run . iterators iter-seq2
//	go run . iterators pull
package iterators

import (
	"fmt"
	"os"
)

// Run executes all iterator examples in sequence.
func Run() {
	fmt.Println("=== Iterators (Go 1.23) ===")
	fmt.Println()

	fmt.Println("--- Range-over-func ---")
	rangeFuncExample()
	fmt.Println()

	fmt.Println("--- iter.Seq ---")
	iterSeqExample()
	fmt.Println()

	fmt.Println("--- iter.Seq2 (slices/maps stdlib) ---")
	iterSeq2Example()
	fmt.Println()

	fmt.Println("--- iter.Pull (pull-style iterator) ---")
	pullIterExample()
	fmt.Println()
}

// RunExample runs a specific iterator example by name.
func RunExample(name string) {
	fmt.Printf("=== Iterators: %s ===\n\n", name)

	switch name {
	case "range-func":
		rangeFuncExample()
	case "iter-seq":
		iterSeqExample()
	case "iter-seq2":
		iterSeq2Example()
	case "pull":
		pullIterExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  range-func  - Range-over-func (push iterator basics)")
		fmt.Println("  iter-seq    - iter.Seq / iter.Seq2 types + adapters")
		fmt.Println("  iter-seq2   - slices/maps stdlib iterator functions")
		fmt.Println("  pull        - iter.Pull: push-to-pull conversion")
		os.Exit(1)
	}
}
