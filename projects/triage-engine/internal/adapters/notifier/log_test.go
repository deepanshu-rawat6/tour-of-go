package notifier_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"tour_of_go/projects/triage-engine/internal/adapters/notifier"
	"tour_of_go/projects/triage-engine/internal/domain"
)

func TestLogNotifier_SendApprovalRequest(t *testing.T) {
	var buf bytes.Buffer
	n := notifier.NewLogNotifier(&buf)

	state := &domain.InvestigationState{
		TicketID:        "T-42",
		Category:        "build_failure",
		DraftedResponse: "Please retry the build.",
	}

	if err := n.SendApprovalRequest(context.Background(), state); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "T-42") {
		t.Fatalf("output missing ticket ID: %s", out)
	}
	if !strings.Contains(out, "/graph/resume") {
		t.Fatalf("output missing approve instruction: %s", out)
	}
	if !strings.Contains(out, "build_failure") {
		t.Fatalf("output missing category: %s", out)
	}
}
