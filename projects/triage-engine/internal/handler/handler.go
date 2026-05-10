package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"tour_of_go/projects/triage-engine/internal/domain"
)

// Engine is the interface the handlers depend on — satisfied by *graph.TriageEngine.
type Engine interface {
	Start(ctx context.Context, ticket domain.TicketData) (*domain.InvestigationState, error)
	Resume(ctx context.Context, ticketID string, approved bool) (*domain.InvestigationState, error)
}

type webhookRequest struct {
	domain.TicketData
}

type webhookResponse struct {
	TicketID string `json:"ticket_id"`
	Status   string `json:"status"`
}

func WebhookTicket(engine Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req webhookRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		state, err := engine.Start(r.Context(), req.TicketData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(webhookResponse{TicketID: state.TicketID, Status: state.Status}) //nolint:errcheck
	}
}

type resumeRequest struct {
	TicketID string `json:"ticket_id"`
	Approved bool   `json:"approved"`
}

type resumeResponse struct {
	TicketID        string `json:"ticket_id"`
	Status          string `json:"status"`
	DraftedResponse string `json:"drafted_response"`
}

func ResumeGraph(engine Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req resumeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		state, err := engine.Resume(r.Context(), req.TicketID, req.Approved)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if state == nil {
			http.Error(w, "ticket not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resumeResponse{ //nolint:errcheck
			TicketID:        state.TicketID,
			Status:          state.Status,
			DraftedResponse: state.DraftedResponse,
		})
	}
}
