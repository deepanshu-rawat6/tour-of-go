package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8081"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Printf("\n--- incoming request ---\n%s\n---\n", dump)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
			"upstream":          addr,
			"x-internal-user":   r.Header.Get("X-Internal-User-Id"),
			"x-internal-role":   r.Header.Get("X-Internal-User-Role"),
			"x-request-id":      r.Header.Get("X-Request-ID"),
			"authorization":     r.Header.Get("Authorization"), // should be empty
		})
	})

	fmt.Printf("mock upstream listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
