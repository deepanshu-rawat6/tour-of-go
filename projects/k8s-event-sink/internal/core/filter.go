package core

import "github.com/tour-of-go/k8s-event-sink/internal/config"

// hardcodedSeverity maps common K8s event reasons to severity levels.
// These are the defaults; config.Severity overrides them.
var hardcodedSeverity = map[string]string{
	// Critical — always alert
	"OOMKilled":          "critical",
	"CrashLoopBackOff":   "critical",
	"NodeNotReady":       "critical",
	"Evicted":            "critical",
	"FailedScheduling":   "critical",
	"BackOff":            "critical",
	"NodeMemoryPressure": "critical",
	"NodeDiskPressure":   "critical",

	// Warning — store but don't page
	"Unhealthy":       "warning",
	"FailedMount":     "warning",
	"FailedAttachVolume": "warning",
	"NetworkNotReady": "warning",

	// Ignore — normal lifecycle noise
	"Scheduled":   "ignore",
	"Pulling":     "ignore",
	"Pulled":      "ignore",
	"Created":     "ignore",
	"Started":     "ignore",
	"Killing":     "ignore",
}

// Filter classifies and filters incoming events.
type Filter struct {
	cfg *config.Config
}

func NewFilter(cfg *config.Config) *Filter {
	return &Filter{cfg: cfg}
}

// Apply returns the event with Severity set, or nil if the event should be dropped.
func (f *Filter) Apply(event Event) *Event {
	// Drop all Normal events by default.
	if event.Type == "Normal" {
		return nil
	}

	// Check config ignore list.
	for _, r := range f.cfg.IgnoreReasons {
		if r == event.Reason {
			return nil
		}
	}

	// Determine severity: config overrides take precedence over hardcoded defaults.
	severity := "warning" // default for Warning-type events
	if s, ok := f.cfg.Severity[event.Reason]; ok {
		severity = s
	} else if s, ok := hardcodedSeverity[event.Reason]; ok {
		severity = s
	}

	if severity == "ignore" {
		return nil
	}

	event.Severity = severity
	return &event
}
