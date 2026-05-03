# 06-message-queue

An in-memory pub/sub message broker with a TCP server interface.

## Architecture

```mermaid
graph LR
    P[Producer\ncmd/producer] -->|PUB events msg\n| TCP[TCP Server\n:9001]
    TCP --> B[Broker\ntopics map]
    B -->|fan-out| CH1[chan Message\nsubscriber 1]
    B --> CH2[chan Message\nsubscriber 2]
    CH1 -->|MSG events msg| C1[Consumer 1]
    CH2 -->|MSG events msg| C2[Consumer 2]
```

## Protocol

```
# Publish
PUB <topic> <payload>\n  →  OK\n

# Subscribe
SUB <topic>\n            →  OK\n
                         ←  MSG <topic> <payload>\n  (for each message)
```

## Quick Start

```bash
make run-server    # start broker on :9001
make run-consumer  # subscribe to "events"
make run-producer  # publish 5 messages
```

## Docs

- [`docs/deep-dive.md`](./docs/deep-dive.md)
