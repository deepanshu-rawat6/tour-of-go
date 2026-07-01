package io_bufio

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// streamingExample demonstrates composing readers and writers to build
// streaming pipelines without loading the full payload into memory.
//
// The key insight: every transformation (compress, hash, encrypt, log)
// is just another io.Reader or io.Writer wrapping the previous one.
// Data flows through the chain one chunk at a time.
func streamingExample() {
	fmt.Println("--- streaming with TeeReader & MultiWriter ---")

	// ── 1. io.TeeReader ──────────────────────────────────────────────────────
	// Like the Unix `tee` command: every byte read from the returned reader
	// is simultaneously written to a secondary writer.
	// Perfect for computing a hash or audit log while streaming data.
	source := strings.NewReader("streaming payload data")
	var audit bytes.Buffer
	teeReader := io.TeeReader(source, &audit)

	// Primary consumer: just reads from teeReader
	var dest bytes.Buffer
	io.Copy(&dest, teeReader)

	fmt.Printf("  destination: %q\n", dest.String())
	fmt.Printf("  audit copy:  %q\n", audit.String())
	fmt.Println("  (both received every byte — no double-read of the source)")

	// ── 2. io.MultiWriter ────────────────────────────────────────────────────
	// Fan out: write once, land in multiple sinks.
	var primary, backup bytes.Buffer
	mw := io.MultiWriter(&primary, &backup)
	fmt.Fprintf(mw, "critical record: id=%d", 42)

	fmt.Printf("\n  primary: %q\n", primary.String())
	fmt.Printf("  backup:  %q\n", backup.String())

	// ── 3. Chaining transformations ──────────────────────────────────────────
	// Imagine: gzip reader → limit reader → tee reader (for hash) → dest
	// Each layer adds behaviour; no layer knows about the others.
	payload := strings.NewReader("AAABBBCCCDDDEEE (imagine this is gzipped)")
	limited := io.LimitReader(payload, 20) // cap at 20 bytes

	var hashLog bytes.Buffer
	teed := io.TeeReader(limited, &hashLog)

	var result bytes.Buffer
	n, _ := io.Copy(&result, teed)

	fmt.Printf("\n  Chained pipeline (limit=20):\n")
	fmt.Printf("    bytes transferred: %d\n", n)
	fmt.Printf("    result:   %q\n", result.String())
	fmt.Printf("    hashLog:  %q\n", hashLog.String())

	// ── 4. io.Pipe ── connecting goroutines ──────────────────────────────────
	pipesExample()
}

// pipesExample demonstrates io.Pipe for connecting a producer goroutine to a
// consumer goroutine with a synchronous, unbuffered byte channel.
func pipesExample() {
	fmt.Println("\n--- io.Pipe ---")

	// io.Pipe creates a synchronous, in-memory pipe.
	// PipeWriter and PipeReader are connected: a write blocks until the
	// reader consumes the data. No intermediate buffer is allocated.
	pr, pw := io.Pipe()

	done := make(chan string)

	// Producer goroutine — writes to pw
	go func() {
		defer pw.Close() // closing signals EOF to the reader
		items := []string{"chunk-1", "chunk-2", "chunk-3"}
		for _, item := range items {
			fmt.Fprintf(pw, "[%s]", item)
		}
	}()

	// Consumer goroutine — reads from pr (acts like any io.Reader)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, pr)
		done <- buf.String()
	}()

	result := <-done
	fmt.Printf("  pipe result: %q\n", result)

	fmt.Println("\n💡 Use io.Pipe when:")
	fmt.Println("   • One goroutine generates data, another consumes it.")
	fmt.Println("   • You need to pass an io.Reader to a library (e.g., http.NewRequest)")
	fmt.Println("     but your data arrives asynchronously.")
	fmt.Println("   • You want zero-copy streaming between two stages.")
}
