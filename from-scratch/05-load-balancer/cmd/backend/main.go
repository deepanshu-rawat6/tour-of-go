// Command backend starts a simple HTTP backend that identifies itself by port.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := envOr("PORT", "9010")
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "backend:%s\n", port)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	log.Printf("backend on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
