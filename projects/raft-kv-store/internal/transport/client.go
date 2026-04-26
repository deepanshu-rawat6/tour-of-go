package transport

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/tour-of-go/raft-kv-store/internal/raft"
	pb "github.com/tour-of-go/raft-kv-store/internal/transport/pb"
)

// Client implements raft.Transport by sending gRPC calls to peers.
type Client struct {
	mu    sync.Mutex
	conns map[string]*grpc.ClientConn
	addrs map[string]string // peerID → gRPC address
}

func NewClient(peers map[string]string) *Client {
	return &Client{
		conns: make(map[string]*grpc.ClientConn),
		addrs: peers,
	}
}

func (c *Client) conn(peerID string) (pb.RaftServiceClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if conn, ok := c.conns[peerID]; ok {
		return pb.NewRaftServiceClient(conn), nil
	}
	addr := c.addrs[peerID]
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	c.conns[peerID] = conn
	return pb.NewRaftServiceClient(conn), nil
}

func (c *Client) RequestVote(ctx context.Context, peerID string, args *raft.RequestVoteArgs) (*raft.RequestVoteReply, error) {
	stub, err := c.conn(peerID)
	if err != nil {
		return nil, err
	}
	resp, err := stub.RequestVote(ctx, &pb.RequestVoteRequest{
		Term:         args.Term,
		CandidateId:  args.CandidateID,
		LastLogIndex: args.LastLogIndex,
		LastLogTerm:  args.LastLogTerm,
	})
	if err != nil {
		return nil, err
	}
	return &raft.RequestVoteReply{Term: resp.Term, VoteGranted: resp.VoteGranted}, nil
}

func (c *Client) AppendEntries(ctx context.Context, peerID string, args *raft.AppendEntriesArgs) (*raft.AppendEntriesReply, error) {
	stub, err := c.conn(peerID)
	if err != nil {
		return nil, err
	}
	entries := make([]*pb.LogEntry, len(args.Entries))
	for i, e := range args.Entries {
		entries[i] = &pb.LogEntry{Term: e.Term, Index: e.Index, Command: e.Command}
	}
	resp, err := stub.AppendEntries(ctx, &pb.AppendEntriesRequest{
		Term:         args.Term,
		LeaderId:     args.LeaderID,
		PrevLogIndex: args.PrevLogIndex,
		PrevLogTerm:  args.PrevLogTerm,
		Entries:      entries,
		LeaderCommit: args.LeaderCommit,
	})
	if err != nil {
		return nil, err
	}
	return &raft.AppendEntriesReply{Term: resp.Term, Success: resp.Success, ConflictIndex: resp.ConflictIndex}, nil
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, conn := range c.conns {
		conn.Close()
	}
}
