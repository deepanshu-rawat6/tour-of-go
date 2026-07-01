// slices_pkg.go — Tour of the "slices" package (Go 1.21+).
//
// The slices package provides generic functions for working with slices of
// any ordered or comparable type. It replaces common hand-rolled helpers
// like contains(), indexOf(), etc.
//
// Key insight: all mutating functions (Sort, Reverse, Replace, Delete,
// Insert) operate *in place* and return the (possibly re-allocated) slice.
// Always capture the return value: s = slices.Delete(s, i, j).
package stdlib_collections

import (
	"cmp"
	"fmt"
	"slices"
)

// slicesExample walks through every major function in the slices package.
func slicesExample() {
	fmt.Println("--- slices package (Go 1.21+) ---")

	// -------------------------------------------------------------------------
	// 1. slices.Contains — O(n) linear search for a value.
	//    Use this when the slice is UNSORTED or small. For sorted slices,
	//    prefer BinarySearch (see below).
	// -------------------------------------------------------------------------
	fruits := []string{"apple", "banana", "cherry"}
	fmt.Printf("Contains 'banana': %v\n", slices.Contains(fruits, "banana"))   // true
	fmt.Printf("Contains 'durian':  %v\n", slices.Contains(fruits, "durian")) // false

	// -------------------------------------------------------------------------
	// 2. slices.Index — returns the first index of v, or -1 if not present.
	//    O(n) linear scan. Like strings.Index but generic.
	// -------------------------------------------------------------------------
	nums := []int{10, 20, 30, 20, 10}
	fmt.Printf("Index of 20: %d\n", slices.Index(nums, 20)) // 1 (first occurrence)
	fmt.Printf("Index of 99: %d\n", slices.Index(nums, 99)) // -1

	// -------------------------------------------------------------------------
	// 3. slices.Sort — in-place sort for any cmp.Ordered type
	//    (integers, floats, strings). Uses pdqsort (pattern-defeating quicksort).
	//    NOT stable: equal elements may be reordered.
	// -------------------------------------------------------------------------
	words := []string{"cherry", "apple", "banana", "apricot"}
	fmt.Printf("Before Sort: %v\n", words)
	slices.Sort(words)
	fmt.Printf("After  Sort: %v\n", words) // [apple apricot banana cherry]

	// -------------------------------------------------------------------------
	// 4. slices.SortFunc — sort with a custom comparator.
	//    The comparator must follow the cmp.Compare convention:
	//      -1 if a < b, 0 if a == b, +1 if a > b
	//    Great for structs or case-insensitive sorts.
	// -------------------------------------------------------------------------
	type Person struct{ Name string; Age int }
	people := []Person{
		{"Alice", 30},
		{"Bob", 25},
		{"Charlie", 35},
		{"Dave", 25},
	}
	// Sort by Age ascending, then by Name ascending as tiebreaker.
	slices.SortFunc(people, func(a, b Person) int {
		// cmp.Or tries each comparator and returns the first non-zero.
		return cmp.Or(
			cmp.Compare(a.Age, b.Age),  // primary key: age
			cmp.Compare(a.Name, b.Name), // tiebreaker: name
		)
	})
	fmt.Printf("SortFunc by Age,Name: %v\n", people)
	// [{Bob 25} {Dave 25} {Alice 30} {Charlie 35}]

	// -------------------------------------------------------------------------
	// 5. slices.SortStableFunc — like SortFunc but STABLE.
	//    Stable sort preserves the original order of equal elements.
	//    Slightly slower than unstable sort but necessary when order matters.
	// -------------------------------------------------------------------------
	data := []Person{{"Eve", 25}, {"Frank", 25}, {"Grace", 30}}
	slices.SortStableFunc(data, func(a, b Person) int {
		return cmp.Compare(a.Age, b.Age)
	})
	fmt.Printf("SortStableFunc (age only): %v\n", data)
	// Order of equal-age elements is preserved: Eve before Frank.

	// -------------------------------------------------------------------------
	// 6. slices.BinarySearch — O(log n) search in a SORTED slice.
	//    Returns (index, found). If not found, index is where it would be
	//    inserted to keep the slice sorted (useful for insertion).
	//
	//    PRECONDITION: slice must be sorted (call slices.Sort first).
	// -------------------------------------------------------------------------
	sorted := []int{1, 3, 5, 7, 9, 11}
	idx, found := slices.BinarySearch(sorted, 7)
	fmt.Printf("BinarySearch(7):  idx=%d, found=%v\n", idx, found) // idx=3, found=true
	idx, found = slices.BinarySearch(sorted, 6)
	fmt.Printf("BinarySearch(6):  idx=%d, found=%v\n", idx, found) // idx=3, found=false (insert position)

	// -------------------------------------------------------------------------
	// 7. slices.Compact — remove consecutive duplicate elements.
	//    Like Unix `uniq`. Operates in-place; returns updated slice.
	//    NOTE: only removes *adjacent* duplicates, so sort first for full dedup.
	// -------------------------------------------------------------------------
	withDups := []int{1, 1, 2, 3, 3, 3, 4, 1, 1} // trailing 1,1 won't be removed without sort
	compacted := slices.Compact(withDups)
	fmt.Printf("Compact: %v\n", compacted) // [1 2 3 4 1]

	// For a full dedup, sort first:
	all := []int{3, 1, 4, 1, 5, 9, 2, 6, 5, 3, 5}
	slices.Sort(all)
	deduped := slices.Compact(all)
	fmt.Printf("Sort+Compact (full dedup): %v\n", deduped) // [1 2 3 4 5 6 9]

	// -------------------------------------------------------------------------
	// 8. slices.CompactFunc — remove consecutive duplicates with custom equality.
	//    Useful for case-insensitive dedup or struct equality.
	// -------------------------------------------------------------------------
	mixed := []string{"Apple", "apple", "Banana", "banana", "Cherry"}
	slices.SortFunc(mixed, func(a, b string) int {
		// Sort case-insensitively first
		if a < b { return -1 }
		if a > b { return 1 }
		return 0
	})
	_ = slices.CompactFunc(mixed, func(a, b string) bool {
		// Treat as equal if same lowered value — demonstration only
		return len(a) == len(b) // simplified; real code: strings.EqualFold(a, b)
	})
	fmt.Printf("CompactFunc result: %v\n", mixed)

	// -------------------------------------------------------------------------
	// 9. slices.Clip — shrink the slice's capacity to match its length.
	//    Releases excess backing array memory. Important when you return
	//    a sub-slice from a large buffer and don't need the extra capacity.
	// -------------------------------------------------------------------------
	big := make([]int, 3, 100) // len=3, cap=100 — wastes memory
	big[0], big[1], big[2] = 1, 2, 3
	clipped := slices.Clip(big)
	fmt.Printf("Clip: len=%d, cap=%d (was cap=100)\n", len(clipped), cap(clipped)) // len=3, cap=3

	// -------------------------------------------------------------------------
	// 10. slices.Grow — pre-allocate capacity.
	//     Ensures the slice has room for at least n more elements.
	//     Use before a loop that appends many elements to avoid reallocations.
	// -------------------------------------------------------------------------
	s := []int{1, 2, 3}
	s = slices.Grow(s, 10) // ensure capacity for 10 more elements
	fmt.Printf("Grow: len=%d, cap>=%d\n", len(s), len(s)+10) // len=3, cap>=13

	// -------------------------------------------------------------------------
	// 11. slices.Reverse — reverse elements in place.
	// -------------------------------------------------------------------------
	rev := []int{1, 2, 3, 4, 5}
	slices.Reverse(rev)
	fmt.Printf("Reverse: %v\n", rev) // [5 4 3 2 1]

	// -------------------------------------------------------------------------
	// 12. slices.Replace — replace a subslice [i, j) with new elements.
	//     s = slices.Replace(s, i, j, newElems...)
	//     Elements at indices i..j-1 are replaced. The result may be a
	//     different slice header if reallocation was needed.
	// -------------------------------------------------------------------------
	r := []string{"a", "b", "c", "d", "e"}
	r = slices.Replace(r, 1, 3, "X", "Y", "Z") // replace [b, c] with [X, Y, Z]
	fmt.Printf("Replace [1,3) with X,Y,Z: %v\n", r) // [a X Y Z d e]

	// -------------------------------------------------------------------------
	// 13. slices.Delete — delete elements at indices [i, j).
	//     Like Replace with no replacement elements.
	//     IMPORTANT: elements after j are shifted left; the backing array
	//     may still hold old values at the end (use Clip to trim cap).
	// -------------------------------------------------------------------------
	d := []string{"a", "b", "c", "d", "e"}
	d = slices.Delete(d, 1, 3) // delete indices 1 and 2 (b, c)
	fmt.Printf("Delete [1,3): %v\n", d) // [a d e]

	// -------------------------------------------------------------------------
	// 14. slices.Insert — insert elements at position i.
	//     Everything from i onward shifts right.
	// -------------------------------------------------------------------------
	ins := []string{"a", "d", "e"}
	ins = slices.Insert(ins, 1, "b", "c") // insert b, c at index 1
	fmt.Printf("Insert at 1: %v\n", ins) // [a b c d e]

	// -------------------------------------------------------------------------
	// 15. slices.Max / slices.Min — find the maximum/minimum in one pass.
	//     Panics on empty slice (no zero value is safe for generics here).
	// -------------------------------------------------------------------------
	vals := []int{3, 1, 4, 1, 5, 9, 2, 6}
	fmt.Printf("Max: %d, Min: %d\n", slices.Max(vals), slices.Min(vals)) // 9, 1

	// -------------------------------------------------------------------------
	// 16. slices.Equal / slices.EqualFunc — compare two slices element-by-element.
	//     Equal uses == ; EqualFunc uses a custom predicate.
	// -------------------------------------------------------------------------
	a := []int{1, 2, 3}
	b := []int{1, 2, 3}
	c := []int{1, 2, 4}
	fmt.Printf("Equal(a,b): %v\n", slices.Equal(a, b)) // true
	fmt.Printf("Equal(a,c): %v\n", slices.Equal(a, c)) // false

	// EqualFunc: compare by absolute value
	x := []int{1, -2, 3}
	y := []int{1, 2, -3}
	abs := func(n int) int {
		if n < 0 { return -n }
		return n
	}
	fmt.Printf("EqualFunc(abs): %v\n", slices.EqualFunc(x, y, func(a, b int) bool {
		return abs(a) == abs(b)
	})) // true

	// -------------------------------------------------------------------------
	// 17. slices.Concat (Go 1.22) — concatenate multiple slices into one new slice.
	//     Equivalent to append(append(a, b...), c...) but cleaner.
	// -------------------------------------------------------------------------
	p1 := []int{1, 2, 3}
	p2 := []int{4, 5}
	p3 := []int{6, 7, 8, 9}
	concatenated := slices.Concat(p1, p2, p3)
	fmt.Printf("Concat: %v\n", concatenated) // [1 2 3 4 5 6 7 8 9]
}
