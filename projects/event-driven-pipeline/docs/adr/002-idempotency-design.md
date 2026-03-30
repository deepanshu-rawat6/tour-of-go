# ADR-002: Redis SET NX for Idempotency

**Status:** Accepted

## Decision

Use Redis `SET key 1 NX EX <ttl>` for idempotency tracking.

## Why Redis over DB?

- O(1) lookup vs SQL query
- TTL-based automatic cleanup (no need for a cleanup job)
- Atomic SET NX prevents race conditions between concurrent consumers

## Key Design

`idempotency:<idempotencyKey>` — set by the producer, checked by the consumer.

The producer is responsible for generating stable idempotency keys (e.g., `order-42`). This is the same pattern as Stripe's Idempotency-Key header.
