package encoding

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ── Custom time format ────────────────────────────────────────────────────────

// Date wraps time.Time to marshal/unmarshal as "YYYY-MM-DD" instead of RFC3339.
type Date struct {
	time.Time
}

const dateLayout = "2006-01-02"

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Time.Format(dateLayout))
}

func (d *Date) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return fmt.Errorf("Date.UnmarshalJSON: %w", err)
	}
	d.Time = t
	return nil
}

// ── Enum as string ────────────────────────────────────────────────────────────

// Status is an integer enum that marshals to/from human-readable strings.
type Status int

const (
	StatusPending Status = iota
	StatusActive
	StatusClosed
)

var statusNames = map[Status]string{
	StatusPending: "pending",
	StatusActive:  "active",
	StatusClosed:  "closed",
}

var statusValues = map[string]Status{
	"pending": StatusPending,
	"active":  StatusActive,
	"closed":  StatusClosed,
}

func (s Status) MarshalJSON() ([]byte, error) {
	name, ok := statusNames[s]
	if !ok {
		return nil, fmt.Errorf("unknown Status %d", s)
	}
	return json.Marshal(name)
}

func (s *Status) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	v, ok := statusValues[strings.ToLower(name)]
	if !ok {
		return fmt.Errorf("unknown Status name %q", name)
	}
	*s = v
	return nil
}

// ── Sensitive field — always redact on marshal ────────────────────────────────

// APIKey stores a secret but never exposes it via JSON.
type APIKey struct {
	value string
}

func NewAPIKey(v string) APIKey { return APIKey{value: v} }

func (k APIKey) MarshalJSON() ([]byte, error) {
	return json.Marshal("[REDACTED]")
}

func (k *APIKey) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &k.value)
}

// ── Polymorphic / discriminated union ────────────────────────────────────────

// Notification dispatches to different sub-types based on "kind".
type Notification struct {
	Kind    string `json:"kind"`
	Payload any    `json:"-"` // decoded separately
}

type EmailNotif struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
}
type PushNotif struct {
	DeviceID string `json:"device_id"`
	Title    string `json:"title"`
}

func (n *Notification) UnmarshalJSON(data []byte) error {
	// First pass: decode only the discriminator field.
	var raw struct {
		Kind    string          `json:"kind"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	n.Kind = raw.Kind

	// Second pass: decode the payload into the correct concrete type.
	switch raw.Kind {
	case "email":
		var e EmailNotif
		if err := json.Unmarshal(raw.Payload, &e); err != nil {
			return err
		}
		n.Payload = e
	case "push":
		var p PushNotif
		if err := json.Unmarshal(raw.Payload, &p); err != nil {
			return err
		}
		n.Payload = p
	default:
		return fmt.Errorf("unknown notification kind %q", raw.Kind)
	}
	return nil
}

// ── Putting it all together ───────────────────────────────────────────────────

type Contract struct {
	ID        string  `json:"id"`
	Status    Status  `json:"status"`
	StartDate Date    `json:"start_date"`
	SecretKey APIKey  `json:"secret_key"`
}

func jsonCustomMarshalingExample() {
	fmt.Println("--- custom MarshalJSON / UnmarshalJSON ---")

	// ── 1. Custom Date type ───────────────────────────────────────────────────
	c := Contract{
		ID:        "CTR-001",
		Status:    StatusActive,
		StartDate: Date{time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)},
		SecretKey: NewAPIKey("sk_live_abc123"),
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	fmt.Printf("Marshal:\n%s\n", b)
	// StartDate → "2024-03-15", Status → "active", SecretKey → "[REDACTED]"

	raw := `{"id":"CTR-002","status":"closed","start_date":"2023-11-01","secret_key":"sk_live_xyz"}`
	var c2 Contract
	_ = json.Unmarshal([]byte(raw), &c2)
	fmt.Printf("\nUnmarshal: id=%s status=%v date=%s key=%q\n",
		c2.ID, c2.Status, c2.StartDate.Format(dateLayout), c2.SecretKey.value)

	// ── 2. Polymorphic notification ───────────────────────────────────────────
	emailJSON := `{"kind":"email","payload":{"to":"alice@x.com","subject":"Hello"}}`
	var notif Notification
	_ = json.Unmarshal([]byte(emailJSON), &notif)
	fmt.Printf("\nPolymorphic: kind=%q payload=%+v\n", notif.Kind, notif.Payload)

	fmt.Println("\n💡 Implement MarshalJSON / UnmarshalJSON on the VALUE type (not pointer)")
	fmt.Println("   for marshal, and on the POINTER type for unmarshal.")
	fmt.Println("   Rule: if the method modifies the receiver, it must be a pointer.")
}
