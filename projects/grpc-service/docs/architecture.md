# Architecture

## Overview

gRPC server + client demonstrating Protobuf-defined services with unary and server-streaming RPCs.

## Components

```
proto/greeter.proto    → Service definition (Protobuf IDL)
gen/                   → Generated Go code (protoc-gen-go + protoc-gen-go-grpc)
server/main.go         → gRPC server implementation
client/main.go         → gRPC client with both call patterns
```

## Key Concepts

- **Protobuf**: Language-neutral serialization (smaller + faster than JSON)
- **Unary RPC**: Single request → single response
- **Server streaming**: Single request → stream of responses
- **HTTP/2**: Multiplexed streams over a single TCP connection
