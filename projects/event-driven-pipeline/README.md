# event-driven-pipeline

An event processing pipeline with exactly-once semantics, circuit breaking, backpressure, retry with exponential backoff, and a dead letter queue — built on NATS JetStream.

---

## Architecture

```mermaid
graph LR
    P[Producer\ncmd/producer] -->|publish| N[NATS JetStream\nsubject: events.*]
    N -->|at-least-once delivery| C[Consumer\ncmd/consumer]
    C --> PROC[Processor]
    PROC -->|check| IDEM[Redis\nidempotency store]
    IDEM -->|duplicate| SKIP[skip + ack]
    IDEM -->|new| CB[Circuit Breaker\nClosed/Open/Half-Open]
    CB -->|closed| H[EventHandler\ndownstream call]
    H -->|success| MARK[mark processed\nRedis TTL 24h]
    H -->|fail × 3| DLQ[DLQ\nevents.dlq]
    CB -->|open| DLQ
```

## Processing Flow

```mermaid
sequenceDiagram
    participant N as NATS JetStream
    participant P as Processor
    participant R as Redis
    participant H as Handler
    participant D as DLQ

    N->>P: deliver event
    P->>R: IsProcessed(idempotencyKey)?
    alt already processed
        R-->>P: true
        P->>N: ack (skip)
    else new event
        R-->>P: false
        P->>H: Handle(event) [circuit breaker]
        alt success
            H-->>P: nil
            P->>R: MarkProcessed(key, 24h)
            P->>N: ack
        else fail after 3 retries
            P->>D: Publish to DLQ
            P->>N: ack
        end
    end
```

## Key Concepts

- **Exactly-once** — Redis idempotency key prevents duplicate processing even if NATS redelivers.
- **Circuit Breaker** — 3-state (Closed/Open/Half-Open). Opens after 5 failures, retries after 30s.
- **Backpressure** — bounded `chan *Event` buffer. `Submit` returns an error if full — signals the consumer to slow down.
- **Exponential backoff** — 100ms → 200ms → 400ms between retries before DLQ.

## Quick Start

```bash
docker-compose up -d   # starts NATS + Redis
go run ./cmd/consumer &
go run ./cmd/producer
```

## Docs

- [`docs/adr/001-nats-over-kafka.md`](./docs/adr/001-nats-over-kafka.md)
- [`docs/adr/002-idempotency-design.md`](./docs/adr/002-idempotency-design.md)
