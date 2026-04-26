package core

import (
	"context"
	"time"
)

// Event is the canonical domain model for a Kubernetes event.
type Event struct {
	ID        string
	Namespace string
	Pod       string // InvolvedObject.Name
	Reason    string
	Message   string
	Type      string // "Normal" or "Warning"
	Severity  string // "critical", "warning", "ignore"
	Count     int    // deduplicated occurrence count
	FirstSeen time.Time
	LastSeen  time.Time
}

// QueryFilter constrains event queries.
type QueryFilter struct {
	Namespace string
	Pod       string
	Reason    string
	Severity  string
	Since     time.Time
	Until     time.Time
	Limit     int
}

// StoragePort persists events for long-term structured queries.
type StoragePort interface {
	Save(ctx context.Context, event Event) error
	Query(ctx context.Context, f QueryFilter) ([]Event, error)
}

// SearchPort indexes events for full-text search.
type SearchPort interface {
	Index(event Event) error
	Search(query string) ([]Event, error)
}

// AlerterPort sends notifications for critical/warning events.
type AlerterPort interface {
	Notify(ctx context.Context, event Event) error
}
