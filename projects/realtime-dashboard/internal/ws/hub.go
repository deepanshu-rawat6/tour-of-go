// Package ws implements the WebSocket hub pattern.
// One hub goroutine manages all connected clients and broadcasts messages.
// This prevents slow clients from blocking the broadcaster.
package ws

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"
)

// Message is the JSON envelope sent to all connected browsers.
type Message struct {
	Type string `json:"type"` // "concurrency_update" | "job_update"
	Data any    `json:"data"`
}

// Hub maintains the set of active WebSocket clients and broadcasts messages.
type Hub struct {
	mu        sync.RWMutex
	clients   map[*Client]struct{}
	broadcast chan Message
	log       *slog.Logger
}

func NewHub() *Hub {
	return &Hub{
		clients:   make(map[*Client]struct{}),
		broadcast: make(chan Message, 64),
		log:       slog.Default(),
	}
}

// Run starts the hub's event loop. Call in a goroutine.
func (h *Hub) Run() {
	for msg := range h.broadcast {
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		h.mu.RLock()
		for client := range h.clients {
			select {
			case client.send <- data:
			default:
				// Slow client — drop message rather than block
				h.log.Debug("dropping message for slow client")
			}
		}
		h.mu.RUnlock()
	}
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msg Message) {
	select {
	case h.broadcast <- msg:
	default:
	}
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

// Client is a single WebSocket connection.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

