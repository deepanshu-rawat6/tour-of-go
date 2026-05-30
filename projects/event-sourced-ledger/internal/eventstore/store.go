package eventstore

import (
	"fmt"
	"sync"
	"time"
)

type EventType string

const (
	AccountOpened    EventType = "AccountOpened"
	MoneyDeposited   EventType = "MoneyDeposited"
	MoneyWithdrawn   EventType = "MoneyWithdrawn"
	MoneyTransferred EventType = "MoneyTransferred"
)

type Event struct {
	ID          int
	AggregateID string
	Type        EventType
	Data        map[string]any
	Version     int
	Timestamp   time.Time
}

// Store is an append-only in-memory event store with optimistic concurrency.
type Store struct {
	events  []Event
	streams map[string][]Event // aggregateID → events
	mu      sync.RWMutex
	nextID  int
}

func New() *Store {
	return &Store{streams: make(map[string][]Event)}
}

// Append adds events with optimistic concurrency check.
func (s *Store) Append(aggregateID string, expectedVersion int, events []Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current := len(s.streams[aggregateID])
	if current != expectedVersion {
		return fmt.Errorf("concurrency conflict: expected version %d, got %d", expectedVersion, current)
	}

	for i := range events {
		s.nextID++
		events[i].ID = s.nextID
		events[i].AggregateID = aggregateID
		events[i].Version = current + i + 1
		events[i].Timestamp = time.Now()

		s.events = append(s.events, events[i])
		s.streams[aggregateID] = append(s.streams[aggregateID], events[i])
	}
	return nil
}

// Load returns all events for an aggregate.
func (s *Store) Load(aggregateID string) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.streams[aggregateID]
}

// AllEvents returns all events in order (for projections).
func (s *Store) AllEvents() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Event, len(s.events))
	copy(result, s.events)
	return result
}
