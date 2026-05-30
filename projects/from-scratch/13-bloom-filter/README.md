# 13 — Bloom Filter & HyperLogLog

Probabilistic data structures used everywhere in distributed systems: cache miss prevention, deduplication at scale, and cardinality estimation.

---

## Bloom Filter

A space-efficient probabilistic set that answers "is X in the set?" with:
- **Definitely NOT in set** (100% accurate)
- **Probably in set** (false positive possible)

```mermaid
graph LR
    KEY[key: user:42] --> H1[hash1 → bit 3]
    KEY --> H2[hash2 → bit 7]
    KEY --> H3[hash3 → bit 11]
    
    H1 --> BIT[Bit Array: 0 0 0 1 0 0 0 1 0 0 0 1 0 0 0 0]
    H2 --> BIT
    H3 --> BIT
```

```mermaid
sequenceDiagram
    participant App
    participant BF as Bloom Filter
    participant DB as Database
    
    App->>BF: Contains("user:42")?
    BF-->>App: NO → definitely not in DB
    Note over App: Skip DB query entirely
    
    App->>BF: Contains("user:7")?
    BF-->>App: YES (maybe)
    App->>DB: SELECT * FROM users WHERE id=7
    DB-->>App: result (or empty if false positive)
```

### Use Cases
- **Cache miss prevention**: Check bloom filter before hitting DB
- **Distributed dedup**: Has this event been processed?
- **Spell checkers**: Is this word in the dictionary?
- **Network routers**: Is this IP in the blocklist?

### False Positive Rate

```
FPR ≈ (1 - e^(-kn/m))^k

m = bit array size
n = number of elements inserted
k = number of hash functions
Optimal k = (m/n) * ln(2)
```

| Elements | Bits | Hash funcs | FPR |
|----------|------|-----------|-----|
| 1,000 | 9,585 | 7 | 1% |
| 1,000,000 | 9,585,058 | 7 | 1% |
| 1,000,000 | 4,792,529 | 3 | 10% |

---

## HyperLogLog

Estimates the **cardinality** (count of distinct elements) of a set using only ~12KB of memory — even for billions of elements.

```mermaid
graph TD
    ELEM[Element] --> HASH[hash → 64-bit]
    HASH --> REG[First p bits → register index]
    HASH --> RUN[Remaining bits → count leading zeros + 1]
    RUN --> MAX[registers: max of observed run lengths]
    MAX --> EST[Estimate = α * m² * harmonic mean correction]
```

**Accuracy**: ±1.04/√m standard error (m = number of registers)
- 1024 registers (1KB) → ~3.25% error
- 16384 registers (12KB) → ~0.81% error

### Use Cases
- Count unique visitors (without storing all IPs)
- Count distinct queries per day
- Redis `PFADD` / `PFCOUNT` uses HyperLogLog

---

## Running

```bash
make run    # demos both structures
make test   # verify false positive rates
```
