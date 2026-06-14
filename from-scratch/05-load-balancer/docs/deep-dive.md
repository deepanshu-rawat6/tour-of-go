# 05-load-balancer: Deep Dive

## What is L7 Load Balancing?

L7 (application layer) load balancing operates on HTTP — it can inspect headers, paths, and cookies to make routing decisions. L4 load balancing only sees TCP/IP.

```mermaid
graph TD
    subgraph L4 Load Balancer
        L4[TCP packets\nroute by IP/port]
    end
    subgraph L7 Load Balancer
        L7[HTTP requests\nroute by path/header/cookie]
    end
    CLIENT[Client] --> L7
    L7 --> B1[Backend 1]
    L7 --> B2[Backend 2]
```

## Round Robin

Requests cycle through backends in order. Uses an atomic counter for lock-free operation:

```mermaid
graph LR
    REQ1[Request 1] -->|idx=1 mod 3| B1[Backend 1]
    REQ2[Request 2] -->|idx=2 mod 3| B2[Backend 2]
    REQ3[Request 3] -->|idx=3 mod 3| B3[Backend 3]
    REQ4[Request 4] -->|idx=4 mod 3| B1
    UNHEALTHY[Backend 2 unhealthy] -->|skip| B3
```

The `atomic.Uint64` counter increments on every request — no mutex needed.

## Least Connections

Routes to the backend with the fewest active connections:

```mermaid
graph TD
    REQ[New Request] --> SCAN[scan all healthy backends]
    SCAN --> COMPARE{compare\nActiveConns}
    COMPARE -->|B1=5 B2=2 B3=8| SELECT[select B2\nlowest conns]
    SELECT --> INC[B2.ActiveConns++]
    INC --> PROXY[proxy request]
    PROXY --> DEC[B2.ActiveConns--\ndefer]
```

`ActiveConns` is an `atomic.Int64` — incremented before the request, decremented via `defer` after.

## Health Checker

A background goroutine polls each backend's `/health` endpoint:

```mermaid
sequenceDiagram
    participant HC as Health Checker
    participant B1 as Backend 1
    participant B2 as Backend 2

    loop every 5 seconds
        HC->>B1: GET /health
        B1-->>HC: 200 OK
        HC->>HC: B1.SetHealthy(true)
        HC->>B2: GET /health
        Note over B2: B2 is down
        HC->>HC: B2.SetHealthy(false)
    end

    Note over HC: Next request skips B2
```

## Reverse Proxy Flow

```mermaid
graph LR
    CLIENT[Client] -->|HTTP request| LB[Load Balancer\nhttputil.ReverseProxy]
    LB -->|Director: rewrite URL| UPSTREAM[Selected Backend]
    UPSTREAM -->|response| LB
    LB -->|forward response| CLIENT
```

`httputil.ReverseProxy` handles connection pooling, header forwarding, and response streaming. The `Director` function is the only customization point — it rewrites the request URL to point to the selected backend.
