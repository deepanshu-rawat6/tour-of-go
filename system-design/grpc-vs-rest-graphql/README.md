# gRPC vs REST vs GraphQL — Decision Guide

> **SDE-1/SDE-2** — Asked in every senior backend interview. Know the trade-offs cold.

---

## Quick Decision Flowchart

```
Is it internal microservice-to-microservice?
├── YES → gRPC (binary, typed contracts, streaming)
└── NO → Is it a public API for diverse clients?
         ├── YES → Does the client need to shape its own queries?
         │         ├── YES → GraphQL (BFF, mobile, rapidly evolving frontend)
         │         └── NO  → REST (universal, cacheable, simple CRUD)
         └── NO → Is low latency / streaming critical?
                  ├── YES → gRPC
                  └── NO  → REST
```

---

## Side-by-Side Comparison

| Dimension | gRPC | REST | GraphQL |
|-----------|------|------|---------|
| **Protocol** | HTTP/2 + Protocol Buffers (binary) | HTTP/1.1 or HTTP/2 (text/JSON) | HTTP/1.1 or HTTP/2 (text/JSON) |
| **Contract** | `.proto` schema — strongly typed, code-generated | OpenAPI / Swagger — optional, not enforced | SDL (Schema Definition Language) — enforced by server |
| **Type safety** | Compile-time (generated stubs) | Runtime (JSON is untyped) | Runtime (schema-validated) |
| **Browser support** | ❌ Requires grpc-web proxy | ✅ Native | ✅ Native |
| **Streaming** | ✅ 4 modes (unary, server, client, bidirectional) | ⚠️ SSE for server push only | ⚠️ Subscriptions (WebSocket-based) |
| **Caching** | ❌ Hard (binary POST, no HTTP cache semantics) | ✅ HTTP caching (GET + ETags + CDN) | ⚠️ Per-query caching is complex |
| **Over-fetching** | Fixed payload per RPC | ⚠️ Common (endpoint returns all fields) | ✅ Client specifies exact fields |
| **Under-fetching** | Fixed payload per RPC | ⚠️ Common (requires multiple calls) | ✅ Single query for nested data |
| **Error model** | Rich: `status.Code` + `google.rpc.Status` details | HTTP status codes + JSON body | Always HTTP 200; errors in `errors[]` array |
| **Tooling** | grpcurl, Postman (gRPC), BloomRPC | Postman, curl, Swagger UI | GraphiQL, Apollo Studio, Altair |
| **Learning curve** | Medium (proto, codegen, interceptors) | Low (universal knowledge) | Medium (SDL, resolvers, DataLoader) |
| **Interoperability** | Requires proto-compatible runtime | Any HTTP client | Any HTTP client |
| **Payload size** | Small (Protobuf binary ~3-10x smaller) | Larger (JSON verbose) | Larger (JSON verbose) |
| **Versioning** | Proto backward-compatible fields | URL path / header / content-type | Schema evolution (additive) |

---

## gRPC Deep Dive

### When to Use gRPC

- Internal **microservice-to-microservice** communication
- **Streaming** workloads: live feeds, telemetry, file upload/download
- **Mobile clients** where bandwidth is constrained
- Systems that require **strict, evolving contracts** (proto fields are numbered)
- Polyglot environments — codegen for Go, Java, Python, Rust, etc.

### Key Strengths

```proto
// Protobuf gives you typed contracts with version-safe field numbers
syntax = "proto3";

service OrderService {
  rpc CreateOrder (CreateOrderRequest) returns (Order);                    // unary
  rpc StreamOrders (StreamOrdersRequest) returns (stream Order);           // server stream
  rpc UploadItems (stream Item) returns (UploadSummary);                   // client stream
  rpc Chat (stream Message) returns (stream Message);                      // bidirectional
}
```

```go
// Go server — generated stub, implement the interface
type orderServer struct {
    pb.UnimplementedOrderServiceServer
    repo OrderRepository
}

func (s *orderServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.Order, error) {
    // Interceptors already handled auth, logging, recovery before this runs
    order, err := s.repo.Create(ctx, req)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "create order: %v", err)
    }
    return order, nil
}
```

### Key Weaknesses

- **Not browser-native** — needs `grpc-web` + Envoy/nginx proxy
- Binary protocol makes **manual debugging harder** (use grpcurl or reflection)
- **HTTP/2 only** — some load balancers / firewalls block H2
- Proto schema changes require **regenerating code** across all services
- **No built-in HTTP caching** — every call goes to the backend

### gRPC in Go — Production Setup

