package ports_test

import (
	"context"

	"tour_of_go/projects/triage-engine/internal/domain"
	"tour_of_go/projects/triage-engine/internal/ports"
)

var _ ports.StateCheckpointer = (*stubCheckpointer)(nil)
var _ ports.KnowledgeBase = (*stubKB)(nil)
var _ ports.DiagnosticTools = (*stubDiag)(nil)
var _ ports.LLMClient = (*stubLLM)(nil)
var _ ports.Notifier = (*stubNotifier)(nil)

type stubCheckpointer struct{}

func (s *stubCheckpointer) Save(_ context.Context, _ *domain.InvestigationState) error { return nil }
func (s *stubCheckpointer) Load(_ context.Context, _ string) (*domain.InvestigationState, error) {
	return nil, nil
}

type stubKB struct{}

func (s *stubKB) Search(_ context.Context, _ string, _ int) ([]string, error) { return nil, nil }
func (s *stubKB) Index(_ context.Context, _, _ string) error                  { return nil }

type stubDiag struct{}

func (s *stubDiag) CheckBuildStatus(_ context.Context, _ string) (string, error) { return "", nil }

type stubLLM struct{}

func (s *stubLLM) Categorize(_ context.Context, _ domain.TicketData) (string, error) {
	return "", nil
}
func (s *stubLLM) DraftResponse(_ context.Context, _ domain.TicketData, _ []string, _ string) (string, error) {
	return "", nil
}
func (s *stubLLM) Embed(_ context.Context, _ string) ([]float32, error) { return nil, nil }

type stubNotifier struct{}

func (s *stubNotifier) SendApprovalRequest(_ context.Context, _ *domain.InvestigationState) error {
	return nil
}
