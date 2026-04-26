# service-mesh-sidecar

A TCP reverse proxy sidecar with token bucket rate limiting, a 3-state circuit breaker, Prometheus metrics, and health checks — demonstrating how service mesh proxies (Envoy, Linkerd) work under the hood.

---

## Architecture

```mermaid
graph LR
    Client -->|TCP| PROXY[Sidecar Proxy\n:8080]
    PROXY --> RL[Token Bucket\nRate Limiter\nper client IP]
    RL -->|allowed| CB[Circuit Breaker\nClosed / Open / Half-Open]
    CB -->|closed| UP[Upstream Service\n:9090]
    CB -->|open| ERR[drop connection]
    RL -->|rate limited| DROP[drop connection]
    PROXY --> M[Prometheus Metrics\n:9091 /metrics]
    PROXY --> H[Health Check\n:9091 /health]
```

## Circuit Breaker States

```mermaid
stateDiagram-v2
    [*] --> Closed
    Closed --> Open : failures >= threshold
    Open --> HalfOpen : retryAfter elapsed
    HalfOpen --> Closed : success
    HalfOpen --> Open : failure
```

## Key Concepts

- **Token Bucket** — per-IP rate limiting. Each IP gets a bucket refilled at a fixed rate. Requests that exceed the bucket are dropped.
- **Circuit Breaker** — protects the upstream from cascading failures. Opens after N failures, probes with one request after a cooldown.
- **Bidirectional proxy** — `io.Copy` in two goroutines. On Linux, uses `splice(2)` for zero-copy kernel-level transfer.
- **Goroutine-per-connection** — cheap in Go (2KB stack). Each accepted TCP connection gets its own goroutine.

## Quick Start

```bash
make run
# Proxy listens on :8080, metrics on :9091
curl http://localhost:9091/health
curl http://localhost:9091/metrics
```

## Docs

- [`docs/deep-dive.md`](./docs/deep-dive.md)
