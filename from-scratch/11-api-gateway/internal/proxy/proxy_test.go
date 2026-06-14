package proxy_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tour_of_go/projects/from-scratch/11-api-gateway/internal/auth"
	"tour_of_go/projects/from-scratch/11-api-gateway/internal/middleware"
	"tour_of_go/projects/from-scratch/11-api-gateway/internal/proxy"
)

func TestProxy_InjectsHeaders(t *testing.T) {
	// Upstream backend that captures received headers.
	var (
		gotUserID  string
		gotRole    string
		gotReqID   string
		gotAuthHdr string
	)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = r.Header.Get("X-Internal-User-Id")
		gotRole = r.Header.Get("X-Internal-User-Role")
		gotReqID = r.Header.Get("X-Request-ID")
		gotAuthHdr = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	a := auth.NewAuthenticator("secret", time.Hour)
	token, _ := a.Issue("alice", "admin")

	// Build handler chain: RequestID → Auth → proxy
	h := middleware.RequestID(
		middleware.Auth(a)(
			proxy.New(backend.URL),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Request-ID", "trace-abc")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if gotUserID != "alice" {
		t.Fatalf("X-Internal-User-Id: want alice, got %q", gotUserID)
	}
	if gotRole != "admin" {
		t.Fatalf("X-Internal-User-Role: want admin, got %q", gotRole)
	}
	if gotReqID != "trace-abc" {
		t.Fatalf("X-Request-ID: want trace-abc, got %q", gotReqID)
	}
	if gotAuthHdr != "" {
		t.Fatalf("Authorization header must be stripped, got %q", gotAuthHdr)
	}
}
