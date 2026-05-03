// Command aggregator starts the log aggregator (TCP :9002 for ingest, HTTP :8085 for query).
package main

import (
	"log"
	"net/http"

	"tour_of_go/projects/from-scratch/08-log-aggregator/internal/aggregator"
	"tour_of_go/projects/from-scratch/08-log-aggregator/internal/handler"
)

func main() {
	a := aggregator.New()
	go func() {
		if err := a.StartTCP(":9002"); err != nil {
			log.Fatal(err)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/logs", handler.Logs(a))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	log.Println("aggregator: TCP :9002 (ingest), HTTP :8085 (query)")
	log.Fatal(http.ListenAndServe(":8085", mux))
}
