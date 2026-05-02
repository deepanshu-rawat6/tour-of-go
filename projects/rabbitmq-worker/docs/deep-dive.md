# rabbitmq-worker: Deep Dive

## AMQP Topology

```
Producer
  │
  └─► tasks (direct exchange, durable)
          │
          └─► tasks.queue (durable, x-dead-letter-exchange=tasks.dlx)
                    │
                    ├─► Consumer Workers (prefetch=5, manual ack)
                    │         │
                    │         ├─► success: Ack(false)
                    │         ├─► transient fail (retries < 3): Nack(false, requeue=true)
                    │         └─► exhausted (retries ≥ 3): Nack(false, requeue=false)
                    │                                              │
                    └─────────────────────────────────────────────┘
                                                                   │
                                                    tasks.dlx (fanout exchange, durable)
                                                                   │
                                                    tasks.dlq (durable) ← inspect failed tasks
```

---

## Prefetch / QoS

```go
ch.Qos(prefetch, 0, false)
```

Without prefetch, RabbitMQ pushes all queued messages to the consumer at once. With `prefetch=5`, the broker holds back messages until the consumer has fewer than 5 unacked messages. This provides natural backpressure.

| prefetch | Effect |
|---|---|
| 0 | Unlimited push — consumer can be overwhelmed |
| 1 | Strict round-robin — one message at a time per consumer |
| 5 | Balanced — 5 in-flight per consumer, good throughput |

---

## Manual Acknowledgement

```go
// Success — remove from queue
d.Ack(false)

// Transient failure — put back in queue for retry
d.Nack(false, true)  // requeue=true

// Exhausted retries — route to DLX (no requeue)
d.Nack(false, false) // requeue=false → DLX
```

The `false` first argument means "only this message" (not bulk ack).

---

## Dead Letter Exchange (DLX)

When a message is nacked with `requeue=false`, RabbitMQ routes it to the exchange specified in the queue's `x-dead-letter-exchange` argument. The DLX is a fanout exchange that routes all dead letters to `tasks.dlq`.

The `x-death` header is automatically added by RabbitMQ and contains the death count, reason, and original queue. The consumer reads this to determine retry count:

```go
deaths := d.Headers["x-death"].([]any)
count := deaths[0].(amqp.Table)["count"].(int64)
```

---

## Graceful Shutdown

```
SIGTERM received
  → signal.NotifyContext cancels ctx
  → consumer.Close() closes the AMQP channel
  → deliveries channel is closed by the broker client
  → worker goroutines exit their range loop
  → pool.Wait() blocks until all in-flight messages are acked/nacked
  → process exits cleanly
```

No messages are lost: in-flight messages that haven't been acked are requeued by RabbitMQ when the connection closes (AMQP's "at-least-once" guarantee).
