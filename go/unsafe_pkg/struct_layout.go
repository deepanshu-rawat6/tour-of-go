package unsafe_pkg

// struct_layout.go — padding, field ordering, cache-line awareness
//
// Why does field order matter?
//   The Go spec requires each field to be aligned to a multiple of its own
//   size. To satisfy this, the compiler inserts invisible "padding" bytes
//   between fields. Reordering fields from largest to smallest typically
//   minimises padding.
//
// Alignment rule:
//   A field of type T must start at an address that is a multiple of
//   unsafe.Alignof(T). On most 64-bit platforms:
//     bool, int8, uint8  → 1-byte alignment
//     int16, uint16      → 2-byte alignment
//     int32, uint32, float32 → 4-byte alignment
//     int64, uint64, float64, pointer → 8-byte alignment
//
// Cache-line awareness (64 bytes on x86/ARM):
//   When two goroutines update fields in the SAME cache line concurrently,
//   the CPU must synchronise the line between cores even though no Go-level
//   lock is held. This is called FALSE SHARING and is a common cause of
//   unexpected performance degradation on multi-core systems.
//   Fix: add padding to push hot fields onto separate cache lines.
//
// sync/atomic alignment requirement:
//   Atomic operations on 64-bit values (atomic.AddInt64, etc.) require the
//   value to be 64-bit aligned. This is guaranteed for package-level and
//   heap-allocated variables, but not for fields in structs allocated on a
//   32-bit-aligned address (e.g. a struct embedded in another). Always put
//   64-bit atomic fields FIRST in a struct to guarantee alignment.

import (
	"fmt"
	"reflect"
	"unsafe"
)

// -----------------------------------------------------------------------------
// 1. Same fields, different order → different sizes
// -----------------------------------------------------------------------------

// wastefulLayout has the fields in an order that maximises padding.
// Layout on a 64-bit system:
//   A  bool     @ offset 0  (1 byte + 7 bytes padding = 8 bytes)
//   B  float64  @ offset 8  (8 bytes)
//   C  bool     @ offset 16 (1 byte + 7 bytes padding = 8 bytes)
//   D  float64  @ offset 24 (8 bytes)
//   E  int8     @ offset 32 (1 byte + 7 bytes padding = 8 bytes)
//   TOTAL: 40 bytes
type wastefulLayout struct {
	A bool
	B float64
	C bool
	D float64
	E int8
}

// efficientLayout has the same fields packed largest-first.
// Layout on a 64-bit system:
//   B  float64  @ offset 0  (8 bytes)
//   D  float64  @ offset 8  (8 bytes)
//   A  bool     @ offset 16 (1 byte)
//   C  bool     @ offset 17 (1 byte)
//   E  int8     @ offset 18 (1 byte + 5 bytes padding to reach size 24)
//   TOTAL: 24 bytes  (40% smaller!)
type efficientLayout struct {
	B float64
	D float64
	A bool
	C bool
	E int8
}

// -----------------------------------------------------------------------------
// 2. False sharing — cache line contention (conceptual, not run concurrently)
// -----------------------------------------------------------------------------

// falseSharingPair: two counters on the SAME 64-byte cache line.
// If goroutine 1 writes Counter1 and goroutine 2 writes Counter2, the CPU
// must bounce the entire cache line between cores → serialises what should be
// parallel updates.
type falseSharingPair struct {
	Counter1 int64 // hot field — goroutine 1 owns this
	Counter2 int64 // hot field — goroutine 2 owns this
}

// noFalseSharingPair: each counter lives on its own cache line.
// Padding ensures Counter1 and Counter2 are in different 64-byte cache lines.
const cacheLineSize = 64

type noFalseSharingPair struct {
	Counter1 int64
	// _pad1 pushes Counter2 onto the next cache line.
	// Size of Counter1 is 8 bytes; we need 64-8 = 56 bytes of padding.
	_pad1    [cacheLineSize - unsafe.Sizeof(int64(0))]byte
	Counter2 int64
	_pad2    [cacheLineSize - unsafe.Sizeof(int64(0))]byte
}

