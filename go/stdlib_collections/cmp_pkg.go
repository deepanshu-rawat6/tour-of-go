// cmp_pkg.go — Tour of the "cmp" package (Go 1.21+).
//
// The cmp package provides generic comparison utilities for ordered types.
// It centralises comparison logic that was previously scattered across
// hand-written helper functions.
//
// cmp.Ordered constraint — the set of types that support < > <= >= ==:
//   ~int | ~int8 | ~int16 | ~int32 | ~int64 |
//   ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
//   ~float32 | ~float64 |
//   ~string
//
// Any named type built on these (e.g. type Celsius float64) also satisfies
// cmp.Ordered because of the ~ (underlying type constraint).
package stdlib_collections

import (
	"cmp"
	"fmt"
	"slices"
)

// cmpExample demonstrates every function in the cmp package plus real-world
// usage patterns for multi-key sorting.
func cmpExample() {
	fmt.Println("--- cmp package (Go 1.21+) ---")

	// -------------------------------------------------------------------------
	// 1. cmp.Compare(a, b) — generic three-way comparison.
	//
	//    Returns:
	//      -1  if a < b
	//       0  if a == b
	//      +1  if a > b
	//
	//    Equivalent to strings.Compare but works for any cmp.Ordered type.
	//    This is THE comparator function to pass to slices.SortFunc.
	// -------------------------------------------------------------------------
	fmt.Println("cmp.Compare:")
	fmt.Printf("  Compare(1, 2)   = %d\n", cmp.Compare(1, 2))       // -1
	fmt.Printf("  Compare(5, 5)   = %d\n", cmp.Compare(5, 5))       //  0
	fmt.Printf("  Compare(9, 3)   = %d\n", cmp.Compare(9, 3))       // +1
	fmt.Printf("  Compare(\"a\",\"b\") = %d\n", cmp.Compare("a", "b")) // -1

	// -------------------------------------------------------------------------
	// 2. Equality — use == directly (cmp has no Equal function).
	//
	//    The cmp package does NOT provide a cmp.Equal function.
	//    Use == for equality checks on cmp.Ordered types.
	//    cmp.Compare(a, b) == 0 is equivalent to a == b.
	// -------------------------------------------------------------------------
	fmt.Println("Equality with == (cmp has no Equal):")
	fmt.Printf("  3 == 3:          %v\n", 3 == 3)              // true
	fmt.Printf("  3 == 4:          %v\n", 3 == 4)              // false
	fmt.Printf("  cmp.Compare equivalent: %v\n", cmp.Compare(3, 3) == 0) // true

	// -------------------------------------------------------------------------
	// 3. cmp.Or(a, b, c, ...) — returns the first non-zero value (Go 1.22).
	//
	//    "Non-zero" means != 0 for numbers, != "" for strings, != false for
	//    bools, etc. (the zero value for that type).
	//
	//    Primary use case 1: DEFAULT VALUES
	//      timeout := cmp.Or(cfg.Timeout, 30*time.Second)
	//      // if cfg.Timeout == 0, use 30s as default
	//
	//    Primary use case 2: MULTI-KEY SORT COMPARATORS (see below)
	//      return cmp.Or(cmp.Compare(a.Last, b.Last), cmp.Compare(a.First, b.First))
	// -------------------------------------------------------------------------
	fmt.Println("cmp.Or:")
	fmt.Printf("  Or(0, 0, 42)       = %d\n", cmp.Or(0, 0, 42))            // 42
	fmt.Printf("  Or(\"\", \"\", \"hello\") = %q\n", cmp.Or("", "", "hello"))   // "hello"
	fmt.Printf("  Or(7, 42)          = %d\n", cmp.Or(7, 42))               // 7 (first non-zero)

	// Default value pattern:
	userTimeout := 0 // user didn't set a timeout
	effective := cmp.Or(userTimeout, 30)
	fmt.Printf("  effective timeout  = %d (default fallback)\n", effective) // 30

	// -------------------------------------------------------------------------
	// 4. cmp.Compare as a sort comparator with slices.SortFunc.
	//
	//    BEFORE (Go 1.20 style) — verbose, error-prone:
	//
	//      slices.SortFunc(nums, func(a, b int) int {
	//          if a < b { return -1 }
	//          if a > b { return  1 }
	//          return 0
	//      })
	//
	//    AFTER (Go 1.21 style) — one-liner using cmp.Compare:
	//
	//      slices.SortFunc(nums, cmp.Compare)
	//
	// -------------------------------------------------------------------------
	fmt.Println("Sort with cmp.Compare (one-liner):")
	nums := []int{9, 3, 7, 1, 4, 1, 5, 9, 2, 6}
	slices.SortFunc(nums, cmp.Compare) // pass cmp.Compare directly
	fmt.Printf("  sorted: %v\n", nums)

	// Descending: negate the comparator (swap a and b)
	slices.SortFunc(nums, func(a, b int) int { return cmp.Compare(b, a) })
	fmt.Printf("  descending: %v\n", nums)

	// -------------------------------------------------------------------------
	// 5. Multi-key sort with cmp.Or(cmp.Compare(...), cmp.Compare(...))
	//
	//    This is the cleanest way to sort by multiple fields:
	//      - First compare by the primary key.
	//      - If equal (returns 0), cmp.Or moves to the next comparison.
	//      - Chain as many keys as needed.
	//
	//    Old way (Go 1.20):
	//      if a.Last != b.Last { return cmp.Compare(a.Last, b.Last) }
	//      return cmp.Compare(a.First, b.First)
	//
	//    New way (Go 1.22):
	//      return cmp.Or(cmp.Compare(a.Last, b.Last), cmp.Compare(a.First, b.First))
	// -------------------------------------------------------------------------
	fmt.Println("Multi-key sort (Last, then First):")

	type Name struct {
		First string
		Last  string
	}

	names := []Name{
		{"Alice", "Smith"},
		{"Bob", "Jones"},
		{"Charlie", "Smith"},  // same Last as Alice
		{"Alice", "Jones"},    // same as Bob's last name
		{"David", "Smith"},    // same Last as Alice + Charlie
	}

	// Sort by Last name ascending, then First name ascending as tiebreaker.
	slices.SortFunc(names, func(a, b Name) int {
		return cmp.Or(
			cmp.Compare(a.Last, b.Last),   // primary key
			cmp.Compare(a.First, b.First), // tiebreaker
		)
	})

	fmt.Println("  Sorted results:")
	for _, n := range names {
		fmt.Printf("    %s %s\n", n.First, n.Last)
	}
	// Expected:
	//   Alice Jones
	//   Bob   Jones
	//   Alice Smith
	//   Charlie Smith
	//   David Smith

	// -------------------------------------------------------------------------
	// 6. Three-key sort example — age, last name, first name.
	//    cmp.Or chains work the same way with three or more fields.
	// -------------------------------------------------------------------------
	fmt.Println("Three-key sort (Age, Last, First):")

	type Employee struct {
		First string
		Last  string
		Age   int
	}

	employees := []Employee{
		{"Alice", "Smith", 30},
		{"Bob", "Jones", 30},    // same age as Alice
		{"Charlie", "Smith", 25},
		{"Alice", "Jones", 30},  // same age, same first as Alice Smith
		{"David", "Smith", 25},  // same age as Charlie
	}

	slices.SortFunc(employees, func(a, b Employee) int {
		return cmp.Or(
			cmp.Compare(a.Age, b.Age),     // primary: age ascending
			cmp.Compare(a.Last, b.Last),   // secondary: last name
			cmp.Compare(a.First, b.First), // tertiary: first name
		)
	})

	for _, e := range employees {
		fmt.Printf("  %d  %-8s %-8s\n", e.Age, e.Last, e.First)
	}

	// -------------------------------------------------------------------------
	// 7. The cmp.Ordered constraint — what types satisfy it.
	//
	//    cmp.Ordered is defined as:
	//
	//      type Ordered interface {
	//          ~int | ~int8 | ~int16 | ~int32 | ~int64 |
	//          ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
	//          ~float32 | ~float64 |
	//          ~string
	//      }
	//
	//    The ~ means "any type whose underlying type is this". So:
	//
	//      type Celsius float64  // satisfies cmp.Ordered ✓
	//      type UserID  int64    // satisfies cmp.Ordered ✓
	//
	//    Structs, slices, pointers do NOT satisfy cmp.Ordered —
	//    use slices.SortFunc with a custom comparator for those.
	// -------------------------------------------------------------------------
	fmt.Println("cmp.Ordered — named types with primitive underlying types:")

	type Celsius float64 // ~float64 satisfies cmp.Ordered
	type Kelvin float64

	temps := []Celsius{100.0, 0.0, 37.0, -40.0, 20.0}
	slices.SortFunc(temps, func(a, b Celsius) int {
		// cmp.Compare works because Celsius has underlying type float64
		return cmp.Compare(a, b)
	})
	fmt.Printf("  Sorted Celsius: %v\n", temps)

	// -------------------------------------------------------------------------
	// 8. Old vs New comparator pattern — side by side.
	// -------------------------------------------------------------------------
	fmt.Println("Before cmp (Go 1.20 verbose comparator):")
	fmt.Println(`
  // OLD: manual three-way comparator
  slices.SortFunc(people, func(a, b Person) int {
      if a.Age < b.Age { return -1 }
      if a.Age > b.Age { return  1 }
      // tiebreaker
      if a.Name < b.Name { return -1 }
      if a.Name > b.Name { return  1 }
      return 0
  })

  // NEW: cmp.Or + cmp.Compare
  slices.SortFunc(people, func(a, b Person) int {
      return cmp.Or(cmp.Compare(a.Age, b.Age), cmp.Compare(a.Name, b.Name))
  })`)

	_ = Kelvin(273.15) // suppress unused warning
}
