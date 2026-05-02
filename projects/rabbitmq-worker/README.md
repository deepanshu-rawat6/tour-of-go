# rabbitmq-worker

A task worker system demonstrating **RabbitMQ** (AMQP) with durable exchanges, dead letter exchanges (DLX), QoS prefetch, manual acknowledgement, and graceful shutdown.

---

## Architecture

```mermaid
graph LR
    P[Producer\ncmd/producer] -->|publish persistent| EX[tasks\ndirect exchange]
    EX -->|routing key: task| Q[tasks.queue\ndurable + DLX binding]
    Q -->|prefetch=5\nautoAck=false| W1[Worker 1]
    Q --> W2[Worker 2]
    Q --> W3[Worker 3]
    W1 -->|success| ACK[Ack]
    W2 -->|fail retries lt 3| NACK_R[Nack requeue]
    W3 -->|fail retries ≥ 3| NACK_D[Nack no-requeue]
    NACK_R -->|back to queue| Q
    NACK_D --> DLX[tasks.dlx\nfanout exchange]
    DLX --> DLQ[tasks.dlq\ndead letter queue]
```

## Message Lifecycle

```mermaid
sequenceDiagram
    participant P as Producer
    participant Q as tasks.queue
    participant W as Worker
    participant DLX as tasks.dlx
    participant DLQ as tasks.dlq

    P->>Q: publish(task, persistent)
    Q->>W: deliver(task) [prefetch limit]
    alt success
        W->>Q: Ack(false)
    else transient failure (retries < 3)
        W->>Q: Nack(false, requeue=true)
        Q->>W: redeliver (x-death count++)
    else exhausted (retries ≥ 3)
        W->>Q: Nack(false, requeue=false)
        Q->>DLX: route dead letter
        DLX->>DLQ: store for inspection
    end
```

## Key Concepts

- **Durable exchange + queue** — survive broker restart; messages with `DeliveryMode: Persistent` survive too
- **Dead Letter Exchange (DLX)** — messages that exhaust retries are routed to `tasks.dlx` → `tasks.dlq` for inspection
- **QoS Prefetch** — `channel.Qos(5, 0, false)` limits unacked messages per consumer, providing backpressure
- **Manual ack** — `Ack(false)` on success, `Nack(false, requeue)` on failure — no message lost on crash
- **Graceful shutdown** — SIGTERM → stop consuming → drain in-flight → close connection

## Quick Start

```bash
# Start RabbitMQ
make docker-up

# Start consumer (in one terminal)
make run-consumer

# Publish 10 tasks (in another terminal)
make run-producer

# Inspect queues at http://localhost:15672 (guest/guest)
# tasks.queue — active messages
# tasks.dlq   — failed messages after 3 retries
```

## Docs

- [`docs/deep-dive.md`](./docs/deep-dive.md)
- [`docs/adr/001-rabbitmq-over-sqs.md`](./docs/adr/001-rabbitmq-over-sqs.md)
