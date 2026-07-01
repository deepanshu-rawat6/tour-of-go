package encoding

// streaming.go — json.Encoder and json.Decoder for large/streaming payloads
//
// KEY INSIGHT:
//   json.Marshal / json.Unmarshal  → buffers the ENTIRE value in memory as []byte
//   json.NewEncoder / NewDecoder   → writes/reads incrementally to any io.Writer / io.Reader
//
// Rule of thumb:
//   • Small, known-size payloads in memory → Marshal / Unmarshal
//   • HTTP response body, file, network stream, huge arrays → Encoder / Decoder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ── toReader is a package-level helper used across this package's examples.
// It converts a plain string into an *strings.Reader so we can pass it to
// any function expecting an io.Reader without creating a temporary variable.
func toReader(s string) *strings.Reader {
	return strings.NewReader(s)
}

// ── Domain types ─────────────────────────────────────────────────────────────

// LogEntry represents one entry in a stream of structured logs.
type LogEntry struct {
	Level   string `json:"level"`
	Message string `json:"msg"`
	Service string `json:"service"`
}

// Product is used for the streaming-array decode example.
type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// jsonStreamingExample demonstrates the four key streaming APIs:
//
//  1. json.NewEncoder  — write JSON to any io.Writer (e.g. HTTP body, file)
//  2. json.NewDecoder  — read JSON from any io.Reader
//  3. SetIndent / DisallowUnknownFields options
//  4. Token-by-token parsing with dec.Token() for huge JSON arrays
func jsonStreamingExample() {
	fmt.Println("--- JSON streaming (Encoder / Decoder) ---")

	encodeToBuffer()
	fmt.Println()

	decodeFromStream()
	fmt.Println()

	strictDecoding()
	fmt.Println()

	tokenByTokenParsing()
	fmt.Println()

	fmt.Println("💡 Encoder vs Marshal — when to choose:")
	fmt.Println("   Use json.NewEncoder when:")
	fmt.Println("   • Writing directly to http.ResponseWriter, os.File, net.Conn")
	fmt.Println("   • You want to avoid allocating a large []byte buffer for the whole body")
	fmt.Println("   • You need to write a stream of newline-delimited JSON (NDJSON) objects")
	fmt.Println()
	fmt.Println("   Use json.NewDecoder when:")
	fmt.Println("   • Reading from resp.Body, os.File, or any io.Reader")
	fmt.Println("   • Parsing a large JSON array element-by-element (constant memory)")
	fmt.Println("   • You need dec.Token() to walk an array of unknown size")
	fmt.Println()
	fmt.Println("   Use json.Marshal / Unmarshal when:")
	fmt.Println("   • You already have the full payload as []byte or string in memory")
	fmt.Println("   • The payload is small and you want the simplest API")
}

// ── 1. Encoding to a buffer (simulates writing to http.ResponseWriter) ────────

func encodeToBuffer() {
	fmt.Println("  [1] json.NewEncoder — streaming encode to an io.Writer")

	// bytes.Buffer implements io.Writer.  In production this would be
	// http.ResponseWriter or an os.File — no difference to the encoder.
	var buf bytes.Buffer

	enc := json.NewEncoder(&buf)

	// SetIndent makes the output human-readable.
	// First arg = prefix per line (useful for nested structures), second = indent unit.
	enc.SetIndent("", "  ")

	// Encode three log entries one by one — each Encode call writes one JSON
	// object followed by a newline.  The underlying io.Writer is written to
	// incrementally; the full output is never held in a single []byte.
	entries := []LogEntry{
		{Level: "info", Message: "server started", Service: "api"},
		{Level: "warn", Message: "high latency", Service: "db"},
		{Level: "error", Message: "connection refused", Service: "cache"},
	}

	for _, e := range entries {
		// enc.Encode(v) is equivalent to:
		//   b, _ := json.Marshal(v)
		//   w.Write(append(b, '\n'))
		// but without the intermediate []byte allocation for the whole stream.
		if err := enc.Encode(e); err != nil {
			fmt.Println("encode error:", err)
			return
		}
	}

	fmt.Printf("  Encoded %d NDJSON objects (%d bytes):\n", len(entries), buf.Len())
	// Print first 3 lines to keep output short
	lines := strings.SplitN(buf.String(), "\n", 4)
	for _, l := range lines[:3] {
		fmt.Printf("    %s\n", l)
	}
}

// ── 2. Decoding from a stream ─────────────────────────────────────────────────

