package raft

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

// Transport is the interface the election module uses to send RPCs.
// Implemented by the gRPC transport layer.
type Transport interface {
	RequestVote(ctx context.Context, peerID string, req *RequestVoteArgs) (*RequestVoteReply, error)
	AppendEntries(ctx context.Context, peerID string, req *AppendEntriesArgs) (*AppendEntriesReply, error)
}

// RequestVoteArgs / Reply mirror the protobuf types but are transport-agnostic.
type RequestVoteArgs struct {
	Term         uint64
	CandidateID  string
	LastLogIndex uint64
	LastLogTerm  uint64
}

type RequestVoteReply struct {
	Term        uint64
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term         uint64
	LeaderID     string
	PrevLogIndex uint64
	PrevLogTerm  uint64
	Entries      []LogEntry
	LeaderCommit uint64
}

type AppendEntriesReply struct {
	Term          uint64
	Success       bool
	ConflictIndex uint64
}

// Run starts the Raft main loop. Blocks until ctx is cancelled.
func (n *Node) Run(ctx context.Context, transport Transport, electionMinMs, electionMaxMs, heartbeatMs int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n.mu.Lock()
		role := n.role
		n.mu.Unlock()

		switch role {
		case Follower:
			n.runFollower(ctx, electionMinMs, electionMaxMs)
		case Candidate:
			n.runCandidate(ctx, transport, electionMinMs, electionMaxMs)
		case Leader:
			n.runLeader(ctx, transport, heartbeatMs)
		}
	}
}

func (n *Node) electionTimeout(minMs, maxMs int) time.Duration {
	ms := minMs + rand.Intn(maxMs-minMs)
	return time.Duration(ms) * time.Millisecond
}

func (n *Node) runFollower(ctx context.Context, minMs, maxMs int) {
	timer := time.NewTimer(n.electionTimeout(minMs, maxMs))
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-n.heartbeatCh:
			// Reset timer on valid heartbeat
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(n.electionTimeout(minMs, maxMs))
		case <-timer.C:
			// Timeout — become candidate
			n.mu.Lock()
			n.role = Candidate
			n.mu.Unlock()
			return
		}
	}
}

func (n *Node) runCandidate(ctx context.Context, transport Transport, minMs, maxMs int) {
	n.mu.Lock()
	n.currentTerm++
	n.votedFor = n.id
	term := n.currentTerm
	lastIdx := n.lastLogIndex()
	lastTerm := n.lastLogTerm()
	peers := n.peers
	n.mu.Unlock()

	votes := 1 // vote for self
	majority := (len(peers)+1)/2 + 1
	var voteMu sync.Mutex
	wonElection := make(chan struct{}, 1)

	for _, peer := range peers {
		peer := peer
		go func() {
			reply, err := transport.RequestVote(ctx, peer, &RequestVoteArgs{
				Term:         term,
				CandidateID:  n.id,
				LastLogIndex: lastIdx,
				LastLogTerm:  lastTerm,
			})
			if err != nil {
				return
			}
			n.mu.Lock()
			if reply.Term > n.currentTerm {
				n.becomeFollower(reply.Term)
				n.mu.Unlock()
				return
			}
			n.mu.Unlock()
			if reply.VoteGranted {
				voteMu.Lock()
				votes++
				if votes >= majority {
					select {
					case wonElection <- struct{}{}:
					default:
					}
				}
				voteMu.Unlock()
			}
		}()
	}

	timer := time.NewTimer(n.electionTimeout(minMs, maxMs))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-n.stepDownCh:
		// Received higher term — already set to Follower
		return
	case <-wonElection:
		n.mu.Lock()
		if n.role == Candidate && n.currentTerm == term {
			n.becomeLeader()
		}
		n.mu.Unlock()
	case <-timer.C:
		// Split vote — restart election (role stays Candidate)
	}
}

func (n *Node) runLeader(ctx context.Context, transport Transport, heartbeatMs int) {
	ticker := time.NewTicker(time.Duration(heartbeatMs) * time.Millisecond)
	defer ticker.Stop()

	// Send immediate heartbeat on becoming leader
	n.sendHeartbeats(ctx, transport)

	for {
		select {
		case <-ctx.Done():
			return
		case <-n.stepDownCh:
			return
		case <-ticker.C:
			n.sendHeartbeats(ctx, transport)
		}
	}
}

func (n *Node) sendHeartbeats(ctx context.Context, transport Transport) {
	n.mu.Lock()
	term := n.currentTerm
	leaderID := n.id
	commitIndex := n.commitIndex
	peers := n.peers
	n.mu.Unlock()

	for _, peer := range peers {
		peer := peer
		go func() {
			reply, err := transport.AppendEntries(ctx, peer, &AppendEntriesArgs{
				Term:         term,
				LeaderID:     leaderID,
				LeaderCommit: commitIndex,
				// Empty entries = heartbeat
			})
			if err != nil {
				return
			}
			if reply.Term > term {
				n.mu.Lock()
				n.becomeFollower(reply.Term)
				n.mu.Unlock()
			}
		}()
	}
}
