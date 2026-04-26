package bleveadapter

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/tour-of-go/k8s-event-sink/internal/core"
)

func TestIndexAndSearch(t *testing.T) {
	idx, err := New(filepath.Join(t.TempDir(), "test.bleve"))
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()

	events := []core.Event{
		{ID: "e1", Namespace: "default", Pod: "api-pod", Reason: "OOMKilled",
			Message: "container killed due to memory limit", Severity: "critical",
			Count: 1, LastSeen: time.Now()},
		{ID: "e2", Namespace: "default", Pod: "db-pod", Reason: "CrashLoopBackOff",
			Message: "back-off restarting failed container", Severity: "critical",
			Count: 3, LastSeen: time.Now()},
		{ID: "e3", Namespace: "kube-system", Pod: "dns-pod", Reason: "Unhealthy",
			Message: "readiness probe failed connection refused", Severity: "warning",
			Count: 1, LastSeen: time.Now()},
	}
	for _, e := range events {
		if err := idx.Index(e); err != nil {
			t.Fatal(err)
		}
	}

	results, err := idx.Search("memory")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Error("expected at least 1 result for 'memory' search")
	}
	if results[0].ID != "e1" {
		t.Errorf("expected e1 for memory search, got %s", results[0].ID)
	}
}

func TestSearch_NoResults(t *testing.T) {
	idx, err := New(filepath.Join(t.TempDir(), "test.bleve"))
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()

	results, err := idx.Search("nonexistentterm12345")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
