package io_bufio

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// basicReadersExample demonstrates the io.Reader interface and common
// implementations: strings.Reader, bytes.Reader, io.LimitReader, io.MultiReader.
//
// The io.Reader contract:
//   Read(p []byte) (n int, err error)
//
// A Read call fills p with up to len(p) bytes and returns how many were filled.
// It returns io.EOF when there is no more data — this is NOT an error, it is
// the normal end-of-stream signal.
func basicReadersExample() {
	fmt.Println("--- io.Reader basics ---")

	// ── 1. strings.Reader ────────────────────────────────────────────────────
	// The most common in-memory reader. Implements io.Reader, io.ReaderAt,
	// io.Seeker, and io.WriterTo.
	r := strings.NewReader("Hello, Go io!")
	buf := make([]byte, 5)

	fmt.Printf("strings.Reader (%d bytes total):\n", r.Len())
	for {
		n, err := r.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("read error:", err)
			break
		}
		fmt.Printf("  read %d bytes: %q\n", n, buf[:n])
	}

	// ── 2. bytes.Reader ──────────────────────────────────────────────────────
	// Identical to strings.Reader but works over []byte.
	data := []byte{0x47, 0x6F, 0x21} // "Go!"
	br := bytes.NewReader(data)
	all, _ := io.ReadAll(br) // io.ReadAll reads until EOF, returns []byte
	fmt.Printf("\nbytes.Reader → io.ReadAll: %s\n", all)

	// ── 3. io.LimitReader ────────────────────────────────────────────────────
	// Wraps any Reader and stops after N bytes.
	// Useful for protecting against oversized uploads.
	src := strings.NewReader("this string is longer than 10 bytes")
	limited := io.LimitReader(src, 10)
	snippet, _ := io.ReadAll(limited)
	fmt.Printf("\nio.LimitReader (10 bytes): %q\n", snippet)

	// ── 4. io.MultiReader ────────────────────────────────────────────────────
	// Chains multiple readers; reads them in sequence as if they were one.
	// Common for prepending headers to a body stream.
	header := strings.NewReader("[HEADER] ")
	body := strings.NewReader("body content")
	multi := io.MultiReader(header, body)
	combined, _ := io.ReadAll(multi)
	fmt.Printf("\nio.MultiReader: %q\n", combined)

	// ── 5. io.SectionReader ──────────────────────────────────────────────────
	// Reads a slice [off, off+n) from any io.ReaderAt.
	// Common for parsing binary formats (ZIP files, ELF headers).
	full := strings.NewReader("AAABBBCCCDDDEEE")
	section := io.NewSectionReader(full, 3, 6) // bytes 3–9 → "BBBCCC"
	sliced, _ := io.ReadAll(section)
	fmt.Printf("\nio.SectionReader [3:9]: %q\n", sliced)

	// ── Key takeaway ─────────────────────────────────────────────────────────
	fmt.Println("\n💡 io.Reader is the universal input abstraction.")
	fmt.Println("   Anything that can produce bytes implements it:")
	fmt.Println("   files, network connections, HTTP bodies, gzip streams…")
}
