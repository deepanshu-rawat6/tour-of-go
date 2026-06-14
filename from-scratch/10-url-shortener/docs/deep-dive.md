# 10-url-shortener: Deep Dive

## Full Request Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant RL as Rate Limiter\n04-rate-limiter
    participant H as HTTP Handler
    participant CACHE as RESP Cache\n07-distributed-cache
    participant MQ as Message Queue\n06-message-queue
    participant SCHED as Scheduler\n09-task-scheduler

    C->>RL: POST /shorten {"url":"https://..."}
    RL->>RL: token bucket Allow?
    alt rate limited
        RL-->>C: 429 Too Many Requests
    else allowed
        RL->>H: forward request
        H->>H: shortener.Code() → "abc123"
        H->>CACHE: SET abc123 https://... EX 86400
        CACHE-->>H: +OK
        H->>MQ: PUB url.created abc123:https://...
        H-->>C: {"short":"abc123","url":"https://..."}
    end

    C->>RL: GET /abc123
    RL->>H: forward
    H->>CACHE: GET abc123
    CACHE-->>H: $N\r\nhttps://...
    H->>MQ: PUB url.clicked abc123
    H-->>C: 301 Location: https://...

    loop every minute
        SCHED->>SCHED: analytics flush tick
    end
```

## Component Integration Map

```mermaid
graph TD
    subgraph 10-url-shortener
        RL[ratelimit.Middleware\ntoken bucket 10 req/s]
        H[HTTP Handler\nshorten + redirect]
        IC[inMemCache\nfallback]
    end

    subgraph 04-rate-limiter
        TB[TokenBucket\nratelimit package]
    end

    subgraph 07-distributed-cache
        RESP[RESP TCP server\n:6380]
    end

    subgraph 06-message-queue
        MQS[MQ TCP server\n:9001]
    end

    subgraph 09-task-scheduler
        SCHED[Scheduler\ntaskscheduler package]
    end

    RL --> TB
    H -->|cache.Client TCP| RESP
    H -->|mq.Client TCP| MQS
    H -->|fallback| IC
    SCHED --> H
```

## Short Code Generation

```mermaid
graph LR
    RAND[crypto/rand\n6 random bytes] --> B64[base64.URLEncoding\n8 chars] --> TRIM[first 6 chars\nabc123]
```

`crypto/rand` is used (not `math/rand`) to prevent predictable codes. Base64 URL encoding avoids `+` and `/` which are problematic in URLs.

## Cache Interface (Dependency Inversion)

The handler depends on the `Cache` interface, not the concrete `*cache.Client`:

```mermaid
graph LR
    H[Handler] -->|Cache interface| IFACE{Cache\nSet Get Del}
    IFACE -->|production| CLIENT[cache.Client\nRESP TCP]
    IFACE -->|test| MOCK[mockCache\nin-memory map]
    IFACE -->|fallback| INMEM[inMemCache\nin-memory map]
```

This is the Dependency Inversion Principle in action — the handler is testable without a running RESP server.

## Graceful Degradation

```mermaid
graph TD
    START[server start] --> CDIAL{cache.Dial\n:6380}
    CDIAL -->|success| CRESP[use RESP client]
    CDIAL -->|fail| CFALLBACK[use inMemCache\nlog warning]
    START --> MDIAL{mq.Dial\n:9001}
    MDIAL -->|success| MQCLIENT[publish analytics]
    MDIAL -->|fail| MQNIL[mqClient = nil\nanalytics disabled]
```

The server starts and serves traffic even if the cache or MQ is unavailable — it degrades gracefully.
