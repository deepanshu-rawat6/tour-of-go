package core

import (
	"testing"

	"github.com/tour-of-go/k8s-event-sink/internal/config"
)

func newFilter(ignoreReasons []string, severity map[string]string) *Filter {
	return NewFilter(&config.Config{
		IgnoreReasons: ignoreReasons,
		Severity:      severity,
	})
}

func event(typ, reason string) Event {
	return Event{ID: "test", Type: typ, Reason: reason, Namespace: "default", Pod: "pod-1"}
}

func TestFilter_DropsNormal(t *testing.T) {
	f := newFilter(nil, nil)
	if f.Apply(event("Normal", "Started")) != nil {
		t.Error("expected Normal event to be dropped")
	}
}

func TestFilter_PassesWarning(t *testing.T) {
	f := newFilter(nil, nil)
	e := f.Apply(event("Warning", "CrashLoopBackOff"))
	if e == nil {
		t.Fatal("expected Warning event to pass")
	}
	if e.Severity != "critical" {
		t.Errorf("expected critical severity, got %s", e.Severity)
	}
}

func TestFilter_IgnoreReasons(t *testing.T) {
	f := newFilter([]string{"Unhealthy"}, nil)
	if f.Apply(event("Warning", "Unhealthy")) != nil {
		t.Error("expected Unhealthy to be ignored via config")
	}
}

func TestFilter_ConfigSeverityOverride(t *testing.T) {
	f := newFilter(nil, map[string]string{"BackOff": "warning"})
	e := f.Apply(event("Warning", "BackOff"))
	if e == nil {
		t.Fatal("expected event to pass")
	}
	if e.Severity != "warning" {
		t.Errorf("expected config override to set warning, got %s", e.Severity)
	}
}

func TestFilter_HardcodedIgnore(t *testing.T) {
	f := newFilter(nil, nil)
	// "Pulling" is hardcoded as ignore
	if f.Apply(event("Warning", "Pulling")) != nil {
		t.Error("expected hardcoded ignore reason to be dropped")
	}
}

func TestFilter_UnknownReasonDefaultsToWarning(t *testing.T) {
	f := newFilter(nil, nil)
	e := f.Apply(event("Warning", "SomeUnknownReason"))
	if e == nil {
		t.Fatal("expected unknown Warning reason to pass")
	}
	if e.Severity != "warning" {
		t.Errorf("expected default warning severity, got %s", e.Severity)
	}
}
