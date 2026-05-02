// Package handler provides HTTP handlers for the secure-api.
// Single Responsibility Principle: each handler does exactly one thing.
package handler

import (
	"encoding/json"
	"net/http"

	"tour_of_go/projects/secure-api/internal/middleware"
	"tour_of_go/projects/secure-api/internal/ports"
)

// Health is a public liveness probe — no auth required.
func Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Me returns the authenticated user's claims. Requires Auth middleware.
func Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "no claims in context", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"user_id":    claims.UserID(),
		"roles":      claims.Roles(),
		"expires_at": claims.ExpiresAt(),
	})
}

// TokenRequest is the OAuth2 password grant request body.
type TokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Token handles POST /oauth2/token — validates credentials and returns a JWT.
func Token(authn ports.UserAuthenticator, issuer ports.TokenIssuer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req TokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		claims, err := authn.Authenticate(req.Username, req.Password)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		tok, err := issuer.Issue(claims)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": tok.AccessToken(),
			"token_type":   tok.TokenType(),
			"expires_in":   tok.ExpiresIn(),
		})
	}
}
