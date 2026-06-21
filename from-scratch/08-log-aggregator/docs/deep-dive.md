# 08-log-aggregator: Deep Dive

## System Overview

```mermaid
graph LR
    APP[Application<br>writes to file] -->|append| FILE[/tmp/app.log]
    FILE -->|poll 100ms| TAILER[Tailer<br>seek to end on start]
    TAILER -->|new lines| SHIPPER[Shipper<br>TCP client]
    SHIPPER -->|SOURCE\tLINE<br>| AGG[Aggregator<br>TCP :9002]
    AGG --> STORE[In-memory store<br>[]LogEntry]
    STORE -->|search| HTTP[HTTP API<br>:8085 /logs]
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
        T->>F: ReadString('<br>')
        alt new line available
            F-->>T: "ERROR: something<br>"
            T->>T: Lines ← "ERROR: something<br>"
        else no new data
            F-->>T: io.EOF
            Note over T: sleep, try again
        end
    end
```

## Shipper Protocol

Simple tab-separated format: `SOURCE\tLOG_LINE<br>`

```mermaid
graph LR
    TAILER[Tailer<br>Lines chan] -->|line| SHIPPER[Shipper]
    SHIPPER -->|fmt.Fprintf<br>app1\tERROR: ...<br>| CONN[net.Conn<br>TCP to aggregator]
```

## Aggregator: Ingest + Search

```mermaid
graph TD
    TCP[TCP :9002] -->|accept| CONN[net.Conn per shipper]
    CONN -->|scan lines| PARSE[split SOURCE\tLINE]
    PARSE --> INGEST[Ingest<br>append LogEntry]
    INGEST --> STORE[sync.RWMutex<br>[]LogEntry]

    HTTP[GET /logs?q=error&source=app1] --> SEARCH[Search<br>reverse scan]
    STORE --> SEARCH
    SEARCH -->|filter by query + source| RESULTS[[]LogEntry]
    RESULTS --> JSON[JSON response]
```

## Search Algorithm

Search scans from newest to oldest (reverse slice iteration) and stops at `limit`:

```mermaid
graph TD
    START[i = len-1] --> CHECK{i >= 0 AND<br>len results < limit}
    CHECK -->|yes| FILTER{source match<br>AND query match}
    FILTER -->|yes| APPEND[append to results]
    FILTER -->|no| SKIP[skip]
    APPEND & SKIP --> DEC[i--]
    DEC --> CHECK
    CHECK -->|no| RETURN[return results]
```

Reverse scan means the most recent logs appear first in results — the natural expectation for log queries.
