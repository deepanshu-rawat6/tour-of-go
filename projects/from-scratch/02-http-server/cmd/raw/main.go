// Command raw starts the hand-rolled HTTP/1.1 server on :8080.
package main

import (
	"log"
	"tour_of_go/projects/from-scratch/02-http-server/internal/raw"
)

func main() {
	s := raw.New(":8080")
	s.Handle("/", func(w *raw.ResponseWriter, r *raw.Request) {
		w.Header("Content-Type", "application/json")
		w.Write(`{"message":"hello from raw HTTP/1.1 server"}`)
	})
	s.Handle("/health", func(w *raw.ResponseWriter, r *raw.Request) {
		w.Write("ok")
	})
	log.Println("raw HTTP server on :8080")
	log.Fatal(s.Start())
}
