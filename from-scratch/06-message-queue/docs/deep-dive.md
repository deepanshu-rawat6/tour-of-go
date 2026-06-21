# 06-message-queue: Deep Dive

## Pub/Sub Model

Publishers and subscribers are decoupled — they don't know about each other:

```mermaid
graph LR
    P1[Producer A] -->|PUB orders| BROKER[Broker]
    P2[Producer B] -->|PUB orders| BROKER
    BROKER -->|fan-out| S1[Subscriber 1<br>orders channel]
    BROKER -->|fan-out| S2[Subscriber 2<br>orders channel]
    BROKER -->|NOT delivered| S3[Subscriber 3<br>payments channel]
```

## Broker Internals

```mermaid
graph TD
    BROKER[Broker<br>subs map] --> T1[topic: orders<br>slice of channels]
    BROKER --> T2[topic: payments<br>slice of channels]
    T1 --> CH1[chan Message<br>buf=64]
    T1 --> CH2[chan Message<br>buf=64]
    T2 --> CH3[chan Message<br>buf=64]
```

`Subscribe` appends a new buffered channel to the topic's slice. `Publish` iterates the slice and sends to each channel with a non-blocking `select` — slow subscribers drop messages rather than blocking the publisher.

## TCP Protocol

```mermaid
sequenceDiagram
    participant P as Producer
    participant S as Server
    participant C as Consumer

    C->>S: SUB events<br>
    S-->>C: OK<br>

    P->>S: PUB events hello-world<br>
    S-->>P: OK<br>
    S-->>C: MSG events hello-world<br>

    P->>S: PUB events second-message<br>
    S-->>P: OK<br>
    S-->>C: MSG events second-message<br>
```

## Fan-out with Goroutines

When a consumer subscribes, the server spawns a goroutine to forward messages from the broker channel to the TCP connection:

```mermaid
graph LR
    BROKER[Broker<br>chan Message] -->|range ch| FWD[forward goroutine<br>per subscription]
    FWD -->|fmt.Fprintf| CONN[net.Conn<br>TCP to consumer]
```

## Slow Subscriber Handling

```mermaid
graph TD
    PUB[Publish] --> ITER[iterate subscribers]
    ITER --> SELECT{select}
    SELECT -->|ch ← msg<br>channel has space| DELIVER[delivered]
    SELECT -->|default<br>channel full| DROP[dropped<br>no blocking]
```

This prevents one slow consumer from blocking all other consumers or the publisher.
