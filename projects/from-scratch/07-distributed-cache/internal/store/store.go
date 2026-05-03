// Package store implements a thread-safe KV store with TTL expiry.
package store

import (
	"sync"
	"time"
)

type entry struct {
	value     string
	expiresAt time.Time // zero = no expiry
}

// Store is a thread-safe in-memory key-value store with TTL support.
type Store struct {
	mu   sync.RWMutex
	data map[string]entry
}

func New() *Store {
	s := &Store{data: make(map[string]entry)}
	go s.reap()
	return s
}

func (s *Store) Set(key, val string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := entry{value: val}
	if ttl > 0 {
		e.expiresAt = time.Now().Add(ttl)
	}
	s.data[key] = e
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok {
		return "", false
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		return "", false
	}
	return e.value, true
}

func (s *Store) Del(keys ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, k := range keys {
		if _, ok := s.data[k]; ok {
			delete(s.data, k)
			n++
		}
	}
	return n
}

func (s *Store) Exists(key string) bool {
	_, ok := s.Get(key)
	return ok
}

func (s *Store) TTL(key string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok {
		return -2 // key does not exist
	}
	if e.expiresAt.IsZero() {
		return -1 // no expiry
	}
	remaining := time.Until(e.expiresAt)
	if remaining <= 0 {
		return -2
	}
	return int64(remaining.Seconds())
}

func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data))
	now := time.Now()
	for k, e := range s.data {
		if e.expiresAt.IsZero() || now.Before(e.expiresAt) {
			keys = append(keys, k)
		}
	}
	return keys
}

func (s *Store) reap() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for k, e := range s.data {
			if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
				delete(s.data, k)
			}
		}
		s.mu.Unlock()
	}
}