```go
// Interceptor chain (logging → auth → recovery → handler)
s := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        loggingInterceptor,
        authInterceptor,
        recoveryInterceptor,
    ),
    grpc.ChainStreamInterceptor(
        streamLoggingInterceptor,
        streamAuthInterceptor,
    ),
)

// Deadline propagation — ALWAYS set a deadline on outgoing calls
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
resp, err := client.CreateOrder(ctx, req)
if status.Code(err) == codes.DeadlineExceeded {
    // handle timeout separately from other errors
}
```

---

## REST Deep Dive

### When to Use REST

- **Public APIs** consumed by diverse clients (browsers, mobile, third-party)
- **Simple CRUD** operations that map cleanly to HTTP verbs
- When **HTTP caching** (CDN, ETags, Cache-Control) is valuable
- Teams that need minimal tooling or onboarding friction
- External partner integrations

### Key Strengths

```go
// stdlib net/http — no dependencies, universally understood
mux := http.NewServeMux()
mux.HandleFunc("GET /v1/orders/{id}", getOrder)
mux.HandleFunc("POST /v1/orders", createOrder)
mux.HandleFunc("PATCH /v1/orders/{id}", updateOrder)
mux.HandleFunc("DELETE /v1/orders/{id}", deleteOrder)

// HTTP semantics map naturally to CRUD
// GET    → read   (idempotent, cacheable)
// POST   → create (not idempotent)
// PUT    → replace (idempotent)
// PATCH  → partial update
// DELETE → remove (idempotent)
```

```go
// RFC 7807 Problem Details — standard error response
type ProblemDetail struct {
    Type     string `json:"type"`
    Title    string `json:"title"`
    Status   int    `json:"status"`
    Detail   string `json:"detail"`
    Instance string `json:"instance,omitempty"`
}

// GET /v1/orders/999 → 404
// {"type":"https://api.example.com/errors/not-found","title":"Order Not Found","status":404,"detail":"Order 999 does not exist"}
```

### Key Weaknesses

- **Over-fetching**: `/users/1` returns 20 fields when you need 3
- **Under-fetching**: need user + orders + addresses = 3 round trips
- **No streaming** built in (SSE is one-way; WebSocket breaks REST semantics)
- Versioning is awkward: `/v1/`, `/v2/`, or header-based — all have trade-offs
- No enforced schema by default (OpenAPI is optional/external)

### REST Versioning Comparison

```
/v1/orders   — URL path versioning (most common, cache-friendly, breaks REST purity)
Accept: application/vnd.api+json;version=2  — content-type negotiation (pure but complex)
API-Version: 2  — custom header (clean but not cacheable without Vary header)
```

---

## GraphQL Deep Dive

### When to Use GraphQL

- **BFF (Backend for Frontend)** — mobile + web need different shapes of the same data
- **Rapidly evolving frontends** where field requirements change weekly
- **Reducing round trips** for deeply nested data (social feed, dashboard)
- When the client knows best what it needs

### Key Strengths

```graphql
# Client asks for exactly what it needs — nothing more
query GetUserDashboard($userId: ID!) {
  user(id: $userId) {
    name
    email
    recentOrders(limit: 5) {
      id
      status
      total
      items {
        name
        quantity
      }
    }
  }
}
```

```go
// Go — gqlgen (code-first, type-safe)
// Define schema → gqlgen generates resolvers → implement them
type queryResolver struct{ *Resolver }

func (r *queryResolver) User(ctx context.Context, id string) (*model.User, error) {
    return r.userRepo.FindByID(ctx, id)
}

// DataLoader pattern — REQUIRED to prevent N+1
// Without DataLoader: 1 query for users + N queries for each user's orders
// With DataLoader: batch all order lookups into one IN clause
var orderLoader = dataloader.New(func(ctx context.Context, keys []string) []*dataloader.Result {
    orders, _ := r.orderRepo.FindByUserIDs(ctx, keys) // single batched query
    // map results back to keys...
})
```

### Key Weaknesses

- **N+1 problem** — resolvers run per-node; MUST use DataLoader or you'll hammer the DB
- **HTTP caching is broken** — everything is POST to `/graphql`; CDNs can't cache
- **Rate limiting complexity** — limit by query depth/complexity, not endpoint count
- **Attack surface** — deeply nested queries can be a DoS vector; add depth limits
- **Error model is non-standard** — always HTTP 200, errors in response body

```go
// Depth limiting — MUST add in production
directives.IntrospectionDirective(schema)
extension.FixedComplexityLimit(100)   // reject queries over complexity 100
extension.FixedQueryDepthLimit(10)    // reject queries deeper than 10 levels
```

### GraphQL vs REST Error Handling

```json
// REST — HTTP status codes are meaningful
// 404 Not Found, 400 Bad Request, 401 Unauthorized

// GraphQL — always HTTP 200, even on errors
{
  "data": { "user": null },
  "errors": [
    {
      "message": "user not found",
      "locations": [{"line": 2, "column": 3}],
      "path": ["user"],
      "extensions": { "code": "NOT_FOUND" }
    }
  ]
}
```

