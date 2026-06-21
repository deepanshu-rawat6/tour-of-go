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
    TIME[time.Now] -->|elapsed × rate| REFILL[add tokens<br>min burst]
    REFILL --> BUCKET[token bucket<br>current tokens]
    REQ[Request] -->|Allow| CHECK{tokens ≥ 1?}
    CHECK -->|yes| CONSUME[tokens--<br>return true]
    CHECK -->|no| REJECT[return false<br>429]
    BUCKET --> CHECK
```

**Best for**: APIs that allow short bursts (e.g., 10 req/s with burst of 50).

### Leaky Bucket

A buffered channel acts as the bucket. A background goroutine drains it at a constant rate:

```mermaid
graph LR
    REQ[Request] -->|Allow| CHAN{channel<br>full?}
    CHAN -->|space available| ENQUEUE[queue ← struct{}<br>return true]
    CHAN -->|full| REJECT[return false<br>429]
    DRAIN[ticker goroutine<br>every 1/rate] -->|drain one| CHAN
```

**Best for**: Traffic shaping — output rate is always constant regardless of input bursts.

### Fixed Window

Counts requests per time window. Resets at window boundary:

```mermaid
graph LR
    REQ[Request] --> KEY[window key<br>now / window_ns]
    KEY --> MAP[counts map<br>key → count]
    MAP -->|count < limit| ALLOW[count++<br>return true]
    MAP -->|count ≥ limit| REJECT[return false]
    TICK[new window] -->|new key| MAP
```

**Problem**: Boundary burst — 2× limit requests possible at window boundary.

### Sliding Window Log

Stores a timestamp for every request. Evicts old entries on each check:

```mermaid
graph LR
    REQ[Request] --> EVICT[evict entries<br>older than window]
    EVICT --> CHECK{len logs<br>< limit?}
    CHECK -->|yes| APPEND[append now<br>return true]
    CHECK -->|no| REJECT[return false]
```

**Best for**: Strict per-window limits. **Cost**: O(requests) memory.

## Algorithm Comparison

```mermaid
graph TD
    subgraph Memory
        TB_M[Token Bucket<br>O1] 
        LB_M[Leaky Bucket<br>O capacity]
        FW_M[Fixed Window<br>O1]
        SW_M[Sliding Window<br>O requests]
    end

    subgraph Burst
        TB_B[Token Bucket<br>Yes - configurable]
        LB_B[Leaky Bucket<br>No - constant rate]
        FW_B[Fixed Window<br>At boundary only]
        SW_B[Sliding Window<br>No]
    end
```

## Middleware Chain

```mermaid
graph LR
    REQ[HTTP Request] --> RL[RateLimit middleware<br>l.Allow?]
    RL -->|true| NEXT[next handler]
    RL -->|false| 429[429 Too Many Requests]
    NEXT --> RESP[HTTP Response]
```
