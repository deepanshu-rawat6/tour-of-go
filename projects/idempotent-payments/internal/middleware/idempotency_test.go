package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tour_of_go/projects/idempotent-payments/internal/domain"
	"tour_of_go/projects/idempotent-payments/internal/middleware"
)

// mockStore is a simple in-memory IdempotencyStore for testing.
type mockStore struct {
	records map[string]*domain.IdempotencyRecord
	saved   *domain.IdempotencyRecord
}

func newMockStore() *mockStore {
	return &mockStore{records: make(map[string]*domain.IdempotencyRecord)}
}

func (m *mockStore) Get(_ context.Context, key string) (*domain.IdempotencyRecord, error) {
	return m.records[key], nil
}

func (m *mockStore) Save(_ context.Context, rec *domain.IdempotencyRecord) error {
	m.saved = rec
	m.records[rec.Key] = rec
	return nil
}

func (m *mockStore) DeleteExpired(_ context.Context) (int64, error) { return 0, nil }

func TestIdempotency_MissingHeader(t *testing.T) {
	store := newMockStore()
	h := middleware.Idempotency(store, time.Hour)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/payments", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestIdempotency_CacheHit(t *testing.T) {
	store := newMockStore()
	store.records["key-1"] = &domain.IdempotencyRecord{
		Key:        "key-1",
		StatusCode: 201,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       []byte(`{"id":"cached"}`),
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	handlerCalled := false
	h := middleware.Idempotency(store, time.Hour)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/payments", nil)
	req.Header.Set("Idempotency-Key", "key-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != 201 {
		t.Fatalf("want 201, got %d", rr.Code)
	}
	if string(rr.Body.Bytes()) != `{"id":"cached"}` {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
	if handlerCalled {
		t.Fatal("handler must not be called on cache hit")
	}
}

func TestIdempotency_CacheMiss(t *testing.T) {
	store := newMockStore()
	h := middleware.Idempotency(store, time.Hour)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"new"}`)) //nolint:errcheck
	}))

	req := httptest.NewRequest(http.MethodPost, "/payments", nil)
	req.Header.Set("Idempotency-Key", "key-new")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d", rr.Code)
	}
	if store.saved == nil {
		t.Fatal("expected response to be stored")
	}
	if store.saved.StatusCode != 201 {
		t.Fatalf("stored status: want 201, got %d", store.saved.StatusCode)
	}
	if string(store.saved.Body) != `{"id":"new"}` {
		t.Fatalf("stored body mismatch: %s", store.saved.Body)
	}
}
