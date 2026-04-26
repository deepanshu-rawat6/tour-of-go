package raft

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tour-of-go/raft-kv-store/internal/kv"
	"github.com/tour-of-go/raft-kv-store/internal/wal"
)

// mockTransport grants votes to all RequestVote calls.
type mockTransport struct {
	grantVote bool
	higherTerm uint64
}

func (m *mockTransport) RequestVote(_ context.Context, _ string, req *RequestVoteArgs) (*RequestVoteReply, error) {
	if m.higherTerm > 0 {
		return &RequestVoteReply{Term: m.higherTerm, VoteGranted: false}, nil
	}
	return &RequestVoteReply{Term: req.Term, VoteGranted: m.grantVote}, nil
}

func (m *mockTransport) AppendEntries(_ context.Context, _ string, _ *AppendEntriesArgs) (*AppendEntriesReply, error) {
	return nil, errors.New("not implemented")
}

func newNode(t *testing.T, id string, peers []string) *Node {
	t.Helper()
	w, _ := wal.Open(t.TempDir())
	t.Cleanup(func() { w.Close() })
	n, _ := NewNode(id, peers, w, kv.New())
	return n
}

func TestElection_WinsWithMajority(t *testing.T) {
	n := newNode(t, "n1", []string{"n2", "n3"})
	transport := &mockTransport{grantVote: true}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Force candidate state
	n.mu.Lock()
	n.role = Candidate
	n.mu.Unlock()

	go n.Run(ctx, transport, 50, 100, 20)

	// Wait for leader election
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		role, _, _, _ := n.State()
		if role == Leader {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("node did not become leader within 1s")
}

func TestElection_StepsDownOnHigherTerm(t *testing.T) {
	n := newNode(t, "n1", []string{"n2", "n3"})
	n.mu.Lock()
	n.currentTerm = 1
	n.becomeLeader()
	n.mu.Unlock()

	// Simulate receiving a higher term
	n.mu.Lock()
	n.becomeFollower(5)
	n.mu.Unlock()

	role, term, _, _ := n.State()
	if role != Follower {
		t.Errorf("expected Follower, got %s", role)
	}
	if term != 5 {
		t.Errorf("expected term 5, got %d", term)
	}
}
