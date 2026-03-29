# Go Internals & System Design: Master Table of Contents

Welcome to the deep-dive series. This guide is organized sequentially to take you from understanding how Go works under the hood to designing massive, high-throughput distributed systems.

---

## 🟢 Phase 1: Go Internals & Runtime Mechanics
*Understand the language "magic" and avoid common pitfalls.*

1.  [**Go Quirks & Twisters**](./go-internals/quirks/README.md) - Nil interfaces, variable shadowing, and slice capacity traps.
2.  [**Deep Dive into `defer`**](./go-internals/defer/README.md) - Rules of defer, LIFO execution, and resource cleanup.
3.  [**Understanding `fmt.Println()`**](./go-internals/fmt.Println()/README.md) - Multiple return values and reliable logging.
4.  [**Expert Runtime Deep Dive**](./go-internals/expert-deep-dive/README.md) - G-M-P Scheduler, Tricolor GC, Netpoller, and CPU Cache lines.
5.  [**Reflection & Type Systems**](./go-internals/reflection/README.md) - Building generic libraries and understanding `reflect`.
6.  [**Concurrency Orchestration**](./go-internals/go-internals/concurrency-deep-dive/README.md) - `errgroup`, `sync.Pool`, and Load Shedding.

---

## 🔵 Phase 2: Idiomatic Design Patterns
*Learn to write clean, testable, and maintainable Go code.*

1.  [**Basic Go Patterns**](./design-patterns/patterns/README.md) - Functional Options, Generators, and Worker Pools.
2.  [**Concurrency Patterns**](./design-patterns/concurrency-patterns/README.md) - Pipelines and Fan-out/Fan-in.
3.  [**Industry-Standard Patterns**](./design-patterns/industry-patterns/README.md) - Middleware, Strategy, and Circuit Breakers.
4.  [**Error Handling Mastery**](./design-patterns/error-handling/README.md) - Context wrapping, `%w`, and custom error types.
5.  [**Engineering Best Practices**](./design-patterns/additional-patterns/README.md) - Dependency Injection, Observer, and Factory patterns.

---

## 🔴 Phase 3: System Design & Platform Ops
*Architecting for scale, reliability, and high throughput.*

1.  [**Distributed Systems Design**](./system-design/distributed-systems/README.md) - Idempotency, Leasing, and Dead Letter Queues.
2.  [**High-Throughput Architecture**](./system-design/high-throughput-systems/README.md) - Sharding, CQRS, WAL, and Batching.
3.  [**Rate Limiting Deep Dive**](./system-design/rate-limiting-deep-dive/README.md) - Implementing Token Buckets, Leaky Buckets, and Sliding Windows.
4.  [**Zero-Downtime Deployment**](./system-design/zero-downtime/README.md) - Graceful draining, SIGTERM, and K8s lifecycle.
5.  [**Go for Platform Ops & SRE**](./system-design/platform-ops/README.md) - Kubernetes Operators, System Signals, and Prometheus Observability.

---

## 🚀 Recommended Learning Path
If you are preparing for a **Platform Engineering** or **Senior Backend** role, I recommend this order:
1.  **Internals** (Phase 1) to master the runtime.
2.  **Concurrency Patterns** (Phase 2) to master orchestration.
3.  **Platform Ops** (Phase 3) to understand K8s and OS interactions.
4.  **System Design** (Phase 3) to master high-level architecture.
