package io_bufio

import (
	"bufio"
	"fmt"
	"strings"
)

// bufioReaderExample shows why buffered I/O matters and how to use
// bufio.Reader and bufio.Scanner.
//
// The problem: every call to io.Reader.Read() may be a syscall.
// Reading a file character-by-character without buffering = one syscall per byte.
// bufio.Reader wraps any io.Reader with an in-memory buffer (default 4096 bytes),
// reducing syscalls dramatically.
func bufioReaderExample() {
	fmt.Println("--- bufio.Reader & Scanner ---")

	raw := strings.NewReader("line one\nline two\nline three\n")

	// ── 1. bufio.NewReader ───────────────────────────────────────────────────
	br := bufio.NewReader(raw)

	// ReadString reads until the delimiter (inclusive).
	// Returns partial data + error if EOF is hit before the delimiter.
	for {
		line, err := br.ReadString('\n')
		if line != "" {
			fmt.Printf("  ReadString: %q\n", line)
		}
		if err != nil { // io.EOF on last line
			break
		}
	}

	// ── 2. bufio.Scanner — the idiomatic line reader ──────────────────────────
	// Scanner is higher-level than Reader: it hides the EOF handling.
	// Default split function is ScanLines (strips the '\n').
	csv := strings.NewReader("alice,30\nbob,25\ncharlie,35")
	scanner := bufio.NewScanner(csv)
	fmt.Println("\n  Scanner (ScanLines):")
	for scanner.Scan() {
		fmt.Printf("    line: %q\n", scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("  scanner error:", err)
	}

	// ── 3. Custom split function: ScanWords ──────────────────────────────────
	words := strings.NewReader("the quick brown fox")
	ws := bufio.NewScanner(words)
	ws.Split(bufio.ScanWords)
	fmt.Print("\n  ScanWords: ")
	for ws.Scan() {
		fmt.Printf("%q ", ws.Text())
	}
	fmt.Println()

	// ── 4. bufio.Scanner with larger buffer ──────────────────────────────────
	// Default max token size is 64 KB. For longer lines (log lines, base64
	// blobs) you must call scanner.Buffer() before the first Scan().
	bigLine := strings.Repeat("x", 128*1024) // 128 KB line
	s := bufio.NewScanner(strings.NewReader(bigLine))
	s.Buffer(make([]byte, 256*1024), 256*1024) // raise the cap
	s.Scan()
	fmt.Printf("\n  Long-line scanner: read %d bytes (buffer expanded)\n", len(s.Bytes()))

	fmt.Println("\n💡 Use bufio.Scanner for line-by-line reading.")
	fmt.Println("   Use bufio.Reader when you need Peek, ReadByte, or ReadRune.")
}

// bufioWriterExample shows buffered writing and why you must always Flush.
func bufioWriterExample() {
	fmt.Println("--- bufio.Writer ---")

	var out strings.Builder
	bw := bufio.NewWriterSize(&out, 16) // tiny 16-byte buffer for demo

	writes := []string{"hello", ", ", "buffered", " world"}
	for _, s := range writes {
		n, _ := bw.WriteString(s)
		fmt.Printf("  WriteString(%q) → %d bytes written to buffer\n", s, n)
		fmt.Printf("    buffered=%d available=%d\n", bw.Buffered(), bw.Available())
	}

	// Data may still be sitting in the buffer — Flush sends it to the
	// underlying writer. ALWAYS call Flush (ideally via defer).
	fmt.Println("\n  Flushing…")
	_ = bw.Flush()
	fmt.Printf("  out.String() = %q\n", out.String())

	// ── Defer Flush pattern (production idiom) ───────────────────────────────
	fmt.Println("\n  Production pattern — defer bw.Flush():")
	fmt.Println(`
    func writeReport(w io.Writer) error {
        bw := bufio.NewWriter(w)
        defer bw.Flush()          // ← always runs, even on early return

        bw.WriteString("header\n")
        // ... write many small items ...
        return nil
    }`)

	fmt.Println("\n💡 Without Flush, the last chunk of data stays in the buffer")
	fmt.Println("   and is silently lost when the function returns.")
}
