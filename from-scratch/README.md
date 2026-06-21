# From Scratch Series

Build fundamental distributed systems components from the ground up in Go — no magic, no frameworks, just `net`, `sync`, and the standard library.

Each project builds on the previous one. The series culminates in a URL shortener that integrates the rate limiter, cache, message queue, and task scheduler you built along the way. Project 11 introduces `go-chi/chi` to show where a router earns its place.

---

## Learning Path

```mermaid
graph LR
    T[01-tcp-server<br/>raw net.Listener] --> H[02-http-server<br/>HTTP/1.1 on TCP]
    H --> W[03-websocket-chat<br/>hub pattern]
    H --> R[04-rate-limiter<br/>4 algorithms]
    H --> L[05-load-balancer<br/>round-robin + least-conn]
    T --> M[06-message-queue<br/>pub/sub + TCP]
    T --> C[07-distributed-cache<br/>RESP protocol]
    T --> A[08-log-aggregator<br/>tail + ship + query]
    H --> S[09-task-scheduler<br/>cron + HTTP API]
    R & C & M & S --> U[10-url-shortener<br/>capstone]
    R & L --> G[11-api-gateway<br/>chi · JWT · ReverseProxy]
    C --> CH[12-consistent-hash<br/>virtual nodes · ring]
    C --> BF[13-bloom-filter<br/>probabilistic DS]
    CH --> CRDT[14-crdt<br/>eventual consistency]
    M & C --> CL[15-commit-log<br/>mmap · sendfile · segments]
```

---

## Projects

| # | Project | Level | What you build | Key concepts |
|---|---------|-------|---------------|--------------|
| 01 | [`01-tcp-server`](./01-tcp-server/) | `SDE-1` | Raw TCP echo server | `net.Listener`, goroutine-per-conn, `io.Copy` |
| 02 | [`02-http-server`](./02-http-server/) | `SDE-1` | HTTP/1.1 parser on TCP + stdlib comparison | Request line parsing, routing, response writing |
| 03 | [`03-websocket-chat`](./03-websocket-chat/) | `SDE-1` | Multi-room chat server | Hub pattern, broadcast, gorilla/websocket |
| 04 | [`04-rate-limiter`](./04-rate-limiter/) | `SDE-1` | All 4 rate limiting algorithms | Token bucket, leaky bucket, fixed window, sliding window |
| 05 | [`05-load-balancer`](./05-load-balancer/) | `SDE-1` | L7 reverse proxy | Round-robin, least-connections, health checks |
| 06 | [`06-message-queue`](./06-message-queue/) | `SDE-1` | In-memory pub/sub + TCP server | Broker, topics, fan-out, custom protocol |
| 07 | [`07-distributed-cache`](./07-distributed-cache/) | `SDE-1` | Redis-compatible KV store | RESP protocol, TTL eviction, `redis-cli` compatible |
| 08 | [`08-log-aggregator`](./08-log-aggregator/) | `SDE-1` | Log tail → ship → aggregate → query | File tailer, TCP shipper, in-memory store, HTTP search |
| 09 | [`09-task-scheduler`](./09-task-scheduler/) | `SDE-1` | Cron-like task scheduler | Cron parser, tick loop, HTTP API |
| 10 | [`10-url-shortener`](./10-url-shortener/) | `SDE-1` | URL shortener (capstone) | Integrates 04 + 07 + 06 + 09 |
| 11 | [`11-api-gateway`](./11-api-gateway/) | `SDE-1` | Edge API Gateway (JWT auth, rate limiting, reverse proxy) | `go-chi/chi` middleware composability, HS256 JWT, `httputil.ReverseProxy` Director, `context.Context` identity propagation, `replace` directive |
| 12 | [`12-consistent-hash`](./12-consistent-hash/) | `SDE-2` | Consistent hashing ring | Virtual nodes, crc32, binary search, minimal redistribution |
| 13 | [`13-bloom-filter`](./13-bloom-filter/) | `SDE-2` | Bloom filter + HyperLogLog | Probabilistic membership, cardinality estimation, false positive rates |
| 14 | [`14-crdt`](./14-crdt/) | `SDE-2` | Conflict-Free Replicated Data Types | G-Counter, PN-Counter, LWW-Register, eventual consistency |
| 15 | [`15-commit-log`](./15-commit-log/) | `SDE-2` | Log-structured message broker (Mini-Kafka) | Append-only segments, mmap index, sendfile zero-copy, consumer offsets |

