package domain

import "time"

// Status constants for InvestigationState.
const (
	StatusCategorizing  = "categorizing"
	StatusRetrieving    = "retrieving"
	StatusDiagnosing    = "diagnosing"
	StatusDrafting      = "drafting"
	StatusAwaitingHuman = "awaiting_human"
	StatusExecuting     = "executing"
	StatusCompleted     = "completed"
	StatusRejected      = "rejected"
)

// ApprovalStatus constants.
const (
	ApprovalPending  = "pending"
	ApprovalApproved = "approved"
	ApprovalRejected = "rejected"
)

type TicketData struct {
	ID          string
	Summary     string
	Description string
	Reporter    string
	Priority    string
	CreatedAt   time.Time
}

type Message struct {
	Role    string
	Content string
}

type InvestigationState struct {
	TicketID         string
	Status           string
	TicketData       TicketData
	Category         string
	RunbookChunks    []string
	DiagnosticResult string
	DraftedResponse  string
	Messages         []Message
	ApprovalStatus   string
	CurrentNode      string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
