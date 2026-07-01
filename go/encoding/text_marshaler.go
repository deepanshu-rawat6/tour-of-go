package encoding

import (
	"encoding"
	"encoding/json"
	"fmt"
	"strings"
)

// ── encoding.TextMarshaler / TextUnmarshaler ──────────────────────────────────
//
// These interfaces are used by:
//   - encoding/json (as a fallback when MarshalJSON is not defined)
//   - encoding/xml
//   - encoding/csv
//   - flag package
//   - database/sql (via sql.Scanner / driver.Valuer, but TextMarshaler is similar)
//
// Signature:
//   MarshalText() (text []byte, err error)
//   UnmarshalText(text []byte) error

// Color is a type-safe enum that uses TextMarshaler to produce human-readable
// values across all text-based encodings (JSON, XML, YAML, etc.) — without
// implementing separate MarshalJSON / MarshalXML methods for each.
type Color int

const (
	Red Color = iota
	Green
	Blue
)

var colorNames = map[Color]string{Red: "red", Green: "green", Blue: "blue"}
var colorValues = map[string]Color{"red": Red, "green": Green, "blue": Blue}

// MarshalText implements encoding.TextMarshaler.
func (c Color) MarshalText() ([]byte, error) {
	name, ok := colorNames[c]
	if !ok {
		return nil, fmt.Errorf("unknown Color %d", int(c))
	}
	return []byte(name), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (c *Color) UnmarshalText(text []byte) error {
	v, ok := colorValues[strings.ToLower(string(text))]
	if !ok {
		return fmt.Errorf("unknown Color %q", text)
	}
	*c = v
	return nil
}

// Verify at compile time that Color satisfies both interfaces.
var _ encoding.TextMarshaler = Color(0)
var _ encoding.TextUnmarshaler = (*Color)(nil)

// Palette groups colors; since Color implements TextMarshaler, json.Marshal
// will call MarshalText automatically (JSON wraps the result in a string).
type Palette struct {
	Name      string  `json:"name"`
	Primary   Color   `json:"primary"`
	Secondary Color   `json:"secondary"`
	All       []Color `json:"all"`
}

// ── NetworkAddr — a common real-world use case ───────────────────────────────
// Custom network address that marshals as "host:port" string.

type NetworkAddr struct {
	Host string
	Port int
}

func (a NetworkAddr) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%s:%d", a.Host, a.Port)), nil
}

func (a *NetworkAddr) UnmarshalText(text []byte) error {
	parts := strings.SplitN(string(text), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid address %q, want host:port", text)
	}
	a.Host = parts[0]
	_, err := fmt.Sscanf(parts[1], "%d", &a.Port)
	return err
}

var _ encoding.TextMarshaler = NetworkAddr{}
var _ encoding.TextUnmarshaler = (*NetworkAddr)(nil)

func textMarshalerExample() {
	fmt.Println("--- encoding.TextMarshaler / TextUnmarshaler ---")

	// ── 1. Color enum round-trip via JSON ────────────────────────────────────
	p := Palette{
		Name:      "vivid",
		Primary:   Red,
		Secondary: Blue,
		All:       []Color{Red, Green, Blue},
	}
	b, _ := json.MarshalIndent(p, "", "  ")
	fmt.Printf("Palette JSON:\n%s\n", b)
	// "primary": "red"  ← MarshalText called automatically by json package

	var p2 Palette
	_ = json.Unmarshal(b, &p2)
	fmt.Printf("Decoded: name=%q primary=%v secondary=%v all=%v\n\n",
		p2.Name, colorNames[p2.Primary], colorNames[p2.Secondary], p2.All)

	// ── 2. NetworkAddr as a JSON map key ─────────────────────────────────────
	// json.Marshal can only use string-keyed maps. If the key type implements
	// TextMarshaler, the key is encoded as a string automatically.
	routes := map[NetworkAddr]string{
		{Host: "10.0.0.1", Port: 8080}: "web",
		{Host: "10.0.0.2", Port: 9090}: "metrics",
	}
	rb, _ := json.MarshalIndent(routes, "", "  ")
	fmt.Printf("NetworkAddr as map key:\n%s\n", rb)

	// ── 3. Manual MarshalText / UnmarshalText ────────────────────────────────
	addr := NetworkAddr{Host: "localhost", Port: 3306}
	text, _ := addr.MarshalText()
	fmt.Printf("\nMarshalText: %q\n", text)

	var addr2 NetworkAddr
	_ = addr2.UnmarshalText(text)
	fmt.Printf("UnmarshalText: %+v\n", addr2)

	fmt.Println("\n💡 Prefer TextMarshaler over MarshalJSON when:")
	fmt.Println("   • You want the same representation across JSON, XML, CSV, flags.")
	fmt.Println("   • Your type needs to be used as a JSON map key.")
	fmt.Println("   • The text representation is the natural human-readable form.")
}
