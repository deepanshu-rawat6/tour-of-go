package notifier

import (
	"context"
	"fmt"
	"io"
	"os"

	"tour_of_go/projects/triage-engine/internal/domain"
)

type LogNotifier struct {
	w io.Writer
}

func NewLogNotifier(w io.Writer) *LogNotifier {
	if w == nil {
		w = os.Stdout
	}
	return &LogNotifier{w: w}
}

func (n *LogNotifier) SendApprovalRequest(_ context.Context, state *domain.InvestigationState) error {
	_, err := fmt.Fprintf(n.w,
		"[APPROVAL REQUIRED] ticket=%s category=%s\nDraft: %s\nTo approve: POST /graph/resume {\"ticket_id\":%q,\"approved\":true}\n",
		state.TicketID, state.Category, state.DraftedResponse, state.TicketID,
	)
	return err
}