func decodeFromStream() {
	fmt.Println("  [2] json.NewDecoder — streaming decode from an io.Reader")

	// A newline-delimited JSON stream (NDJSON) — common in log pipelines,
	// Kubernetes API watches, Docker events, etc.
	ndjson := `{"id":1,"name":"Widget A","price":9.99}
{"id":2,"name":"Widget B","price":19.99}
{"id":3,"name":"Widget C","price":4.50}`

	// strings.NewReader gives us an io.Reader from a string.
	// In production: dec := json.NewDecoder(resp.Body)
	dec := json.NewDecoder(strings.NewReader(ndjson))

	var total float64
	var count int

	// dec.More() returns true while there is another value in the current
	// array/stream — it peeks ahead without consuming input.
	for dec.More() {
		var p Product
		// Decode consumes exactly one JSON value from the stream.
		// Subsequent calls pick up from where the previous one left off.
		if err := dec.Decode(&p); err != nil {
			fmt.Println("decode error:", err)
			return
		}
		total += p.Price
		count++
		fmt.Printf("    decoded: id=%d name=%-10q price=%.2f\n", p.ID, p.Name, p.Price)
	}
	fmt.Printf("  Total: %.2f across %d products\n", total, count)
}

// ── 3. Strict decoding — reject unknown fields ────────────────────────────────

func strictDecoding() {
	fmt.Println("  [3] dec.DisallowUnknownFields() — strict schema validation")

	// By default the decoder silently ignores JSON keys with no matching struct
	// field.  DisallowUnknownFields() turns that into an error, which is useful
	// in APIs that want to catch client typos like "prodcut_id" instead of "product_id".
	dec := json.NewDecoder(toReader(`{"id":10,"name":"Gadget","price":99.0,"colour":"red"}`))
	dec.DisallowUnknownFields() // "colour" has no struct field → error

	var p Product
	if err := dec.Decode(&p); err != nil {
		fmt.Printf("  strict decode error (expected): %v\n", err)
	}

	// Without DisallowUnknownFields — same input, silently succeeds:
	dec2 := json.NewDecoder(toReader(`{"id":10,"name":"Gadget","price":99.0,"colour":"red"}`))
	var p2 Product
	_ = dec2.Decode(&p2)
	fmt.Printf("  lenient decode:  id=%d name=%q (colour silently ignored)\n", p2.ID, p2.Name)
}

// ── 4. Token-by-token parsing ─────────────────────────────────────────────────
//
// dec.Token() reads one JSON token at a time:
//   json.Delim   → '[', ']', '{', '}'
//   string       → object key or string value
//   float64      → number
//   bool         → boolean
//   nil          → null
//
// This is the only way to process arrays so large they would OOM if decoded
// into a []T slice all at once.  Memory usage is O(1) per element, not O(n).

func tokenByTokenParsing() {
	fmt.Println("  [4] dec.Token() — walk a huge JSON array without loading it all into memory")

	// Simulate a JSON file with a top-level array of products.
	// In practice this could be gigabytes returned by an HTTP endpoint.
	bigArray := `[
  {"id":100,"name":"Alpha","price":1.0},
  {"id":101,"name":"Beta","price":2.0},
  {"id":102,"name":"Gamma","price":3.0}
]`

	dec := json.NewDecoder(strings.NewReader(bigArray))

	// Step 1: consume the opening '[' token.
	// We MUST do this before calling Decode in the loop, otherwise Decode
	// would try to decode the whole array into a single value.
	openBracket, err := dec.Token()
	if err != nil {
		fmt.Println("  token error:", err)
		return
	}
	fmt.Printf("  Opening token: %v (type %T)\n", openBracket, openBracket)
	// openBracket is json.Delim('[')

	// Step 2: loop while there are more elements in the current array.
	// dec.More() peeks at the next token: returns false when it sees ']'.
	var products []Product
	for dec.More() {
		var p Product
		// Decode reads exactly one element from the array position.
		// Memory: only one Product at a time is held during decoding,
		// even if the array has millions of entries.
		if err := dec.Decode(&p); err != nil {
			fmt.Println("  decode error:", err)
			return
		}
		products = append(products, p)
	}

	// Step 3: consume the closing ']' token.
	closeBracket, err := dec.Token()
	if err != nil {
		fmt.Println("  token error:", err)
		return
	}
	fmt.Printf("  Closing token: %v (type %T)\n", closeBracket, closeBracket)

	fmt.Printf("  Parsed %d products token-by-token:\n", len(products))
	for _, p := range products {
		fmt.Printf("    id=%d name=%-6q price=%.1f\n", p.ID, p.Name, p.Price)
	}

	// Bonus: raw token walk (no struct — useful for inspecting unknown schemas)
	rawJSON := `{"event":"click","count":42,"tags":["web","mobile"]}`
	fmt.Println("\n  Raw token walk of arbitrary JSON:")
	walkTokens(strings.NewReader(rawJSON), "  ")
}

// walkTokens prints every token from an io.Reader — demonstrates the full
// token stream for educational purposes.
func walkTokens(r io.Reader, indent string) {
	dec := json.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("%serror: %v\n", indent, err)
			break
		}
		// Each token carries its Go type.
		fmt.Printf("%s  token: %-10v  type: %T\n", indent, tok, tok)
	}
}
