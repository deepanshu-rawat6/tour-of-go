# 08-log-aggregator: Deep Dive

## System Overview

```mermaid
graph LR
    APP[Application\nwrites to file] -->|append| FILE[/tmp/app.log]
    FILE -->|poll 100ms| TAILER[Tailer\nseek to end on start]
    TAILER -->|new lines| SHIPPER[Shipper\nTCP client]
    SHIPPER -->|SOURCE\tLINE\n| AGG[Aggregator\nTCP :9002]
    AGG --> STORE[In-memory store\n[]LogEntry]
    STORE -->|search| HTTP[HTTP API\n:8085 /logs]
    HTTP -->|JSON| QUERY[curl / browser]
```

## Tailer: Poll-Based File Watching

The tailer seeks to the end of the file on startup (so it only tails new lines), then polls every 100ms:

```mermaid
sequenceDiagram
    participant T as Tailer
    participant F as File

    T->>F: Open file
    T->>F: Seek to end (io.SeekEnd)
    loop every 100ms
        T->>F: ReadString('\n')
        alt new line available
            F-->>T: "ERROR: something\n"
            T->>T: Lines ← "ERROR: something\n"
        else no new data
            F-->>T: io.EOF
            Note over T: sleep, try again
        end
    end
```

## Shipper Protocol

Simple tab-separated format: `SOURCE\tLOG_LINE\n`

```mermaid
graph LR
    TAILER[Tailer\nLines chan] -->|line| SHIPPER[Shipper]
    SHIPPER -->|fmt.Fprintf\napp1\tERROR: ...\n| CONN[net.Conn\nTCP to aggregator]
```

## Aggregator: Ingest + Search

```mermaid
graph TD
    TCP[TCP :9002] -->|accept| CONN[net.Conn per shipper]
    CONN -->|scan lines| PARSE[split SOURCE\tLINE]
    PARSE --> INGEST[Ingest\nappend LogEntry]
    INGEST --> STORE[sync.RWMutex\n[]LogEntry]

    HTTP[GET /logs?q=error&source=app1] --> SEARCH[Search\nreverse scan]
    STORE --> SEARCH
    SEARCH -->|filter by query + source| RESULTS[[]LogEntry]
    RESULTS --> JSON[JSON response]
```

## Search Algorithm

Search scans from newest to oldest (reverse slice iteration) and stops at `limit`:

```mermaid
graph TD
    START[i = len-1] --> CHECK{i >= 0 AND\nlen results < limit}
    CHECK -->|yes| FILTER{source match\nAND query match}
    FILTER -->|yes| APPEND[append to results]
    FILTER -->|no| SKIP[skip]
    APPEND & SKIP --> DEC[i--]
    DEC --> CHECK
    CHECK -->|no| RETURN[return results]
```

Reverse scan means the most recent logs appear first in results — the natural expectation for log queries.
