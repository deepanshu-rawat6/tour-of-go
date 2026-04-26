package tracker

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/tour-of-go/tf-drift-detector/internal/poller"
)

// DriftRecord is a persisted record of a known drifted resource.
type DriftRecord struct {
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
}

// Tracker maintains stateful drift records across polling cycles.
type Tracker struct {
	mu      sync.Mutex
	known   map[string]DriftRecord // key = type:id
	statePath string
}

// New creates a Tracker, loading persisted state from statePath if it exists.
func New(statePath string) *Tracker {
	t := &Tracker{
		known:     make(map[string]DriftRecord),
		statePath: statePath,
	}
	t.load()
	return t
}

// Update compares current poll results against known drift state.
// Returns newly drifted resources and newly resolved resources.
func (t *Tracker) Update(results []poller.DriftResult) (newDrift, resolved []poller.DriftResult) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	currentDrifted := make(map[string]bool)

	for _, r := range results {
		key := fmt.Sprintf("%s:%s", r.Resource.Type, r.Resource.ID)
		if r.Drifted {
			currentDrifted[key] = true
			if _, known := t.known[key]; !known {
				// New drift — not seen before
				newDrift = append(newDrift, r)
				t.known[key] = DriftRecord{
					ResourceType: r.Resource.Type,
					ResourceID:   r.Resource.ID,
					FirstSeen:    now,
					LastSeen:     now,
				}
			} else {
				// Update last seen
				rec := t.known[key]
				rec.LastSeen = now
				t.known[key] = rec
			}
		}
	}

	// Find resolved: was drifted before, now clean
	for key, rec := range t.known {
		if !currentDrifted[key] {
			// Find the original result to return
			for _, r := range results {
				if fmt.Sprintf("%s:%s", r.Resource.Type, r.Resource.ID) == key {
					resolved = append(resolved, r)
					break
				}
			}
			_ = rec
			delete(t.known, key)
		}
	}

	return newDrift, resolved
}

// Persist writes the current drift state to disk.
func (t *Tracker) Persist() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	data, err := json.MarshalIndent(t.known, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(t.statePath, data, 0644)
}

func (t *Tracker) load() {
	data, err := os.ReadFile(t.statePath)
	if err != nil {
		return // file doesn't exist yet — start fresh
	}
	json.Unmarshal(data, &t.known)
}

// KnownCount returns the number of currently tracked drifted resources.
func (t *Tracker) KnownCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.known)
}
