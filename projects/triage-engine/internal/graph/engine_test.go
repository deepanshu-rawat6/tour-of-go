package graph_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"tour_of_go/projects/triage-engine/internal/domain"
	"tour_of_go/projects/triage-engine/internal/graph"
)

// --- mocks ---

type mockCheckpointer struct {
	saved  *domain.InvestigationState
	stored map[string]*domain.InvestigationState
}

func newMockCheckpointer() *mockCheckpointer {
	return &mockCheckpointer{stored: make(map[string]*domain.InvestigationState)}
}

func (m *mockCheckpointer) Save(_ context.Context, s *domain.InvestigationState) error {
	m.saved = s
	m.stored[s.TicketID] = s
	return nil
}
func (m *mockCheckpointer) Load(_ context.Context, id string) (*domain.InvestigationState, error) {
	return m.stored[id], nil
}

type mockKB struct{ called bool }

func (m *mockKB) Search(_ context.Context, _ string, _ int) ([]string, error) {
	m.called = true
	return []string{"runbook chunk 1"}, nil
}
func (m *mockKB) Index(_ context.Context, _, _ string) error { return nil }

type mockDiag struct{ called bool }

func (m *mockDiag) CheckBuildStatus(_ context.Context, _ string) (string, error) {
	m.called = true
	return "build #42: SUCCESS", nil
}

type mockLLM struct {
	categorizeCalled bool
	draftCalled      bool
	categorizeErr    error
}

func (m *mockLLM) Categorize(_ context.Context, _ domain.TicketData) (string, error) {
	m.categorizeCalled = true
	return "build_failure", m.categorizeErr
}
func (m *mockLLM) DraftResponse(_ context.Context, _ domain.TicketData, _ []string, _ string) (string, error) {
	m.draftCalled = true
	return "Please restart the build.", nil
}
func (m *mockLLM) Embed(_ context.Context, _ string) ([]float32, error) { return nil, nil }

type mockNotifier struct{ called bool }

func (m *mockNotifier) SendApprovalRequest(_ context.Context, _ *domain.InvestigationState) error {
	m.called = true
	return nil
}

func newEngine(cp *mockCheckpointer, kb *mockKB, diag *mockDiag, llm *mockLLM, n *mockNotifier) *graph.TriageEngine {
	return graph.NewTriageEngine(cp, kb, diag, llm, n)
}

func testTicket() domain.TicketData {
	return domain.TicketData{ID: "T-1", Summary: "Build stuck", Reporter: "alice", CreatedAt: time.Now()}
}

// --- tests ---

func TestStart_RunsThroughAllNodes(t *testing.T) {
	cp := newMockCheckpointer()
	kb := &mockKB{}
	diag := &mockDiag{}
	llm := &mockLLM{}
	n := &mockNotifier{}
	engine := newEngine(cp, kb, diag, llm, n)

	state, err := engine.Start(context.Background(), testTicket())
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != domain.StatusAwaitingHuman {
		t.Fatalf("want %s, got %s", domain.StatusAwaitingHuman, state.Status)
	}
	if !llm.categorizeCalled {
		t.Fatal("categorize not called")
	}
	if !kb.called {
		t.Fatal("knowledge base not called")
	}
	if !diag.called {
		t.Fatal("diagnostic not called")
	}
	if !llm.draftCalled {
		t.Fatal("draft not called")
	}
	if !n.called {
		t.Fatal("notifier not called")
	}
	if cp.saved == nil {
		t.Fatal("state not saved")
	}
	if state.Category != "build_failure" {
		t.Fatalf("want build_failure, got %s", state.Category)
	}
	if state.DraftedResponse != "Please restart the build." {
		t.Fatalf("unexpected draft: %s", state.DraftedResponse)
	}
}

func TestResume_Approved(t *testing.T) {
	cp := newMockCheckpointer()
	engine := newEngine(cp, &mockKB{}, &mockDiag{}, &mockLLM{}, &mockNotifier{})

	state, _ := engine.Start(context.Background(), testTicket())
	// state is now saved at StatusAwaitingHuman

	final, err := engine.Resume(context.Background(), state.TicketID, true)
	if err != nil {
		t.Fatal(err)
	}
	if final.Status != domain.StatusCompleted {
		t.Fatalf("want completed, got %s", final.Status)
	}
	if final.ApprovalStatus != domain.ApprovalApproved {
		t.Fatalf("want approved, got %s", final.ApprovalStatus)
	}
}

func TestResume_Rejected(t *testing.T) {
	cp := newMockCheckpointer()
	engine := newEngine(cp, &mockKB{}, &mockDiag{}, &mockLLM{}, &mockNotifier{})

	state, _ := engine.Start(context.Background(), testTicket())

	final, err := engine.Resume(context.Background(), state.TicketID, false)
	if err != nil {
		t.Fatal(err)
	}
	if final.Status != domain.StatusRejected {
		t.Fatalf("want rejected, got %s", final.Status)
	}
	if final.ApprovalStatus != domain.ApprovalRejected {
		t.Fatalf("want rejected approval, got %s", final.ApprovalStatus)
	}
}

func TestResume_WrongStatus(t *testing.T) {
	cp := newMockCheckpointer()
	// Manually store a state that is NOT awaiting_human.
	cp.stored["T-2"] = &domain.InvestigationState{
		TicketID: "T-2",
		Status:   domain.StatusCompleted,
	}
	engine := newEngine(cp, &mockKB{}, &mockDiag{}, &mockLLM{}, &mockNotifier{})

	_, err := engine.Resume(context.Background(), "T-2", true)
	if err == nil {
		t.Fatal("expected error for wrong status")
	}
}

func TestStart_LLMError(t *testing.T) {
	cp := newMockCheckpointer()
	llm := &mockLLM{categorizeErr: errors.New("openai timeout")}
	engine := newEngine(cp, &mockKB{}, &mockDiag{}, llm, &mockNotifier{})

	_, err := engine.Start(context.Background(), testTicket())
	if err == nil {
		t.Fatal("expected error from LLM")
	}
	if cp.saved != nil {
		t.Fatal("state must not be saved on error")
	}
}
