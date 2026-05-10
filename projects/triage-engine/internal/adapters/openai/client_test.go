package openai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tour_of_go/projects/triage-engine/internal/adapters/openai"
	"tour_of_go/projects/triage-engine/internal/domain"
)

func chatServer(reply string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": reply}},
			},
		})
	}))
}

func embedServer(vec []float32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"data": []map[string]any{{"embedding": vec}},
		})
	}))
}

func newTestClient(srv *httptest.Server) *openai.Client {
	c := openai.NewClient("test-key", "gpt-4o-mini", "text-embedding-3-small")
	c.SetBaseURL(srv.URL)
	return c
}

func TestCategorize(t *testing.T) {
	srv := chatServer("build_failure")
	defer srv.Close()

	client := newTestClient(srv)
	ticket := domain.TicketData{ID: "T-1", Summary: "Build stuck", Description: "Pipeline failed", CreatedAt: time.Now()}

	category, err := client.Categorize(context.Background(), ticket)
	if err != nil {
		t.Fatal(err)
	}
	if category != "build_failure" {
		t.Fatalf("want build_failure, got %s", category)
	}
}

func TestDraftResponse_IncludesRunbookContext(t *testing.T) {
	var capturedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
		msgs := body["messages"].([]any)
		capturedBody = msgs[0].(map[string]any)["content"].(string)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "Retry the build."}},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(srv)
	ticket := domain.TicketData{ID: "T-1", Summary: "Build stuck", CreatedAt: time.Now()}
	draft, err := client.DraftResponse(context.Background(), ticket, []string{"runbook step 1"}, "build #42: FAILED")
	if err != nil {
		t.Fatal(err)
	}
	if draft != "Retry the build." {
		t.Fatalf("unexpected draft: %s", draft)
	}
	if !strings.Contains(capturedBody, "runbook step 1") {
		t.Fatal("system prompt must include runbook context")
	}
	if !strings.Contains(capturedBody, "build #42: FAILED") {
		t.Fatal("system prompt must include diagnostic result")
	}
}

func TestEmbed(t *testing.T) {
	expected := []float32{0.1, 0.2, 0.3}
	srv := embedServer(expected)
	defer srv.Close()

	client := newTestClient(srv)
	vec, err := client.Embed(context.Background(), "some text")
	if err != nil {
		t.Fatal(err)
	}
	if len(vec) != 3 {
		t.Fatalf("want 3 dims, got %d", len(vec))
	}
	for i, v := range expected {
		if vec[i] != v {
			t.Fatalf("dim %d: want %f, got %f", i, v, vec[i])
		}
	}
}
