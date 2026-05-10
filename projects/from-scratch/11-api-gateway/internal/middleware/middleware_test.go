package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tour_of_go/projects/from-scratch/11-api-gateway/internal/auth"
	"tour_of_go/projects/from-scratch/11-api-gateway/internal/middleware"
)

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// --- RequestID ---

func TestRequestID_GeneratesIfMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	middleware.RequestID(okHandler).ServeHTTP(rr, req)

	id := rr.Header().Get("X-Request-ID")
	if id == "" {
		t.Fatal("expected X-Request-ID to be set")
	}
}

func TestRequestID_PreservesExisting(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "my-trace-id")
	rr := httptest.NewRecorder()
	middleware.RequestID(okHandler).ServeHTTP(rr, req)

	if rr.Header().Get("X-Request-ID") != "my-trace-id" {
		t.Fatal("expected existing X-Request-ID to be preserved")
	}
}

// --- Auth ---

func newAuthn() *auth.Authenticator {
	return auth.NewAuthenticator("test-secret", time.Hour)
}

func TestAuth_ValidToken(t *testing.T) {
	a := newAuthn()
	token, _ := a.Issue("alice", "user")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	middleware.Auth(a)(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
}

func TestAuth_MissingHeader(t *testing.T) {
	a := newAuthn()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	middleware.Auth(a)(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	a := newAuthn()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-token")
	rr := httptest.NewRecorder()
	middleware.Auth(a)(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

// --- RequireRole ---

func TestRequireRole_Allowed(t *testing.T) {
	a := newAuthn()
	token, _ := a.Issue("alice", "admin")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	h := middleware.Auth(a)(middleware.RequireRole("admin")(okHandler))
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
}

func TestRequireRole_Forbidden(t *testing.T) {
	a := newAuthn()
	token, _ := a.Issue("bob", "user")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	h := middleware.Auth(a)(middleware.RequireRole("admin")(okHandler))
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rr.Code)
	}
}
