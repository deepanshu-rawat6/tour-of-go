package unsafe_pkg

// pointer_arithmetic.go — unsafe.Pointer, uintptr, string/slice headers
//
// The three safe unsafe.Pointer conversion rules (from the spec):
//
//   1. *T1 → unsafe.Pointer → *T2
//      Reinterpret the memory at the same address as a different type.
//
//   2. unsafe.Pointer → uintptr (for Println/fmt only)
//      Convert to an integer for PRINTING only. Never dereference later.
//
//   3. unsafe.Pointer → uintptr → arithmetic → unsafe.Pointer
//      Walk struct fields or array elements by offset.
//      ⚠️  MUST be done in a SINGLE EXPRESSION — see GC rule below.
//
// THE GC RULE (critical):
//   uintptr is just an integer — it is NOT a GC root.
//   If you store a pointer as uintptr and a GC runs before you convert it
//   back, the GC may move or collect the object (the runtime is free to do
//   this in future versions of Go). The address in your uintptr is now stale.
//
//   Safe: unsafe.Pointer(uintptr(p) + offset)   ← single expression, no GC point
//   UNSAFE: u := uintptr(p); <anything>; unsafe.Pointer(u)  ← GC can run between

import (
	"fmt"
	"unsafe"
)

// -----------------------------------------------------------------------------
// 1. unsafe.Sizeof, unsafe.Alignof, unsafe.Offsetof
// -----------------------------------------------------------------------------

// point3D is a simple struct used to demonstrate layout queries.
type point3D struct {
	X float64 // 8 bytes
	Y float64 // 8 bytes
	Z float64 // 8 bytes
}

// mixedStruct demonstrates alignment padding.
type mixedStruct struct {
	A bool    // 1 byte  @ offset 0
	// 7 bytes padding (B must be 8-byte aligned)
	B float64 // 8 bytes @ offset 8
	C int32   // 4 bytes @ offset 16
	// 4 bytes padding (struct size must be multiple of its largest alignment = 8)
}

func showSizes() {
	var p point3D
	var m mixedStruct

	fmt.Println("  [unsafe] Sizeof / Alignof / Offsetof:")
	fmt.Printf("    point3D: Sizeof=%d  Alignof=%d\n",
		unsafe.Sizeof(p), unsafe.Alignof(p))
	fmt.Printf("    point3D.X: Offsetof=%d\n", unsafe.Offsetof(p.X))
	fmt.Printf("    point3D.Y: Offsetof=%d\n", unsafe.Offsetof(p.Y))
	fmt.Printf("    point3D.Z: Offsetof=%d\n", unsafe.Offsetof(p.Z))
	fmt.Println()
	fmt.Printf("    mixedStruct: Sizeof=%d  Alignof=%d\n",
		unsafe.Sizeof(m), unsafe.Alignof(m))
	fmt.Printf("    mixedStruct.A: Sizeof=1, Offsetof=%d\n", unsafe.Offsetof(m.A))
	fmt.Printf("    mixedStruct.B: Sizeof=8, Offsetof=%d (7 bytes padding before B)\n",
		unsafe.Offsetof(m.B))
	fmt.Printf("    mixedStruct.C: Sizeof=4, Offsetof=%d\n", unsafe.Offsetof(m.C))
	fmt.Printf("    total: 1+7(pad)+8+4+4(pad) = %d bytes\n", unsafe.Sizeof(m))
}

// -----------------------------------------------------------------------------
// 2. Walking struct fields by address arithmetic
// -----------------------------------------------------------------------------

// walkFields demonstrates accessing struct fields by computing their addresses
// via unsafe pointer arithmetic.
func walkFields() {
	p := point3D{X: 1.1, Y: 2.2, Z: 3.3}
	base := unsafe.Pointer(&p)

	fmt.Println("  [unsafe] walking point3D fields by address:")

	// SAFE: the arithmetic and conversion happen in a single expression.
	// The GC sees unsafe.Pointer (a GC root), not a bare uintptr.
	xPtr := (*float64)(unsafe.Pointer(uintptr(base) + unsafe.Offsetof(p.X)))
	yPtr := (*float64)(unsafe.Pointer(uintptr(base) + unsafe.Offsetof(p.Y)))
	zPtr := (*float64)(unsafe.Pointer(uintptr(base) + unsafe.Offsetof(p.Z)))

	fmt.Printf("    X=%v  Y=%v  Z=%v\n", *xPtr, *yPtr, *zPtr)

	// Mutate via pointer — same memory as p.Z.
	*zPtr = 9.9
	fmt.Printf("    After *zPtr=9.9: p.Z=%v\n", p.Z)
}

