package encoding

import (
	"encoding/json"
	"fmt"
)

// User demonstrates struct tags and field visibility rules.
// Only exported fields are included by default.
type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"` // omitted when empty string
	Password  string `json:"-"`               // always excluded
	CreatedAt string `json:"created_at"`
}

// jsonBasicExample covers the fundamentals of encoding/json:
//   - Marshal (struct → JSON bytes)
//   - Unmarshal (JSON bytes → struct)
//   - Struct tags: json:"name", omitempty, "-"
//   - map and slice encoding
//   - json.RawMessage for deferred parsing
func jsonBasicExample() {
	fmt.Println("--- JSON basics ---")

	// ── 1. Marshal ───────────────────────────────────────────────────────────
	u := User{
		ID:       42,
		Name:     "Alice",
		Email:    "",        // omitted via omitempty
		Password: "secret", // excluded via "-"
	}
	b, err := json.Marshal(u)
	if err != nil {
		fmt.Println("marshal error:", err)
		return
	}
	fmt.Printf("Marshal:   %s\n", b)
	// → {"id":42,"name":"Alice","created_at":""}
	// Notice: email is gone (omitempty), password is gone ("-")

	// ── 2. MarshalIndent for human-readable output ────────────────────────────
	pretty, _ := json.MarshalIndent(u, "", "  ")
	fmt.Printf("\nMarshalIndent:\n%s\n", pretty)

	// ── 3. Unmarshal ─────────────────────────────────────────────────────────
	raw := `{"id":7,"name":"Bob","email":"bob@example.com","password":"ignore-me","created_at":"2024-01-15"}`
	var decoded User
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		fmt.Println("unmarshal error:", err)
		return
	}
	fmt.Printf("\nUnmarshal: id=%d name=%q email=%q password=%q\n",
		decoded.ID, decoded.Name, decoded.Email, decoded.Password)
	// Password stays empty even though it was in the JSON — the "-" tag blocks it.

	// ── 4. Unknown fields are silently ignored by default ─────────────────────
	withExtra := `{"id":1,"name":"Carol","unknown_field":"ignored"}`
	var u2 User
	_ = json.Unmarshal([]byte(withExtra), &u2)
	fmt.Printf("\nUnknown field silently ignored: %+v\n", u2)

	// ── 5. map[string]any for dynamic/unknown JSON ────────────────────────────
	dynamic := `{"service":"payments","version":3,"enabled":true}`
	var m map[string]any
	_ = json.Unmarshal([]byte(dynamic), &m)
	fmt.Printf("\nmap[string]any: %v\n", m)
	// Numbers unmarshal as float64 by default in map[string]any!
	fmt.Printf("  'version' type: %T, value: %v\n", m["version"], m["version"])

	// ── 6. json.Number to avoid float64 precision loss ───────────────────────
	dec := json.NewDecoder(toReader(`{"id":9999999999999999}`))
	dec.UseNumber() // keeps numbers as json.Number (string) instead of float64
	var precise map[string]json.Number
	_ = dec.Decode(&precise)
	fmt.Printf("\njson.Number:  %s (no float64 precision loss)\n", precise["id"])

	// ── 7. json.RawMessage — defer parsing part of a payload ─────────────────
	// Common for polymorphic events where "payload" depends on "type".
	type Event struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"` // kept as raw bytes
	}
	eventJSON := `{"type":"transfer","payload":{"from":"alice","to":"bob","amount":100}}`
	var ev Event
	_ = json.Unmarshal([]byte(eventJSON), &ev)
	fmt.Printf("\njson.RawMessage: type=%q payload=%s\n", ev.Type, ev.Payload)

	fmt.Println("\n💡 Key rules:")
	fmt.Println("   • Only exported fields are marshaled.")
	fmt.Println("   • json:\"-\" completely hides a field in both directions.")
	fmt.Println("   • omitempty skips zero values (0, \"\", false, nil, empty slice).")
	fmt.Println("   • Numbers in map[string]any come back as float64 — use json.Number or a typed struct.")
}
