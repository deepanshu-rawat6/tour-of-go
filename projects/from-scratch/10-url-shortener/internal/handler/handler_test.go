package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"tour_of_go/projects/from-scratch/10-url-shortener/internal/handler"
)

type mockCache struct{ data map[string]string }

func newMockCache() *mockCache { return &mockCache{data: make(map[string]string)} }

func (m *mockCache) Set(k, v string, _ int) error { m.data[k] = v; return nil }
func (m *mockCache) Get(k string) (string, error) {
	v, ok := m.data[k]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return v, nil
}
func (m *mockCache) Del(k string) error { delete(m.data, k); return nil }

func TestShorten_And_Redirect(t *testing.T) {
	mc := newMockCache()
	h := handler.NewWithCache(mc, nil, 3600)
	mux := http.NewServeMux()
	h.Register(mux)

	body, _ := json.Marshal(map[string]string{"url": "https://example.com"})
	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("shorten: want 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	code := resp["short"]
	if code == "" {
		t.Fatal("expected short code in response")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/"+code, nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusMovedPermanently {
		t.Fatalf("redirect: want 301, got %d", rr2.Code)
	}
	if rr2.Header().Get("Location") != "https://example.com" {
		t.Fatalf("want Location: https://example.com, got %s", rr2.Header().Get("Location"))
	}
}

func TestShorten_MissingURL(t *testing.T) {
	mc := newMockCache()
	h := handler.NewWithCache(mc, nil, 3600)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}
