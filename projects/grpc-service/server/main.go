package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	pb "tour_of_go/projects/grpc-service/gen/greeter"

	"google.golang.org/grpc"
)

type greeterServer struct {
	pb.UnimplementedGreeterServer
}

// SayHello handles a unary RPC call
func (s *greeterServer) SayHello(_ context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	log.Printf("SayHello called with name=%q", req.GetName())
	return &pb.HelloResponse{
		Message: fmt.Sprintf("Hello, %s! (from gRPC server)", req.GetName()),
	}, nil
}

// SayHelloStream handles a server-streaming RPC call
func (s *greeterServer) SayHelloStream(req *pb.HelloRequest, stream pb.Greeter_SayHelloStreamServer) error {
	log.Printf("SayHelloStream called with name=%q", req.GetName())
	for i := 1; i <= 3; i++ {
		msg := &pb.HelloResponse{
			Message: fmt.Sprintf("Stream message %d: Hello, %s!", i, req.GetName()),
		}
		if err := stream.Send(msg); err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &greeterServer{})

	log.Println("gRPC server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
