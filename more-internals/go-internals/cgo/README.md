# cgo & FFI: Bridging Go and C

The Foreign Function Interface (FFI) allows Go to interact with C libraries, enabling the use of high-performance existing drivers like `sqlite3`, `libnet`, or custom C code. However, this power comes with a significant runtime cost.

---

## 🏗️ How cgo Works

When you import `"C"`, Go invokes the `cgo` tool to generate Go and C glue code.

```go
package main

/*
#include <stdio.h>
#include <stdlib.h>

void hello() {
    printf("Hello from C!\n");
}
*/
import "C"

func main() {
    C.hello()
}
```

---

## 📉 The Hidden Overhead

Calling a C function from Go is **not** a simple jump. It involves a complex dance:

1.  **Stack Switch**: Go uses growable stacks (starts at ~2KB), while C uses fixed-size OS stacks. Go must switch to an OS stack before calling C.
2.  **Scheduler Management**: Go's scheduler needs to know that a goroutine is entering C code, as C can block indefinitely.
3.  **Argument Passing**: Complex Go types (slices, strings, pointers) must be converted into C-compatible pointers.
4.  **Register Saving**: Go must save all its CPU registers to ensure C doesn't overwrite them.

**Result**: A `cgo` call is roughly **40-60x slower** than a regular Go function call.

---

## ⚠️ Safety and Pitfalls

1.  **Pointer Rules**: Never pass a Go pointer to C if that pointer contains another Go pointer (unless specifically allowed). Go's Garbage Collector (GC) moves objects around, and C won't know where they went.
2.  **Memory Management**: Memory allocated in C (`malloc`) is **not** managed by Go's GC. You **must** manually free it.
3.  **Panic/Signals**: C code that crashes will crash the entire Go process. Go's `panic`/`recover` will not catch C crashes.

---

## 🚀 Optimization Strategies

*   **Batching**: Instead of calling C in a loop, pass an array of data and do the work in a single C call.
*   **Static Linking**: Prefer static linking for C libraries to avoid runtime dependency issues.
*   **Avoid String Conversion**: Passing strings requires a `C.CString` call, which involves an allocation and a copy.

---

## 🛠️ Real-World Examples
*   **`database/sql` drivers**: Many high-performance DB drivers (like SQLite) are `cgo`-based.
*   **Networking**: High-performance packet capture (libpcap) often requires `cgo`.
*   **Cryptography**: Interfacing with OpenSSL or hardware security modules (HSMs).
