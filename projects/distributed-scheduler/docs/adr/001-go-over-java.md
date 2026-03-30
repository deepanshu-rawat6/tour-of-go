# ADR-001: Go over Java/Spring Boot for the Job Scheduler

**Status:** Accepted  
**Date:** 2026-03-31

## Context

The original job scheduler is a Spring Boot 3.4.3 / Java 21 application. We are porting it to Go as a learning exercise and to explore the architectural differences.

## Decision

Reimplement the scheduler in Go.

## Rationale

| Concern | Java/Spring Boot | Go |
|---|---|---|
| Concurrency model | ThreadPoolExecutor (OS threads, ~1MB each) | Goroutines (M:N scheduled, ~2KB each) |
| Startup time | 5-15 seconds (JVM warmup, Spring context) | <100ms (single binary) |
| Memory footprint | 256MB+ (JVM heap + Spring overhead) | 20-50MB |
| Dependency injection | Spring IoC container (annotation magic) | Manual constructor injection (explicit) |
| Scheduled tasks | `@Scheduled` annotation | `time.Ticker` + goroutine (explicit lifecycle) |
| Distributed lock | Redisson (heavy client) | `go-redis` + SET NX (minimal) |
| Embedded search | Hibernate Search + Lucene | Bleve (pure Go, same architecture) |
| Deployment | Fat JAR + JVM | Single static binary |
| Observability | Micrometer + Prometheus | `prometheus/client_golang` |

## Key Go Idioms That Replace Spring Patterns

- `@Service` singleton → struct with constructor, wired in `main.go`
- `@Scheduled` → `time.Ticker` goroutine with `context.Context` cancellation
- `ThreadPoolExecutor` → bounded channel semaphore
- `ConcurrentHashMap.newKeySet()` → `sync.Map`
- `synchronized` block → `sync.Mutex` / `sync.RWMutex`
- `AtomicInteger` → `atomic.Int64`
- `@PreDestroy` → `signal.Notify(SIGTERM)` + context cancellation

## Consequences

- No framework magic — every dependency is explicit in `main.go`
- Easier to understand the startup sequence (no annotation scanning)
- Context propagation is explicit (every function takes `ctx context.Context`)
- No reflection-based DI — compile-time errors instead of runtime panics
