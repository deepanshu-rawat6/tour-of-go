# Go Internals & System Design: Master Table of Contents

Welcome to the deep-dive series. This guide is organized sequentially to take you from understanding how Go works under the hood to designing massive, high-throughput distributed systems.

---

## 🟢 Phase 1: Go Internals & Runtime Mechanics
*Understand the language "magic" and avoid common pitfalls.*

1.  [**Go Quirks & Twisters**](./go-internals/quirks/README.md) - Nil interfaces, variable shadowing, and slice capacity traps.
2.  [**Deep Dive into `defer`**](./go-internals/defer/README.md) - Rules of defer, LIFO execution, and resource cleanup.
3.  [**Interface Memory Layout**](./go-internals/interfaces/README.md) - Detailed look at `itab` and data pointers; understanding the cost of abstraction.
4.  [**Expert Runtime Deep Dive**](./go-internals/expert-deep-dive/README.md) - G-M-P Scheduler, Tricolor GC, Netpoller, and CPU Cache lines.
5.  [**Reflection & Type Systems**](./go-internals/reflection/README.md) - Using `reflect` and `unsafe` for generic libraries and ORMs.
6.  [**cgo & FFI (Foreign Function Interface)**](./go-internals/cgo/README.md) - The cost of calling C code and overhead in drivers (sqlite3, networking).
7.  [**Assembly in Go (Plan9)**](./go-internals/assembly/README.md) - Talking directly to the CPU with Go's unique assembly syntax.
8.  [**Concurrency Orchestration**](./go-internals/concurrency-deep-dive/README.md) - `errgroup`, `sync.Pool`, and Load Shedding.
9.  [**Production Go**](./go-internals/production-go/README.md) - HTTP timeouts, pprof profiling, escape analysis, `sync/atomic`, lock-free patterns, graceful shutdown.
10. [**Testing Patterns**](./go-internals/testing-patterns/README.md) - Table-driven tests, mocks, testcontainers-go, benchmarks, fuzz testing.
11. [**Module Patterns**](./go-internals/module-patterns/README.md) - `go.work` workspaces, `internal/` packages, `replace` directives, monorepo strategies.

---

## 🔵 Phase 2: Idiomatic Design Patterns
*Learn to write clean, testable, and maintainable Go code.*

1.  [**Basic Go Patterns**](./design-patterns/patterns/README.md) - Functional Options, Generators, and Worker Pools.
2.  [**Error Handling Mastery**](./design-patterns/error-handling/README.md) - Beyond `if err != nil`; using `errors.Is`/`As` and carrying "Platform Context" (Retryable vs. Fatal).
3.  [**Plugin Architecture**](./design-patterns/plugins/README.md) - Using `hashicorp/go-plugin` (RPC) or WASM for extensible systems.
4.  [**Data Access Object (DAO) & Repository**](./design-patterns/dao/README.md) - Layering business logic to swap databases (MySQL vs. MongoDB) seamlessly.
5.  [**Industry-Standard Patterns**](./design-patterns/industry-patterns/README.md) - Middleware, Strategy, and Circuit Breakers.
6.  [**Engineering Best Practices**](./design-patterns/additional-patterns/README.md) - Dependency Injection, Observer, and Factory patterns.
7.  [**Concurrency Patterns**](./design-patterns/concurrency-patterns/README.md) - Pipeline, fan-out/fan-in, and context propagation.
8.  [**The Twelve-Factor App**](./design-patterns/twelve-factor/README.md) - Cloud-native principles applied to Go: env config, stateless processes, port binding, disposability.
9.  [**API Design**](./design-patterns/api-design/README.md) - REST best practices, versioning, pagination (cursor vs offset), error responses (RFC 7807), OpenAPI.

---

## 🔴 Phase 3: System Design & Platform Ops
*Architecting for scale, reliability, and high throughput.*

**Fundamentals (start here):**

1.  [**Networking Fundamentals**](./system-design/networking/README.md) - DNS resolution, TCP/TLS handshakes, HTTP/2 multiplexing, mTLS certificate rotation.
2.  [**Consistency Models**](./system-design/consistency-models/README.md) - Strong vs eventual vs causal, linearizability, quorum reads/writes, CAP theorem.
3.  [**Database Internals**](./system-design/database-internals/README.md) - B-tree, LSM-tree storage engines, query optimization, connection pooling, sharding strategies.
4.  [**Database Migrations**](./system-design/database-migrations/README.md) - Zero-downtime schema changes, expand-contract pattern, golang-migrate.
5.  [**Auth Deep Dive**](./system-design/auth-deep-dive/README.md) - OAuth2 flows, PKCE, JWT best practices, refresh token rotation, session management.
6.  [**Message Queue Patterns**](./system-design/message-queue-patterns/README.md) - Kafka vs NATS vs SQS vs RabbitMQ, ordering, exactly-once, DLQ, backpressure.
7.  [**Capacity Planning**](./system-design/capacity-planning/README.md) - Back-of-envelope calculations, Little's Law, QPS/storage/bandwidth estimation.

