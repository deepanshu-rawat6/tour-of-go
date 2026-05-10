//go:build integration

package graph_test

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	pgadapter "tour_of_go/projects/triage-engine/internal/adapters/postgres"
	"tour_of_go/projects/triage-engine/internal/adapters/diagnostic"
	"tour_of_go/projects/triage-engine/internal/adapters/notifier"
	"tour_of_go/projects/triage-engine/internal/domain"
	"tour_of_go/projects/triage-engine/internal/graph"
)

func setupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	migrationSQL, err := os.ReadFile("../../migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	ctr, err := tcpostgres.Run(ctx,
		"pgvector/pgvector:pg16",
		tcpostgres.WithDatabase("triage"),
		tcpostgres.WithUsername("triage"),
		tcpostgres.WithPassword("triage"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("docker unavailable: %v", err)
	}
	t.Cleanup(func() { ctr.Terminate(ctx) }) //nolint:errcheck

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, string(migrationSQL)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	return pool
}

// mockLLM returns deterministic responses — no real OpenAI calls.
type mockLLM struct{}

func (m *mockLLM) Categorize(_ context.Context, _ domain.TicketData) (string, error) {
	return "build_failure", nil
}
func (m *mockLLM) DraftResponse(_ context.Context, _ domain.TicketData, _ []string, _ string) (string, error) {
	return "Please retry the build pipeline.", nil
}
func (m *mockLLM) Embed(_ context.Context, _ string) ([]float32, error) {
	v := make([]float32, 1536)
	for i := range v {
		v[i] = 0.01
	}
	return v, nil
}

func buildEngine(t *testing.T, pool *pgxpool.Pool) *graph.TriageEngine {
	t.Helper()
	diagSrv := diagnostic.NewMockServer()
	t.Cleanup(diagSrv.Close)

	llm := &mockLLM{}
	checkpointer := pgadapter.NewCheckpointer(pool)
	kb := pgadapter.NewKnowledgeRepo(pool, llm)
	diag := diagnostic.NewClient(diagSrv.URL)
	n := notifier.NewLogNotifier(&bytes.Buffer{})

	return graph.NewTriageEngine(checkpointer, kb, diag, llm, n)
}

func testTicket(id string) domain.TicketData {
	return domain.TicketData{
		ID: id, Summary: "Build stuck", Description: "Pipeline failed on step 3",
		Reporter: "alice", Priority: "high", CreatedAt: time.Now(),
	}
}

func TestFullTriageFlow_Approved(t *testing.T) {
	pool := setupDB(t)
	engine := buildEngine(t, pool)
	ctx := context.Background()

	state, err := engine.Start(ctx, testTicket("T-INT-1"))
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != domain.StatusAwaitingHuman {
		t.Fatalf("want awaiting_human, got %s", state.Status)
	}
	if state.Category != "build_failure" {
		t.Fatalf("want build_failure, got %s", state.Category)
	}
	if state.DraftedResponse == "" {
		t.Fatal("expected drafted response")
	}

	// Verify state is persisted in DB.
	cp := pgadapter.NewCheckpointer(pool)
	persisted, err := cp.Load(ctx, "T-INT-1")
	if err != nil {
		t.Fatal(err)
	}
	if persisted == nil {
		t.Fatal("state not persisted in DB")
	}
	if persisted.Status != domain.StatusAwaitingHuman {
		t.Fatalf("persisted status: want awaiting_human, got %s", persisted.Status)
	}

	// Resume with approval.
	final, err := engine.Resume(ctx, "T-INT-1", true)
	if err != nil {
		t.Fatal(err)
	}
	if final.Status != domain.StatusCompleted {
		t.Fatalf("want completed, got %s", final.Status)
	}
	if final.ApprovalStatus != domain.ApprovalApproved {
		t.Fatalf("want approved, got %s", final.ApprovalStatus)
	}

	// Verify final state persisted.
	completed, _ := cp.Load(ctx, "T-INT-1")
	if completed.Status != domain.StatusCompleted {
		t.Fatalf("DB: want completed, got %s", completed.Status)
	}
}

func TestFullTriageFlow_Rejected(t *testing.T) {
	pool := setupDB(t)
	engine := buildEngine(t, pool)
	ctx := context.Background()

	state, err := engine.Start(ctx, testTicket("T-INT-2"))
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != domain.StatusAwaitingHuman {
		t.Fatalf("want awaiting_human, got %s", state.Status)
	}

	final, err := engine.Resume(ctx, "T-INT-2", false)
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

func TestConcurrentTickets(t *testing.T) {
	pool := setupDB(t)
	engine := buildEngine(t, pool)
	ctx := context.Background()

	// Two tickets processed concurrently — each should reach StatusAwaitingHuman independently.
	type result struct {
		state *domain.InvestigationState
		err   error
	}
	ch := make(chan result, 2)

	for _, id := range []string{"T-INT-3", "T-INT-4"} {
		go func(ticketID string) {
			s, e := engine.Start(ctx, testTicket(ticketID))
			ch <- result{s, e}
		}(id)
	}

	for i := 0; i < 2; i++ {
		r := <-ch
		if r.err != nil {
			t.Errorf("concurrent ticket error: %v", r.err)
			continue
		}
		if r.state.Status != domain.StatusAwaitingHuman {
			t.Errorf("want awaiting_human, got %s for ticket %s", r.state.Status, r.state.TicketID)
		}
	}
}
