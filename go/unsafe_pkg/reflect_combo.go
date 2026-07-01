package unsafe_pkg

// reflect_combo.go — reflect + unsafe: unexported fields, headers, type punning
//
// ⚠️  This is the riskiest use of unsafe. Understand every rule before using
//    any of these techniques in production code.

import (
	"fmt"
	"math"
	"reflect"
	"unsafe"
)

// -----------------------------------------------------------------------------
// 1. Setting unexported struct fields via reflect + unsafe
//
// Use cases in real projects:
//   • ORMs / struct mappers that need to populate private fields.
//   • Test helpers that inject mock state into library structs.
//   • Implementing Copy() on structs whose authors didn't export everything.
//
// RULES:
//   a. The struct must be addressable (pointer or reflect.New).
//   b. You must KNOW the field layout won't change between Go versions.
//      Using this on stdlib structs is fragile and can break on any update.
//   c. This bypasses the encapsulation intentionally provided by the author.
//      Document why it's necessary and write tests that verify layout.
// -----------------------------------------------------------------------------

// privateFields contains unexported fields — in real code these might be from
// a library you don't control.
type privateFields struct {
	name  string // unexported
	score int    // unexported
	Tag   string // exported (no unsafe needed)
}

// setUnexportedField sets field at index i in the struct pointed to by v.
// v must be a reflect.Value of kind Ptr to a struct.
func setUnexportedField(v reflect.Value, fieldIndex int, value any) {
	// v.Elem() unwraps the pointer to get the struct Value.
	field := v.Elem().Field(fieldIndex)

	// reflect.Value.Set panics on unexported fields. We work around this by
	// getting the field's address as an unsafe.Pointer and constructing a new
	// reflect.Value that is addressable and exported (NewAt does this).
	//
	// reflect.NewAt(T, ptr) creates a *T Value pointing at ptr.
	// .Elem() dereferences it to T. .Set() sets the value.
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).
		Elem().
		Set(reflect.ValueOf(value))
}

func showUnexportedFieldAccess() {
	fmt.Println("  [reflect+unsafe] setting unexported fields:")

	pf := &privateFields{Tag: "visible"}
	fmt.Printf("    before: name=%q score=%d tag=%q\n", pf.name, pf.score, pf.Tag)

	rv := reflect.ValueOf(pf)
	setUnexportedField(rv, 0, "alice")   // field 0: name
	setUnexportedField(rv, 1, 42)        // field 1: score

	fmt.Printf("    after:  name=%q score=%d tag=%q\n", pf.name, pf.score, pf.Tag)
}

// -----------------------------------------------------------------------------
// 2. reflect.StringHeader and reflect.SliceHeader (deprecated in Go 1.20)
//
// These types exposed the internal layout of strings and slices via reflect.
// They are deprecated because they used uintptr for Data (not a GC root),
// which was unsafe. The modern replacements use unsafe.Pointer directly.
//
// Deprecated (still works but avoid in new code):
//   reflect.StringHeader{Data uintptr, Len int}
//   reflect.SliceHeader{Data uintptr, Len int, Cap int}
//
// Modern replacements (Go 1.17+):
//   unsafe.String(ptr *byte, len IntegerType) string  — slice-of-bytes→string
//   unsafe.StringData(str string) *byte               — string→backing array
//   unsafe.SliceData(slice []T) *T                    — slice→backing array
//   unsafe.Slice(ptr *T, len IntegerType) []T          — pointer+len→slice
// -----------------------------------------------------------------------------

