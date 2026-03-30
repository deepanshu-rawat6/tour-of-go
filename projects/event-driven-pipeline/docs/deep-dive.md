# Event-Driven Pipeline: Deep Dive

A NATS JetStream-based event processing pipeline with exactly-once semantics, circuit breaking, and dead letter queue handling.

---

## Architecture

```mermaid
graph LR
    P[Producer] -->|publish| NATS[NATS JetStream\nevents.orders]
    NATS -->|pull| C[Consumer]
    C -->|submit| BUF[Bounded Buffer\nchan *Event]
    BUF -->|drain| W1[Worker 1]
    BUF -->|drain| W2[Worker 2]
    BUF -->|drain| W3[Worker 3]
    W1 --> IDEM[Redis\nIdempotency Check]
    W1 --> CB[Circuit Breaker]
    CB --> H[EventHandler]
    CB -->|open| DLQ[DLQ\nevents.dlq]
    W1 -->|max retries| DLQ
```

---

## Exactly-Once Processing

```mermaid
sequenceDiagram
    Consumer->>Redis: EXISTS idempotency:<key>
    alt key exists
        Redis-->>Consumer: 1 (already processed)
        Note over Consumer: Skip — duplicate event
    else key absent
        Redis-->>Consumer: 0
        Consumer->>Handler: Handle(event)
        Handler-->>Consumer: success
        Consumer->>Redis: SET idempotency:<key> 1 EX 86400
    end
```

The idempotency key is set by the **producer** (e.g., `order-42`). If the consumer crashes after processing but before ACKing, NATS redelivers the event. The idempotency check prevents double-processing.

---

## Circuit Breaker States

```mermaid
stateDiagram-v2
    [*] --> Closed
    Closed --> Open : failures >= threshold
    Open --> HalfOpen : retryAfter elapsed
    HalfOpen --> Closed : success
    HalfOpen --> Open : failure
```

When the circuit is Open, events are immediately sent to the DLQ instead of waiting for timeouts.

---

## Backpressure

The bounded buffer channel (`chan *Event, bufferSize`) provides backpressure:
- If the buffer is full, `Submit()` returns an error
- The consumer NACKs the NATS message
- NATS redelivers after the NAK delay
- This prevents memory exhaustion under load

See [ADR-003](./adr/003-backpressure-design.md).

---

## OTel Trace Propagation

The trace ID travels through NATS message headers:

```
Producer:
  msg.Header.Set("traceparent", span.SpanContext().TraceID().String())

Consumer:
  traceID := msg.Headers().Get("traceparent")
  event.TraceID = traceID
  // Use traceID to continue the trace in the handler
```

This connects to the `projects/otel-tracing/` project — the same W3C TraceContext propagation pattern.
