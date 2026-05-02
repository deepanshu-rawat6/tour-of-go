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

---

## 🔵 Phase 2: Idiomatic Design Patterns
*Learn to write clean, testable, and maintainable Go code.*

1.  [**Basic Go Patterns**](./design-patterns/patterns/README.md) - Functional Options, Generators, and Worker Pools.
2.  [**Error Handling Mastery**](./design-patterns/error-handling/README.md) - Beyond `if err != nil`; using `errors.Is`/`As` and carrying "Platform Context" (Retryable vs. Fatal).
3.  [**Plugin Architecture**](./design-patterns/plugins/README.md) - Using `hashicorp/go-plugin` (RPC) or WASM for extensible systems.
4.  [**Data Access Object (DAO) & Repository**](./design-patterns/dao/README.md) - Layering business logic to swap databases (MySQL vs. MongoDB) seamlessly.
5.  [**Industry-Standard Patterns**](./design-patterns/industry-patterns/README.md) - Middleware, Strategy, and Circuit Breakers.
6.  [**Engineering Best Practices**](./design-patterns/additional-patterns/README.md) - Dependency Injection, Observer, and Factory patterns.

---

## 🔴 Phase 3: System Design & Platform Ops
*Architecting for scale, reliability, and high throughput.*

1.  [**eBPF with Go**](./system-design/ebpf/README.md) - High-performance networking and security probes in the Linux Kernel.
2.  [**Service Discovery & Gossip Protocols**](./system-design/discovery/README.md) - How nodes find each other without a central DB (Consul/Serf).
3.  [**Zero-Downtime Deployment**](./system-design/zero-downtime/README.md) - Graceful draining, SIGTERM, and K8s Liveness/Readiness probes.
4.  [**Distributed Tracing (OpenTelemetry)**](./system-design/tracing/README.md) - Propagating Trace IDs across microservices to find bottlenecks.
5.  [**Rate Limiting Deep Dive**](./system-design/rate-limiting-deep-dive/README.md) - Implementing Token Buckets, Leaky Buckets, and Sliding Windows.
6.  [**High-Throughput Architecture**](./system-design/high-throughput-systems/README.md) - Sharding, CQRS, WAL, and Batching.
7.  [**Go for Platform Ops & SRE**](./system-design/platform-ops/README.md) - Kubernetes Operators, System Signals, and Prometheus Observability.

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

---

### 🔵 Stage 2: Resilient Architecture (The Idiomatic Series)
*Transition from "making it work" to "making it maintainable and resilient."*

9.  **Advanced Concurrency Orchestration**: Master `errgroup` for lifecycle management and `sync.Pool` to reduce GC pressure in high-frequency paths.
10. **Error Handling Mastery**: Move beyond `if err != nil`. Implement error wrapping with `%w` and build custom error types that distinguish between "Retryable" and "Fatal" states.
11. **Functional Options & Configuration**: Use the Functional Options pattern to build clean, extensible APIs for your platform components.
12. **Plugin Architectures**: Learn to build extensible systems using gRPC/RPC (HashiCorp) or WASM, allowing users to extend your core binary safely.
13. **Data Layering (DAO/Repository)**: Decouple your domain logic from persistence. Learn to swap databases (e.g., SQL to Mongo) without touching a single line of business logic.
14. **Resiliency Patterns**: Implement Circuit Breakers, Retries with Exponential Backoff, and Load Shedding to protect your services from cascading failures.

---

### 🔴 Stage 3: Platform Engineering & SRE (The Scale Series)
*Architecting for the cloud-native era and high-throughput environments.*

15. **eBPF Observability**: Use Go to load probes into the Linux Kernel. Master deep-kernel networking, security auditing, and zero-overhead tracing.
16. **Service Discovery & Gossip**: Understand how nodes find each other in a decentralized cluster using SWIM/Gossip protocols (Serf/Consul).
17. **Zero-Downtime Logic**: Master the Kubernetes lifecycle. Implement graceful connection draining and Liveness/Readiness probes that actually reflect system health.
18. **Distributed Tracing (OpenTelemetry)**: Learn to propagate Trace IDs across service boundaries to map request journeys and find 99th-percentile latency bottlenecks.
19. **Advanced Rate Limiting**: Implement Token Buckets and Sliding Windows to protect your APIs from "Thundering Herd" problems.
20. **High-Throughput Optimizations**: Master Write-Ahead Logging (WAL), CQRS, and Sharding strategies for building databases or high-speed message brokers.
21. **Operator Pattern & Controllers**: Learn to extend Kubernetes by writing custom Controllers and Operators in Go to manage complex infrastructure automatically.

