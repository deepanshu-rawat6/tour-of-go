package hub_test

import (
	"sync"
	"testing"
	"time"

	"tour_of_go/projects/from-scratch/03-websocket-chat/internal/hub"
)

type mockClient struct {
	mu       sync.Mutex
	received []string
	room     string
}

func (m *mockClient) Send(b []byte) {
	m.mu.Lock()
	m.received = append(m.received, string(b))
	m.mu.Unlock()
}
func (m *mockClient) Room() string { return m.room }

func TestHub_BroadcastToRoom(t *testing.T) {
	h := hub.New()
	go h.Run()

	c1 := &mockClient{room: "general"}
	c2 := &mockClient{room: "general"}
	c3 := &mockClient{room: "other"}

	h.Join(c1, "general")
	h.Join(c2, "general")
	h.Join(c3, "other")
	time.Sleep(10 * time.Millisecond)

	h.Publish(hub.Message{Room: "general", Sender: "alice", Text: "hi"})
	time.Sleep(20 * time.Millisecond)

	c1.mu.Lock()
	defer c1.mu.Unlock()
	c2.mu.Lock()
	defer c2.mu.Unlock()
	c3.mu.Lock()
	defer c3.mu.Unlock()

	if len(c1.received) != 1 {
		t.Fatalf("c1 want 1 msg, got %d", len(c1.received))
	}
	if len(c2.received) != 1 {
		t.Fatalf("c2 want 1 msg, got %d", len(c2.received))
	}
	if len(c3.received) != 0 {
		t.Fatalf("c3 want 0 msgs (different room), got %d", len(c3.received))
	}
}

func TestHub_Leave(t *testing.T) {
	h := hub.New()
	go h.Run()

	c := &mockClient{room: "r"}
	h.Join(c, "r")
	time.Sleep(10 * time.Millisecond)
	h.Leave(c)
	time.Sleep(10 * time.Millisecond)
	h.Publish(hub.Message{Room: "r", Sender: "x", Text: "msg"})
	time.Sleep(20 * time.Millisecond)

	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.received) != 0 {
		t.Fatalf("want 0 after leave, got %d", len(c.received))
	}
}
