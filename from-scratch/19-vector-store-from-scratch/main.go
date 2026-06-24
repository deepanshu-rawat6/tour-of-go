// FS-19: Vector Store from Scratch
// An in-memory vector store with HTTP API, metadata filtering, and disk persistence.
// Builds on FS-18 (embeddings). This is what pgvector/Pinecone do under the hood.
// Run: export OPENAI_API_KEY=sk-... && go run .
package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

// Record is one stored item: text + its embedding + arbitrary metadata.
type Record struct {
	ID       string            `json:"id"`
	Text     string            `json:"text"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Vector   []float32         `json:"-"` // not in JSON responses
}

// Store holds all records in memory, protected by a RWMutex.
type Store struct {
	mu      sync.RWMutex
	records []Record
	client  *openai.Client
}

func NewStore(client *openai.Client) *Store { return &Store{client: client} }

func main() {
	client := openai.NewClient(mustEnv("OPENAI_API_KEY"))
	store := NewStore(client)

	// Optionally load from disk on startup
	if path := os.Getenv("STORE_PATH"); path != "" {
		if err := store.load(path); err != nil {
			fmt.Fprintf(os.Stderr, "load store: %v\n", err)
		} else {
			fmt.Printf("Loaded %d records from %s\n", len(store.records), path)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /insert", store.handleInsert)   // add a record
	mux.HandleFunc("GET /search", store.handleSearch)     // semantic search
	mux.HandleFunc("DELETE /delete", store.handleDelete)  // remove by ID
	mux.HandleFunc("GET /list", store.handleList)         // list all IDs
	mux.HandleFunc("POST /save", store.handleSave)        // persist to disk
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]int{"records": len(store.records)})
	})

	fmt.Printf("Vector store :8082 (%d records)\n", len(store.records))
	http.ListenAndServe(":8082", mux)
}

// POST /insert  {"id":"doc1","text":"...","metadata":{"source":"k8s"}}
func (s *Store) handleInsert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       string            `json:"id"`
		Text     string            `json:"text"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
		http.Error(w, "need id + text", 400)
		return
	}

	vec, err := s.embed(r.Context(), req.Text)
	if err != nil {
		http.Error(w, "embed: "+err.Error(), 500)
		return
	}

	s.mu.Lock()
	// Remove existing record with same ID (upsert)
	s.records = removeByID(s.records, req.ID)
	s.records = append(s.records, Record{
		ID: req.ID, Text: req.Text, Metadata: req.Metadata, Vector: vec,
	})
	s.mu.Unlock()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": req.ID})
}

// GET /search?q=query&k=5&source=k8s
// Optional metadata filter: any query param that isn't q/k is treated as metadata filter.
func (s *Store) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "missing ?q=", 400)
		return
	}
	k, _ := strconv.Atoi(r.URL.Query().Get("k"))
	if k <= 0 {
		k = 5
	}

	// Build metadata filter from remaining query params
	filter := map[string]string{}
	for key, vals := range r.URL.Query() {
		if key != "q" && key != "k" {
			filter[key] = vals[0]
		}
	}

	queryVec, err := s.embed(r.Context(), q)
	if err != nil {
		http.Error(w, "embed: "+err.Error(), 500)
		return
	}

	type hit struct {
		Record
		Score float64 `json:"score"`
	}

	s.mu.RLock()
	hits := make([]hit, 0, len(s.records))
	for _, rec := range s.records {
		// Metadata filter: all filter keys must match
		if !matchesFilter(rec.Metadata, filter) {
			continue
		}
		hits = append(hits, hit{
			Record: rec,
			Score:  cosineSimilarity(queryVec, rec.Vector),
		})
	}
	s.mu.RUnlock()

	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > k {
		hits = hits[:k]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hits)
}

// DELETE /delete?id=doc1
func (s *Store) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	s.mu.Lock()
	before := len(s.records)
	s.records = removeByID(s.records, id)
	deleted := before - len(s.records)
	s.mu.Unlock()
	json.NewEncoder(w).Encode(map[string]int{"deleted": deleted})
}

// GET /list
func (s *Store) handleList(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	ids := make([]string, len(s.records))
	for i, rec := range s.records {
		ids[i] = rec.ID
	}
	s.mu.RUnlock()
	json.NewEncoder(w).Encode(ids)
}

// POST /save?path=./store.bin
func (s *Store) handleSave(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "store.bin"
	}
	if err := s.save(path); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"saved": path})
}

// ── Persistence: binary format ────────────────────────────────────────────
// Format per record: [id_len:4][id][text_len:4][text][vec_len:4][float32s...][meta_json_len:4][meta_json]

func (s *Store) save(path string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, rec := range s.records {
		writeStr(f, rec.ID)
		writeStr(f, rec.Text)
		writeVec(f, rec.Vector)
		metaJSON, _ := json.Marshal(rec.Metadata)
		writeBytes(f, metaJSON)
	}
	return nil
}

func (s *Store) load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for {
		id, err := readStr(f)
		if err != nil {
			break
		}
		text, _ := readStr(f)
		vec, _ := readVec(f)
		metaJSON, _ := readBytes(f)
		var meta map[string]string
		json.Unmarshal(metaJSON, &meta)
		s.records = append(s.records, Record{ID: id, Text: text, Vector: vec, Metadata: meta})
	}
	return nil
}

// ── Embedding (same as FS-18) ─────────────────────────────────────────────

func (s *Store) embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := s.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.SmallEmbedding3,
	})
	if err != nil {
		return nil, err
	}
	f32 := make([]float32, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		f32[i] = float32(v)
	}
	return f32, nil
}

// ── Cosine similarity (same as FS-18) ─────────────────────────────────────

func cosineSimilarity(a, b []float32) float64 {
	var dot, magA, magB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		magA += float64(a[i]) * float64(a[i])
		magB += float64(b[i]) * float64(b[i])
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

// ── Helpers ───────────────────────────────────────────────────────────────

func removeByID(recs []Record, id string) []Record {
	out := recs[:0]
	for _, r := range recs {
		if r.ID != id {
			out = append(out, r)
		}
	}
	return out
}

func matchesFilter(meta, filter map[string]string) bool {
	for k, v := range filter {
		if meta[k] != v {
			return false
		}
	}
	return true
}

func writeStr(f *os.File, s string) {
	b := []byte(s)
	binary.Write(f, binary.LittleEndian, uint32(len(b)))
	f.Write(b)
}

func writeVec(f *os.File, v []float32) {
	binary.Write(f, binary.LittleEndian, uint32(len(v)))
	binary.Write(f, binary.LittleEndian, v)
}

func writeBytes(f *os.File, b []byte) {
	binary.Write(f, binary.LittleEndian, uint32(len(b)))
	f.Write(b)
}

func readStr(f *os.File) (string, error) {
	b, err := readBytes(f)
	return string(b), err
}

func readVec(f *os.File) ([]float32, error) {
	var n uint32
	if err := binary.Read(f, binary.LittleEndian, &n); err != nil {
		return nil, err
	}
	v := make([]float32, n)
	binary.Read(f, binary.LittleEndian, v)
	return v, nil
}

func readBytes(f *os.File) ([]byte, error) {
	var n uint32
	if err := binary.Read(f, binary.LittleEndian, &n); err != nil {
		return nil, err
	}
	b := make([]byte, n)
	f.Read(b)
	return b, nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintln(os.Stderr, key+" not set")
		os.Exit(1)
	}
	return v
}
