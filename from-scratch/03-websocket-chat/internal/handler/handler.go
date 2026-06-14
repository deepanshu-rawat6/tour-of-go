// Package handler upgrades HTTP connections to WebSocket and wires them to the hub.
package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"tour_of_go/projects/from-scratch/03-websocket-chat/internal/hub"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type wsClient struct {
	conn *websocket.Conn
	send chan []byte
	room string
}

func (c *wsClient) Send(b []byte) {
	select {
	case c.send <- b:
	default:
	}
}
func (c *wsClient) Room() string { return c.room }

// Chat handles WebSocket upgrades and wires clients to the hub.
func Chat(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		room := r.URL.Query().Get("room")
		if room == "" {
			room = "general"
		}
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "anonymous"
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("upgrade:", err)
			return
		}

		c := &wsClient{conn: conn, send: make(chan []byte, 32), room: room}
		h.Join(c, room)
		defer func() {
			h.Leave(c)
			conn.Close()
		}()

		// Writer goroutine
		go func() {
			for msg := range c.send {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			}
		}()

		// Reader loop
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				break
			}
			var payload struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(raw, &payload); err != nil {
				payload.Text = string(raw)
			}
			h.Publish(hub.Message{Room: room, Sender: name, Text: payload.Text})
		}
	}
}
