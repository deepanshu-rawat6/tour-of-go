// Command shipper tails a log file and ships lines to the aggregator.
package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"tour_of_go/projects/from-scratch/08-log-aggregator/internal/tailer"
)

func main() {
	path := envOr("LOG_FILE", "/tmp/app.log")
	source := envOr("SOURCE", "app1")
	aggAddr := envOr("AGG_ADDR", ":9002")

	conn, err := net.Dial("tcp", aggAddr)
	if err != nil {
		log.Fatalf("connect to aggregator: %v", err)
	}
	defer conn.Close()

	tl := tailer.New(path)
	go tl.Start()
	log.Printf("shipping %s → %s as source=%s", path, aggAddr, source)
	for line := range tl.Lines {
		fmt.Fprintf(conn, "%s\t%s\n", source, line)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
