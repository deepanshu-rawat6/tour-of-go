// Package handler provides the HTTP query API for the aggregator.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"tour_of_go/projects/from-scratch/08-log-aggregator/internal/aggregator"
)

func Logs(a *aggregator.Aggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		query := q.Get("q")
		source := q.Get("source")
		limit := 100
		if l := q.Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil {
				limit = n
			}
		}
		entries := a.Search(query, source, limit)
		lines := make([]string, len(entries))
		for i, e := range entries {
			lines[i] = aggregator.FormatEntry(e)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"count": len(lines), "logs": lines})
	}
}
