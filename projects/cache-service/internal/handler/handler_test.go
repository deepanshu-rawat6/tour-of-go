package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tour_of_go/projects/cache-service/internal/cache"
	"tour_of_go/projects/cache-service/internal/handler"
	"tour_of_go/projects/cache-service/internal/store"
)

func newHandler(t *testing.T) *handler.Handler {
	t.Helper()
	lru := cache.NewLRU(10)
	t.Cleanup(lru.Close)
	mem := store.NewMemory()
	ca := cache.NewCacheAside(lru, mem, time.Minute)
	return handler.New(ca)
}

func TestHandler_SetAndGet(t *testing.T) {
	h := newHandler(t)
	mux := http.NewServeMux()
	h.Register(mux)

	// PUT
	body, _ := json.Marshal(map[string]string{"value": "bar"})
	req := httptest.NewRequest(http.MethodPut, "/cache/foo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("PUT want 204, got %d", rr.Code)
	}

	// GET — cache-aside: key was written to store, not cache; first GET populates cache
	req2 := httptest.NewRequest(http.MethodGet, "/cache/foo", nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("GET want 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(rr2.Body).Decode(&resp)
	if resp["value"] != "bar" {
		t.Fatalf("want bar, got %s", resp["value"])
	}
}

func TestHandler_GetMissing(t *testing.T) {
	h := newHandler(t)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/cache/missing", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rr.Code)
	}
}

func TestHandler_Delete(t *testing.T) {
	h := newHandler(t)
	mux := http.NewServeMux()
	h.Register(mux)

	body, _ := json.Marshal(map[string]string{"value": "v"})
	req := httptest.NewRequest(http.MethodPut, "/cache/k", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	httptest.NewRecorder() // discard
	mux.ServeHTTP(httptest.NewRecorder(), req)

	req2 := httptest.NewRequest(http.MethodDelete, "/cache/k", nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusNoContent {
		t.Fatalf("DELETE want 204, got %d", rr2.Code)
	}
}

func TestHandler_Stats(t *testing.T) {
	h := newHandler(t)
	mux := http.NewServeMux()
	h.Register(mux)

	// Trigger a miss
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/cache/x", nil))

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	var stats map[string]int64
	json.NewDecoder(rr.Body).Decode(&stats)
	if stats["misses"] != 1 {
		t.Fatalf("want 1 miss, got %d", stats["misses"])
	}
}
