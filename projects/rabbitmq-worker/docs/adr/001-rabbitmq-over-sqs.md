# ADR-001: RabbitMQ over SQS for Message Queue Learning

**Status:** Accepted

## Decision

Use RabbitMQ (AMQP) instead of AWS SQS for this learning project.

## Rationale

| Concern | AWS SQS | RabbitMQ |
|---|---|---|
| AMQP protocol | No (proprietary HTTP API) | Yes — industry-standard protocol |
| Dead Letter Exchange | Via separate DLQ + redrive policy | Native DLX with `x-dead-letter-exchange` header |
| Prefetch / QoS | No (visibility timeout only) | `channel.Qos(prefetch, 0, false)` |
| Manual ack | No (visibility timeout) | `msg.Ack(false)` / `msg.Nack(false, requeue)` |
| Local dev | Requires AWS credentials / LocalStack | `docker run rabbitmq:3-management` |
| Exchange types | None (queue-only) | direct, fanout, topic, headers |
| Learning value | AWS-specific patterns | Universal AMQP patterns (used by Celery, Spring AMQP, etc.) |

## Consequences

- RabbitMQ requires Docker for local development; SQS would work with AWS credentials.
- The AMQP patterns learned here (DLX, prefetch, manual ack) transfer directly to other AMQP brokers (ActiveMQ, Azure Service Bus AMQP mode).
- For production AWS workloads, SQS + Lambda or SQS + ECS workers are the idiomatic choice.
