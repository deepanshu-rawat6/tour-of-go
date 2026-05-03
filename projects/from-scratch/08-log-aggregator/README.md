# 08-log-aggregator

Tail a log file → ship over TCP → aggregate → query via HTTP.

## Architecture

```mermaid
graph LR
    F[Log File\n/tmp/app.log] -->|poll 100ms| TL[Tailer]
    TL -->|new lines| SH[Shipper\ncmd/shipper]
    SH -->|SOURCE\tLINE\n| AGG[Aggregator\nTCP :9002]
    AGG --> STORE[In-memory store\n[]LogEntry]
    STORE -->|search| API[HTTP API\n:8085 /logs]
    API -->|JSON| CLIENT[curl / browser]
```

## Quick Start

```bash
# Start aggregator
make run-aggregator

# Start shipper (in another terminal)
LOG_FILE=/tmp/app.log SOURCE=myapp make run-shipper

# Write some logs
echo "ERROR: something broke" >> /tmp/app.log
echo "INFO: all good" >> /tmp/app.log

# Query
curl "localhost:8085/logs?q=ERROR"
curl "localhost:8085/logs?source=myapp&limit=10"
```

## Docs

- [`docs/deep-dive.md`](./docs/deep-dive.md)
