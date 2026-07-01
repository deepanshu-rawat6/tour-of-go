// Package unsafe_pkg explores Go's unsafe package and its interaction with
// the reflect package and the GC.
//
// ⚠️  IMPORTANT: unsafe bypasses Go's type safety guarantees. Code in this
// package is intentionally educational. In production, reach for unsafe only
// when you have measurable performance evidence and no safe alternative.
//
// Topics covered:
//   - pointer_arithmetic — Sizeof, Alignof, Offsetof; string/slice headers;
//                          the GC rule about uintptr vs unsafe.Pointer.
//   - struct_layout      — padding, field ordering, cache-line awareness.
//   - reflect_combo      — unexported field access, SliceHeader/StringHeader,
//                          modern unsafe.String/unsafe.SliceData, type punning.
package unsafe_pkg

import (
	"fmt"
	"os"
)

// Run executes all unsafe examples in order.
func Run() {
	fmt.Println("=== unsafe Package Deep Dive ===")
	fmt.Println()

	pointerArithmeticExample()
	fmt.Println()

	structLayoutExample()
	fmt.Println()

	reflectComboExample()
	fmt.Println()
}

// RunExample dispatches to a single named example.
// Usage: go run . unsafe_pkg pointer
func RunExample(name string) {
	fmt.Printf("=== unsafe: %s ===\n\n", name)

	switch name {
	case "pointer":
		pointerArithmeticExample()
	case "struct-layout":
		structLayoutExample()
	case "reflect-combo":
		reflectComboExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  pointer       - Sizeof/Alignof/Offsetof, string header, []byte↔string zero-copy")
		fmt.Println("  struct-layout - Padding, field order, size differences, cache-line awareness")
		fmt.Println("  reflect-combo - Unexported fields, SliceHeader, type punning float64↔uint64")
		os.Exit(1)
	}
}
