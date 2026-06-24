// FS-18: Embeddings from Scratch
// Demonstrates: what is an embedding, cosine similarity math, semantic search.
// Run: export OPENAI_API_KEY=sk-... && go run .
package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"sort"

	openai "github.com/sashabaranov/go-openai"
)

// A small corpus — in real life this would be your docs/KB
var corpus = []string{
	"Kubernetes pods are the smallest deployable units",
	"A Pod contains one or more containers sharing network and storage",
	"The kubectl apply command creates or updates resources from a YAML file",
	"HPA scales the number of pod replicas based on CPU or custom metrics",
	"StatefulSets provide stable pod names and persistent storage per pod",
	"Error handling in Go uses multiple return values",
	"The error interface has a single Error() string method",
	"Use errors.Is and errors.As for wrapped errors in Go",
	"Goroutines are lightweight threads managed by the Go runtime",
	"Channels provide type-safe communication between goroutines",
	"Redis is an in-memory data structure store used as a cache",
	"PostgreSQL supports MVCC for concurrent reads and writes",
	"Docker containers isolate processes using Linux namespaces and cgroups",
	"A Dockerfile defines how to build a container image layer by layer",
}

type Result struct {
	Text       string
	Similarity float64
}

func main() {
	client := openai.NewClient(mustEnv("OPENAI_API_KEY"))
	ctx := context.Background()

	// Step 1: Embed the entire corpus (once, upfront)
	fmt.Printf("Embedding %d documents...\n", len(corpus))
	corpusVecs, err := embedBatch(ctx, client, corpus)
	if err != nil {
		fmt.Fprintln(os.Stderr, "embed corpus:", err)
		os.Exit(1)
	}
	fmt.Println("Done. Enter a query to search semantically.\n")

	// Step 2: Interactive semantic search loop
	for {
		fmt.Print("Query (or 'quit'): ")
		var query string
		fmt.Scanln(&query)
		if query == "" || query == "quit" {
			break
		}

		// Embed the query with the same model
		queryVec, err := embedOne(ctx, client, query)
		if err != nil {
			fmt.Fprintln(os.Stderr, "embed query:", err)
			continue
		}

		// Compute cosine similarity between query and every corpus item
		results := make([]Result, len(corpus))
		for i, docVec := range corpusVecs {
			results[i] = Result{
				Text:       corpus[i],
				Similarity: cosineSimilarity(queryVec, docVec),
			}
		}

		// Sort by similarity descending
		sort.Slice(results, func(i, j int) bool {
			return results[i].Similarity > results[j].Similarity
		})

		// Print top 3
		fmt.Println("\nTop 3 results:")
		for _, r := range results[:3] {
			bar := progressBar(r.Similarity, 20)
			fmt.Printf("  %.3f %s  %s\n", r.Similarity, bar, r.Text)
		}
		fmt.Println()
	}
}

// ── Embedding functions ───────────────────────────────────────────────────

func embedOne(ctx context.Context, client *openai.Client, text string) ([]float32, error) {
	vecs, err := embedBatch(ctx, client, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func embedBatch(ctx context.Context, client *openai.Client, texts []string) ([][]float32, error) {
	resp, err := client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: texts,
		Model: openai.SmallEmbedding3, // text-embedding-3-small, 1536 dims
	})
	if err != nil {
		return nil, err
	}
	result := make([][]float32, len(texts))
	for _, d := range resp.Data {
		f32 := make([]float32, len(d.Embedding))
		for i, v := range d.Embedding {
			f32[i] = float32(v)
		}
		result[d.Index] = f32
	}
	return result, nil
}

// ── Cosine similarity — implemented from scratch ──────────────────────────

// cosineSimilarity = dot(A, B) / (|A| * |B|)
// Range: -1 (opposite) to 1 (identical). For text embeddings: 0 to 1.
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

func progressBar(v float64, width int) string {
	filled := int(v * float64(width))
	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return bar
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintln(os.Stderr, key+" not set")
		os.Exit(1)
	}
	return v
}
