package raft

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/tour-of-go/raft-kv-store/internal/wal"
)

// Propose appends a command to the leader's log and waits for it to be committed.
// Returns an error if this node is not the leader.
func (n *Node) Propose(ctx context.Context, command []byte, transport Transport) (string, error) {
	n.mu.Lock()
	if n.role != Leader {
		leaderID := n.leaderID
		n.mu.Unlock()
		return "", &NotLeaderError{LeaderID: leaderID}
	}

	// Append to local log
	entry := LogEntry{
		Term:    n.currentTerm,
		Index:   n.lastLogIndex() + 1,
		Command: command,
	}
	n.log = append(n.log, entry)
	if err := n.wal.Append(wal.Entry{Term: entry.Term, Index: entry.Index, Command: entry.Command}); err != nil {
		n.mu.Unlock()
		return "", fmt.Errorf("WAL append: %w", err)
	}
	commitTarget := entry.Index
	term := n.currentTerm
	peers := n.peers
	n.mu.Unlock()

	// Replicate to followers
	n.replicateToFollowers(ctx, transport, term, peers)

	// Wait for commit
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-n.commitCh:
		case <-time.After(10 * time.Millisecond):
		}
		n.mu.Lock()
		if n.commitIndex >= commitTarget {
			n.mu.Unlock()
			// Parse command to return the value for PUT
			var cmd struct {
				Op    string `json:"op"`
				Value string `json:"value"`
			}
			json.Unmarshal(command, &cmd)
			return cmd.Value, nil
		}
		n.mu.Unlock()
	}
	return "", fmt.Errorf("commit timeout for index %d", commitTarget)
}

func (n *Node) replicateToFollowers(ctx context.Context, transport Transport, term uint64, peers []string) {
	var wg sync.WaitGroup
	acks := 1 // leader counts itself
	majority := (len(peers)+1)/2 + 1
	var mu sync.Mutex
	committed := false

	for _, peer := range peers {
		peer := peer
		wg.Add(1)
		go func() {
			defer wg.Done()
			n.mu.Lock()
			nextIdx := n.nextIndex[peer]
			prevIdx := nextIdx - 1
			prevTerm := n.termOf(prevIdx)
			var entries []LogEntry
			for _, e := range n.log {
				if e.Index >= nextIdx {
					entries = append(entries, e)
				}
			}
			commitIdx := n.commitIndex
			n.mu.Unlock()

			reply, err := transport.AppendEntries(ctx, peer, &AppendEntriesArgs{
				Term:         term,
				LeaderID:     n.id,
				PrevLogIndex: prevIdx,
				PrevLogTerm:  prevTerm,
				Entries:      entries,
				LeaderCommit: commitIdx,
			})
			if err != nil {
				return
			}

			n.mu.Lock()
			defer n.mu.Unlock()

			if reply.Term > n.currentTerm {
				n.becomeFollower(reply.Term)
				return
			}
			if reply.Success && len(entries) > 0 {
				last := entries[len(entries)-1]
				n.nextIndex[peer] = last.Index + 1
				n.matchIndex[peer] = last.Index

				mu.Lock()
				acks++
				if !committed && acks >= majority {
					committed = true
					mu.Unlock()
					n.advanceCommitIndex()
				} else {
					mu.Unlock()
				}
			} else if !reply.Success {
				// Back off nextIndex
				if reply.ConflictIndex > 0 {
					n.nextIndex[peer] = reply.ConflictIndex
				} else if n.nextIndex[peer] > 1 {
					n.nextIndex[peer]--
				}
			}
		}()
	}
}

// advanceCommitIndex finds the highest index replicated to a majority and commits.
func (n *Node) advanceCommitIndex() {
	// Find highest N such that matchIndex[majority] >= N and log[N].term == currentTerm
	for idx := n.lastLogIndex(); idx > n.commitIndex; idx-- {
		if n.termOf(idx) != n.currentTerm {
			continue
		}
		count := 1
		for _, peer := range n.peers {
			if n.matchIndex[peer] >= idx {
				count++
			}
		}
		majority := (len(n.peers)+1)/2 + 1
		if count >= majority {
			n.commitIndex = idx
			n.applyCommitted()
			select {
			case n.commitCh <- struct{}{}:
			default:
			}
			break
		}
	}
}

