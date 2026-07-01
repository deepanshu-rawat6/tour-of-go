package io_bufio

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

// basicWritersExample demonstrates the io.Writer interface and common
// implementations: bytes.Buffer, strings.Builder, os.Stdout, io.MultiWriter,
// io.Discard.
//
// The io.Writer contract:
//   Write(p []byte) (n int, err error)
//
// Write must return a non-nil error if it returns n < len(p).
// It must NOT modify the slice p, even temporarily.
func basicWritersExample() {
	fmt.Println("--- io.Writer basics ---")

	// ── 1. bytes.Buffer ──────────────────────────────────────────────────────
	// The workhorse in-memory Writer (and Reader). Grows as needed.
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "item=%d ", 1)
	fmt.Fprintf(&buf, "item=%d ", 2)
	fmt.Fprintf(&buf, "item=%d", 3)
	fmt.Printf("bytes.Buffer: %q\n", buf.String())

	// Reset reuses the underlying memory — avoids allocation on repeated use.
	buf.Reset()
	fmt.Fprintf(&buf, "reused: %s", "✓")
	fmt.Printf("after Reset: %q\n", buf.String())

	// ── 2. strings.Builder ───────────────────────────────────────────────────
	// Optimised for building strings. Never copies on Grow.
	// Prefer over bytes.Buffer when the final result is a string.
	var sb strings.Builder
	words := []string{"Go", "is", "fast"}
	for i, w := range words {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(w)
	}
	fmt.Printf("\nstrings.Builder: %q\n", sb.String())

	// ── 3. os.Stdout is an io.Writer ─────────────────────────────────────────
	// os.File implements io.Writer, so you can pass it anywhere a Writer is
	// accepted — log files, config output, piped CLIs.
	fmt.Fprintln(os.Stdout, "\nos.Stdout (via fmt.Fprintln): hello from io.Writer")

	// ── 4. io.MultiWriter ────────────────────────────────────────────────────
	// Fans-out writes to multiple Writers simultaneously.
	// Classic pattern: tee logs to a file AND stdout.
	var logBuf bytes.Buffer
	tee := io.MultiWriter(os.Stdout, &logBuf)
	fmt.Fprintln(tee, "io.MultiWriter: written to stdout AND logBuf")
	fmt.Printf("logBuf captured: %q\n", logBuf.String())

	// ── 5. io.Discard ────────────────────────────────────────────────────────
	// A Writer that silently discards everything written to it.
	// Useful in benchmarks or when you must drain a reader but don't need data.
	n, _ := fmt.Fprintln(io.Discard, "this goes nowhere")
	fmt.Printf("\nio.Discard: wrote %d bytes, got nothing back\n", n)

	// ── 6. Writing from a Reader to a Writer ─────────────────────────────────
	// io.Copy is the canonical way. It allocates an internal 32 KB buffer.
	src := strings.NewReader("streaming source data")
	var dst bytes.Buffer
	written, _ := io.Copy(&dst, src)
	fmt.Printf("\nio.Copy: transferred %d bytes → %q\n", written, dst.String())

	fmt.Println("\n💡 fmt.Fprintf, io.Copy, json.NewEncoder — all take io.Writer.")
	fmt.Println("   This is Go's composability: swap bytes.Buffer for a network")
	fmt.Println("   connection and your code doesn't change.")
}
