package raft

import (
	"sync"

	"github.com/tour-of-go/raft-kv-store/internal/kv"
	"github.com/tour-of-go/raft-kv-store/internal/wal"
)

// Role represents the Raft node role.
type Role int

const (
	Follower Role = iota
	Candidate
	Leader
)

func (r Role) String() string {
	return [...]string{"Follower", "Candidate", "Leader"}[r]
}

// LogEntry is an in-memory log entry (mirrors wal.Entry).
type LogEntry struct {
	Term    uint64
	Index   uint64
	Command []byte
}

// Node holds all Raft state for a single node.
type Node struct {
	mu sync.Mutex

	// Identity
	id    string
	peers []string // peer node IDs

	// Persistent state (must survive restarts)
	currentTerm uint64
	votedFor    string
	log         []LogEntry // index 0 is a sentinel (term=0, index=0)

	// Volatile state
	role        Role
	commitIndex uint64
	lastApplied uint64
	leaderID    string

	// Leader-only volatile state
	nextIndex  map[string]uint64
	matchIndex map[string]uint64

	// Dependencies
	wal *wal.WAL
	kv  *kv.Store

	// Channels for internal signalling
	stepDownCh  chan struct{} // leader → follower
	heartbeatCh chan struct{} // received valid AppendEntries
	commitCh    chan struct{} // new entries committed (notify waiting proposals)
}

// NewNode creates a Node and recovers state from the WAL.
func NewNode(id string, peers []string, w *wal.WAL, store *kv.Store) (*Node, error) {
	n := &Node{
		id:          id,
		peers:       peers,
		role:        Follower,
		log:         []LogEntry{{Term: 0, Index: 0}}, // sentinel at index 0
		nextIndex:   make(map[string]uint64),
		matchIndex:  make(map[string]uint64),
		wal:         w,
		kv:          store,
		stepDownCh:  make(chan struct{}, 1),
		heartbeatCh: make(chan struct{}, 1),
		commitCh:    make(chan struct{}, 16),
	}
	if err := n.recoverFromWAL(); err != nil {
		return nil, err
	}
	return n, nil
}

// recoverFromWAL replays the WAL into the in-memory log and applies committed entries.
func (n *Node) recoverFromWAL() error {
	entries, err := n.wal.ReadAll()
	if err != nil {
		return err
	}
	for _, e := range entries {
		n.log = append(n.log, LogEntry{Term: e.Term, Index: e.Index, Command: e.Command})
	}
	// On recovery we treat all WAL entries as committed (simplification:
	// a proper implementation would persist commitIndex separately).
	if len(entries) > 0 {
		last := entries[len(entries)-1]
		n.commitIndex = last.Index
		n.lastApplied = last.Index
		for _, e := range entries {
			if len(e.Command) > 0 {
				n.kv.Apply(e.Command) //nolint:errcheck
			}
		}
	}
	return nil
}

// --- Accessors (all require mu held by caller or are safe to call externally) ---

func (n *Node) ID() string { return n.id }

func (n *Node) State() (role Role, term uint64, leaderID string, commitIndex uint64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.role, n.currentTerm, n.leaderID, n.commitIndex
}

func (n *Node) lastLogIndex() uint64 { return n.log[len(n.log)-1].Index }
func (n *Node) lastLogTerm() uint64  { return n.log[len(n.log)-1].Term }

// termOf returns the term of the entry at logIndex, or 0 if out of range.
func (n *Node) termOf(logIndex uint64) uint64 {
	for i := len(n.log) - 1; i >= 0; i-- {
		if n.log[i].Index == logIndex {
			return n.log[i].Term
		}
	}
	return 0
}

// becomeFollower transitions to Follower and updates term.
func (n *Node) becomeFollower(term uint64) {
	n.role = Follower
	n.currentTerm = term
	n.votedFor = ""
	select {
	case n.stepDownCh <- struct{}{}:
	default:
	}
}

// becomeLeader transitions to Leader and initialises leader state.
func (n *Node) becomeLeader() {
	n.role = Leader
	n.leaderID = n.id
	nextIdx := n.lastLogIndex() + 1
	for _, p := range n.peers {
		n.nextIndex[p] = nextIdx
		n.matchIndex[p] = 0
	}
}
