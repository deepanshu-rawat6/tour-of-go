// Command server starts a raw TCP echo server on :8080.
package main

import (
	"log"
	"tour_of_go/projects/from-scratch/01-tcp-server/internal/server"
)

func main() {
	s := server.New(":8080")
	log.Fatal(s.Start())
}
