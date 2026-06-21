# 07-distributed-cache: Deep Dive

## RESP Protocol

Redis Serialization Protocol (RESP) uses a type prefix byte followed by data:

```
+OK\r<br>              → Simple String "OK"
-ERR message\r<br>     → Error
:42\r<br>              → Integer 42
$5\r<br>hello\r<br>      → Bulk String "hello" (5 bytes)
$-1\r<br>              → Null Bulk String
*3\r<br>$3\r<br>SET\r<br>$3\r<br>foo\r<br>$3\r<br>bar\r<br>  → Array ["SET","foo","bar"]
```

```mermaid
graph TD
    BYTE[First byte] -->|+| SS[Simple String<br>read until CRLF]
    BYTE -->|-| ERR[Error<br>read until CRLF]
    BYTE -->|:| INT[Integer<br>parse int64]
    BYTE -->|$| BULK[Bulk String<br>read length then bytes]
    BYTE -->|*| ARR[Array<br>parse N values recursively]
```

## Command Execution Flow

```mermaid
graph TD
    CLIENT[redis-cli / client] -->|RESP array| TCP[TCP :6380]
    TCP --> PARSER[resp.Parse<br>bufio.Reader]
    PARSER --> CMD[resp.Command<br>extract cmd + args]
    CMD --> ROUTER{switch cmd}
    ROUTER -->|SET| SET[store.Set<br>key value ttl]
    ROUTER -->|GET| GET[store.Get<br>return value or null]
    ROUTER -->|DEL| DEL[store.Del<br>return count]
    ROUTER -->|TTL| TTL[store.TTL<br>return seconds]
    ROUTER -->|KEYS| KEYS[store.Keys<br>return array]
    SET & GET & DEL & TTL & KEYS --> WRITER[resp.Writer<br>format response]
    WRITER --> CLIENT
```

## KV Store with TTL

```mermaid
graph TD
    SET[Set key value ttl] --> ENTRY[entry<br>value + expiresAt]
    ENTRY --> MAP[sync.RWMutex<br>map key→entry]
    GET[Get key] --> MAP
    MAP -->|found + not expired| VALUE[return value]
    MAP -->|not found or expired| MISS[return false]
    REAPER[background goroutine<br>ticker 1s] -->|scan + delete expired| MAP
```

## Concurrency Model

```mermaid
graph LR
    C1[Client 1<br>goroutine] -->|RLock for reads| RWMU[sync.RWMutex]
    C2[Client 2<br>goroutine] -->|RLock for reads| RWMU
    C3[Client 3<br>goroutine] -->|Lock for writes| RWMU
    REAPER[Reaper<br>goroutine] -->|Lock for deletes| RWMU
    RWMU --> MAP[map key→entry]
```

Multiple readers can hold `RLock` simultaneously. A writer (`Set`, `Del`, reaper) acquires exclusive `Lock`.

## redis-cli Compatibility

Because we speak RESP, any Redis client works:

```bash
redis-cli -p 6380 SET foo bar EX 60
# → +OK

redis-cli -p 6380 GET foo
# → $3\r<br>bar

redis-cli -p 6380 TTL foo
# → :58

redis-cli -p 6380 KEYS
# → *1\r<br>$3\r<br>foo
```
