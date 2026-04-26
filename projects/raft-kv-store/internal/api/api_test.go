package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tour-of-go/raft-kv-store/internal/kv"
	"github.com/tour-of-go/raft-kv-store/internal/raft"
)

type mockNode struct {
	role     raft.Role
	leaderID string
}

func (m *mockNode) Propose(_ context.Context, _ []byte, _ raft.Transport) (string, error) {
	return "ok", nil
}
func (m *mockNode) State() (raft.Role, uint64, string, uint64) {
	return m.role, 1, m.leaderID, 0
}
func (m *mockNode) ID() string { return "node1" }

func TestAPI_GetNotFound(t *testing.T) {
	store := kv.New()
	h := New(&mockNode{role: raft.Leader}, store, nil, nil)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/kv/missing", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_PutAndGet_Leader(t *testing.T) {
	store := kv.New()
	store.Apply([]byte(`{"op":"put","key":"foo","value":"bar"}`))
	h := New(&mockNode{role: raft.Leader}, store, nil, nil)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/kv/foo", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "bar" {
		t.Errorf("expected bar, got %s", w.Body.String())
	}
}

func TestAPI_WriteOnFollower_Redirects(t *testing.T) {
	store := kv.New()
	peers := map[string]string{"leader1": "localhost:8001"}
	h := New(&mockNode{role: raft.Follower, leaderID: "leader1"}, store, nil, peers)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPut, "/kv/foo", strings.NewReader("bar"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected 307, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "localhost:8001") {
		t.Errorf("expected redirect to leader, got %s", w.Header().Get("Location"))
	}
}

func TestAPI_Status(t *testing.T) {
	store := kv.New()
	h := New(&mockNode{role: raft.Leader}, store, nil, nil)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Leader") {
		t.Errorf("expected Leader in status: %s", w.Body.String())
	}
}
