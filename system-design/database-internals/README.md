# Database Internals

How storage engines, query optimization, connection pooling, and sharding actually work — with Go implementations.

---

## Storage Engines

### B-Tree (PostgreSQL, MySQL InnoDB)

B-trees are the backbone of relational databases. Every index you create is a B-tree.

```mermaid
graph TD
    ROOT["Root [30 | 60]"] --> L1["[10 | 20]"]
    ROOT --> L2["[40 | 50]"]
    ROOT --> L3["[70 | 80]"]
    L1 --> D1["data pages"]
    L2 --> D2["data pages"]
    L3 --> D3["data pages"]
```

**Properties:**
- Balanced — all leaves at same depth → O(log N) lookups
- High fan-out — each node holds many keys → fewer disk reads
- In-place updates — modify pages directly (needs WAL for crash safety)
- Great for: point lookups, range scans, read-heavy workloads

### LSM-Tree (RocksDB, LevelDB, Cassandra)

LSM-trees optimize for writes by buffering in memory and flushing sorted runs to disk.

```mermaid
graph TD
    W[Write] --> MEM[MemTable<br>red-black tree<br>in-memory sorted]
    MEM -->|flush when full| L0[Level 0<br>SSTables unsorted]
    L0 -->|compaction| L1[Level 1<br>SSTables sorted, no overlap]
    L1 -->|compaction| L2[Level 2<br>larger sorted runs]
    
    R[Read] --> MEM
    R --> BF[Bloom Filter<br>per-SSTable]
    BF -->|maybe exists| L0
    BF --> L1
    BF --> L2
```

**Properties:**
- Sequential writes only → 10-100x faster writes than B-tree
- Read amplification — must check multiple levels
- Bloom filters reduce unnecessary disk reads
- Compaction reclaims space and merges levels
- Great for: write-heavy workloads, time-series, logs

### Comparison

| Aspect | B-Tree | LSM-Tree |
|--------|--------|----------|
| Write | O(log N) random I/O | O(1) amortized sequential |
| Read | O(log N) single path | O(log N) × levels |
| Space | ~50% page utilization | Compaction overhead |
| Use case | OLTP, read-heavy | Write-heavy, append-only |

---

## Query Optimization

```mermaid
graph LR
    SQL[SQL Query] --> PARSE[Parser<br>AST]
    PARSE --> PLAN[Planner<br>logical plan]
    PLAN --> OPT[Optimizer<br>cost-based]
    OPT --> EXEC[Executor<br>physical plan]
    
    OPT --> IDX[Index Scan<br>vs Seq Scan]
    OPT --> JOIN[Nested Loop<br>vs Hash Join<br>vs Merge Join]
    OPT --> STATS[Table Statistics<br>row count, cardinality]
```

**Key rules:**
- `EXPLAIN ANALYZE` — always check the actual execution plan
- Index on columns in WHERE, JOIN, ORDER BY
- Covering indexes avoid heap lookups
- Composite index order matters: `(a, b)` helps `WHERE a=1 AND b=2` but not `WHERE b=2`

---

## Connection Pooling

```mermaid
sequenceDiagram
    participant App as App (100 goroutines)
    participant Pool as Connection Pool (max=20)
    participant DB as PostgreSQL (max_connections=100)
    
    App->>Pool: Acquire()
    Pool-->>App: *sql.DB conn (reused)
    App->>DB: SELECT ...
    DB-->>App: rows
    App->>Pool: Release (defer rows.Close)
    
    Note over Pool: Idle connections kept warm
    Note over Pool: MaxOpenConns, MaxIdleConns, ConnMaxLifetime
```

---

## Sharding Strategies

```mermaid
graph TD
    REQ[Request] --> ROUTER[Shard Router]
    ROUTER -->|hash(user_id) % N| HASH[Hash Sharding<br>uniform distribution<br>no range queries]
    ROUTER -->|user_id 1-1M| RANGE[Range Sharding<br>range queries OK<br>hot spots possible]
    ROUTER -->|geography| GEO[Geo Sharding<br>data locality<br>compliance]
    
    HASH --> S1[(Shard 1)]
    HASH --> S2[(Shard 2)]
    HASH --> S3[(Shard 3)]
```

| Strategy | Pros | Cons |
|----------|------|------|
| Hash | Uniform distribution | No range queries, resharding is painful |
| Range | Range scans, easy splits | Hot spots on recent data |
| Directory | Flexible routing | Single point of failure (directory) |

---

## Go Implementation

See `btree.go` for a minimal B-tree, `lsm.go` for an LSM-tree with memtable + SSTable flush, and `pool.go` for connection pool configuration patterns.

```bash
go test -v -race ./...
go run .
```
