// Package hub implements the WebSocket hub pattern.
// One hub goroutine owns all client state — no locks needed.
package hub

import "sync"

// Message is a chat message with sender info.
type Message struct {
	Room   string
	Sender string
	Text   string
}

// Client represents a connected WebSocket peer.
type Client interface {
	Send([]byte)
	Room() string
}

// Hub manages clients and broadcasts messages within rooms.
type Hub struct {
	mu      sync.RWMutex
	rooms   map[string]map[Client]struct{}
	join    chan joinMsg
	leave   chan Client
	publish chan Message
}

type joinMsg struct {
	client Client
	room   string
}

func New() *Hub {
	return &Hub{
		rooms:   make(map[string]map[Client]struct{}),
		join:    make(chan joinMsg, 16),
		leave:   make(chan Client, 16),
		publish: make(chan Message, 64),
	}
}

// Run processes hub events. Call in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case j := <-h.join:
			h.mu.Lock()
			if h.rooms[j.room] == nil {
				h.rooms[j.room] = make(map[Client]struct{})
			}
			h.rooms[j.room][j.client] = struct{}{}
			h.mu.Unlock()

		case c := <-h.leave:
			h.mu.Lock()
			for room, clients := range h.rooms {
				delete(clients, c)
				if len(clients) == 0 {
					delete(h.rooms, room)
				}
			}
			h.mu.Unlock()

		case msg := <-h.publish:
			h.mu.RLock()
			for c := range h.rooms[msg.Room] {
				c.Send([]byte(msg.Sender + ": " + msg.Text))
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Join(c Client, room string)  { h.join <- joinMsg{c, room} }
func (h *Hub) Leave(c Client)              { h.leave <- c }
func (h *Hub) Publish(msg Message)         { h.publish <- msg }