**Resilience & Performance:**

8.  [**Rate Limiting Deep Dive**](./system-design/rate-limiting-deep-dive/README.md) - Implementing Token Buckets, Leaky Buckets, and Sliding Windows.
9.  [**Backpressure & Flow Control**](./system-design/backpressure/README.md) - Adaptive load shedding, bounded channels, semaphores, AIMD, context timeouts.
10. [**High-Throughput Architecture**](./system-design/high-throughput-systems/README.md) - Sharding, CQRS, WAL, and Batching.
11. [**Zero-Downtime Deployment**](./system-design/zero-downtime/README.md) - Graceful draining, SIGTERM, and K8s Liveness/Readiness probes.

**Observability & Operations:**

12. [**Observability Guide**](./system-design/observability-guide/README.md) - Structured logging with `slog`, trace ID propagation, Prometheus metrics, log levels.
13. [**Distributed Tracing (OpenTelemetry)**](./system-design/tracing/README.md) - Propagating Trace IDs across microservices to find bottlenecks.
14. [**Incident Response & SRE**](./system-design/incident-response/README.md) - SLI/SLO/SLA, error budgets, burn rate alerts, runbook templates, on-call practices.

**Infrastructure & Deployment:**

15. [**CI/CD for Go Services**](./system-design/cicd/README.md) - Multi-stage Docker, GitHub Actions, semantic versioning, GoReleaser, deployment strategies.
16. [**Terraform Basics**](./system-design/terraform-basics/README.md) - Module structure, remote state, workspaces, lifecycle rules, deploying Go services.
17. [**Helm Charts**](./system-design/helm-charts/README.md) - Chart anatomy, values, Go templates, environment overrides, CI/CD integration.

**Advanced (distributed systems & kernel):**

18. [**Service Discovery & Gossip Protocols**](./system-design/discovery/README.md) - How nodes find each other without a central DB (Consul/Serf).
19. [**Go for Platform Ops & SRE**](./system-design/platform-ops/README.md) - Kubernetes Operators, System Signals, and Prometheus Observability.
20. [**eBPF with Go**](./system-design/ebpf/README.md) - High-performance networking and security probes in the Linux Kernel.

---

## 🏃 Runnable Code

The theory above is backed by executable Go programs in [`runnable/`](./runnable/README.md):

```shell
go run ./more-internals/runnable/concurrency-patterns/   # Pipeline, fan-out/fan-in
go run ./more-internals/runnable/design-patterns/        # Functional options, circuit breaker, singleflight
go run ./more-internals/runnable/system-design/          # Token bucket + sliding window rate limiter
```

## 🚀 Projects (Runnable Platform Projects)

Put it all together with standalone mini-projects in [`../projects/`](../projects/):

| Project | Connects to |
|---------|-------------|
| [`grpc-service/`](../projects/grpc-service/) | Platform Ops & SRE |
| [`otel-tracing/`](../projects/otel-tracing/) | Distributed Tracing (OpenTelemetry) |
| [`k8s-controller/`](../projects/k8s-controller/) | Go for Platform Ops & SRE |
| [`secure-api/`](../projects/secure-api/) | SOLID, TDD, Immutability, Security (JWT/OAuth2/mTLS) |
| [`cache-service/`](../projects/cache-service/) | Caching Strategies (LRU, TTL, cache-aside, write-through, singleflight) |
| [`rabbitmq-worker/`](../projects/rabbitmq-worker/) | Message Queues (AMQP, DLX, prefetch, manual ack) |
| [`from-scratch/`](../projects/from-scratch/) | From-Scratch Series: TCP, HTTP, WebSocket, Rate Limiter, Load Balancer, MQ, Cache, Log Aggregator, Scheduler, URL Shortener |

---

## 🚀 Recommended Learning Path (In-Depth)
Follow this sequential roadmap to transition from a Go developer to a **Platform Engineer** or **Senior Backend Architect**.

---

### 🟢 Stage 1: Runtime Mastery (The Foundation)
*Before building distributed systems, you must understand the machine you are building on.*