---

## Running Any Project

```bash
cd 01-tcp-server
make build   # compile
make test    # run tests with -race
make run     # start the server
```

---

## Project Architectures

### 01. TCP Server

```mermaid
graph TD
    CLIENT[Client<br>telnet / nc] -->|TCP connect| LISTENER[net.Listener<br>:9000]
    LISTENER -->|Accept| CONN[net.Conn]
    CONN --> GOR[goroutine<br>io.Copy echo]
    GOR -->|response| CLIENT
```

Raw TCP echo server. One goroutine per connection, `io.Copy` reflects data back.

---

### 02. HTTP Server

```mermaid
graph TD
    CLIENT[Client<br>curl] -->|TCP| CONN[net.Conn]
    CONN --> PARSE[Parse Request Line<br>GET /path HTTP/1.1]
    PARSE --> ROUTE[Router<br>path → handler]
    ROUTE --> HANDLER[Handler<br>write response]
    HANDLER -->|HTTP/1.1 200 OK| CLIENT
```

HTTP/1.1 parser built on raw TCP. Parses request line, routes to handlers, writes properly formatted responses.

---

### 03. WebSocket Chat

```mermaid
graph TD
    C1[Client 1] -->|WS| HUB[Hub<br>sync.Mutex<br>rooms map]
    C2[Client 2] -->|WS| HUB
    C3[Client 3] -->|WS| HUB
    HUB -->|broadcast| C1
    HUB -->|broadcast| C2
    HUB -->|broadcast| C3
    
    C1 -->|join room:general| ROOM[Room<br>set of connections]
```

Multi-room chat with hub pattern. Hub manages rooms, broadcasts messages to all clients in a room.

---

### 04. Rate Limiter

```mermaid
graph LR
    REQ[Request] --> RL{Rate Limiter}
    RL -->|Token Bucket| TB[Refill tokens/sec<br>Consume 1 per request]
    RL -->|Leaky Bucket| LB[Fixed drain rate<br>Queue overflow → reject]
    RL -->|Fixed Window| FW[Count per time window<br>Reset at boundary]
    RL -->|Sliding Window| SW[Weighted count<br>Previous + current window]
    TB & LB & FW & SW --> DECISION{Allow?}
    DECISION -->|yes| PASS[200 OK]
    DECISION -->|no| REJECT[429 Too Many Requests]
```

All 4 rate limiting algorithms implemented and benchmarked.

---

### 05. Load Balancer

```mermaid
graph LR
    CLIENT[Clients] --> LB[Load Balancer<br>:8080]
    LB -->|round-robin| B1[Backend :9001]
    LB -->|least-conn| B2[Backend :9002]
    LB -->|health check| B3[Backend :9003]
    
    HC[Health Checker<br>time.Ticker] -->|GET /health| B1
    HC --> B2
    HC --> B3
    HC -->|mark unhealthy| LB
```

L7 reverse proxy with round-robin, least-connections, and periodic health checks.

---

### 06. Message Queue

```mermaid
graph TD
    PUB[Publisher] -->|TCP| BROKER[Broker<br>topics map]
    BROKER --> T1[Topic: orders<br>fan-out]
    BROKER --> T2[Topic: events]
    T1 --> SUB1[Subscriber 1]
    T1 --> SUB2[Subscriber 2]
    T2 --> SUB3[Subscriber 3]
```

In-memory pub/sub with custom TCP text protocol. Broker manages topics, fan-out to all subscribers.

---

### 07. Distributed Cache

```mermaid
graph LR
    CLIENT[redis-cli] -->|RESP protocol| SERVER[TCP Server<br>:6379]
    SERVER --> PARSE[RESP Parser<br>*3\r<br>$3\r<br>SET...]
    PARSE --> STORE[KV Store<br>sync.RWMutex]
    STORE --> TTL[TTL Reaper<br>time.Ticker<br>expire keys]
```

Redis-compatible KV store. Speaks RESP protocol — works with `redis-cli`. Supports GET, SET, DEL, EXPIRE, TTL.

---

### 08. Log Aggregator

```mermaid
graph LR
    FILE[Log File] --> TAIL[File Tailer<br>fsnotify]
    TAIL -->|TCP| SHIP[Shipper]
    SHIP -->|TCP| AGG[Aggregator<br>in-memory store]
    AGG --> HTTP[HTTP API<br>GET /search?q=error]
```

