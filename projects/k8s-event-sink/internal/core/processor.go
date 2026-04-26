package core

import (
	"context"
	"log/slog"
)

// EventProcessor is the central coordinator.
// It wires Filter → DedupEngine → Storage + Alerter.
type EventProcessor struct {
	filter  *Filter
	dedup   *DedupEngine
	storage StoragePort
	search  SearchPort
	alerter AlerterPort
}

func NewEventProcessor(
	filter *Filter,
	dedup *DedupEngine,
	storage StoragePort,
	search SearchPort,
	alerter AlerterPort,
) *EventProcessor {
	return &EventProcessor{
		filter:  filter,
		dedup:   dedup,
		storage: storage,
		search:  search,
		alerter: alerter,
	}
}

// Process is called by the K8s informer for each incoming event.
func (p *EventProcessor) Process(ctx context.Context, event Event) {
	// Step 1: Filter — drop noise, classify severity.
	filtered := p.filter.Apply(event)
	if filtered == nil {
		return
	}
	// Step 2: Dedup — leaky bucket; onForward is called for first + summary events.
	p.dedup.Process(ctx, *filtered)
}

// OnForward is registered as the dedup engine's callback.
// Called for: first occurrence (forwarded immediately) + summary events (window expiry).
func (p *EventProcessor) OnForward(ctx context.Context, event Event) {
	// Persist to SQLite.
	if err := p.storage.Save(ctx, event); err != nil {
		slog.Error("storage save failed", "event_id", event.ID, "error", err)
	}
	// Index in Bleve.
	if err := p.search.Index(event); err != nil {
		slog.Error("search index failed", "event_id", event.ID, "error", err)
	}
	// Alert.
	if err := p.alerter.Notify(ctx, event); err != nil {
		slog.Error("alerter notify failed", "event_id", event.ID, "error", err)
	}
}
