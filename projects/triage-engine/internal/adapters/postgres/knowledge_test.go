//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"tour_of_go/projects/triage-engine/internal/adapters/postgres"
	"tour_of_go/projects/triage-engine/internal/domain"
)

// mockEmbedder returns a fixed 1536-dim vector for any input.
type mockEmbedder struct{}

func (m *mockEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	v := make([]float32, 1536)
	for i := range v {
		v[i] = 0.01
	}
	return v, nil
}
func (m *mockEmbedder) Categorize(_ context.Context, _ domain.TicketData) (string, error) {
	return "", nil
}
func (m *mockEmbedder) DraftResponse(_ context.Context, _ domain.TicketData, _ []string, _ string) (string, error) {
	return "", nil
}

func TestKnowledgeRepo_IndexAndSearch(t *testing.T) {
	pool := setupContainer(t)
	repo := postgres.NewKnowledgeRepo(pool, &mockEmbedder{})
	ctx := context.Background()

	if err := repo.Index(ctx, "doc-1", "Restart the Jenkins build by clicking Rebuild."); err != nil {
		t.Fatal(err)
	}

	chunks, err := repo.Search(ctx, "build stuck", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("want 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != "Restart the Jenkins build by clicking Rebuild." {
		t.Fatalf("unexpected chunk: %s", chunks[0])
	}
}

func TestKnowledgeRepo_SearchEmpty(t *testing.T) {
	pool := setupContainer(t)
	repo := postgres.NewKnowledgeRepo(pool, &mockEmbedder{})

	chunks, err := repo.Search(context.Background(), "anything", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 0 {
		t.Fatalf("want 0 chunks, got %d", len(chunks))
	}
}

func TestKnowledgeRepo_IndexIdempotent(t *testing.T) {
	pool := setupContainer(t)
	repo := postgres.NewKnowledgeRepo(pool, &mockEmbedder{})
	ctx := context.Background()

	if err := repo.Index(ctx, "doc-2", "First version."); err != nil {
		t.Fatal(err)
	}
	// Re-index same ID with updated content — should not error.
	if err := repo.Index(ctx, "doc-2", "Updated version."); err != nil {
		t.Fatalf("re-index should not error: %v", err)
	}

	chunks, _ := repo.Search(ctx, "version", 3)
	if len(chunks) != 1 {
		t.Fatalf("want 1 chunk after upsert, got %d", len(chunks))
	}
}
