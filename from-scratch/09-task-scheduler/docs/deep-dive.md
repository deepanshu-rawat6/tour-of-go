# 09-task-scheduler: Deep Dive

## Cron Expression Format

```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12)
│ │ │ │ ┌───────────── day of week (0-6, Sun=0)
│ │ │ │ │
* * * * *
```

Supported syntax:

| Pattern | Meaning |
|---|---|
| `*` | every value |
| `*/5` | every 5th value |
| `1-5` | range 1 through 5 |
| `1,3,5` | specific values |
| `30 9 * * 1-5` | 9:30 on weekdays |

## Cron Parser

```mermaid
graph TD
    EXPR[cron expression\n5 fields] --> SPLIT[strings.Fields]
    SPLIT --> F1[minute field\nparseField 0-59]
    SPLIT --> F2[hour field\nparseField 0-23]
    SPLIT --> F3[day field\nparseField 1-31]
    SPLIT --> F4[month field\nparseField 1-12]
    SPLIT --> F5[weekday field\nparseField 0-6]
    F1 & F2 & F3 & F4 & F5 --> BITS[bool arrays\nO1 match lookup]
```

Each field is parsed into a `[]bool` array. Matching is O(1) — just index into the array.

## Scheduler Tick Loop

```mermaid
graph TD
    START[Start ctx] --> TICKER[time.NewTicker\nevery second]
    TICKER -->|now| TICK[tick now]
    TICK --> SCAN[scan all tasks\nRLock]
    SCAN --> MATCH{schedule.Match\nnow}
    MATCH -->|true| FIRE[go task.Fn\nnew goroutine]
    MATCH -->|false| SKIP[skip]
    FIRE & SKIP --> TICKER
    CTX[ctx.Done] --> STOP[return]
```

The scheduler ticks every second but cron expressions have minute-level granularity. A task with `* * * * *` fires once per minute (when `second=0` of that minute is the first tick in that minute). In practice, the tick at second 0 of each minute matches.

## HTTP API Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant H as HTTP Handler
    participant S as Scheduler

    C->>H: POST /tasks {"id":"cleanup","expr":"*/5 * * * *"}
    H->>S: s.Add(id, name, expr, fn)
    S->>S: cron.Parse(expr)
    S-->>H: nil (success)
    H-->>C: 201 Created

    C->>H: GET /tasks
    H->>S: s.List()
    S-->>H: []*Task
    H-->>C: JSON array

    C->>H: DELETE /tasks/cleanup
    H->>S: s.Remove("cleanup")
    H-->>C: 204 No Content
```

## Concurrency Safety

```mermaid
graph LR
    TICK[tick goroutine] -->|RLock| RWMU[sync.RWMutex]
    ADD[Add goroutine] -->|Lock| RWMU
    REMOVE[Remove goroutine] -->|Lock| RWMU
    LIST[List goroutine] -->|RLock| RWMU
    RWMU --> MAP[map id→Task]
```

Task functions run in their own goroutines (`go task.Fn()`), so a slow task doesn't block the tick loop.
