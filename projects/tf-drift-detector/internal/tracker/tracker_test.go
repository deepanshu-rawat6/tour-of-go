package tracker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tour-of-go/tf-drift-detector/internal/diff"
	"github.com/tour-of-go/tf-drift-detector/internal/poller"
	"github.com/tour-of-go/tf-drift-detector/internal/state"
)

func driftResult(id string, drifted bool) poller.DriftResult {
	return poller.DriftResult{
		Resource: state.ManagedResource{Type: "aws_instance", ID: id},
		Drifted:  drifted,
		Fields:   []diff.DriftField{{Path: "instance_type", Expected: "t3.micro", Actual: "t3.large"}},
	}
}

func TestTracker_NewDrift(t *testing.T) {
	tr := New(filepath.Join(t.TempDir(), "state.json"))
	newDrift, resolved := tr.Update([]poller.DriftResult{driftResult("i-1", true)})
	if len(newDrift) != 1 {
		t.Errorf("expected 1 new drift, got %d", len(newDrift))
	}
	if len(resolved) != 0 {
		t.Errorf("expected 0 resolved, got %d", len(resolved))
	}
}

func TestTracker_SecondCycleSilent(t *testing.T) {
	tr := New(filepath.Join(t.TempDir(), "state.json"))
	tr.Update([]poller.DriftResult{driftResult("i-1", true)})
	// Second cycle — same drift, should be silent
	newDrift, resolved := tr.Update([]poller.DriftResult{driftResult("i-1", true)})
	if len(newDrift) != 0 {
		t.Errorf("expected 0 new drift on second cycle, got %d", len(newDrift))
	}
	if len(resolved) != 0 {
		t.Errorf("expected 0 resolved, got %d", len(resolved))
	}
}

func TestTracker_Resolved(t *testing.T) {
	tr := New(filepath.Join(t.TempDir(), "state.json"))
	tr.Update([]poller.DriftResult{driftResult("i-1", true)})
	// Third cycle — drift resolved
	_, resolved := tr.Update([]poller.DriftResult{driftResult("i-1", false)})
	if len(resolved) != 1 {
		t.Errorf("expected 1 resolved, got %d", len(resolved))
	}
}

func TestTracker_PersistAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	tr := New(path)
	tr.Update([]poller.DriftResult{driftResult("i-1", true)})
	if err := tr.Persist(); err != nil {
		t.Fatal(err)
	}

	// Verify file was written
	if _, err := os.Stat(path); err != nil {
		t.Fatal("state file not created")
	}

	// Load into new tracker — should know about i-1
	tr2 := New(path)
	if tr2.KnownCount() != 1 {
		t.Errorf("expected 1 known drift after load, got %d", tr2.KnownCount())
	}
	// i-1 is already known — second cycle should be silent
	newDrift, _ := tr2.Update([]poller.DriftResult{driftResult("i-1", true)})
	if len(newDrift) != 0 {
		t.Errorf("expected 0 new drift after load, got %d", len(newDrift))
	}
}