1.  **Go Quirks & Memory Safety**: Master the "Gotchas" (nil interfaces, shadowing). *Goal: Write bug-free, predictable code.*
2.  **The `defer` Lifecycle**: Understand the LIFO execution and the performance difference between stack vs. heap allocation of defer records.
3.  **Interface Internals (`itab`)**: Deep-dive into how Go handles polymorphism. Understand why an interface is a pair of pointers and how dynamic dispatch works.
4.  **The G-M-P Scheduler**: Learn how Go manages thousands of goroutines with M:N scheduling. Master the concepts of work-stealing and preemption.
5.  **Garbage Collection (Tricolor Mark & Sweep)**: Understand the trade-offs of low-latency GC and how to minimize "Stop The World" (STW) pauses.
6.  **Reflection & Unsafe**: Learn when to break the type system to build high-performance tools (ORMs, Encoders) and the safety risks involved.
7.  **cgo & FFI Boundaries**: Understand the 50x cost of context-switching between Go and C. Learn to batch calls to minimize this overhead.
8.  **Plan9 Assembly**: Learn to read Go's assembly to verify compiler optimizations and write micro-optimized SIMD code.
9.  **Production Go**: HTTP timeout tuning, pprof profiling, escape analysis, `sync/atomic` for lock-free patterns, graceful multi-component shutdown.
10. **Testing Patterns**: Table-driven tests, interface mocking, testcontainers for integration tests, benchmarks, and fuzz testing.
11. **Module Patterns**: `go.work` workspaces, `internal/` package boundaries, `replace` directives, and monorepo versioning.

---

### 🔵 Stage 2: Resilient Architecture (The Idiomatic Series)
*Transition from "making it work" to "making it maintainable and resilient."*

12. **Advanced Concurrency Orchestration**: Master `errgroup` for lifecycle management and `sync.Pool` to reduce GC pressure in high-frequency paths.
13. **Error Handling Mastery**: Move beyond `if err != nil`. Implement error wrapping with `%w` and build custom error types that distinguish between "Retryable" and "Fatal" states.
14. **Functional Options & Configuration**: Use the Functional Options pattern to build clean, extensible APIs for your platform components.
15. **Plugin Architectures**: Learn to build extensible systems using gRPC/RPC (HashiCorp) or WASM, allowing users to extend your core binary safely.
16. **Data Layering (DAO/Repository)**: Decouple your domain logic from persistence. Learn to swap databases (e.g., SQL to Mongo) without touching a single line of business logic.
17. **Resiliency Patterns**: Implement Circuit Breakers, Retries with Exponential Backoff, and Load Shedding to protect your services from cascading failures.
18. **The Twelve-Factor App**: Apply cloud-native principles — env-based config, stateless processes, port binding, disposability, dev/prod parity.
19. **API Design**: REST versioning, cursor-based pagination, RFC 7807 error responses, OpenAPI documentation, rate limit headers.

---

### 🔴 Stage 3: Platform Engineering & SRE (The Scale Series)
*Architecting for the cloud-native era and high-throughput environments.*

20. **Networking Fundamentals**: DNS resolution, TCP/TLS handshakes, HTTP/2 multiplexing, connection pooling, mTLS certificate rotation.
21. **Consistency Models**: Strong vs eventual vs causal consistency, linearizability, quorum reads/writes, CAP theorem trade-offs.
22. **Database Internals**: B-tree and LSM-tree storage engines, query optimization with EXPLAIN, connection pooling, sharding strategies.
23. **Auth & Security**: OAuth2 + PKCE flows, JWT best practices, refresh token rotation, session management, RBAC.
24. **Message Queue Patterns**: Kafka vs NATS vs SQS vs RabbitMQ, delivery semantics, ordering guarantees, DLQ, backpressure.
25. **Capacity Planning**: Back-of-envelope calculations, Little's Law, QPS/storage/bandwidth estimation for system design.
26. **Rate Limiting & Backpressure**: Token Buckets, Sliding Windows, adaptive load shedding (AIMD), bounded channels, context timeouts.
27. **High-Throughput Optimizations**: Master Write-Ahead Logging (WAL), CQRS, Event Sourcing, and Sharding strategies.
28. **Observability**: Structured logging with `slog`, trace ID propagation, Prometheus metrics, distributed tracing with OpenTelemetry.
29. **Incident Response & SRE**: SLI/SLO/SLA definitions, error budgets, burn rate alerts, runbook design, on-call practices.
30. **Zero-Downtime Deployment**: Graceful connection draining, SIGTERM handling, K8s Liveness/Readiness probes, rolling updates.
31. **CI/CD**: Multi-stage Docker builds, GitHub Actions pipelines, GoReleaser, semantic versioning, canary deployments.
32. **Infrastructure as Code**: Terraform module structure, remote state, workspaces. Helm chart anatomy, values, templates.
33. **Service Discovery & Gossip**: Decentralized node discovery using SWIM/Gossip protocols (Serf/Consul).
34. **Operator Pattern & Controllers**: Extend Kubernetes by writing custom Controllers and Operators in Go.
35. **eBPF Observability**: Load probes into the Linux Kernel for deep-kernel networking, security auditing, and zero-overhead tracing.

