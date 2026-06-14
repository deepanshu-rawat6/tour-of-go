package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tour_of_go/projects/from-scratch/11-api-gateway/internal/auth"
	"tour_of_go/projects/from-scratch/11-api-gateway/internal/handler"
)

func TestHealthCheck(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.HealthCheck(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp) //nolint:errcheck
	if resp["status"] != "ok" {
		t.Fatalf("want ok, got %s", resp["status"])
	}
}

func TestLogin_IssuesToken(t *testing.T) {
	a := auth.NewAuthenticator("secret", time.Hour)
	h := handler.Login(a)

	body, _ := json.Marshal(map[string]string{"username": "alice", "role": "admin"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp) //nolint:errcheck
	if resp["token"] == "" {
		t.Fatal("expected token in response")
	}
	// Validate the issued token.
	claims, err := a.Validate(resp["token"])
	if err != nil {
		t.Fatalf("issued token invalid: %v", err)
	}
	if claims.UserID != "alice" || claims.Role != "admin" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestLogin_MissingUsername(t *testing.T) {
	a := auth.NewAuthenticator("secret", time.Hour)
	h := handler.Login(a)

	body, _ := json.Marshal(map[string]string{"role": "user"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestListRoutes(t *testing.T) {
	routes := map[string]string{"users": "http://localhost:8081"}
	h := handler.ListRoutes(routes)

	req := httptest.NewRequest(http.MethodGet, "/admin/routes", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp) //nolint:errcheck
	if resp["users"] != "http://localhost:8081" {
		t.Fatalf("unexpected routes: %v", resp)
	}
}
