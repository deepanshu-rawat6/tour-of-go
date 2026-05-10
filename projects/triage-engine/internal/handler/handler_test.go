package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tour_of_go/projects/triage-engine/internal/domain"
	"tour_of_go/projects/triage-engine/internal/handler"
)

type mockEngine struct {
	startState  *domain.InvestigationState
	resumeState *domain.InvestigationState
	startErr    error
	resumeErr   error
}

func (m *mockEngine) Start(_ context.Context, ticket domain.TicketData) (*domain.InvestigationState, error) {
	if m.startErr != nil {
		return nil, m.startErr
	}
	return m.startState, nil
}

func (m *mockEngine) Resume(_ context.Context, _ string, _ bool) (*domain.InvestigationState, error) {
	return m.resumeState, m.resumeErr
}

func TestWebhookTicket_Success(t *testing.T) {
	eng := &mockEngine{
		startState: &domain.InvestigationState{
			TicketID: "T-1", Status: domain.StatusAwaitingHuman,
		},
	}
	h := handler.WebhookTicket(eng)

	body, _ := json.Marshal(map[string]any{"ID": "T-1", "Summary": "Build stuck"})
	req := httptest.NewRequest(http.MethodPost, "/webhooks/ticket", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("want 202, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp) //nolint:errcheck
	if resp["ticket_id"] != "T-1" {
		t.Fatalf("want T-1, got %s", resp["ticket_id"])
	}
	if resp["status"] != domain.StatusAwaitingHuman {
		t.Fatalf("want awaiting_human, got %s", resp["status"])
	}
}

func TestResumeGraph_Approved(t *testing.T) {
	eng := &mockEngine{
		resumeState: &domain.InvestigationState{
			TicketID: "T-1", Status: domain.StatusCompleted, DraftedResponse: "Retry the build.",
		},
	}
	h := handler.ResumeGraph(eng)

	body, _ := json.Marshal(map[string]any{"ticket_id": "T-1", "approved": true})
	req := httptest.NewRequest(http.MethodPost, "/graph/resume", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp) //nolint:errcheck
	if resp["status"] != domain.StatusCompleted {
		t.Fatalf("want completed, got %s", resp["status"])
	}
}

func TestResumeGraph_NotFound(t *testing.T) {
	eng := &mockEngine{resumeState: nil}
	h := handler.ResumeGraph(eng)

	body, _ := json.Marshal(map[string]any{"ticket_id": "unknown", "approved": true})
	req := httptest.NewRequest(http.MethodPost, "/graph/resume", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rr.Code)
	}
}
