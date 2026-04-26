package transport

import (
	"context"
	"net"

	"google.golang.org/grpc"

	"github.com/tour-of-go/raft-kv-store/internal/raft"
	pb "github.com/tour-of-go/raft-kv-store/internal/transport/pb"
)

// RaftHandler is implemented by the Raft node to handle incoming RPCs.
type RaftHandler interface {
	HandleAppendEntries(args *raft.AppendEntriesArgs) *raft.AppendEntriesReply
	HandleRequestVote(args *raft.RequestVoteArgs) *raft.RequestVoteReply
}

// Server wraps a gRPC server exposing the Raft service.
type Server struct {
	pb.UnimplementedRaftServiceServer
	handler RaftHandler
	grpc    *grpc.Server
}

func NewServer(handler RaftHandler) *Server {
	s := &Server{handler: handler}
	s.grpc = grpc.NewServer()
	pb.RegisterRaftServiceServer(s.grpc, s)
	return s
}

func (s *Server) Serve(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.grpc.Serve(ln)
}

func (s *Server) Stop() { s.grpc.GracefulStop() }

func (s *Server) AppendEntries(_ context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	entries := make([]raft.LogEntry, len(req.Entries))
	for i, e := range req.Entries {
		entries[i] = raft.LogEntry{Term: e.Term, Index: e.Index, Command: e.Command}
	}
	reply := s.handler.HandleAppendEntries(&raft.AppendEntriesArgs{
		Term:         req.Term,
		LeaderID:     req.LeaderId,
		PrevLogIndex: req.PrevLogIndex,
		PrevLogTerm:  req.PrevLogTerm,
		Entries:      entries,
		LeaderCommit: req.LeaderCommit,
	})
	return &pb.AppendEntriesResponse{
		Term:          reply.Term,
		Success:       reply.Success,
		ConflictIndex: reply.ConflictIndex,
	}, nil
}

func (s *Server) RequestVote(_ context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	reply := s.handler.HandleRequestVote(&raft.RequestVoteArgs{
		Term:         req.Term,
		CandidateID:  req.CandidateId,
		LastLogIndex: req.LastLogIndex,
		LastLogTerm:  req.LastLogTerm,
	})
	return &pb.RequestVoteResponse{
		Term:        reply.Term,
		VoteGranted: reply.VoteGranted,
	}, nil
}
