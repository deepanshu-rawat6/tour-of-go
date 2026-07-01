// Package fmtprintln is a deep-dive into fmt.Println and the fmt package.
//
// The folder is named "fmt.Println()" (with parens) which is unusual but
// valid in a filesystem. The package name cannot contain dots or parens,
// so we use "fmtprintln".
//
// This file is intentionally a standalone learning reference.
// Read it directly — it cannot be imported (parens in path are invalid import chars).
// Run: go run go/go-internals/fmt.Println\(\)/fmt_println.go  ← won't work as-is
// Instead, copy the Run() function into a main package to execute it.
//
//go:build ignore

package fmtprintln

import (
	"errors"
	"fmt"
	"os"
)

// ─────────────────────────────────────────────────────────────────────────────
// TYPES USED IN EXAMPLES
// ─────────────────────────────────────────────────────────────────────────────

// Temperature satisfies the fmt.Stringer interface by implementing String().
// When fmt.Println (or any fmt function) encounters a value that satisfies
// fmt.Stringer, it calls .String() automatically instead of the default
// struct representation.
type Temperature struct {
	Celsius float64
}

// String satisfies fmt.Stringer.
// fmt.Stringer interface: type Stringer interface { String() string }
func (t Temperature) String() string {
	return fmt.Sprintf("%.1f°C (%.1f°F)", t.Celsius, t.Celsius*9/5+32)
}

// Point satisfies fmt.GoStringer.
// fmt.GoStringer interface: type GoStringer interface { GoString() string }
// GoString is called when the %#v verb is used — it should return a
// Go syntax representation of the value (like what go/printer would emit).
type Point struct {
	X, Y int
}

// GoString satisfies fmt.GoStringer.
func (p Point) GoString() string {
	return fmt.Sprintf("Point{X: %d, Y: %d}", p.X, p.Y)
}

