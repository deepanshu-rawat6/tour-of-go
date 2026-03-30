# ADR-001: NATS JetStream over Kafka

**Status:** Accepted

## Decision

Use NATS JetStream instead of Apache Kafka.

## Rationale

| Concern | Kafka | NATS JetStream |
|---|---|---|
| Operational complexity | High (ZooKeeper/KRaft, brokers) | Low (single binary) |
| Go client quality | `confluent-kafka-go` (CGo) | `nats.go` (pure Go) |
| Persistence | Yes (log-based) | Yes (JetStream) |
| Exactly-once | Complex (transactions) | Built-in (dedup window) |
| Learning focus | Kafka-specific concepts | Go concurrency patterns |

For a learning project, NATS lets us focus on the Go patterns (channels, goroutines, context) rather than Kafka operational concerns.
