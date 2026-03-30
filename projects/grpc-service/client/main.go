package main

import (
	"context"
	"io"
	"log"
	"time"

	pb "tour_of_go/projects/grpc-service/gen/greeter"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to the gRPC server (insecure for local dev)
	conn, err := grpc.NewClient("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewGreeterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Unary RPC
	log.Println("--- Unary RPC ---")
	resp, err := client.SayHello(ctx, &pb.HelloRequest{Name: "Gopher"})
	if err != nil {
		log.Fatalf("SayHello failed: %v", err)
	}
	log.Println("Response:", resp.GetMessage())

	// Server-streaming RPC
	log.Println("\n--- Server-Streaming RPC ---")
	stream, err := client.SayHelloStream(ctx, &pb.HelloRequest{Name: "Gopher"})
	if err != nil {
		log.Fatalf("SayHelloStream failed: %v", err)
	}
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("stream error: %v", err)
		}
		log.Println("Stream:", msg.GetMessage())
	}
}
