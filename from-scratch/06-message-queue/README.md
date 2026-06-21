# 06-message-queue

An in-memory pub/sub message broker with a TCP server interface.

## Architecture

```mermaid
graph LR
    P[Producer<br>cmd/producer] -->|PUB events msg<br>| TCP[TCP Server<br>:9001]
    TCP --> B[Broker<br>topics map]
    B -->|fan-out| CH1[chan Message<br>subscriber 1]
    B --> CH2[chan Message<br>subscriber 2]
    CH1 -->|MSG events msg| C1[Consumer 1]
    CH2 -->|MSG events msg| C2[Consumer 2]
```

## Protocol

```
# Publish
PUB <topic> <payload><br>  →  OK<br>

# Subscribe
SUB <topic><br>            →  OK<br>
                         ←  MSG <topic> <payload><br>  (for each message)
```

## Quick Start

```bash
make run-server    # start broker on :9001
make run-consumer  # subscribe to "events"
make run-producer  # publish 5 messages
```

## Docs

- [`docs/deep-dive.md`](./docs/deep-dive.md)
