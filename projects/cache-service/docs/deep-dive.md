# cache-service: Deep Dive

## LRU Cache Internals

The LRU cache uses two data structures working together for O(1) operations:

```
map["b"] ──────────────────────────────────────────┐
map["a"] ──────────────────────────────────────────┼──┐
                                                   ↓  ↓
list: [front] ← c ↔ b ↔ a → [back/LRU]
```

- **Get**: look up element in map → move to front → O(1)
- **Set**: if exists, update + move to front; if new, push front + evict back if over capacity → O(1)
- **Evict**: remove `list.Back()` + delete from map → O(1)

### TTL Reaper Goroutine

```go
func (c *LRU) reap() {
    ticker := time.NewTicker(time.Second)
    for {
        select {
        case <-ticker.C:
            // Scan from back (oldest) to front, remove expired entries
        case <-c.stop:
            return
        }
    }
}
```

Scanning from back-to-front is efficient because older entries (more likely expired) are at the back. The reaper runs every second; `Get` also checks expiry inline for immediate eviction.

---

## Strategy Comparison

| | Cache-Aside | Write-Through | Write-Back |
|---|---|---|---|
| **Read** | Cache → miss → store → populate | Always cache | Always cache |
| **Write** | Store only | Cache + store atomically | Cache only; async flush |
| **Consistency** | Eventual (stale possible) | Strong | Eventual (data loss risk) |
| **Use case** | Read-heavy, tolerate stale | Read-heavy, need consistency | Write-heavy |
| **This project** | ✅ Default | ✅ Implemented | ❌ Not implemented |

---

## Singleflight: Preventing Cache Stampede

Without singleflight, a cache miss under high concurrency causes a "thundering herd":

```
1000 goroutines miss "popular-key"
→ 1000 concurrent DB queries
→ DB overload
```

With singleflight:

```
1000 goroutines miss "popular-key"
→ 1 DB query (first goroutine)
→ 999 goroutines wait and share the result
→ DB sees 1 query
```

The `group.Do(key, fn)` call deduplicates by key. The `shared` return value is `true` for all goroutines that shared a result.

---

## Redis vs LRU Backend

| | In-Memory LRU | Redis |
|---|---|---|
| **Latency** | ~100ns | ~1ms (network) |
| **Capacity** | Process memory | Configurable (GB) |
| **Persistence** | None (lost on restart) | RDB/AOF snapshots |
| **Multi-instance** | Per-process (no sharing) | Shared across all instances |
| **Use case** | Single-instance, low latency | Distributed, shared cache |

Switch with `CACHE_BACKEND=redis`.