// AppError is a custom error type that wraps an underlying cause.
type AppError struct {
	Code    int
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("AppError %d: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("AppError %d: %s", e.Code, e.Message)
}

// Unwrap allows errors.Is / errors.As to traverse the chain.
func (e *AppError) Unwrap() error { return e.Cause }

// ─────────────────────────────────────────────────────────────────────────────
// RUN — entry point
// ─────────────────────────────────────────────────────────────────────────────

// Run executes all fmt.Println deep-dive examples.
func Run() {
	fmt.Println("=== fmt.Println deep-dive ===")
	fmt.Println()

	signatureDemo()
	fmt.Println()

	nestedPrintlnDemo()
	fmt.Println()

	printVariantsDemo()
	fmt.Println()

	sprintVariantsDemo()
	fmt.Println()

	stderrAndErrorfDemo()
	fmt.Println()

	stringerDemo()
	fmt.Println()

	goStringerDemo()
	fmt.Println()

	verbsDemo()
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// EXAMPLES
// ─────────────────────────────────────────────────────────────────────────────

// signatureDemo explains the signature: func Println(a ...any) (n int, err error).
//
// fmt.Println:
//   - accepts any number of arguments of any type (...any)
//   - separates arguments with a single space
//   - appends a trailing newline
//   - returns (n int, err error)
//     where n = number of bytes written, err = write error (almost always nil
//     when writing to stdout, but could be non-nil for custom writers)
func signatureDemo() {
	fmt.Println("--- Signature: func Println(a ...any) (n int, err error) ---")

	// The return values are almost always ignored in practice.
	n, err := fmt.Println("hello, world")
	fmt.Printf("  bytes written: %d, error: %v\n", n, err) // 13 (12 chars + newline), <nil>

	// Multiple args: space-separated automatically
	n, err = fmt.Println("one", "two", "three")
	fmt.Printf("  bytes written: %d, error: %v\n", n, err) // 14 (13 chars + newline)
}

// nestedPrintlnDemo shows the famous "fmt.Println(fmt.Println(...))" puzzle.
//
// The inner fmt.Println("hello") runs first:
//   1. It prints "hello\n" to stdout.
//   2. It returns (6, nil) — 6 bytes ("hello\n").
//
// The outer fmt.Println then receives (6, nil) as its arguments:
//   3. It prints "6 <nil>\n" to stdout (space-separated, plus newline).
//
// Total output:
//   hello
//   6 <nil>
func nestedPrintlnDemo() {
	fmt.Println("--- Nested fmt.Println(fmt.Println(\"hello\")) ---")
	fmt.Println("Output:")
	fmt.Println(fmt.Println("hello")) // prints: hello\n6 <nil>\n
}

// printVariantsDemo shows the differences between Print, Println, Printf.
//
//   Print:   no separator between args unless neither is a string; no newline
//   Println: space-separated args; always adds newline
//   Printf:  format string with verbs; no automatic newline
func printVariantsDemo() {
	fmt.Println("--- Print vs Println vs Printf ---")

	// Print: no newline, no space between adjacent string args
	fmt.Print("a", "b", "c")          // abc
	fmt.Print(1, 2, 3)                 // 1 2 3 (spaces added when neither is a string)
	fmt.Println()                      // just newline to flush the line

	// Println: always adds spaces and newline
	fmt.Println("a", "b", "c")        // a b c\n
	fmt.Println(1, 2, 3)              // 1 2 3\n

	// Printf: uses format verbs, NO automatic newline
	fmt.Printf("name=%s age=%d\n", "Alice", 30) // name=Alice age=30
	fmt.Printf("pi=%.4f\n", 3.14159265)          // pi=3.1416
}

// sprintVariantsDemo shows Sprint, Sprintf, Sprintln.
//
// The "S" variants build a string in memory without writing to any writer.
// Use them when you need to construct a string for logging, returning, etc.
func sprintVariantsDemo() {
	fmt.Println("--- Sprint, Sprintf, Sprintln (return string) ---")

	// Sprint: same as Print but returns the string
	s1 := fmt.Sprint("hello", "world")    // "helloworld" — no space between strings
	s2 := fmt.Sprint(1, 2, 3)             // "1 2 3" — spaces between non-strings
	fmt.Printf("  Sprint:    %q\n", s1)
	fmt.Printf("  Sprint:    %q\n", s2)

	// Sprintf: format string → returns string
	msg := fmt.Sprintf("user %s has %d points", "Bob", 42)
	fmt.Printf("  Sprintf:   %q\n", msg)

	// Sprintln: same as Println but returns the string (includes trailing \n)
	line := fmt.Sprintln("one", "two", "three")
	fmt.Printf("  Sprintln:  %q\n", line) // "one two three\n"

	// Common pattern: build a log message then pass to logger
	logMsg := fmt.Sprintf("[ERROR] failed to connect to %s:%d", "localhost", 5432)
	_ = logMsg // would pass to logger
	fmt.Printf("  log msg:   %q\n", logMsg)
}

// stderrAndErrorfDemo shows fmt.Fprintf to stderr and fmt.Errorf with %w.
func stderrAndErrorfDemo() {
	fmt.Println("--- fmt.Fprintf to stderr + fmt.Errorf with %%w ---")

	// fmt.Fprintf writes to any io.Writer — here we use os.Stderr.
	// This is the standard way to write error messages without using a logger.
	fmt.Fprintf(os.Stderr, "  [stderr] this goes to stderr, not stdout\n")

	// fmt.Errorf creates a new error with a formatted message.
	// The %w verb *wraps* the error — errors.Is and errors.As can unwrap it.
	baseErr := errors.New("connection refused")
	wrapped := fmt.Errorf("dial postgres: %w", baseErr)
	fmt.Printf("  wrapped error:  %v\n", wrapped)
	fmt.Printf("  errors.Is match: %v\n", errors.Is(wrapped, baseErr)) // true

	// Wrapping vs not wrapping:
	//   %w  — wraps:  errors.Is/As can traverse the chain
	//   %v  — formats: error message is embedded but NOT unwrappable
	notWrapped := fmt.Errorf("dial postgres: %v", baseErr) // %v instead of %w
	fmt.Printf("  %%v errors.Is:  %v\n", errors.Is(notWrapped, baseErr)) // false!

	// Custom error with wrapping
	appErr := &AppError{
		Code:    503,
		Message: "database unavailable",
		Cause:   baseErr,
	}
	fmt.Printf("  AppError:       %v\n", appErr)
	fmt.Printf("  Unwrap works:   %v\n", errors.Is(appErr, baseErr)) // true via Unwrap()
}

// stringerDemo shows fmt.Stringer — the most important fmt interface.
//
// When you implement String() string on a type, fmt functions will call it
// automatically for any format verb that triggers the default representation
// (%v, %s, fmt.Println, etc.).
func stringerDemo() {
	fmt.Println("--- fmt.Stringer interface ---")

	t := Temperature{Celsius: 100.0}

	// All of these call t.String() automatically:
	fmt.Println(t)                         // 100.0°C (212.0°F)
	fmt.Printf("%v\n", t)                  // 100.0°C (212.0°F)  — %v calls String()
	fmt.Printf("%s\n", t)                  // 100.0°C (212.0°F)  — %s calls String()
	fmt.Printf("temp: %v\n", t)            // temp: 100.0°C (212.0°F)

	// %T shows the type name, not the string representation:
	fmt.Printf("%T\n", t)                  // fmtprintln.Temperature

	// Without Stringer, fmt.Println would print: {100}
	// With Stringer, it prints the human-readable form.
}

// goStringerDemo shows fmt.GoStringer — the %#v (debug) verb.
//
// GoString() should return a Go-syntax representation: how you'd write
// the value in source code. Used for debugging and logging.
func goStringerDemo() {
	fmt.Printf("--- fmt.GoStringer interface (%%#v verb) ---\n")

	p := Point{X: 3, Y: 7}

	fmt.Printf("%%v  (default):   %v\n", p)   // {3 7}
	fmt.Printf("%%+v (with names): %+v\n", p) // {X:3 Y:7}
	fmt.Printf("%%#v (GoString):  %#v\n", p)  // Point{X: 3, Y: 7}  ← calls GoString()

	// Without GoStringer, %#v would print: fmtprintln.Point{X:3, Y:7}
	// With GoStringer, we control the output.
}

// verbsDemo shows the most important format verbs.
func verbsDemo() {
	fmt.Println("--- Key format verbs ---")

	n := 255
	f := 3.14159
	s := "hello"
	b := true

	// Integer verbs
	fmt.Printf("  %%d  decimal:    %d\n", n)   // 255
	fmt.Printf("  %%b  binary:     %b\n", n)   // 11111111
	fmt.Printf("  %%o  octal:      %o\n", n)   // 377
	fmt.Printf("  %%x  hex (lower): %x\n", n)  // ff
	fmt.Printf("  %%X  hex (upper): %X\n", n)  // FF
	fmt.Printf("  %%08b padded:    %08b\n", n) // 11111111 (already 8 bits)

	// Float verbs
	fmt.Printf("  %%f  fixed:      %f\n", f)   // 3.141590
	fmt.Printf("  %%.2f 2 decimal: %.2f\n", f) // 3.14
	fmt.Printf("  %%e  scientific: %e\n", f)   // 3.141590e+00
	fmt.Printf("  %%g  shortest:   %g\n", f)   // 3.14159

	// String / bytes verbs
	fmt.Printf("  %%s  string:     %s\n", s)  // hello
	fmt.Printf("  %%q  quoted:     %q\n", s)  // "hello"
	fmt.Printf("  %%x  hex bytes:  %x\n", s)  // 68656c6c6f

	// General verbs
	fmt.Printf("  %%v  default:    %v\n", b)           // true
	fmt.Printf("  %%T  type name:  %T\n", f)           // float64
	fmt.Printf("  %%p  pointer:    %p\n", &n)          // 0x... (address)
}
