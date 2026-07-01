// maps_pkg.go — Tour of the "maps" package (Go 1.21+).
//
// The maps package provides generic utilities for working with Go maps.
// It replaces common patterns like manually iterating to copy or compare maps.
//
// Key gotcha: maps.Keys and maps.Values return elements in RANDOM order
// because Go map iteration is intentionally non-deterministic. For
// deterministic output, sort the result with slices.Sort (see below).
package stdlib_collections

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// mapsExample walks through every major function in the maps package.
func mapsExample() {
	fmt.Println("--- maps package (Go 1.21+) ---")

	inventory := map[string]int{
		"apple":  5,
		"banana": 3,
		"cherry": 8,
		"date":   2,
	}

	// -------------------------------------------------------------------------
	// 1. maps.Keys — returns all keys as a []K slice (UNORDERED).
	//    The order changes every run. Never assume order without sorting.
	// -------------------------------------------------------------------------
	keys := slices.Collect(maps.Keys(inventory))
	// Keys returns iter.Seq[K], use slices.Collect to materialise into slice
	fmt.Printf("Keys (unordered): %v\n", keys)

	// -------------------------------------------------------------------------
	// 2. maps.Values — returns all values as a []V slice (UNORDERED).
	// -------------------------------------------------------------------------
	vals := slices.Collect(maps.Values(inventory))
	fmt.Printf("Values (unordered): %v\n", vals)

	// -------------------------------------------------------------------------
	// Sorted keys pattern — combine maps.Keys + slices.Sort for
	// deterministic output. This is the canonical Go idiom.
	// -------------------------------------------------------------------------
	sortedKeys := slices.Collect(maps.Keys(inventory))
	slices.Sort(sortedKeys)
	fmt.Print("Sorted keys: ")
	for _, k := range sortedKeys {
		fmt.Printf("%s=%d ", k, inventory[k])
	}
	fmt.Println()
	// Output: apple=5 banana=3 cherry=8 date=2

	// -------------------------------------------------------------------------
	// 3. maps.Clone — shallow copy.
	//    New map, same key/value pointers (for value types: full copy).
	//    Changes to the clone don't affect the original.
	// -------------------------------------------------------------------------
	clone := maps.Clone(inventory)
	clone["elderberry"] = 10 // add to clone only
	fmt.Printf("Original has 'elderberry': %v\n", inventory["elderberry"] != 0) // false
	fmt.Printf("Clone has 'elderberry':    %v\n", clone["elderberry"] != 0)     // true

	// -------------------------------------------------------------------------
	// 4. maps.Copy — merge src into dst, overwriting matching keys.
	//    Think of it as the map equivalent of append for slices.
	//    After Copy, dst contains all keys from both maps.
	// -------------------------------------------------------------------------
	dst := map[string]int{"apple": 99, "fig": 7} // apple will be overwritten
	src := map[string]int{"apple": 5, "grape": 4}
	maps.Copy(dst, src)
	fmt.Printf("After Copy (dst): apple=%d, fig=%d, grape=%d\n",
		dst["apple"], dst["fig"], dst["grape"]) // apple=5 (src wins), fig=7, grape=4

	// -------------------------------------------------------------------------
	// 5. maps.DeleteFunc — delete entries where the predicate returns true.
	//    In-place mutation. Returns nothing.
	//    (The simpler maps.Delete(m, key) doesn't exist; use delete(m, key) builtin.)
	// -------------------------------------------------------------------------
	prices := map[string]float64{
		"apple":  1.50,
		"banana": 0.30, // cheap: will be deleted
		"cherry": 3.00,
		"date":   0.20, // cheap: will be deleted
	}
	maps.DeleteFunc(prices, func(k string, v float64) bool {
		return v < 1.0 // remove items under $1
	})
	priceKeys := slices.Collect(maps.Keys(prices))
	slices.Sort(priceKeys)
	fmt.Printf("After DeleteFunc (price >= $1): %v\n", priceKeys) // [apple cherry]

	// -------------------------------------------------------------------------
	// 6. maps.Equal — deep equality check.
	//    Returns true only if both maps have identical keys AND values.
	//    Uses == for value comparison.
	// -------------------------------------------------------------------------
	m1 := map[string]int{"a": 1, "b": 2}
	m2 := map[string]int{"a": 1, "b": 2}
	m3 := map[string]int{"a": 1, "b": 3} // b differs
	fmt.Printf("Equal(m1,m2): %v\n", maps.Equal(m1, m2)) // true
	fmt.Printf("Equal(m1,m3): %v\n", maps.Equal(m1, m3)) // false

	// -------------------------------------------------------------------------
	// 7. maps.EqualFunc — equality with a custom comparator for values.
	//    Useful when values are structs, pointers, or need fuzzy matching.
	// -------------------------------------------------------------------------
	upper := map[string]string{"greeting": "HELLO", "farewell": "GOODBYE"}
	lower := map[string]string{"greeting": "hello", "farewell": "goodbye"}
	fmt.Printf("EqualFunc (case-insensitive): %v\n",
		maps.EqualFunc(upper, lower, strings.EqualFold)) // true

	// -------------------------------------------------------------------------
	// 8. maps.All / maps.Keys / maps.Values as iter.Seq (Go 1.23 range-over-func)
	//
	// In Go 1.23+, you can range directly over maps.All(m):
	//
	//   for k, v := range maps.All(inventory) {
	//       fmt.Printf("%s: %d\n", k, v)
	//   }
	//
	// maps.Keys(m) and maps.Values(m) also return iter.Seq[K] / iter.Seq[V]
	// which work with range in Go 1.23. Use slices.Collect() to turn them
	// into a []K or []V slice when you need random access.
	//
	// This is the idiomatic Go 1.23 approach — no need to call slices.Collect
	// if you only need to iterate once.
	// -------------------------------------------------------------------------
	fmt.Println("maps.All (range-over-func, Go 1.23):")
	// Use slices.Collect + sort for deterministic demo output
	allKeys := slices.Collect(maps.Keys(inventory))
	slices.Sort(allKeys)
	for _, k := range allKeys {
		fmt.Printf("  %s: %d\n", k, inventory[k])
	}
}