End-to-end log pipeline: tail files → ship over TCP → aggregate → query via HTTP.

---

### 09. Task Scheduler

```mermaid
graph TD
    API[HTTP API<br>POST /tasks] --> SCHED[Scheduler<br>cron expressions]
    SCHED --> TICK[Tick Loop<br>time.Ticker 1s]
    TICK -->|match?| EXEC[Execute Task<br>goroutine]
    EXEC --> LOG[Log Result]
```

Cron-like scheduler with HTTP API for task management. Parses cron expressions, executes on schedule.

---

### 10. URL Shortener (Capstone)

```mermaid
graph TD
    CLIENT[Client] -->|POST /shorten| API[HTTP API]
    API --> RL[Rate Limiter<br>from 04]
    RL --> GEN[Generate Short Code]
    GEN --> CACHE[Distributed Cache<br>from 07]
    API -->|GET /:code| CACHE
    CACHE -->|redirect 301| CLIENT
    
    API --> MQ[Message Queue<br>from 06]
    MQ --> ANALYTICS[Analytics Consumer]
    
    SCHED[Task Scheduler<br>from 09] -->|cleanup expired| CACHE
```

Capstone project integrating rate limiter (04), distributed cache (07), message queue (06), and task scheduler (09).

---

### 11. API Gateway

```mermaid
graph LR
    CLIENT[Client] --> GW[API Gateway<br>:8080]
    GW --> AUTH[JWT Middleware<br>HS256 verify]
    AUTH --> RL[Rate Limiter<br>per-user]
    RL --> PROXY[httputil.ReverseProxy<br>Director func]
    PROXY --> SVC1[Service A<br>:9001]
    PROXY --> SVC2[Service B<br>:9002]
```

Edge gateway with `go-chi/chi` middleware composability, JWT authentication, per-user rate limiting, and reverse proxy routing.

---

### 12. Consistent Hash

```mermaid
graph TD
    KEY[key: user:42] --> HASH[crc32 hash<br>position on ring]
    HASH --> SEARCH[Binary search<br>next clockwise node]
    SEARCH --> NODE[Target Node<br>cache-2]
    
    RING[Hash Ring<br>150 virtual nodes per physical] --> SEARCH
```

Consistent hashing ring with virtual nodes. Demonstrates minimal key redistribution when nodes join/leave.

---

### 13. Bloom Filter

```mermaid
graph TD
    ADD[Add item] --> H1[hash1 → bit 3]
    ADD --> H2[hash2 → bit 7]
    ADD --> H3[hash3 → bit 11]
    H1 & H2 & H3 --> BITS[Bit Array]
    
    CHECK[Contains?] --> BITS
    BITS -->|all bits set| MAYBE[Probably yes]
    BITS -->|any bit unset| NO[Definitely no]
    
    HLL[HyperLogLog] --> REG[16K registers<br>leading zeros]
    REG --> EST[Cardinality estimate<br>±6% error]
```

Bloom filter for probabilistic membership + HyperLogLog for cardinality estimation using 16KB.

---

### 14. CRDT

```mermaid
graph TD
    subgraph G-Counter
        A[Node A: 3] --> MERGE[Merge: max per node]
        B[Node B: 5] --> MERGE
        MERGE --> SUM[Value = sum = 8]
    end
    
    subgraph LWW-Register
        W1[Set alice t=100] --> CMP[Compare timestamps]
        W2[Set bob t=105] --> CMP
        CMP --> WIN[bob wins<br>higher timestamp]
    end
```

Conflict-Free Replicated Data Types. G-Counter (grow-only), PN-Counter (inc+dec), LWW-Register (last-writer-wins). All converge without coordination.

---

### 15. Commit Log (Mini-Kafka)

```mermaid
graph TD
    PROD[Producer] -->|PRODUCE topic 0 msg| TCP[TCP Server<br>:9092]
    TCP --> BROKER[Broker]
    BROKER --> PART[Partition<br>append-only]
    PART --> SEG[Segment<br>.log + .index]
    SEG --> DISK[(Disk<br>sequential writes)]
    
    CONS[Consumer] -->|CONSUME topic 0 offset| TCP
    TCP -->|sendfile zero-copy| CONS
    
    IDX[mmap'd Index<br>offset → byte position] --> SEG
```

Persistent append-only commit log. Messages survive restarts. Consumers pull at their own pace from any offset. mmap'd index for O(1) lookups, sendfile for zero-copy network transfer.
