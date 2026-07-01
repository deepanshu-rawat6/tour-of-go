//go:build go1.23

// iter_seq2.go — stdlib iterator functions: slices.All, maps.All, and friends.
//
// Go 1.23 added iterator-aware functions to the slices and maps packages.
// They all return iter.Seq or iter.Seq2 and work directly with range.
//
// SLICES:
//   slices.All(s)    → iter.Seq2[int, E]   (index, value)
//   slices.Values(s) → iter.Seq[E]         (value only)
//   slices.Collect(seq) → []E              (iterator → slice)
//
// MAPS:
//   maps.All(m)      → iter.Seq2[K, V]     (key, value)
//   maps.Keys(m)     → iter.Seq[K]         (keys only)
//   maps.Values(m)   → iter.Seq[V]         (values only)
//   maps.Collect(seq2) → map[K]V           (iterator → map)
package iterators

import (
	"fmt"
	"maps"
	"slices"
)

// iterSeq2Example demonstrates the stdlib iterator functions.
func iterSeq2Example() {
	fmt.Println("1. slices.All — (index, value)")
	slicesAllExample()

	fmt.Println("\n2. slices.Values — value only")
	slicesValuesExample()

	fmt.Println("\n3. maps.All — (key, value)")
	mapsAllExample()

	fmt.Println("\n4. maps.Keys and maps.Values")
	mapsKeysValuesExample()

	fmt.Println("\n5. Collecting back: slices.Collect, maps.Collect")
	collectingExample()

	fmt.Println("\n6. Real use case: paginated API result as an iterator")
	paginatedAPIExample()
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. slices.All — yields (index, value) pairs
// ─────────────────────────────────────────────────────────────────────────────

func slicesAllExample() {
	fruits := []string{"apple", "banana", "cherry"}

	// slices.All returns iter.Seq2[int, string]
	// The first value is the index, the second is the element.
	for i, v := range slices.All(fruits) {
		fmt.Printf("  [%d] %s\n", i, v)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. slices.Values — yields values only (no index)
// ─────────────────────────────────────────────────────────────────────────────

func slicesValuesExample() {
	primes := []int{2, 3, 5, 7, 11}

	// slices.Values returns iter.Seq[int]
	fmt.Print("  primes: ")
	for p := range slices.Values(primes) {
		fmt.Printf("%d ", p)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. maps.All — yields (key, value) pairs
// ─────────────────────────────────────────────────────────────────────────────

func mapsAllExample() {
	capitals := map[string]string{
		"France": "Paris",
		"Japan":  "Tokyo",
		"Brazil": "Brasília",
	}

	// maps.All returns iter.Seq2[string, string]
	// Note: map iteration order is random.
	fmt.Println("  capitals (order varies):")
	for country, capital := range maps.All(capitals) {
		fmt.Printf("    %s → %s\n", country, capital)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. maps.Keys and maps.Values
// ─────────────────────────────────────────────────────────────────────────────

func mapsKeysValuesExample() {
	inventory := map[string]int{
		"apples":  50,
		"bananas": 23,
		"oranges": 11,
	}

	// maps.Keys returns iter.Seq[K]
	fmt.Print("  items in stock: ")
	for item := range maps.Keys(inventory) {
		fmt.Printf("%s ", item)
	}
	fmt.Println()

	// maps.Values returns iter.Seq[V]
	total := 0
	for qty := range maps.Values(inventory) {
		total += qty
	}
	fmt.Printf("  total quantity: %d\n", total)
}

// ─────────────────────────────────────────────────────────────────────────────
// 5. Collecting back to slices/maps
// ─────────────────────────────────────────────────────────────────────────────

func collectingExample() {
	// Start with a slice
	original := []int{1, 2, 3, 4, 5, 6, 7, 8}

	// Build a filtered+transformed iterator (reusing adapters from iter_seq.go)
	evenDoubled := MapSeq(
		Filter(slices.Values(original), func(n int) bool { return n%2 == 0 }),
		func(n int) int { return n * 2 },
	)

	// slices.Collect drains the iterator into a new slice
	result := slices.Collect(evenDoubled)
	fmt.Printf("  even numbers doubled: %v\n", result)

	// maps.Collect turns an iter.Seq2[K, V] into a map[K]V
	pairs := func(yield func(string, int) bool) {
		data := [][2]any{{"a", 1}, {"b", 2}, {"c", 3}}
		for _, pair := range data {
			if !yield(pair[0].(string), pair[1].(int)) {
				return
			}
		}
	}
	m := maps.Collect(pairs)
	fmt.Printf("  collected map: %v\n", m)
}

// ─────────────────────────────────────────────────────────────────────────────
// 6. Real use case: paginated API as an iterator
// ─────────────────────────────────────────────────────────────────────────────

// APIPage simulates a page of results from a paginated API.
type APIPage struct {
	Items   []string
	HasMore bool
	Cursor  int
}

// fetchPage simulates fetching a page from a remote API.
// In real code this would make an HTTP request.
func fetchPage(cursor int) APIPage {
	allItems := []string{
		"item-A", "item-B", "item-C",
		"item-D", "item-E", "item-F",
		"item-G",
	}
	pageSize := 3

	start := cursor
	if start >= len(allItems) {
		return APIPage{}
	}
	end := start + pageSize
	if end > len(allItems) {
		end = len(allItems)
	}

	return APIPage{
		Items:   allItems[start:end],
		HasMore: end < len(allItems),
		Cursor:  end,
	}
}

// paginatedItems returns an iter.Seq[string] that transparently paginates.
// The caller just sees a stream of items — no pagination logic leaks out.
// This is the key benefit: callers use a simple for-range, not a cursor loop.
func paginatedItems() func(yield func(string) bool) {
	return func(yield func(string) bool) {
		cursor := 0
		for {
			page := fetchPage(cursor)
			for _, item := range page.Items {
				if !yield(item) {
					return // caller broke out early
				}
			}
			if !page.HasMore {
				break // no more pages
			}
			cursor = page.Cursor
		}
	}
}

func paginatedAPIExample() {
	// Caller doesn't know or care about pagination — just iterates items.
	fmt.Println("  all items from paginated API:")
	for item := range paginatedItems() {
		fmt.Printf("    %s\n", item)
	}

	// Early exit also works — only fetches as many pages as needed
	fmt.Print("  first 4 items only: ")
	count := 0
	for item := range paginatedItems() {
		fmt.Printf("%s ", item)
		count++
		if count >= 4 {
			break
		}
	}
	fmt.Println()
}
