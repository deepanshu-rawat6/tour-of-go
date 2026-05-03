package store_test

import (
	"testing"
	"time"

	"tour_of_go/projects/from-scratch/07-distributed-cache/internal/store"
)

func TestStore_SetGet(t *testing.T) {
	s := store.New()
	s.Set("k", "v", 0)
	v, ok := s.Get("k")
	if !ok || v != "v" {
		t.Fatalf("want v ok=true, got %q ok=%v", v, ok)
	}
}

func TestStore_TTLExpiry(t *testing.T) {
	s := store.New()
	s.Set("k", "v", 50*time.Millisecond)
	if _, ok := s.Get("k"); !ok {
		t.Fatal("expected hit before expiry")
	}
	time.Sleep(100 * time.Millisecond)
	if _, ok := s.Get("k"); ok {
		t.Fatal("expected miss after expiry")
	}
}

func TestStore_Del(t *testing.T) {
	s := store.New()
	s.Set("a", "1", 0)
	s.Set("b", "2", 0)
	n := s.Del("a", "b", "missing")
	if n != 2 {
		t.Fatalf("want 2 deleted, got %d", n)
	}
}

func TestStore_Keys(t *testing.T) {
	s := store.New()
	s.Set("x", "1", 0)
	s.Set("y", "2", 0)
	keys := s.Keys()
	if len(keys) != 2 {
		t.Fatalf("want 2 keys, got %d", len(keys))
	}
}
