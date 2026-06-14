package main

import (
	"log"
	"tour_of_go/projects/from-scratch/06-message-queue/internal/queue"
	"tour_of_go/projects/from-scratch/06-message-queue/internal/server"
)

func main() {
	b := queue.New()
	s := server.New(":9001", b)
	log.Fatal(s.Start())
}
