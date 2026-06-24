package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"
	openai "github.com/sashabaranov/go-openai"
)

const (
	chunkSize    = 512  // characters per chunk (approximate tokens)
	chunkOverlap = 50   // overlap between consecutive chunks
	topK         = 5    // number of chunks to retrieve
	minSimilarity = 0.7 // minimum cosine similarity threshold
)

// DB schema (run once):
// CREATE EXTENSION IF NOT EXISTS vector;
// CREATE TABLE IF NOT EXISTS chunks (
//   id SERIAL PRIMARY KEY,
//   source TEXT, chunk_index INT,
//   content TEXT, embedding vector(1536), created_at TIMESTAMPTZ DEFAULT now()
// );
// CREATE INDEX IF NOT EXISTS chunks_embedding_idx ON chunks
//   USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

var (
	db     *pgxpool.Pool
	ai     *openai.Client
)

func main() {
	ctx := context.Background()

	// Init DB
	var err error
	db, err = pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// Init OpenAI client
	ai = openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	// Routes
	http.HandleFunc("POST /index", handleIndex)
	http.HandleFunc("GET /query", handleQuery)
	http.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	slog.Info("RAG pipeline listening", "addr", ":8080")
	http.ListenAndServe(":8080", nil)
}

// ─── Indexing ──────────────────────────────────────────────────────────────

func handleIndex(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Text string `json:"text"` // inline text alternative
	}
	json.NewDecoder(r.Body).Decode(&req)

	var docs []struct{ source, content string }

	if req.Text != "" {
		docs = append(docs, struct{ source, content string }{"inline", req.Text})
	} else {
		// Walk directory
		err := filepath.WalkDir(req.Path, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() { return err }
			if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".txt") {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil { return err }
			docs = append(docs, struct{ source, content string }{path, string(b)})
			return nil
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	total := 0
	for _, doc := range docs {
		chunks := chunk(doc.content, chunkSize, chunkOverlap)
		for i, c := range chunks {
			if err := indexChunk(r.Context(), doc.source, i, c); err != nil {
				slog.Error("index chunk failed", "err", err)
				continue
			}
			total++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"indexed_chunks": total})
}

// chunk splits text into overlapping windows by character count.
func chunk(text string, size, overlap int) []string {
	var chunks []string
	runes := []rune(text)
	for start := 0; start < len(runes); start += size - overlap {
		end := start + size
		if end > len(runes) { end = len(runes) }
		chunks = append(chunks, string(runes[start:end]))
		if end == len(runes) { break }
	}
	return chunks
}

func indexChunk(ctx context.Context, source string, idx int, content string) error {
	// Skip empty / too-short chunks
	if utf8.RuneCountInString(strings.TrimSpace(content)) < 20 {
		return nil
	}

	emb, err := embed(ctx, content)
	if err != nil { return fmt.Errorf("embed: %w", err) }

	_, err = db.Exec(ctx,
		`INSERT INTO chunks (source, chunk_index, content, embedding)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT DO NOTHING`,
		source, idx, content, pgvector.NewVector(emb),
	)
	return err
}

// ─── Query ─────────────────────────────────────────────────────────────────

func handleQuery(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "missing ?q=", 400)
		return
	}

	// 1. Embed the query
	qEmb, err := embed(r.Context(), q)
	if err != nil {
		http.Error(w, "embed failed: "+err.Error(), 500)
		return
	}

	// 2. Vector search — top K chunks above similarity threshold
	rows, err := db.Query(r.Context(), `
		SELECT source, content, 1 - (embedding <=> $1) AS similarity
		FROM chunks
		WHERE 1 - (embedding <=> $1) > $2
		ORDER BY embedding <=> $1
		LIMIT $3
	`, pgvector.NewVector(qEmb), minSimilarity, topK)
	if err != nil {
		http.Error(w, "search failed: "+err.Error(), 500)
		return
	}
	defer rows.Close()

	type result struct {
		Source     string  `json:"source"`
		Content    string  `json:"content"`
		Similarity float64 `json:"similarity"`
	}
	var results []result
	for rows.Next() {
		var r result
		rows.Scan(&r.Source, &r.Content, &r.Similarity)
		results = append(results, r)
	}

	if len(results) == 0 {
		http.Error(w, "no relevant context found", 404)
		return
	}

	// 3. Build prompt
	var ctxBuilder strings.Builder
	for i, r := range results {
		fmt.Fprintf(&ctxBuilder, "[%d] Source: %s\n%s\n\n", i+1, r.Source, r.Content)
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: `You are a helpful assistant. Answer the question based ONLY on the provided context.
If the context doesn't contain enough information, say "I don't have enough information to answer that."
Do not make up facts.`,
		},
		{
			Role: openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("Context:\n---\n%s---\n\nQuestion: %s\n\nAnswer:",
				ctxBuilder.String(), q),
		},
	}

	// 4. Stream LLM response via SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher := w.(http.Flusher)

	// Send retrieved sources first
	sources, _ := json.Marshal(results)
	fmt.Fprintf(w, "event: sources\ndata: %s\n\n", sources)
	flusher.Flush()

	// Stream LLM answer
	stream, err := ai.CreateChatCompletionStream(r.Context(), openai.ChatCompletionRequest{
		Model:    openai.GPT4oMini,
		Messages: messages,
		Stream:   true,
	})
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		return
	}
	defer stream.Close()

	for {
		resp, err := stream.Recv()
		if err == io.EOF { break }
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
			break
		}
		token := resp.Choices[0].Delta.Content
		if token == "" { continue }
		// SSE: data field, double newline terminates event
		tokenJSON, _ := json.Marshal(token)
		fmt.Fprintf(w, "data: %s\n\n", tokenJSON)
		flusher.Flush()
	}

	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}

// ─── Embedding ─────────────────────────────────────────────────────────────

func embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := ai.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.SmallEmbedding3,
	})
	if err != nil { return nil, err }
	emb := resp.Data[0].Embedding
	f32 := make([]float32, len(emb))
	for i, v := range emb { f32[i] = float32(v) }
	return f32, nil
}

// ensure pgx is used for type assertions
var _ pgx.Rows = (pgx.Rows)(nil)
