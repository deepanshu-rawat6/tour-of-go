// Package handler exposes the cache strategies over HTTP.
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
)

// Cache is the minimal interface the handler depends on.
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, val string) error
	Delete(ctx context.Context, key string) error
}

// Handler wraps a Cache and tracks hit/miss stats.
type Handler struct {
	cache Cache
	hits  atomic.Int64
	misses atomic.Int64
}

func New(c Cache) *Handler { return &Handler{cache: c} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/cache/", h.dispatch)
	mux.HandleFunc("/stats", h.stats)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
}

func (h *Handler) dispatch(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/cache/")
	if key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.get(w, r, key)
	case http.MethodPut:
		h.set(w, r, key)
	case http.MethodDelete:
		h.delete(w, r, key)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request, key string) {
	val, err := h.cache.Get(r.Context(), key)
	if err != nil {
		h.misses.Add(1)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	h.hits.Add(1)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"value": val})
}

func (h *Handler) set(w http.ResponseWriter, r *http.Request, key string) {
	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := h.cache.Set(r.Context(), key, body.Value); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, key string) {
	h.cache.Delete(r.Context(), key)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) stats(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{
		"hits":   h.hits.Load(),
		"misses": h.misses.Load(),
	})
}