// -----------------------------------------------------------------------------
// 3. Atomic alignment: put int64 fields first
// -----------------------------------------------------------------------------

// atomicBad: AlignedCounter is at offset 1 if embedded after a bool on a
// 32-bit system → misaligned → panic on 32-bit, silent bug on some ARM.
type atomicBad struct {
	Flag          bool
	AlignedCounter int64 // NOT first — could be misaligned on 32-bit
}

// atomicGood: int64 first → always 8-byte aligned regardless of arch.
type atomicGood struct {
	AlignedCounter int64 // FIRST — guaranteed 8-byte aligned
	Flag          bool
}

// -----------------------------------------------------------------------------
// structLayoutExample
// -----------------------------------------------------------------------------

func structLayoutExample() {
	fmt.Println("--- struct Layout & Padding ---")

	// --- Size comparison ---
	fmt.Println()
	fmt.Println("  [layout] same fields, different order:")
	fmt.Printf("    wastefulLayout:   Sizeof=%d bytes\n", unsafe.Sizeof(wastefulLayout{}))
	fmt.Printf("    efficientLayout:  Sizeof=%d bytes\n", unsafe.Sizeof(efficientLayout{}))

	// --- Field offsets for wastefulLayout ---
	fmt.Println()
	fmt.Println("  [layout] wastefulLayout field offsets (note the gaps):")
	var w wastefulLayout
	wType := reflect.TypeOf(w)
	for i := 0; i < wType.NumField(); i++ {
		f := wType.Field(i)
		fmt.Printf("    %-6s %-8s offset=%-3d size=%d\n",
			f.Name, f.Type.Kind(), f.Offset, f.Type.Size())
	}

	// --- Field offsets for efficientLayout ---
	fmt.Println()
	fmt.Println("  [layout] efficientLayout field offsets (packed tight):")
	var e efficientLayout
	eType := reflect.TypeOf(e)
	for i := 0; i < eType.NumField(); i++ {
		f := eType.Field(i)
		fmt.Printf("    %-6s %-8s offset=%-3d size=%d\n",
			f.Name, f.Type.Kind(), f.Offset, f.Type.Size())
	}

	// --- False sharing ---
	fmt.Println()
	fmt.Printf("  [layout] falseSharingPair:   Sizeof=%d (both counters in same cache line)\n",
		unsafe.Sizeof(falseSharingPair{}))
	fmt.Printf("  [layout] noFalseSharingPair: Sizeof=%d (counters in separate cache lines)\n",
		unsafe.Sizeof(noFalseSharingPair{}))
	fmt.Println("  [layout] Cache line = 64 bytes (x86/ARM64).")
	fmt.Println("           Benchmark with -benchmem -count=5 to see the difference.")

	// --- Atomic alignment ---
	fmt.Println()
	fmt.Printf("  [layout] atomicBad  — AlignedCounter offset=%d\n",
		unsafe.Offsetof(atomicBad{}.AlignedCounter))
	fmt.Printf("  [layout] atomicGood — AlignedCounter offset=%d (always 0 → safe)\n",
		unsafe.Offsetof(atomicGood{}.AlignedCounter))
	fmt.Println("  [layout] Rule: put 64-bit atomic fields FIRST in struct.")

	// --- Quick alignment check helper ---
	fmt.Println()
	fmt.Println("  [layout] Alignment values on this platform:")
	for _, v := range []any{
		bool(false), int8(0), int16(0), int32(0), int64(0),
		float32(0), float64(0), complex64(0), complex128(0),
		uintptr(0), unsafe.Pointer(nil),
	} {
		t := reflect.TypeOf(v)
		fmt.Printf("    %-20s Sizeof=%-3d Alignof=%d\n",
			t.String(), t.Size(), t.Align())
	}
}