// -----------------------------------------------------------------------------
// 3. String header layout: {Data *byte, Len int}
//
// A Go string is a 2-word struct:
//   type StringHeader struct { Data uintptr; Len int }
//
// The DATA pointer is immutable — you cannot legally write through it.
// -----------------------------------------------------------------------------

// stringHeader is our manual copy of reflect.StringHeader's layout.
// We define it here to avoid the reflect import in this file.
type stringHeader struct {
	Data unsafe.Pointer
	Len  int
}

func showStringHeader() {
	s := "hello, unsafe"

	// Reinterpret the string variable as our stringHeader struct.
	// This is safe: we're reading the header, not mutating string data.
	hdr := (*stringHeader)(unsafe.Pointer(&s))

	fmt.Println("  [unsafe] string header:")
	fmt.Printf("    string: %q\n", s)
	fmt.Printf("    Data (ptr to first byte): %p\n", hdr.Data)
	fmt.Printf("    Len: %d\n", hdr.Len)
	fmt.Printf("    Sizeof(string): %d bytes (2 words on 64-bit)\n", unsafe.Sizeof(s))
}

// -----------------------------------------------------------------------------
// 4. []byte ↔ string zero-copy conversion
//
// Normal conversion: string(b) allocates a new backing array and copies.
// unsafe conversion: reuse the same memory, zero allocation.
//
// ⚠️  DANGER: the resulting string MUST NOT outlive the []byte, and the []byte
//   MUST NOT be modified while the string exists. Violating either causes
//   silent memory corruption. Use only in hot paths with tight control over
//   lifetimes (e.g., inside a single function, never stored).
//
// Modern alternative (Go 1.20+): strings.Builder + WriteString, or use
//   unsafe.String (Go 1.17+) which is the officially blessed approach.
// -----------------------------------------------------------------------------

// bytesToStringUnsafe converts []byte to string WITHOUT allocation.
// The caller MUST NOT modify b after calling this function.
func bytesToStringUnsafe(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	// Go 1.17+ officially blessed approach using unsafe.String:
	// return unsafe.String(&b[0], len(b))
	//
	// The manual header approach (shown for educational purposes):
	return *(*string)(unsafe.Pointer(&b))
}

// stringToBytesUnsafe converts string to []byte WITHOUT allocation.
// The caller MUST NOT write to the returned slice — string data is read-only.
func stringToBytesUnsafe(s string) []byte {
	if s == "" {
		return nil
	}
	// sliceHeader layout: {Data unsafe.Pointer, Len int, Cap int}
	// We build a slice header that points at the string's backing array.
	type sliceHeader struct {
		Data unsafe.Pointer
		Len  int
		Cap  int
	}
	sh := (*stringHeader)(unsafe.Pointer(&s))
	return *(*[]byte)(unsafe.Pointer(&sliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len, // Cap == Len: prevents append from writing
	}))
}

func showZeroCopyConversion() {
	b := []byte("zero copy bytes")

	fmt.Println("  [unsafe] []byte ↔ string zero-copy:")
	s := bytesToStringUnsafe(b)
	fmt.Printf("    bytesToStringUnsafe: %q\n", s)

	b2 := stringToBytesUnsafe(s)
	fmt.Printf("    stringToBytesUnsafe: %v\n", b2[:5])
	fmt.Println("    ⚠️  do NOT append or write to b2 — it aliases string memory")
}

// -----------------------------------------------------------------------------
// pointerArithmeticExample — calls all sub-demos
// -----------------------------------------------------------------------------

func pointerArithmeticExample() {
	fmt.Println("--- unsafe: Pointer Arithmetic ---")
	fmt.Println()
	showSizes()
	fmt.Println()
	walkFields()
	fmt.Println()
	showStringHeader()
	fmt.Println()
	showZeroCopyConversion()

	// GC rule reminder
	fmt.Println()
	fmt.Println("  [unsafe] GC RULE SUMMARY:")
	fmt.Println("    SAFE:   ptr2 := (*T)(unsafe.Pointer(uintptr(ptr1) + offset))")
	fmt.Println("    UNSAFE: u := uintptr(ptr1)  // ← GC point here!")
	fmt.Println("            ptr2 := (*T)(unsafe.Pointer(u)) // u may be stale")
	fmt.Println("    Always keep the entire arithmetic in a single expression.")
}
