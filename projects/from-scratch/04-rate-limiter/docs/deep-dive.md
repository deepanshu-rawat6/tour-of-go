# 04-rate-limiter: Deep Dive

## Why Rate Limiting?

Without rate limiting, a single client can exhaust your server's resources. Rate limiting protects against:
- Accidental thundering herd (retry storms)
- Intentional abuse / DDoS
- Downstream service overload

## Algorithm Internals

### Token Bucket

Tokens accumulate at a fixed rate up to a burst capacity. Each request consumes one token:

```mermaid
graph LR
    TIME[time.Now] -->|elapsed × rate| REFILL[add tokens\nmin burst]
    REFILL --> BUCKET[token bucket\ncurrent tokens]
    REQ[Request] -->|Allow| CHECK{tokens ≥ 1?}
    CHECK -->|yes| CONSUME[tokens--\nreturn true]
    CHECK -->|no| REJECT[return false\n429]
    BUCKET --> CHECK
```

**Best for**: APIs that allow short bursts (e.g., 10 req/s with burst of 50).

### Leaky Bucket

A buffered channel acts as the bucket. A background goroutine drains it at a constant rate:

```mermaid
graph LR
    REQ[Request] -->|Allow| CHAN{channel\nfull?}
    CHAN -->|space available| ENQUEUE[queue ← struct{}\nreturn true]
    CHAN -->|full| REJECT[return false\n429]
    DRAIN[ticker goroutine\nevery 1/rate] -->|drain one| CHAN
```

**Best for**: Traffic shaping — output rate is always constant regardless of input bursts.

### Fixed Window

Counts requests per time window. Resets at window boundary:

```mermaid
graph LR
    REQ[Request] --> KEY[window key\nnow / window_ns]
    KEY --> MAP[counts map\nkey → count]
    MAP -->|count < limit| ALLOW[count++\nreturn true]
    MAP -->|count ≥ limit| REJECT[return false]
    TICK[new window] -->|new key| MAP
```

**Problem**: Boundary burst — 2× limit requests possible at window boundary.

### Sliding Window Log

Stores a timestamp for every request. Evicts old entries on each check:

```mermaid
graph LR
    REQ[Request] --> EVICT[evict entries\nolder than window]
    EVICT --> CHECK{len logs\n< limit?}
    CHECK -->|yes| APPEND[append now\nreturn true]
    CHECK -->|no| REJECT[return false]
```

**Best for**: Strict per-window limits. **Cost**: O(requests) memory.

## Algorithm Comparison

```mermaid
graph TD
    subgraph Memory
        TB_M[Token Bucket\nO1] 
        LB_M[Leaky Bucket\nO capacity]
        FW_M[Fixed Window\nO1]
        SW_M[Sliding Window\nO requests]
    end

    subgraph Burst
        TB_B[Token Bucket\nYes - configurable]
        LB_B[Leaky Bucket\nNo - constant rate]
        FW_B[Fixed Window\nAt boundary only]
        SW_B[Sliding Window\nNo]
    end
```

## Middleware Chain

```mermaid
graph LR
    REQ[HTTP Request] --> RL[RateLimit middleware\nl.Allow?]
    RL -->|true| NEXT[next handler]
    RL -->|false| 429[429 Too Many Requests]
    NEXT --> RESP[HTTP Response]
```
