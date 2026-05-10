package ports

import (
	"context"

	"tour_of_go/projects/triage-engine/internal/domain"
)

type StateCheckpointer interface {
	Save(ctx context.Context, state *domain.InvestigationState) error
	Load(ctx context.Context, ticketID string) (*domain.InvestigationState, error)
}

type KnowledgeBase interface {
	Search(ctx context.Context, query string, topK int) ([]string, error)
	Index(ctx context.Context, docID, content string) error
}

type DiagnosticTools interface {
	CheckBuildStatus(ctx context.Context, userID string) (string, error)
}

type LLMClient interface {
	Categorize(ctx context.Context, ticket domain.TicketData) (string, error)
	DraftResponse(ctx context.Context, ticket domain.TicketData, runbook []string, diagnostic string) (string, error)
	Embed(ctx context.Context, text string) ([]float32, error)
}

type Notifier interface {
	SendApprovalRequest(ctx context.Context, state *domain.InvestigationState) error
}
