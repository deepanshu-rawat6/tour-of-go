# 05-load-balancer

L7 reverse proxy with round-robin, least-connections, and background health checks.

## Architecture

```mermaid
graph TD
    C[Client] -->|HTTP| LB[Load Balancer\n:8084]
    LB --> STRAT{Strategy}
    STRAT -->|round-robin| B1[Backend 1\n:9010]
    STRAT -->|round-robin| B2[Backend 2\n:9011]
    STRAT -->|round-robin| B3[Backend 3\n:9012]
    HC[Health Checker\nevery 5s] -->|GET /health| B1
    HC --> B2
    HC --> B3
    B1 & B2 & B3 -->|unhealthy| SKIP[skipped by balancer]
```

## Quick Start

```bash
# Start 3 backends
PORT=9010 make run-backend &
PORT=9011 make run-backend &
PORT=9012 make run-backend &

# Start load balancer (round-robin)
make run-lb

# Test distribution
for i in $(seq 9); do curl -s localhost:8084/; done
# → backend:9010, backend:9011, backend:9012 × 3

# Least-connections
STRATEGY=lc make run-lb
```

## Docs

- [`docs/deep-dive.md`](./docs/deep-dive.md)
