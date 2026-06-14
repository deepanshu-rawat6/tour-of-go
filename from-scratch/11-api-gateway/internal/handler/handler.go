package handler

import (
	"encoding/json"
	"net/http"

	"tour_of_go/projects/from-scratch/11-api-gateway/internal/auth"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"}) //nolint:errcheck
}

type loginRequest struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Login issues a JWT for the given username+role (no real credential check — demo only).
func Login(a *auth.Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Role == "" {
			req.Role = "user"
		}
		token, err := a.Issue(req.Username, req.Role)
		if err != nil {
			http.Error(w, "could not issue token", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": token}) //nolint:errcheck
	}
}

// ListRoutes returns the configured upstream route map.
func ListRoutes(routes map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(routes) //nolint:errcheck
	}
}
