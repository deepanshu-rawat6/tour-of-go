// Command server starts the WebSocket chat server on :8082.
package main

import (
	"log"
	"net/http"

	"tour_of_go/projects/from-scratch/03-websocket-chat/internal/handler"
	"tour_of_go/projects/from-scratch/03-websocket-chat/internal/hub"
)

func main() {
	h := hub.New()
	go h.Run()

	http.HandleFunc("/ws", handler.Chat(h))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/index.html")
	})

	log.Println("chat server on :8082 — open http://localhost:8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
