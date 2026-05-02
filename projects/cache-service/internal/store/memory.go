package store

import (
	"context"
	"errors"
	"sync"
)

// ErrNotFound is returned when a key does not exist in the store.
var ErrNotFound = errors.New("key not found")

// Memory is a thread-safe in-memory Store implementation used for testing.
type Memory struct {
	mu   sync.RWMutex
	data map[string]string
	// Calls counts how many times Get was called — used to verify singleflight dedup.
	Calls int
}

func NewMemory() *Memory { return &Memory{data: make(map[string]string)} }

func (m *Memory) Get(_ context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.Calls++
	v, ok := m.data[key]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (m *Memory) Set(_ context.Context, key, val string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = val
	return nil
}

func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}
