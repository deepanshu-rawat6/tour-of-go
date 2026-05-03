// Package handler provides the URL shortener HTTP handlers.
package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"tour_of_go/projects/from-scratch/10-url-shortener/internal/mq"
	"tour_of_go/projects/from-scratch/10-url-shortener/internal/shortener"
)

// Cache is the storage interface — satisfied by cache.Client or a mock.
type Cache interface {
	Set(key, value string, ttlSec int) error
	Get(key string) (string, error)
	Del(key string) error
}

// Handler wires the shortener with cache and MQ.
type Handler struct {
	cache  Cache
	mq     *mq.Client
	ttlSec int
}

func New(c Cache, m *mq.Client, ttlSec int) *Handler {
	return &Handler{cache: c, mq: m, ttlSec: ttlSec}
}

// NewWithCache is an alias for New — used in tests.
func NewWithCache(c Cache, m *mq.Client, ttlSec int) *Handler { return New(c, m, ttlSec) }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/shorten", h.shorten)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", h.redirect)
}

func (h *Handler) shorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
		http.Error(w, "invalid body: need {\"url\":\"...\"}", http.StatusBadRequest)
		return
	}
	code, err := shortener.Code()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := h.cache.Set(code, body.URL, h.ttlSec); err != nil {
		log.Printf("cache set: %v", err)
		http.Error(w, "cache unavailable", http.StatusServiceUnavailable)
		return
	}
	if h.mq != nil {
		h.mq.Publish("url.created", code+":"+body.URL)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"short": code, "url": body.URL})
}

func (h *Handler) redirect(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/")
	if code == "" {
		http.Error(w, "usage: POST /shorten", http.StatusBadRequest)
		return
	}
	url, err := h.cache.Get(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if h.mq != nil {
		h.mq.Publish("url.clicked", code)
	}
	http.Redirect(w, r, url, http.StatusMovedPermanently)
}
