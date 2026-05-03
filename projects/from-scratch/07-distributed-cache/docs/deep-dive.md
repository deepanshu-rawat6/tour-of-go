# 07-distributed-cache: Deep Dive

## RESP Protocol

Redis Serialization Protocol (RESP) uses a type prefix byte followed by data:

```
+OK\r\n              → Simple String "OK"
-ERR message\r\n     → Error
:42\r\n              → Integer 42
$5\r\nhello\r\n      → Bulk String "hello" (5 bytes)
$-1\r\n              → Null Bulk String
*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n  → Array ["SET","foo","bar"]
```

```mermaid
graph TD
    BYTE[First byte] -->|+| SS[Simple String\nread until CRLF]
    BYTE -->|-| ERR[Error\nread until CRLF]
    BYTE -->|:| INT[Integer\nparse int64]
    BYTE -->|$| BULK[Bulk String\nread length then bytes]
    BYTE -->|*| ARR[Array\nparse N values recursively]
```

## Command Execution Flow

```mermaid
graph TD
    CLIENT[redis-cli / client] -->|RESP array| TCP[TCP :6380]
    TCP --> PARSER[resp.Parse\nbufio.Reader]
    PARSER --> CMD[resp.Command\nextract cmd + args]
    CMD --> ROUTER{switch cmd}
    ROUTER -->|SET| SET[store.Set\nkey value ttl]
    ROUTER -->|GET| GET[store.Get\nreturn value or null]
    ROUTER -->|DEL| DEL[store.Del\nreturn count]
    ROUTER -->|TTL| TTL[store.TTL\nreturn seconds]
    ROUTER -->|KEYS| KEYS[store.Keys\nreturn array]
    SET & GET & DEL & TTL & KEYS --> WRITER[resp.Writer\nformat response]
    WRITER --> CLIENT
```

## KV Store with TTL

```mermaid
graph TD
    SET[Set key value ttl] --> ENTRY[entry\nvalue + expiresAt]
    ENTRY --> MAP[sync.RWMutex\nmap key→entry]
    GET[Get key] --> MAP
    MAP -->|found + not expired| VALUE[return value]
    MAP -->|not found or expired| MISS[return false]
    REAPER[background goroutine\nticker 1s] -->|scan + delete expired| MAP
```

## Concurrency Model

```mermaid
graph LR
    C1[Client 1\ngoroutine] -->|RLock for reads| RWMU[sync.RWMutex]
    C2[Client 2\ngoroutine] -->|RLock for reads| RWMU
    C3[Client 3\ngoroutine] -->|Lock for writes| RWMU
    REAPER[Reaper\ngoroutine] -->|Lock for deletes| RWMU
    RWMU --> MAP[map key→entry]
```

Multiple readers can hold `RLock` simultaneously. A writer (`Set`, `Del`, reaper) acquires exclusive `Lock`.

## redis-cli Compatibility

Because we speak RESP, any Redis client works:

```bash
redis-cli -p 6380 SET foo bar EX 60
# → +OK

redis-cli -p 6380 GET foo
# → $3\r\nbar

redis-cli -p 6380 TTL foo
# → :58

redis-cli -p 6380 KEYS
# → *1\r\n$3\r\nfoo
```
