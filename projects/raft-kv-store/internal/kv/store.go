package kv

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Command is the structure encoded in each Raft log entry.
type Command struct {
	Op    string `json:"op"`    // "put" or "delete"
	Key   string `json:"key"`
	Value string `json:"value"` // empty for delete
}

// Store is the in-memory KV state machine.
type Store struct {
	mu   sync.RWMutex
	data map[string]string
}

func New() *Store {
	return &Store{data: make(map[string]string)}
}

// Apply decodes and applies a committed Raft command to the store.
func (s *Store) Apply(command []byte) (string, error) {
	var cmd Command
	if err := json.Unmarshal(command, &cmd); err != nil {
		return "", fmt.Errorf("invalid command: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch cmd.Op {
	case "put":
		s.data[cmd.Key] = cmd.Value
		return cmd.Value, nil
	case "delete":
		delete(s.data, cmd.Key)
		return "", nil
	default:
		return "", fmt.Errorf("unknown op: %s", cmd.Op)
	}
}

// Get returns the value for a key and whether it exists.
func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

// Snapshot returns a copy of the current state (for status/debug).
func (s *Store) Snapshot() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	return out
}
