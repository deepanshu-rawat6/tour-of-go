# 06-message-queue: Deep Dive

## Pub/Sub Model

Publishers and subscribers are decoupled — they don't know about each other:

```mermaid
graph LR
    P1[Producer A] -->|PUB orders| BROKER[Broker]
    P2[Producer B] -->|PUB orders| BROKER
    BROKER -->|fan-out| S1[Subscriber 1\norders channel]
    BROKER -->|fan-out| S2[Subscriber 2\norders channel]
    BROKER -->|NOT delivered| S3[Subscriber 3\npayments channel]
```

## Broker Internals

```mermaid
graph TD
    BROKER[Broker\nsubs map] --> T1[topic: orders\nslice of channels]
    BROKER --> T2[topic: payments\nslice of channels]
    T1 --> CH1[chan Message\nbuf=64]
    T1 --> CH2[chan Message\nbuf=64]
    T2 --> CH3[chan Message\nbuf=64]
```

`Subscribe` appends a new buffered channel to the topic's slice. `Publish` iterates the slice and sends to each channel with a non-blocking `select` — slow subscribers drop messages rather than blocking the publisher.

## TCP Protocol

```mermaid
sequenceDiagram
    participant P as Producer
    participant S as Server
    participant C as Consumer

    C->>S: SUB events\n
    S-->>C: OK\n

    P->>S: PUB events hello-world\n
    S-->>P: OK\n
    S-->>C: MSG events hello-world\n

    P->>S: PUB events second-message\n
    S-->>P: OK\n
    S-->>C: MSG events second-message\n
```

## Fan-out with Goroutines

When a consumer subscribes, the server spawns a goroutine to forward messages from the broker channel to the TCP connection:

```mermaid
graph LR
    BROKER[Broker\nchan Message] -->|range ch| FWD[forward goroutine\nper subscription]
    FWD -->|fmt.Fprintf| CONN[net.Conn\nTCP to consumer]
```

## Slow Subscriber Handling

```mermaid
graph TD
    PUB[Publish] --> ITER[iterate subscribers]
    ITER --> SELECT{select}
    SELECT -->|ch ← msg\nchannel has space| DELIVER[delivered]
    SELECT -->|default\nchannel full| DROP[dropped\nno blocking]
```

This prevents one slow consumer from blocking all other consumers or the publisher.
