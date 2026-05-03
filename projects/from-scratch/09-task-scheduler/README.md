# 09-task-scheduler

A cron-like task scheduler with HTTP API for managing tasks at runtime.

## Architecture

```mermaid
graph TD
    TICK[time.Ticker\nevery second] --> SCHED[Scheduler\ntick loop]
    SCHED -->|Match cron expr| T1[Task 1\n* * * * *]
    SCHED --> T2[Task 2\n*/5 * * * *]
    SCHED --> TN[Task N]
    T1 & T2 & TN -->|go fn| EXEC[goroutine\nexecute fn]

    API[HTTP API\n:8086] -->|POST /tasks| SCHED
    API -->|GET /tasks| LIST[list tasks]
    API -->|DELETE /tasks/:id| REMOVE[remove task]
```

## Cron Syntax

```
* * * * *
│ │ │ │ └── weekday (0-6, Sun=0)
│ │ │ └──── month (1-12)
│ │ └────── day (1-31)
│ └──────── hour (0-23)
└────────── minute (0-59)

Supported: * (any), */n (every n), n (exact), n-m (range), n,m (list)
```

## Quick Start

```bash
make run   # starts on :8086

# List tasks
curl localhost:8086/tasks

# Add a task
curl -X POST localhost:8086/tasks \
  -H 'Content-Type: application/json' \
  -d '{"id":"cleanup","name":"cleanup job","expr":"*/5 * * * *"}'

# Remove a task
curl -X DELETE localhost:8086/tasks/cleanup
```

## Docs

- [`docs/deep-dive.md`](./docs/deep-dive.md)