func showStringSliceHeaders() {
	fmt.Println("  [reflect+unsafe] string / slice headers:")

	s := "hello world"
	b := []byte{72, 101, 108, 108, 111}

	// DEPRECATED approach (shown for recognition — don't write new code like this):
	//   sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	//   fmt.Println(sh.Data, sh.Len)
	fmt.Println("    (reflect.StringHeader/SliceHeader are deprecated — see below)")

	// MODERN approach (Go 1.17+):
	// unsafe.StringData returns a pointer to the first byte of the string's
	// backing array. If s is "", the result is unspecified (may be nil).
	dataPtr := unsafe.StringData(s)
	fmt.Printf("    unsafe.StringData(%q) = %p (first byte: %c)\n",
		s, dataPtr, *dataPtr)

	// unsafe.SliceData returns a pointer to the backing array of a slice.
	slicePtr := unsafe.SliceData(b)
	fmt.Printf("    unsafe.SliceData([]byte) = %p (first byte: %d)\n",
		slicePtr, *slicePtr)

	// unsafe.String creates a string from a *byte pointer and length.
	// This is the zero-copy, officially-safe way to convert []byte → string.
	// Rule: the resulting string must NOT outlive the []byte.
	reconstructed := unsafe.String(slicePtr, len(b))
	fmt.Printf("    unsafe.String from slice pointer: %q\n", reconstructed)

	// unsafe.Slice creates a []T from a pointer and length.
	// Use to wrap a C array, a memory-mapped region, etc.
	backToSlice := unsafe.Slice(dataPtr, len(s))
	fmt.Printf("    unsafe.Slice back to []byte (first 5): %v\n", backToSlice[:5])
}

// -----------------------------------------------------------------------------
// 3. Type punning — reinterpreting bits without conversion
//
// Type punning = treating the bits of one type as if they are another type.
//
// Float64 bits as Uint64:
//   This is how math.Float64bits and math.Float64frombits are implemented.
//   It's also used in fast inverse-square-root algorithms, NaN boxing, etc.
//
// SAFETY:
//   OK when:
//     • Both types have the same size.
//     • You own all the code (same binary, same arch).
//     • You do NOT serialise the result (endianness + floating-point encoding
//       differ across architectures/languages).
//
//   DANGEROUS when:
//     • Serialised over the network or to disk (endianness).
//     • Shared with C code that has different struct padding rules.
//     • Used as a map key (NaN != NaN in IEEE 754, but same bits).
// -----------------------------------------------------------------------------

// float64ToUint64Bits reinterprets the IEEE 754 bit pattern of f as a uint64.
// This is exactly what math.Float64bits does.
func float64ToUint64Bits(f float64) uint64 {
	// The three-step conversion:
	//   Step 1: *float64    → unsafe.Pointer  (take address, erase type)
	//   Step 2: unsafe.Pointer → *uint64       (give it a new type)
	//   Step 3: *uint64     → uint64           (dereference)
	return *(*uint64)(unsafe.Pointer(&f))
}

// uint64ToFloat64Bits is the inverse — same bit pattern, interpreted as float64.
func uint64ToFloat64Bits(u uint64) float64 {
	return *(*float64)(unsafe.Pointer(&u))
}

func showTypePunning() {
	fmt.Println("  [reflect+unsafe] type punning — float64 ↔ uint64:")

	values := []float64{0.0, 1.0, -1.0, math.Pi, math.Inf(1), math.NaN()}
	for _, v := range values {
		bits := float64ToUint64Bits(v)
		roundTrip := uint64ToFloat64Bits(bits)
		// Verify against stdlib — should always match.
		stdBits := math.Float64bits(v)
		fmt.Printf("    %10.4f → 0x%016X (stdlib=%016X match=%v roundTrip=%v)\n",
			v, bits, stdBits, bits == stdBits, roundTrip == v || math.IsNaN(v))
	}

	fmt.Println()
	fmt.Println("    Serialisation warning:")
	fmt.Println("    0x3FF0000000000000 = 1.0 on little-endian (x86).")
	fmt.Println("    On big-endian (SPARC), the bytes are reversed.")
	fmt.Println("    NEVER write these raw bits to a file/socket for cross-arch exchange.")
}

// -----------------------------------------------------------------------------
// reflectComboExample — orchestrates all three sub-demos
// -----------------------------------------------------------------------------

func reflectComboExample() {
	fmt.Println("--- reflect + unsafe combo ---")
	fmt.Println()
	showUnexportedFieldAccess()
	fmt.Println()
	showStringSliceHeaders()
	fmt.Println()
	showTypePunning()
}
