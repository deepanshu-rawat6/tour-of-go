package domain_test

import (
	"testing"
	"time"

	"tour_of_go/projects/triage-engine/internal/domain"
)

func TestStatusConstants(t *testing.T) {
	statuses := []string{
		domain.StatusCategorizing, domain.StatusRetrieving, domain.StatusDiagnosing,
		domain.StatusDrafting, domain.StatusAwaitingHuman, domain.StatusExecuting,
		domain.StatusCompleted, domain.StatusRejected,
	}
	for _, s := range statuses {
		if s == "" {
			t.Fatal("empty status constant")
		}
	}
	if domain.ApprovalPending == "" || domain.ApprovalApproved == "" || domain.ApprovalRejected == "" {
		t.Fatal("empty approval constant")
	}
}

func TestStructInstantiation(t *testing.T) {
	ticket := domain.TicketData{ID: "T-1", Summary: "Build stuck", Reporter: "alice", CreatedAt: time.Now()}
	state := domain.InvestigationState{
		TicketID:       ticket.ID,
		Status:         domain.StatusCategorizing,
		TicketData:     ticket,
		ApprovalStatus: domain.ApprovalPending,
		Messages:       []domain.Message{{Role: "user", Content: "help"}},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if state.TicketID != "T-1" {
		t.Fatalf("want T-1, got %s", state.TicketID)
	}
	if len(state.Messages) != 1 {
		t.Fatal("expected 1 message")
	}
}
