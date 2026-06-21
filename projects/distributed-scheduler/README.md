# distributed-scheduler

A production-grade distributed job scheduler with Redis-based leader election, a state machine, zombie detection, full-text search, and a hexagonal architecture.

---

## Architecture

```mermaid
graph TD
    API[HTTP API<br>/jobs /search /status] --> MGR[App Manager<br>lifecycle orchestrator]
    MGR -->|acquire| LEASE[Redis Lease<br>single-active-instance]
    LEASE -->|leader only| POOL[Concurrency Pool<br>sync.Mutex + rules map]
    POOL --> SCHED[Scheduler Service<br>state machine]
    SCHED -->|Pending → Running| DEST{Destination}
    DEST --> SQS[AWS SQS]
    DEST --> MEM[In-Memory]
    SCHED -->|persist| PG[(PostgreSQL<br>job state)]
    SEARCH[Bleve Search<br>full-text index] --> PG

    subgraph Crons
        COLD[ColdScheduler<br>pick up pending jobs]
        REFRESH[ConcurrencyRefresher<br>reload rules]
        ZOMBIE[ZombieChecker<br>detect stale Running jobs]
        HB[HeartbeatMarker<br>mark alive jobs]
        LONG[LongPublishedChecker<br>detect stuck jobs]
    end

    MGR --> Crons
```

## State Machine

```mermaid
stateDiagram-v2
    [*] --> Pending : job submitted
    Pending --> Running : scheduler picks up
    Running --> Done : destination ack
    Running --> Failed : max retries exceeded
    Running --> Pending : zombie detected (no heartbeat)
    Failed --> [*]
    Done --> [*]
```

## Key Concepts

- **Redis Lease** — only one scheduler instance runs at a time; others wait. On SIGTERM the lease is released so a standby can take over immediately.
- **Concurrency Pool** — per-job-name concurrency rules enforced in memory with `sync.Mutex`. Prevents a single job type from flooding the destination.
- **Zombie Detection** — a cron checks for `Running` jobs with no recent heartbeat and resets them to `Pending`.
- **Hexagonal Architecture** — `ports/` defines interfaces; `adapters/` has implementations (Redis, Postgres, SQS, in-memory). The domain never imports infrastructure.

## Quick Start

```bash
docker-compose up -d   # starts Postgres + Redis
make run
```

## Docs

- [`docs/deep-dive.md`](./docs/deep-dive.md)
- [`docs/adr/`](./docs/adr/) — ADRs: Go over Java, sync.Mutex over channels, sqlx over ORM
