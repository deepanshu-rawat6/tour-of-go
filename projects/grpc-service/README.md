# gRPC Service

A minimal gRPC server and client in Go demonstrating unary and server-streaming RPCs.

## Concepts

- **Protobuf**: Language-neutral schema for defining services and messages (`proto/greeter.proto`)
- **Unary RPC**: Single request → single response (like a regular function call over the network)
- **Server-Streaming RPC**: Single request → stream of responses (useful for logs, events, progress)
- **Generated code**: `gen/greeter/` contains the Go types and gRPC stubs generated from the proto

## How to Run

```shell
# Terminal 1 — start the server
go run ./server/

# Terminal 2 — run the client
go run ./client/
```

Expected output (client):
```
SayHello Response: Hello, Gopher! (from gRPC server)
Stream: Stream message 1: Hello, Gopher!
Stream: Stream message 2: Hello, Gopher!
Stream: Stream message 3: Hello, Gopher!
```

## Regenerating Proto Code

Install tools:
```shell
brew install protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Regenerate:
```shell
protoc --go_out=gen --go-grpc_out=gen proto/greeter.proto
```

## Key Files

```
proto/greeter.proto          # Service definition (source of truth)
gen/greeter/greeter.pb.go    # Generated message types
gen/greeter/greeter_grpc.pb.go # Generated gRPC client/server interfaces
server/main.go               # gRPC server implementation
client/main.go               # gRPC client
```

## What to Learn Next

- Add interceptors (middleware) for logging and auth
- Add TLS with `credentials.NewTLS()`
- Try bidirectional streaming: `stream (HelloRequest) returns (stream HelloResponse)`
- See [Platform Ops README](../../more-internals/system-design/platform-ops/README.md) for how gRPC fits into K8s
