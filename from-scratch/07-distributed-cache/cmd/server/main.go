package main

import (
	"log"
	"tour_of_go/projects/from-scratch/07-distributed-cache/internal/server"
	"tour_of_go/projects/from-scratch/07-distributed-cache/internal/store"
)

func main() {
	s := server.New(":6380", store.New())
	log.Fatal(s.Start())
}
