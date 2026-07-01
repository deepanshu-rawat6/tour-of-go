//go:build go1.23

// iter_seq.go — iter.Seq and iter.Seq2: the canonical iterator types.
//
// The iter package (Go 1.23) defines two iterator types:
//
//	iter.Seq[V]     = func(yield func(V) bool)
//	iter.Seq2[K, V] = func(yield func(K, V) bool)
//
// These are just TYPE ALIASES for function signatures — there's nothing
// special about them at runtime. They exist so that:
//   - Code is more readable (iter.Seq[int] is clearer than func(func(int)bool))
//   - The standard library has a common vocabulary for iterators
//   - Adapter functions (Filter, Map, Take) can be written generically
//
// The standard library uses these types in:
//   - slices.Values(s) → iter.Seq[E]
//   - slices.All(s)    → iter.Seq2[int, E]
//   - maps.All(m)      → iter.Seq2[K, V]
//   - maps.Keys(m)     → iter.Seq[K]
//   - maps.Values(m)   → iter.Seq[V]
package iterators

import (
	"fmt"
	"iter"
)

// iterSeqExample demonstrates iter.Seq, iter.Seq2, and adapter functions.
func iterSeqExample() {
	fmt.Println("1. iter.Seq from a slice")
	seqFromSlice()

	fmt.Println("\n2. iter.Seq2 from a map")
	seq2FromMap()

	fmt.Println("\n3. Adapter: Filter")
	filterAdapter()

	fmt.Println("\n4. Adapter: Map (transform values)")
	mapAdapter()

	fmt.Println("\n5. Adapter: Take (limit)")
	takeAdapter()

	fmt.Println("\n6. Chaining adapters")
	chainAdapters()
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. iter.Seq from a slice
// ─────────────────────────────────────────────────────────────────────────────

// sliceValues converts a slice to an iter.Seq (single-value iterator).
// This is exactly what slices.Values does in the standard library.
func sliceValues[T any](s []T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range s { // regular for-range over the slice
			if !yield(v) {
				return
			}
		}
	}
}

func seqFromSlice() {
	words := []string{"go", "channels", "iterators", "generics"}
	seq := sliceValues(words)

	fmt.Print("  slice as Seq: ")
	for w := range seq {
		fmt.Printf("%s ", w)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. iter.Seq2 from a map
// ─────────────────────────────────────────────────────────────────────────────

// mapAll converts a map to an iter.Seq2[K, V] (key-value iterator).
// This is exactly what maps.All does in the standard library.
func mapAll[K comparable, V any](m map[K]V) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, v := range m { // regular for-range over the map
			if !yield(k, v) {
				return
			}
		}
	}
}

func seq2FromMap() {
	scores := map[string]int{"alice": 95, "bob": 87, "carol": 92}
	seq2 := mapAll(scores)

	// Note: map iteration order is random — output may vary
	fmt.Println("  map as Seq2 (order varies):")
	for name, score := range seq2 {
		fmt.Printf("    %s: %d\n", name, score)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. Adapter: Filter
// ─────────────────────────────────────────────────────────────────────────────

// Filter returns a new iterator that only yields values for which pred is true.
// This is a lazy adapter: it doesn't compute anything until iterated.
func Filter[V any](seq iter.Seq[V], pred func(V) bool) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range seq { // iterate source
			if pred(v) { // apply predicate
				if !yield(v) { // forward to consumer
					return
				}
			}
		}
	}
}

func filterAdapter() {
	nums := sliceValues([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	evens := Filter(nums, func(n int) bool { return n%2 == 0 })

	fmt.Print("  even numbers: ")
	for n := range evens {
		fmt.Printf("%d ", n)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. Adapter: Map (transform values)
// ─────────────────────────────────────────────────────────────────────────────

// MapSeq applies f to each value and returns a new iterator.
// Named MapSeq to avoid conflict with the built-in map type.
func MapSeq[In, Out any](seq iter.Seq[In], f func(In) Out) iter.Seq[Out] {
	return func(yield func(Out) bool) {
		for v := range seq {
			if !yield(f(v)) {
				return
			}
		}
	}
}

func mapAdapter() {
	nums := sliceValues([]int{1, 2, 3, 4, 5})
	doubled := MapSeq(nums, func(n int) int { return n * 2 })

	fmt.Print("  doubled: ")
	for n := range doubled {
		fmt.Printf("%d ", n)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// 5. Adapter: Take
// ─────────────────────────────────────────────────────────────────────────────

// Take returns an iterator that yields at most n values from seq.
func Take[V any](seq iter.Seq[V], n int) iter.Seq[V] {
	return func(yield func(V) bool) {
		count := 0
		for v := range seq {
			if count >= n {
				return // enough values collected
			}
			if !yield(v) {
				return // consumer broke out
			}
			count++
		}
	}
}

// infiniteInts returns an iterator that counts 0, 1, 2, ... forever.
// We can safely pass this to Take because Take will stop the iteration early.
func infiniteInts() iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; ; i++ {
			if !yield(i) {
				return // consumer stopped
			}
		}
	}
}

func takeAdapter() {
	// Take 5 values from an infinite stream — no goroutines, no channels needed.
	first5 := Take(infiniteInts(), 5)

	fmt.Print("  first 5 from infinite: ")
	for n := range first5 {
		fmt.Printf("%d ", n)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// 6. Chaining adapters (lazy pipeline)
// ─────────────────────────────────────────────────────────────────────────────

// Collect gathers all values from an iterator into a slice.
// This is exactly what slices.Collect does in the standard library.
func Collect[V any](seq iter.Seq[V]) []V {
	var result []V
	for v := range seq {
		result = append(result, v)
	}
	return result
}

func chainAdapters() {
	// Lazy pipeline (no intermediate slices allocated):
	//   infiniteInts → Take(20) → Filter(odd) → Map(square) → Take(5) → Collect
	pipeline := Take(
		MapSeq(
			Filter(
				Take(infiniteInts(), 20), // 0..19
				func(n int) bool { return n%2 != 0 }, // odd numbers: 1,3,5,7,9,11,13,15,17,19
			),
			func(n int) int { return n * n }, // square: 1,9,25,49,81,...
		),
		5, // take first 5 results
	)

	result := Collect(pipeline)
	fmt.Printf("  chain result: %v\n", result)
	// Expected: [1 9 25 49 81]
}
