package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/tour-of-go/k8s-event-sink/internal/core"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func event(id, ns, pod, reason, severity string) core.Event {
	return core.Event{
		ID: id, Namespace: ns, Pod: pod, Reason: reason,
		Message: "test", Type: "Warning", Severity: severity, Count: 1,
		FirstSeen: time.Now(), LastSeen: time.Now(),
	}
}

func TestSaveAndQuery(t *testing.T) {
	s := newStore(t)
	e := event("evt-1", "default", "pod-1", "OOMKilled", "critical")
	if err := s.Save(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	results, err := s.Query(context.Background(), core.QueryFilter{Namespace: "default"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "evt-1" {
		t.Errorf("unexpected ID: %s", results[0].ID)
	}
}

func TestQuery_SeverityFilter(t *testing.T) {
	s := newStore(t)
	s.Save(context.Background(), event("e1", "default", "pod-1", "OOMKilled", "critical"))
	s.Save(context.Background(), event("e2", "default", "pod-2", "Unhealthy", "warning"))

	results, err := s.Query(context.Background(), core.QueryFilter{Severity: "critical"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ID != "e1" {
		t.Errorf("expected 1 critical event, got %d", len(results))
	}
}

func TestQuery_TimeRange(t *testing.T) {
	s := newStore(t)
	past := time.Now().Add(-2 * time.Hour)
	old := core.Event{
		ID: "old", Namespace: "default", Pod: "pod-1", Reason: "OOMKilled",
		Message: "old", Type: "Warning", Severity: "critical", Count: 1,
		FirstSeen: past, LastSeen: past,
	}
	s.Save(context.Background(), old)
	s.Save(context.Background(), event("new", "default", "pod-2", "OOMKilled", "critical"))

	results, err := s.Query(context.Background(), core.QueryFilter{
		Since: time.Now().Add(-1 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ID != "new" {
		t.Errorf("expected 1 recent event, got %d", len(results))
	}
}
