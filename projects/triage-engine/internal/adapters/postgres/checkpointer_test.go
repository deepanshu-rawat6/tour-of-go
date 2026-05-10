//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"tour_of_go/projects/triage-engine/internal/adapters/postgres"
	"tour_of_go/projects/triage-engine/internal/domain"
)

func TestCheckpointer_SaveLoad(t *testing.T) {
	pool := setupContainer(t)
	cp := postgres.NewCheckpointer(pool)
	ctx := context.Background()

	state := &domain.InvestigationState{
		TicketID:        "T-100",
		Status:          domain.StatusAwaitingHuman,
		Category:        "build_failure",
		DraftedResponse: "Please retry the build.",
		ApprovalStatus:  domain.ApprovalPending,
		RunbookChunks:   []string{"chunk1", "chunk2"},
		Messages:        []domain.Message{{Role: "user", Content: "help"}},
		CreatedAt:       time.Now().Truncate(time.Millisecond),
		UpdatedAt:       time.Now().Truncate(time.Millisecond),
	}

	if err := cp.Save(ctx, state); err != nil {
		t.Fatal(err)
	}

	loaded, err := cp.Load(ctx, "T-100")
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil {
		t.Fatal("expected state, got nil")
	}
	if loaded.Status != domain.StatusAwaitingHuman {
		t.Fatalf("want %s, got %s", domain.StatusAwaitingHuman, loaded.Status)
	}
	if loaded.Category != "build_failure" {
		t.Fatalf("category mismatch: %s", loaded.Category)
	}
	if len(loaded.RunbookChunks) != 2 {
		t.Fatalf("want 2 chunks, got %d", len(loaded.RunbookChunks))
	}
}

func TestCheckpointer_LoadMissing(t *testing.T) {
	pool := setupContainer(t)
	cp := postgres.NewCheckpointer(pool)

	got, err := cp.Load(context.Background(), "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected nil for missing ticket")
	}
}

func TestCheckpointer_SaveUpdates(t *testing.T) {
	pool := setupContainer(t)
	cp := postgres.NewCheckpointer(pool)
	ctx := context.Background()

	state := &domain.InvestigationState{
		TicketID: "T-101", Status: domain.StatusAwaitingHuman,
		ApprovalStatus: domain.ApprovalPending,
	}
	cp.Save(ctx, state) //nolint:errcheck

	state.Status = domain.StatusCompleted
	state.ApprovalStatus = domain.ApprovalApproved
	if err := cp.Save(ctx, state); err != nil {
		t.Fatal(err)
	}

	loaded, _ := cp.Load(ctx, "T-101")
	if loaded.Status != domain.StatusCompleted {
		t.Fatalf("want completed, got %s", loaded.Status)
	}
}
