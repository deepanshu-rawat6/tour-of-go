package graph

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tour_of_go/projects/triage-engine/internal/domain"
	"tour_of_go/projects/triage-engine/internal/ports"
)

type TriageEngine struct {
	checkpointer ports.StateCheckpointer
	kb           ports.KnowledgeBase
	diag         ports.DiagnosticTools
	llm          ports.LLMClient
	notifier     ports.Notifier
}

func NewTriageEngine(
	checkpointer ports.StateCheckpointer,
	kb ports.KnowledgeBase,
	diag ports.DiagnosticTools,
	llm ports.LLMClient,
	notifier ports.Notifier,
) *TriageEngine {
	return &TriageEngine{checkpointer, kb, diag, llm, notifier}
}

func (e *TriageEngine) Start(ctx context.Context, ticket domain.TicketData) (*domain.InvestigationState, error) {
	now := time.Now()
	state := &domain.InvestigationState{
		TicketID:       ticket.ID,
		Status:         domain.StatusCategorizing,
		TicketData:     ticket,
		ApprovalStatus: domain.ApprovalPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := e.categorize(ctx, state); err != nil {
		return nil, fmt.Errorf("categorize: %w", err)
	}
	if err := e.retrieveRunbook(ctx, state); err != nil {
		return nil, fmt.Errorf("retrieveRunbook: %w", err)
	}
	if err := e.executeDiagnostic(ctx, state); err != nil {
		return nil, fmt.Errorf("executeDiagnostic: %w", err)
	}
	if err := e.draftResolution(ctx, state); err != nil {
		return nil, fmt.Errorf("draftResolution: %w", err)
	}
	if err := e.awaitHuman(ctx, state); err != nil {
		return nil, fmt.Errorf("awaitHuman: %w", err)
	}
	return state, nil
}

func (e *TriageEngine) Resume(ctx context.Context, ticketID string, approved bool) (*domain.InvestigationState, error) {
	state, err := e.checkpointer.Load(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, nil
	}
	if state.Status != domain.StatusAwaitingHuman {
		return nil, errors.New("ticket is not awaiting human approval")
	}

	state.UpdatedAt = time.Now()
	if !approved {
		state.ApprovalStatus = domain.ApprovalRejected
		state.Status = domain.StatusRejected
		state.CurrentNode = "rejected"
		return state, e.checkpointer.Save(ctx, state)
	}

	state.ApprovalStatus = domain.ApprovalApproved
	state.Status = domain.StatusExecuting
	state.CurrentNode = "execute_action"
	// executeAction: in a real system this would post the drafted response to the ticket.
	// Here we mark it complete — the action is the drafted response itself.
	state.Status = domain.StatusCompleted
	state.CurrentNode = "completed"
	return state, e.checkpointer.Save(ctx, state)
}

func (e *TriageEngine) categorize(ctx context.Context, state *domain.InvestigationState) error {
	state.CurrentNode = "categorize"
	state.Status = domain.StatusCategorizing
	category, err := e.llm.Categorize(ctx, state.TicketData)
	if err != nil {
		return err
	}
	state.Category = category
	return nil
}

func (e *TriageEngine) retrieveRunbook(ctx context.Context, state *domain.InvestigationState) error {
	state.CurrentNode = "retrieve_runbook"
	state.Status = domain.StatusRetrieving
	chunks, err := e.kb.Search(ctx, state.TicketData.Summary+" "+state.TicketData.Description, 3)
	if err != nil {
		return err
	}
	state.RunbookChunks = chunks
	return nil
}

func (e *TriageEngine) executeDiagnostic(ctx context.Context, state *domain.InvestigationState) error {
	state.CurrentNode = "execute_diagnostic"
	state.Status = domain.StatusDiagnosing
	result, err := e.diag.CheckBuildStatus(ctx, state.TicketData.Reporter)
	if err != nil {
		return err
	}
	state.DiagnosticResult = result
	return nil
}

func (e *TriageEngine) draftResolution(ctx context.Context, state *domain.InvestigationState) error {
	state.CurrentNode = "draft_resolution"
	state.Status = domain.StatusDrafting
	draft, err := e.llm.DraftResponse(ctx, state.TicketData, state.RunbookChunks, state.DiagnosticResult)
	if err != nil {
		return err
	}
	state.DraftedResponse = draft
	return nil
}

func (e *TriageEngine) awaitHuman(ctx context.Context, state *domain.InvestigationState) error {
	state.CurrentNode = "await_human"
	state.Status = domain.StatusAwaitingHuman
	if err := e.checkpointer.Save(ctx, state); err != nil {
		return err
	}
	return e.notifier.SendApprovalRequest(ctx, state)
}
