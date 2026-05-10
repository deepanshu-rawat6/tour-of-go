package diagnostic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

// NewMockServer returns an httptest.Server that handles GET /builds/{userID}/latest.
func NewMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/builds/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(buildStatus{ //nolint:errcheck
			Status: "SUCCESS", BuildNumber: 42, DurationMs: 12000,
		})
	})
	return httptest.NewServer(mux)
}
