// Command stdlib starts the net/http server on :8081 for comparison.
package main

import (
	"log"
	"tour_of_go/projects/from-scratch/02-http-server/internal/stdlib"
)

func main() {
	srv := stdlib.New(":8081")
	log.Println("stdlib HTTP server on :8081")
	log.Fatal(srv.ListenAndServe())
}
