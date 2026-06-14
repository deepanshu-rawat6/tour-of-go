// Command server starts the task scheduler with HTTP API on :8086.
package main

import (
	"context"
	"log"
	"net/http"

	"tour_of_go/projects/from-scratch/09-task-scheduler/internal/handler"
	"tour_of_go/projects/from-scratch/09-task-scheduler/internal/scheduler"
)

func main() {
	s := scheduler.New()
	// Seed a demo task
	s.Add("demo", "log every minute", "* * * * *", func() {
		log.Println("[demo task] fired!")
	})

	ctx := context.Background()
	go s.Start(ctx)

	mux := http.NewServeMux()
	handler.Register(mux, s)

	log.Println("task-scheduler on :8086")
	log.Fatal(http.ListenAndServe(":8086", mux))
}