// applyCommitted applies all entries between lastApplied and commitIndex to the KV store.
func (n *Node) applyCommitted() {
	for n.lastApplied < n.commitIndex {
		n.lastApplied++
		for _, e := range n.log {
			if e.Index == n.lastApplied && len(e.Command) > 0 {
				n.kv.Apply(e.Command) //nolint:errcheck
				break
			}
		}
	}
}

// HandleAppendEntries processes an incoming AppendEntries RPC (follower side).
func (n *Node) HandleAppendEntries(args *AppendEntriesArgs) *AppendEntriesReply {
	n.mu.Lock()
	defer n.mu.Unlock()

	reply := &AppendEntriesReply{Term: n.currentTerm}

	if args.Term < n.currentTerm {
		return reply // stale leader
	}

	// Valid leader — reset election timer
	if args.Term > n.currentTerm {
		n.becomeFollower(args.Term)
	}
	n.role = Follower
	n.leaderID = args.LeaderID
	select {
	case n.heartbeatCh <- struct{}{}:
	default:
	}

	// Check prevLog consistency
	if args.PrevLogIndex > 0 {
		if n.lastLogIndex() < args.PrevLogIndex || n.termOf(args.PrevLogIndex) != args.PrevLogTerm {
			reply.ConflictIndex = n.lastLogIndex()
			return reply
		}
	}

	// Append new entries, truncating conflicts
	for _, e := range args.Entries {
		if e.Index <= n.lastLogIndex() {
			if n.termOf(e.Index) != e.Term {
				// Conflict — truncate from here
				n.wal.Truncate(e.Index) //nolint:errcheck
				newLog := []LogEntry{n.log[0]}
				for _, le := range n.log[1:] {
					if le.Index < e.Index {
						newLog = append(newLog, le)
					}
				}
				n.log = newLog
			} else {
				continue // already have this entry
			}
		}
		n.log = append(n.log, e)
		n.wal.Append(wal.Entry{Term: e.Term, Index: e.Index, Command: e.Command}) //nolint:errcheck
	}

	// Advance commit index
	if args.LeaderCommit > n.commitIndex {
		n.commitIndex = min(args.LeaderCommit, n.lastLogIndex())
		n.applyCommitted()
	}

	reply.Success = true
	reply.Term = n.currentTerm
	return reply
}

// HandleRequestVote processes an incoming RequestVote RPC.
func (n *Node) HandleRequestVote(args *RequestVoteArgs) *RequestVoteReply {
	n.mu.Lock()
	defer n.mu.Unlock()

	reply := &RequestVoteReply{Term: n.currentTerm}

	if args.Term < n.currentTerm {
		return reply
	}
	if args.Term > n.currentTerm {
		n.becomeFollower(args.Term)
	}

	// Grant vote if we haven't voted yet (or voted for this candidate)
	// and candidate's log is at least as up-to-date as ours
	canVote := n.votedFor == "" || n.votedFor == args.CandidateID
	logOK := args.LastLogTerm > n.lastLogTerm() ||
		(args.LastLogTerm == n.lastLogTerm() && args.LastLogIndex >= n.lastLogIndex())

	if canVote && logOK {
		n.votedFor = args.CandidateID
		reply.VoteGranted = true
		reply.Term = n.currentTerm
		select {
		case n.heartbeatCh <- struct{}{}:
		default:
		}
	}
	return reply
}

// NotLeaderError is returned when a write is sent to a non-leader.
type NotLeaderError struct{ LeaderID string }

func (e *NotLeaderError) Error() string { return "not leader: " + e.LeaderID }

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
