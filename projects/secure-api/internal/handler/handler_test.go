package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tour_of_go/projects/secure-api/internal/auth"
	"tour_of_go/projects/secure-api/internal/handler"
	"tour_of_go/projects/secure-api/internal/middleware"
)

func setup(t *testing.T) (authn *auth.UserStore, jwt *auth.JWTAdapter) {
	t.Helper()
	jwt = auth.NewJWTAdapter("test-secret", time.Hour)
	authn = auth.NewUserStore(time.Hour)
	if err := authn.AddUser("alice", "pass", []string{"user"}); err != nil {
		t.Fatal(err)
	}
	return
}

func TestToken_ValidCredentials(t *testing.T) {
	authn, jwt := setup(t)
	h := handler.Token(authn, jwt)

	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "pass"})
	req := httptest.NewRequest(http.MethodPost, "/oauth2/token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["access_token"] == "" {
		t.Fatal("expected access_token in response")
	}
}

func TestToken_BadCredentials(t *testing.T) {
	authn, jwt := setup(t)
	h := handler.Token(authn, jwt)

	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/oauth2/token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestMe_WithValidToken(t *testing.T) {
	authn, jwt := setup(t)
	tokenH := handler.Token(authn, jwt)

	// Get a token first
	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "pass"})
	req := httptest.NewRequest(http.MethodPost, "/oauth2/token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	tokenH.ServeHTTP(rr, req)
	var tokenResp map[string]any
	json.NewDecoder(rr.Body).Decode(&tokenResp)
	accessToken := tokenResp["access_token"].(string)

	// Call /me with the token
	meH := middleware.Chain(http.HandlerFunc(handler.Me), middleware.Auth(jwt))
	req2 := httptest.NewRequest(http.MethodGet, "/me", nil)
	req2.Header.Set("Authorization", "Bearer "+accessToken)
	rr2 := httptest.NewRecorder()
	meH.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	var meResp map[string]any
	json.NewDecoder(rr2.Body).Decode(&meResp)
	if meResp["user_id"] != "alice" {
		t.Fatalf("want alice, got %v", meResp["user_id"])
	}
}

func TestMe_MissingToken(t *testing.T) {
	_, jwt := setup(t)
	meH := middleware.Chain(http.HandlerFunc(handler.Me), middleware.Auth(jwt))
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rr := httptest.NewRecorder()
	meH.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}
