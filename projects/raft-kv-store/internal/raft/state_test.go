package raft

import (
	"testing"

	"github.com/tour-of-go/raft-kv-store/internal/kv"
	"github.com/tour-of-go/raft-kv-store/internal/wal"
)

func newTestNode(t *testing.T) *Node {
	t.Helper()
	w, err := wal.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { w.Close() })
	n, err := NewNode("node1", []string{"node2", "node3"}, w, kv.New())
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestNode_InitialState(t *testing.T) {
	n := newTestNode(t)
	role, term, leader, _ := n.State()
	if role != Follower {
		t.Errorf("expected Follower, got %s", role)
	}
	if term != 0 {
		t.Errorf("expected term 0, got %d", term)
	}
	if leader != "" {
		t.Errorf("expected no leader, got %s", leader)
	}
}

func TestNode_BecomeFollower(t *testing.T) {
	n := newTestNode(t)
	n.mu.Lock()
	n.becomeFollower(5)
	n.mu.Unlock()
	_, term, _, _ := n.State()
	if term != 5 {
		t.Errorf("expected term 5, got %d", term)
	}
}

func TestNode_BecomeLeader(t *testing.T) {
	n := newTestNode(t)
	n.mu.Lock()
	n.currentTerm = 1
	n.becomeLeader()
	n.mu.Unlock()
	role, _, leader, _ := n.State()
	if role != Leader {
		t.Errorf("expected Leader, got %s", role)
	}
	if leader != "node1" {
		t.Errorf("expected leader node1, got %s", leader)
	}
}

func TestNode_WALRecovery(t *testing.T) {
	dir := t.TempDir()
	w, _ := wal.Open(dir)
	w.Append(wal.Entry{Term: 1, Index: 1, Command: []byte(`{"op":"put","key":"x","value":"42"}`)})
	w.Append(wal.Entry{Term: 1, Index: 2, Command: []byte(`{"op":"put","key":"y","value":"99"}`)})
	w.Close()

	w2, _ := wal.Open(dir)
	defer w2.Close()
	n, err := NewNode("node1", nil, w2, kv.New())
	if err != nil {
		t.Fatal(err)
	}
	if n.lastLogIndex() != 2 {
		t.Errorf("expected last log index 2, got %d", n.lastLogIndex())
	}
	v, ok := n.kv.Get("x")
	if !ok || v != "42" {
		t.Errorf("expected x=42 after recovery, got %q %v", v, ok)
	}
}
