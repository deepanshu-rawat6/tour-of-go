package kv

import (
	"encoding/json"
	"sync"
	"testing"
)

func cmd(op, key, value string) []byte {
	b, _ := json.Marshal(Command{Op: op, Key: key, Value: value})
	return b
}

func TestStore_PutAndGet(t *testing.T) {
	s := New()
	s.Apply(cmd("put", "foo", "bar"))
	v, ok := s.Get("foo")
	if !ok || v != "bar" {
		t.Errorf("expected bar, got %q %v", v, ok)
	}
}

func TestStore_Delete(t *testing.T) {
	s := New()
	s.Apply(cmd("put", "foo", "bar"))
	s.Apply(cmd("delete", "foo", ""))
	_, ok := s.Get("foo")
	if ok {
		t.Error("expected key to be deleted")
	}
}

func TestStore_ConcurrentReads(t *testing.T) {
	s := New()
	s.Apply(cmd("put", "k", "v"))
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Get("k")
		}()
	}
	wg.Wait()
}

func TestStore_UnknownOp(t *testing.T) {
	s := New()
	if _, err := s.Apply(cmd("cas", "k", "v")); err == nil {
		t.Error("expected error for unknown op")
	}
}