---

## Hybrid Architecture (Real-World Pattern)

Most production systems use **all three**:

```
┌─────────────────────────────────────────────────┐
│                  Clients                         │
│   Browser    Mobile App    Third-party           │
└─────┬─────────────┬───────────────┬─────────────┘
      │ REST/GraphQL│ GraphQL       │ REST
      ▼             ▼               ▼
┌─────────────────────────────────────────────────┐
│              API Gateway / BFF                   │
│   (REST public API + GraphQL for owned clients)  │
└─────┬───────────────────────────────────────────┘
      │ gRPC (internal — typed, fast, streaming)
      ▼
┌────────┐  ┌─────────────┐  ┌──────────────┐
│ Orders │  │  Inventory  │  │  Payments    │
│ svc    │  │  svc        │  │  svc         │
└────────┘  └─────────────┘  └──────────────┘
```

**Rule of thumb:**
- gRPC between microservices
- REST for the public-facing API
- GraphQL for the BFF that serves web/mobile

---

## Performance Numbers (Rough Benchmarks)

| Scenario | gRPC | REST/JSON | GraphQL |
|----------|------|-----------|---------|
| Payload size (same data) | ~3-10x smaller | baseline | baseline |
| Serialization speed | ~5-10x faster | baseline | baseline |
| Latency (internal, same DC) | ~1-3ms | ~3-8ms | ~3-10ms |
| Throughput (req/s, simple RPC) | ~100k-300k | ~50k-150k | ~30k-100k |

> Numbers are ballpark — actual results depend heavily on payload complexity,
> connection reuse, and infrastructure.

---

## Go Tooling

| Stack | Packages |
|-------|----------|
| gRPC | `google.golang.org/grpc`, `google.golang.org/protobuf`, `github.com/grpc-ecosystem/go-grpc-middleware` |
| REST | `net/http` (stdlib), `github.com/go-chi/chi/v5`, `github.com/gorilla/mux` |
| GraphQL | `github.com/99designs/gqlgen` (code-first), `github.com/graph-gophers/graphql-go` |
| OpenAPI | `github.com/deepmap/oapi-codegen` (REST → Go), `github.com/swaggo/swag` |

---

## Real-World Examples

| Company | Internal | External |
|---------|----------|---------|
| Google | gRPC everywhere | REST / gRPC (Cloud APIs) |
| GitHub | — | REST v3 + GraphQL v4 |
| Shopify | — | GraphQL (Storefront + Admin) |
| Stripe | — | REST (versioned, frozen APIs) |
| Twilio | — | REST |
| Netflix | gRPC between services | REST |
| Uber | — | Thrift (custom binary, pre-gRPC) → migrating to gRPC |

---

## Migration Paths

### REST → gRPC
1. Define `.proto` matching existing REST shapes
2. Run both REST and gRPC servers on different ports
3. Migrate internal callers first (service-by-service)
4. Keep REST for external clients behind an API gateway that translates

### REST → GraphQL
1. Build GraphQL schema that mirrors existing REST endpoints
2. Resolvers delegate to existing REST handlers or service layer
3. Run GraphQL alongside REST
4. Migrate BFF clients; keep REST for external/legacy

### Transcoding: gRPC ↔ REST
`grpc-gateway` generates a REST reverse-proxy from proto annotations:
```proto
import "google/api/annotations.proto";

service OrderService {
  rpc GetOrder (GetOrderRequest) returns (Order) {
    option (google.api.http) = {
      get: "/v1/orders/{id}"
    };
  }
}
```
One proto definition → both gRPC server AND REST HTTP/JSON server.

---

## Interview Cheat Sheet

**"When would you use gRPC over REST?"**
> Internal microservices, streaming data, polyglot environments, when you need strict typed contracts and performance matters. Not for public browser-facing APIs without a gateway.

**"What's the N+1 problem in GraphQL?"**
> Each resolver runs independently, so fetching 10 users and their orders fires 1 + 10 = 11 queries. DataLoader batches child queries into a single `WHERE id IN (...)` call.

**"Why is caching hard in GraphQL?"**
> Everything is a POST to `/graphql`. HTTP caches (CDN, proxy) key on URL + method, so they can't cache POST bodies. You need application-level caching (persisted queries + CDN, or a Redis layer in resolvers).

**"What's the difference between gRPC streaming and REST SSE?"**
> gRPC has 4 streaming modes (all bidirectional-capable) over HTTP/2 multiplexed streams, server-controlled. REST SSE is HTTP/1.1 compatible, server-to-client only, and simpler to implement but far more limited.
