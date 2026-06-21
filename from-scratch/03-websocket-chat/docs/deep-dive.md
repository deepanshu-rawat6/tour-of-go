# 03-websocket-chat: Deep Dive

## WebSocket Upgrade

WebSocket starts as an HTTP request and upgrades to a persistent bidirectional connection:

```mermaid
sequenceDiagram
    participant C as Browser
    participant S as Server

    C->>S: GET /ws HTTP/1.1<br>Upgrade: websocket<br>Sec-WebSocket-Key: ...
    S-->>C: 101 Switching Protocols<br>Upgrade: websocket<br>Sec-WebSocket-Accept: ...
    Note over C,S: TCP connection stays open
    C->>S: Text frame: {"text":"hello"}
    S-->>C: Text frame: alice: hello
    S-->>C: Text frame: alice: hello  (broadcast to all)
```

## Hub Pattern

The hub is a single goroutine that owns all client state. No locks needed on the hot path:

```mermaid
graph TD
    C1[Client 1<br>goroutine] -->|join general| HUB[Hub goroutine<br>select loop]
    C2[Client 2<br>goroutine] -->|join general| HUB
    C3[Client 3<br>goroutine] -->|join other| HUB
    C1 -->|publish Message| HUB
    HUB -->|broadcast to general| C1
    HUB -->|broadcast to general| C2
    HUB -->|NOT broadcast to other| C3
```

## Per-Client Goroutines

Each WebSocket connection spawns two goroutines — one for reading, one for writing:

```mermaid
graph LR
    CONN[net.Conn<br>WebSocket] --> RG[Reader goroutine<br>conn.ReadMessage<br>blocks on network]
    CONN --> WG[Writer goroutine<br>conn.WriteMessage<br>drains send channel]
    RG -->|hub.Publish| HUB[Hub]
    HUB -->|client.send chan| WG
```

The writer goroutine uses a buffered `send chan []byte`. If the channel is full (slow client), the message is dropped rather than blocking the hub.

## Room Isolation

```mermaid
graph TD
    HUB[Hub<br>rooms map] --> R1[room: general<br>set of clients]
    HUB --> R2[room: engineering<br>set of clients]
    HUB --> R3[room: random<br>set of clients]
    MSG[Message<br>Room: general] --> HUB
    HUB -->|only sends to| R1
```

## Graceful Disconnect

```mermaid
sequenceDiagram
    participant C as Client
    participant RG as Reader goroutine
    participant HUB as Hub

    C->>RG: Close tab / network drop
    Note over RG: ReadMessage returns error
    RG->>HUB: hub.Leave(client)
    HUB->>HUB: delete client from room
    Note over RG: defer conn.Close()
```
