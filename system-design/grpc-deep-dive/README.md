# gRPC Deep Dive

> Level: SDE-1 → SDE-2 | Protocol: HTTP/2 + Protocol Buffers

---

## Table of Contents

1. [Proto3 Syntax](#1-proto3-syntax)
2. [Service Definitions & Streaming](#2-service-definitions--streaming)
3. [Go Code Generation](#3-go-code-generation)
4. [Interceptors](#4-interceptors)
5. [Deadlines & Timeouts](#5-deadlines--timeouts)
6. [Error Model](#6-error-model)
7. [Metadata](#7-metadata)
8. [Load Balancing](#8-load-balancing)
9. [Health Checks](#9-health-checks)
10. [Reflection](#10-reflection)
11. [TLS and mTLS](#11-tls-and-mtls)
12. [Credentials](#12-credentials)
13. [Connection Management](#13-connection-management)
14. [Proto2 vs Proto3](#14-proto2-vs-proto3)
15. [Versioning & Compatibility](#15-versioning--compatibility)

---

## 1. Proto3 Syntax

### Basic Message

```protobuf
syntax = "proto3";

package order.v1;

option go_package = "github.com/myorg/myapp/gen/order/v1;orderv1";

import "google/protobuf/timestamp.proto";

message Order {
  string  id         = 1;
  string  user_id    = 2;
  Status  status     = 3;
  repeated OrderItem items = 4;
  map<string, string> metadata = 5;
  google.protobuf.Timestamp created_at = 6;

  // reserved field numbers and names — never reuse
  reserved 7, 8;
  reserved "legacy_field";
}

message OrderItem {
  string product_id = 1;
  int32  quantity   = 2;
  double price_usd  = 3;
}
```

### Enums

```protobuf
enum Status {
  STATUS_UNSPECIFIED = 0; // proto3: first value MUST be 0
  STATUS_PENDING     = 1;
  STATUS_CONFIRMED   = 2;
  STATUS_SHIPPED     = 3;
  STATUS_CANCELLED   = 4;
}
```

**Rule:** Always define an `_UNSPECIFIED = 0` sentinel. It's what you get when the field is absent.

### oneof — Mutually Exclusive Fields

```protobuf
message PaymentMethod {
  oneof method {
    CreditCard  credit_card  = 1;
    BankTransfer bank_transfer = 2;
    Crypto      crypto        = 3;
  }
}

message CreditCard {
  string last_four = 1;
  string network   = 2;
}
```

In Go:
```go
switch m := payment.Method.(type) {
case *PaymentMethod_CreditCard:
    fmt.Println(m.CreditCard.LastFour)
case *PaymentMethod_BankTransfer:
    fmt.Println(m.BankTransfer.AccountNumber)
}
```

### map Fields

```protobuf
// Key must be integral or string. Value can be any type.
map<string, int32> stock_levels = 1;
map<string, Product> catalog    = 2;
```

- Map fields cannot be `repeated`.
- Iteration order is not guaranteed.

### reserved

```protobuf
message User {
  string name  = 1;
  // int32 age = 2; // removed in v2 — reserve the number
  reserved 2, 3;
  reserved "age", "dob";
  string email = 4;
}
```

**Why:** Prevents future fields from accidentally reusing old field numbers, which would corrupt existing serialised data.

---

## 2. Service Definitions & Streaming

```protobuf
service OrderService {
  // Unary: one request → one response
  rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse);

  // Server streaming: one request → stream of responses
  rpc WatchOrderStatus(WatchOrderRequest) returns (stream OrderStatusEvent);

  // Client streaming: stream of requests → one response
  rpc UploadOrderItems(stream OrderItemChunk) returns (UploadSummary);

  // Bidirectional streaming: stream ↔ stream
  rpc OrderChat(stream ChatMessage) returns (stream ChatMessage);
}
```

### Unary — Go server

```go
func (s *server) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
    if req.UserId == "" {
        return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
    }
    order, err := s.svc.Create(ctx, req)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "create order: %v", err)
    }
    return &pb.CreateOrderResponse{OrderId: order.ID}, nil
}
```

### Server Streaming

```go
func (s *server) WatchOrderStatus(req *pb.WatchOrderRequest, stream pb.OrderService_WatchOrderStatusServer) error {
    ch := s.svc.Subscribe(req.OrderId)
    defer s.svc.Unsubscribe(req.OrderId, ch)
    for {
        select {
        case event := <-ch:
            if err := stream.Send(&pb.OrderStatusEvent{Status: event.Status}); err != nil {
                return err // client disconnected
            }
        case <-stream.Context().Done():
            return stream.Context().Err()
        }
    }
}
```

### Client Streaming

```go
func (s *server) UploadOrderItems(stream pb.OrderService_UploadOrderItemsServer) error {
    var items []*pb.OrderItem
    for {
        chunk, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
        items = append(items, chunk.Items...)
    }
    summary, err := s.svc.BulkInsert(stream.Context(), items)
    if err != nil {
        return status.Errorf(codes.Internal, "bulk insert: %v", err)
    }
    return stream.SendAndClose(summary)
}
```

### Bidirectional Streaming

```go
func (s *server) OrderChat(stream pb.OrderService_OrderChatServer) error {
    for {
        msg, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return err
        }
        reply := s.svc.Handle(msg)
        if err := stream.Send(reply); err != nil {
            return err
        }
    }
}
```

---

## 3. Go Code Generation

### Install tools

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Generate

```bash
protoc \
  --proto_path=proto \
  --go_out=gen --go_opt=paths=source_relative \
  --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
  proto/order/v1/order.proto
```

### Recommended layout

```
myapp/
├── proto/
│   └── order/v1/order.proto
├── gen/
│   └── order/v1/
│       ├── order.pb.go          # message types
│       └── order_grpc.pb.go     # client + server interfaces
├── internal/
│   └── orderservice/
│       └── server.go            # implements OrderServiceServer
└── cmd/server/main.go
```

### Registering the server

```go
func main() {
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatal(err)
    }
    s := grpc.NewServer()
    pb.RegisterOrderServiceServer(s, &orderservice.Server{})
    reflection.Register(s) // enables grpcurl
    log.Fatal(s.Serve(lis))
}
```

---

## 4. Interceptors

Interceptors are gRPC's middleware — run code before/after each RPC.

### Unary Server Interceptor

```go
func LoggingInterceptor(
    ctx context.Context,
    req interface{},
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (interface{}, error) {
    start := time.Now()
    resp, err := handler(ctx, req)
    log.Printf("method=%s duration=%s err=%v", info.FullMethod, time.Since(start), err)
    return resp, err
}

func AuthInterceptor(
    ctx context.Context,
    req interface{},
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (interface{}, error) {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "missing metadata")
    }
    tokens := md["authorization"]
    if len(tokens) == 0 || !validateToken(tokens[0]) {
        return nil, status.Error(codes.Unauthenticated, "invalid token")
    }
    return handler(ctx, req)
}

func RecoveryInterceptor(
    ctx context.Context,
    req interface{},
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (resp interface{}, err error) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("panic recovered: %v\n%s", r, debug.Stack())
            err = status.Errorf(codes.Internal, "internal panic")
        }
    }()
    return handler(ctx, req)
}
```

### Chaining interceptors

```go
s := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        RecoveryInterceptor,
        LoggingInterceptor,
        AuthInterceptor,
    ),
)
```

### Stream Server Interceptor

```go
func LoggingStreamInterceptor(
    srv interface{},
    ss grpc.ServerStream,
    info *grpc.StreamServerInfo,
    handler grpc.StreamHandler,
) error {
    start := time.Now()
    err := handler(srv, ss)
    log.Printf("stream method=%s duration=%s err=%v", info.FullMethod, time.Since(start), err)
    return err
}
```

### Wrapping a ServerStream (to inject context)

```go
type wrappedStream struct {
    grpc.ServerStream
    ctx context.Context
}

func (w *wrappedStream) Context() context.Context { return w.ctx }

func TenantStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
    md, _ := metadata.FromIncomingContext(ss.Context())
    tenantID := md["x-tenant-id"][0]
    ctx := context.WithValue(ss.Context(), tenantKey{}, tenantID)
    return handler(srv, &wrappedStream{ss, ctx})
}
```

---

## 5. Deadlines & Timeouts

gRPC propagates deadlines automatically across service hops.

```go
// Client sets deadline
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()
resp, err := client.CreateOrder(ctx, req)
if err != nil {
    if status.Code(err) == codes.DeadlineExceeded {
        log.Println("timeout hit")
    }
}
```

```go
// Server respects it
func (s *server) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
    if deadline, ok := ctx.Deadline(); ok {
        log.Printf("deadline in %s", time.Until(deadline))
    }
    // Pass ctx to all downstream calls — they will be cancelled automatically
    result, err := s.db.QueryContext(ctx, "SELECT ...")
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            return nil, status.Error(codes.DeadlineExceeded, "db query timed out")
        }
        return nil, status.Errorf(codes.Internal, "%v", err)
    }
    return result, nil
}
```

**Key rules:**
- Always pass `ctx` down to every I/O call.
- Never ignore `ctx.Err()`.
- Prefer `WithTimeout` on the client; the server just honours it.

---

## 6. Error Model

### Basic status errors

```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// Returning errors
return nil, status.Errorf(codes.NotFound, "order %s not found", orderID)
return nil, status.Errorf(codes.InvalidArgument, "quantity must be > 0")
return nil, status.Errorf(codes.AlreadyExists, "order already exists")
return nil, status.Errorf(codes.PermissionDenied, "access denied")
return nil, status.Errorf(codes.Internal, "unexpected: %v", err)
return nil, status.Errorf(codes.Unavailable, "downstream service down")
return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")

// Reading errors on client
st := status.Convert(err)
fmt.Println(st.Code(), st.Message())

switch status.Code(err) {
case codes.NotFound:
    // 404 equivalent
case codes.InvalidArgument:
    // 400 equivalent
case codes.DeadlineExceeded:
    // timeout
}
```

### Rich errors with details (google.rpc.Status)

```protobuf
import "google/rpc/error_details.proto";
```

```go
import (
    "google.golang.org/grpc/status"
    "google.golang.org/grpc/codes"
    epb "google.golang.org/genproto/googleapis/rpc/errdetails"
)

func invalidArgErr(field, desc string) error {
    st, _ := status.New(codes.InvalidArgument, "validation failed").
        WithDetails(&epb.BadRequest{
            FieldViolations: []*epb.BadRequest_FieldViolation{
                {Field: field, Description: desc},
            },
        })
    return st.Err()
}

// Client side:
st := status.Convert(err)
for _, detail := range st.Details() {
    switch d := detail.(type) {
    case *epb.BadRequest:
        for _, v := range d.FieldViolations {
            fmt.Printf("field=%s: %s\n", v.Field, v.Description)
        }
    }
}
```

### Common codes mapped to HTTP

| gRPC Code | HTTP | Meaning |
|---|---|---|
| OK | 200 | Success |
| InvalidArgument | 400 | Bad input |
| Unauthenticated | 401 | No/bad credentials |
| PermissionDenied | 403 | Authenticated but unauthorised |
| NotFound | 404 | Resource missing |
| AlreadyExists | 409 | Duplicate |
| ResourceExhausted | 429 | Rate limited |
| Internal | 500 | Bug on server |
| Unavailable | 503 | Service down, safe to retry |
| DeadlineExceeded | 504 | Timeout |

---

## 7. Metadata

Metadata is gRPC's equivalent of HTTP headers — key-value pairs sent with the RPC.

### Sending from client

```go
md := metadata.Pairs(
    "authorization", "Bearer "+token,
    "x-request-id", requestID,
    "x-tenant-id", tenantID,
)
ctx := metadata.NewOutgoingContext(context.Background(), md)
resp, err := client.CreateOrder(ctx, req)
```

### Reading on server

```go
func (s *server) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Error(codes.InvalidArgument, "missing metadata")
    }
    requestIDs := md.Get("x-request-id") // returns []string
    if len(requestIDs) > 0 {
        ctx = context.WithValue(ctx, requestIDKey{}, requestIDs[0])
    }
    // ...
}
```

### Sending response headers/trailers from server

```go
func (s *server) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
    // Send header immediately
    header := metadata.Pairs("x-request-id", uuid.New().String())
    grpc.SendHeader(ctx, header)

    // Attach trailer (sent after response)
    trailer := metadata.Pairs("x-server-timing", "db=12ms")
    grpc.SetTrailer(ctx, trailer)

    return &pb.CreateOrderResponse{}, nil
}
```

### Reading headers/trailers on client

```go
var header, trailer metadata.MD
resp, err := client.CreateOrder(
    ctx, req,
    grpc.Header(&header),
    grpc.Trailer(&trailer),
)
```

---

## 8. Load Balancing

gRPC uses **client-side load balancing** by default. The client resolves the target to multiple addresses and picks one per RPC.

### round_robin

```go
conn, err := grpc.Dial(
    "dns:///order-service.default.svc.cluster.local:50051",
    grpc.WithTransportCredentials(insecure.NewCredentials()),
    grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
)
```

### pick_first (default)

Connects to the first resolved address, only failovers if that address goes down. Use `round_robin` for actual load distribution.

### With Kubernetes headless service

A headless service (`clusterIP: None`) returns multiple A records via DNS — each pod's IP. Combined with `dns:///` scheme and `round_robin`, gRPC will distribute RPCs across all pods.

```yaml
# headless service
spec:
  clusterIP: None
  selector:
    app: order-service
```

### Custom resolver (for service discovery)

```go
// Implement resolver.Builder and resolver.Resolver interfaces
// Register with resolver.Register(myBuilder)
// Then use: grpc.Dial("myscheme:///service-name", ...)
```

---

## 9. Health Checks

Standard gRPC health check protocol (`grpc_health_v1`).

```bash
go get google.golang.org/grpc/health
```

```go
import (
    "google.golang.org/grpc/health"
    healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
    s := grpc.NewServer()
    healthSrv := health.NewServer()
    healthpb.RegisterHealthServer(s, healthSrv)

    // Mark service healthy
    healthSrv.SetServingStatus("order.v1.OrderService", healthpb.HealthCheckResponse_SERVING)
    // Mark unhealthy (e.g. DB connection lost)
    healthSrv.SetServingStatus("order.v1.OrderService", healthpb.HealthCheckResponse_NOT_SERVING)
}
```

### Check from grpcurl

```bash
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
grpcurl -plaintext -d '{"service":"order.v1.OrderService"}' localhost:50051 grpc.health.v1.Health/Check
```

### Kubernetes liveness/readiness probe

```yaml
livenessProbe:
  grpc:
    port: 50051
  initialDelaySeconds: 10
readinessProbe:
  grpc:
    port: 50051
    service: "order.v1.OrderService"
```

---

## 10. Reflection

Reflection lets tools discover your service schema at runtime without the `.proto` file.

```go
import "google.golang.org/grpc/reflection"

s := grpc.NewServer()
pb.RegisterOrderServiceServer(s, &server{})
reflection.Register(s) // register reflection service
```

### grpcurl usage

```bash
# List services
grpcurl -plaintext localhost:50051 list

# List methods of a service
grpcurl -plaintext localhost:50051 list order.v1.OrderService

# Describe a method
grpcurl -plaintext localhost:50051 describe order.v1.OrderService.CreateOrder

# Call an RPC
grpcurl -plaintext -d '{"user_id":"u1","items":[{"product_id":"p1","quantity":2}]}' \
  localhost:50051 order.v1.OrderService/CreateOrder
```

**Production note:** Disable reflection in production if your proto definitions are sensitive. Use an API gateway with schema validation instead.

---

## 11. TLS and mTLS

### One-way TLS (server authenticates to client)

```go
// Server
creds, err := credentials.NewServerTLSFromFile("server.crt", "server.key")
s := grpc.NewServer(grpc.Creds(creds))

// Client
creds, err := credentials.NewClientTLSFromFile("ca.crt", "")
conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(creds))
```

### mTLS (mutual — both sides authenticate)

```go
// Server: require client cert
cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
ca, err := os.ReadFile("ca.crt")
pool := x509.NewCertPool()
pool.AppendCertsFromPEM(ca)

tlsCfg := &tls.Config{
    Certificates: []tls.Certificate{cert},
    ClientAuth:   tls.RequireAndVerifyClientCert,
    ClientCAs:    pool,
}
creds := credentials.NewTLS(tlsCfg)
s := grpc.NewServer(grpc.Creds(creds))

// Client: present its own cert
cert, err := tls.LoadX509KeyPair("client.crt", "client.key")
ca, err := os.ReadFile("ca.crt")
pool := x509.NewCertPool()
pool.AppendCertsFromPEM(ca)

tlsCfg := &tls.Config{
    Certificates: []tls.Certificate{cert},
    RootCAs:      pool,
    ServerName:   "order-service",
}
conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
```

---

## 12. Credentials

Two layers:
- **Channel credentials** — secure the transport (TLS/mTLS). Applied at `Dial` time.
- **Per-call credentials** — attach tokens to individual RPCs (JWT, OAuth2).

```go
// Per-call credential
type bearerToken struct{ token string }

func (b bearerToken) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
    return map[string]string{"authorization": "Bearer " + b.token}, nil
}
func (b bearerToken) RequireTransportSecurity() bool { return true } // must use TLS

conn, err := grpc.Dial(
    "localhost:50051",
    grpc.WithTransportCredentials(tlsCreds),
    grpc.WithPerRPCCredentials(bearerToken{token: myJWT}),
)
```

---

## 13. Connection Management

gRPC **multiplexes all RPCs over a single HTTP/2 connection**. You do not need a connection pool like you would with HTTP/1.1.

```go
// One conn, many goroutines — this is correct
conn, err := grpc.Dial(target, opts...)
client := pb.NewOrderServiceClient(conn)

// Use from many goroutines concurrently — safe
go func() { client.CreateOrder(ctx, req1) }()
go func() { client.CreateOrder(ctx, req2) }()
```

### Keepalive

```go
import "google.golang.org/grpc/keepalive"

conn, err := grpc.Dial(target,
    grpc.WithKeepaliveParams(keepalive.ClientParameters{
        Time:                10 * time.Second, // ping interval
        Timeout:             3 * time.Second,  // wait for pong
        PermitWithoutStream: true,             // ping even without active RPCs
    }),
)
```

```go
// Server side
s := grpc.NewServer(
    grpc.KeepaliveParams(keepalive.ServerParameters{
        MaxConnectionIdle:     15 * time.Second,
        MaxConnectionAge:      30 * time.Second,
        MaxConnectionAgeGrace: 5 * time.Second,
        Time:                  5 * time.Second,
        Timeout:               1 * time.Second,
    }),
    grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
        MinTime:             5 * time.Second,
        PermitWithoutStream: true,
    }),
)
```

### When you DO want multiple connections

- Different auth credentials per connection.
- Explicit shard routing (each conn targets a specific backend).
- Otherwise: one `*grpc.ClientConn` per target is correct.

---

## 14. Proto2 vs Proto3

| Feature | Proto2 | Proto3 |
|---|---|---|
| Required fields | Yes (`required`) | No — all fields optional |
| Default values | Explicit (`[default = ...]`) | Language-type zero values |
| `optional` keyword | Yes | Added back in proto3 (with `has` detection) |
| Extensions | Yes | Replaced by `Any` |
| Unknown fields | Preserved (proto3 since 3.5) | Preserved |
| JSON mapping | Optional | Built-in |
| Use today? | Legacy codebases only | Yes — prefer proto3 |

**Interview answer:** Proto3 removed `required` to make forward/backward compatibility easier. A missing required field in proto2 would cause parse failure; proto3 just uses zero values, which is safer for schema evolution.

---

## 15. Versioning & Compatibility

### Safe (backward + forward compatible) changes

- Add a new field (new field number).
- Add a new enum value.
- Rename a field (field number is what matters in binary encoding, not name).
- Add a new RPC method.

### Unsafe changes (breaking)

- Change a field number.
- Change a field type (e.g., `int32` → `string`).
- Remove a field without `reserved`.
- Rename a package or service.

### Versioning strategy

```protobuf
// v1 — initial
package order.v1;

// v2 — breaking change? create a new package
package order.v2;
```

In Go, use separate import paths:
```
gen/order/v1/  ← old clients stay here
gen/order/v2/  ← new clients migrate here
```

### Field presence in proto3

```protobuf
// Use optional to detect "was this field set?"
message UpdateOrderRequest {
  string order_id = 1;
  optional string note = 2; // has_note() available in Go
}
```

```go
if req.Note != nil {
    // note was explicitly set (even to "")
}
```

---

## Quick Reference

```
Unary RPC:         client.Method(ctx, req)
Server stream:     stream.Send(msg) on server, stream.Recv() on client
Client stream:     stream.Recv() on server, stream.Send(msg) on client, stream.CloseAndRecv()
Bidi stream:       stream.Send() + stream.Recv() on both sides

Error:             status.Errorf(codes.NotFound, "msg")
Read error code:   status.Code(err)
Metadata out:      metadata.NewOutgoingContext(ctx, md)
Metadata in:       metadata.FromIncomingContext(ctx)
Deadline:          context.WithTimeout(ctx, 3*time.Second)
Interceptor chain: grpc.ChainUnaryInterceptor(...)
Health:            healthpb.RegisterHealthServer(s, health.NewServer())
Reflection:        reflection.Register(s)
Load balance:      grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`)
```
