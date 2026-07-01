# SQL Optimization

> Level: SDE-1 → SDE-2 | Database: PostgreSQL (concepts apply broadly)

---

## Table of Contents

1. [EXPLAIN / EXPLAIN ANALYZE](#1-explain--explain-analyze)
2. [Index Types](#2-index-types)
3. [Composite Indexes](#3-composite-indexes)
4. [The N+1 Problem](#4-the-n1-problem)
5. [Query Optimization Patterns](#5-query-optimization-patterns)
6. [Connection Pool Tuning in Go](#6-connection-pool-tuning-in-go)
7. [Slow Query Log & pg_stat_statements](#7-slow-query-log--pg_stat_statements)
8. [VACUUM and ANALYZE](#8-vacuum-and-analyze)
9. [Partitioning](#9-partitioning)
10. [Sharding Strategies](#10-sharding-strategies)
11. [Read Replicas in Go](#11-read-replicas-in-go)
12. [Quick Reference Cheatsheet](#12-quick-reference-cheatsheet)

---

## 1. EXPLAIN / EXPLAIN ANALYZE

`EXPLAIN` shows the **query plan** without running the query. `EXPLAIN ANALYZE` runs it and shows actual times.

```sql
-- Show plan only
EXPLAIN SELECT * FROM orders WHERE user_id = 'u1';

-- Run query and show real timings
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT o.id, o.status, u.name
FROM orders o
JOIN users u ON u.id = o.user_id
WHERE o.created_at > NOW() - INTERVAL '7 days';
```

### Reading the plan

```
Gather  (cost=1000.00..20543.21 rows=500 width=64) (actual time=12.3..89.4 rows=482 loops=1)
  ->  Hash Join  (cost=500..18000 rows=500 width=64)
        Hash Cond: (o.user_id = u.id)
        ->  Seq Scan on orders  (cost=0..15000 rows=500 width=48)
              Filter: (created_at > (now() - '7 days'::interval))
              Rows Removed by Filter: 49518
        ->  Hash  (cost=300..300 rows=1000 width=32)
              ->  Seq Scan on users  (cost=0..300 rows=1000 width=32)
```

| Term | Meaning |
|---|---|
| `Seq Scan` | Full table scan — reads every row. Red flag on large tables. |
| `Index Scan` | Uses a B-tree index to find rows. Good. |
| `Index Only Scan` | Reads only the index, no heap fetch. Best. |
| `Bitmap Heap Scan` | Fetches pages identified by a Bitmap Index Scan. Good for range queries. |
| `Hash Join` | Builds hash table from smaller side. Good for large unsorted joins. |
| `Nested Loop` | Iterates outer rows, probes inner. Good when inner is small/indexed. |
| `cost=X..Y` | Estimated startup cost..total cost (arbitrary planner units). |
| `rows=N` | Estimated row count — if very wrong, run `ANALYZE`. |
| `actual time=X..Y` | Real wall-clock ms. |
| `Rows Removed by Filter` | Rows read but discarded — high number = missing index. |

### Key insight

When `rows` estimate diverges wildly from `actual rows`, the planner is using stale statistics. Run:

```sql
ANALYZE orders; -- refresh statistics for one table
ANALYZE;        -- refresh all
```

---

## 2. Index Types

### B-tree (default)

Best for: equality (`=`), range (`<`, `>`, `BETWEEN`), sorting (`ORDER BY`), `LIKE 'prefix%'`.

```sql
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);
```

### Hash

Best for: equality only (`=`). Smaller than B-tree for equality lookups. No range support.

```sql
CREATE INDEX idx_sessions_token ON sessions USING HASH (token);
```

### GIN — Generalised Inverted Index

Best for: `JSONB` containment (`@>`), full-text search (`@@`), arrays (`&&`, `@>`).

```sql
-- JSONB queries
CREATE INDEX idx_products_attrs ON products USING GIN (attributes);
SELECT * FROM products WHERE attributes @> '{"color": "red"}';

-- Full-text search
CREATE INDEX idx_articles_tsv ON articles USING GIN (to_tsvector('english', body));
SELECT * FROM articles WHERE to_tsvector('english', body) @@ to_tsquery('postgres & index');
```

### GiST — Generalised Search Tree

Best for: geometric types, range types, nearest-neighbour (`<->`), PostGIS.

```sql
CREATE INDEX idx_locations_geom ON locations USING GIST (geom);
-- Nearest neighbour:
SELECT * FROM locations ORDER BY geom <-> ST_MakePoint(-74, 40.7) LIMIT 10;
```

### BRIN — Block Range Index

Best for: naturally ordered large tables (time-series, append-only logs). Tiny size, fast build, imprecise (good when rows correlate with physical order).

```sql
CREATE INDEX idx_events_ts ON events USING BRIN (occurred_at);
-- ~hundreds of KB vs GB for B-tree on the same column
```

### When to use which

| Query pattern | Index type |
|---|---|
| `WHERE id = $1` | B-tree or Hash |
| `WHERE created_at > $1` | B-tree |
| `WHERE attrs @> '{"k":"v"}'` (JSONB) | GIN |
| Full-text search | GIN |
| Geo nearest-neighbour | GiST |
| Huge append-only table, range scans | BRIN |

---

## 3. Composite Indexes

### Column order matters — leftmost prefix rule

```sql
-- Index: (user_id, status, created_at)
CREATE INDEX idx_orders_composite ON orders(user_id, status, created_at DESC);
```

This index can satisfy:
- `WHERE user_id = $1` ✅
- `WHERE user_id = $1 AND status = $2` ✅
- `WHERE user_id = $1 AND status = $2 AND created_at > $3` ✅
- `WHERE status = $1` ❌ (skipped leading column — can't use index)
- `WHERE created_at > $1` ❌ (skipped two leading columns)

**Rule:** Put equality columns first, range column last.

### INCLUDE columns (covering index)

Adds extra columns to the index leaf pages without them being part of the key. Enables Index Only Scan.

```sql
CREATE INDEX idx_orders_user_status
ON orders(user_id, status)
INCLUDE (id, total_amount, created_at);

-- Now this query hits only the index, never the table:
SELECT id, total_amount, created_at
FROM orders
WHERE user_id = $1 AND status = 'pending';
```

### Partial indexes

Index only a subset of rows. Smaller, faster to maintain.

```sql
-- Only index active orders
CREATE INDEX idx_orders_active ON orders(user_id, created_at)
WHERE status = 'active';

-- Only index unverified emails
CREATE INDEX idx_users_unverified ON users(email)
WHERE verified = false;
```

---

## 4. The N+1 Problem

### What it is

You query 1 parent record, then run N separate queries for each child — N+1 total queries.

```go
// BAD: N+1
orders, _ := db.QueryContext(ctx, "SELECT id, user_id FROM orders LIMIT 100")
for _, o := range orders {
    var user User
    db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = $1", o.UserID).Scan(&user.Name)
    // 100 separate round-trips to DB!
}
```

### How to detect

Enable query logging in Postgres:
```sql
-- postgresql.conf or:
SET log_min_duration_statement = 10; -- log queries > 10ms
SET log_statement = 'all';           -- log everything (dev only)
```

Or use `pg_stat_statements`:
```sql
SELECT query, calls, mean_exec_time
FROM pg_stat_statements
ORDER BY calls DESC LIMIT 20;
-- If you see the same parameterised query called thousands of times, N+1.
```

### Fix 1: JOIN

```go
// GOOD: single query
rows, _ := db.QueryContext(ctx, `
    SELECT o.id, o.status, u.name
    FROM orders o
    JOIN users u ON u.id = o.user_id
    LIMIT 100
`)
```

### Fix 2: IN clause (batch load)

```go
// Collect IDs first
var userIDs []string
for _, o := range orders {
    userIDs = append(userIDs, o.UserID)
}

// Batch fetch
query, args, _ := sqlx.In("SELECT id, name FROM users WHERE id IN (?)", userIDs)
query = db.Rebind(query) // convert ? to $1,$2,... for postgres
rows, _ := db.QueryContext(ctx, query, args...)
```

### Fix 3: DataLoader pattern (for GraphQL)

Collect all keys within a single request tick, batch-fetch, distribute results.

---

## 5. Query Optimization Patterns

### Expression indexes

```sql
-- Slow: function on column prevents index use
SELECT * FROM users WHERE lower(email) = 'alice@example.com';

-- Fix: expression index
CREATE INDEX idx_users_lower_email ON users(lower(email));
-- Now the query above uses the index
```

### Avoid SELECT *

```sql
-- BAD: fetches all columns, prevents Index Only Scan
SELECT * FROM orders WHERE user_id = $1;

-- GOOD: fetch only what you need
SELECT id, status, total_amount FROM orders WHERE user_id = $1;
```

### Use EXISTS over COUNT

```sql
-- BAD: counts all matching rows
SELECT COUNT(*) FROM orders WHERE user_id = $1;
IF count > 0 { ... }

-- GOOD: stops at first match
SELECT EXISTS(SELECT 1 FROM orders WHERE user_id = $1);
```

### Avoid implicit type casts

```sql
-- BAD: user_id is UUID, $1 is text — cast prevents index use
WHERE user_id = $1

-- FIX in Go: pass correct type
db.QueryContext(ctx, "SELECT ... WHERE user_id = $1", uuid.MustParse(id))
```

### Pagination: keyset > OFFSET

```sql
-- BAD: OFFSET 10000 still reads 10000 rows
SELECT * FROM orders ORDER BY created_at DESC LIMIT 20 OFFSET 10000;

-- GOOD: keyset pagination
SELECT * FROM orders
WHERE created_at < $1  -- last seen value
ORDER BY created_at DESC
LIMIT 20;
```

---

## 6. Connection Pool Tuning in Go

### database/sql

```go
db, err := sql.Open("pgx", dsn)

// Total open connections (idle + in-use) allowed to the DB
db.SetMaxOpenConns(25)

// Connections kept open when idle — avoids reconnect overhead
db.SetMaxIdleConns(10)

// Recycle connections older than this — avoids stale TCP connections
db.SetConnMaxLifetime(5 * time.Minute)

// Close idle connections sitting longer than this
db.SetConnMaxIdleTime(2 * time.Minute)
```

### pgxpool

```go
import "github.com/jackc/pgx/v5/pgxpool"

cfg, _ := pgxpool.ParseConfig(dsn)
cfg.MaxConns           = 25
cfg.MinConns           = 5
cfg.MaxConnLifetime    = 5 * time.Minute
cfg.MaxConnIdleTime    = 2 * time.Minute
cfg.HealthCheckPeriod  = 1 * time.Minute

pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
```

### How to size the pool

**Postgres has a hard `max_connections` limit** (default 100). Every connection uses ~5-10 MB RAM.

**Little's Law:**  
`N = λ × W`
- `N` = number of concurrent connections needed
- `λ` = query throughput (queries/sec)
- `W` = average query latency (sec)

Example: 500 req/s, each query takes 10ms average:
`N = 500 × 0.010 = 5 connections`

In practice, add headroom for spikes:

```
pool_size = (core_count × 2) + effective_spindle_count
```

**PgBouncer rule of thumb:** Keep `MaxOpenConns` per app instance low (10-25). Use PgBouncer in transaction-mode pooling to multiplex thousands of app connections onto fewer Postgres connections.

```go
// Monitoring pool health
stats := db.Stats()
log.Printf("open=%d idle=%d in_use=%d wait=%d wait_duration=%s",
    stats.OpenConnections,
    stats.Idle,
    stats.InUse,
    stats.WaitCount,
    stats.WaitDuration,
)
```

---

## 7. Slow Query Log & pg_stat_statements

### Enable pg_stat_statements

```sql
-- postgresql.conf
shared_preload_libraries = 'pg_stat_statements'
pg_stat_statements.track = all

-- After restart:
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
```

### Find slow queries

```sql
SELECT
    left(query, 80)          AS query,
    calls,
    round(mean_exec_time::numeric, 2) AS mean_ms,
    round(total_exec_time::numeric, 2) AS total_ms,
    round(stddev_exec_time::numeric, 2) AS stddev_ms,
    rows / calls             AS avg_rows
FROM pg_stat_statements
WHERE calls > 100
ORDER BY mean_exec_time DESC
LIMIT 20;
```

### auto_explain

Logs EXPLAIN ANALYZE automatically for slow queries — no code changes needed.

```sql
-- postgresql.conf
shared_preload_libraries = 'auto_explain'
auto_explain.log_min_duration = 100  -- log queries > 100ms
auto_explain.log_analyze = true
auto_explain.log_buffers = true
```

---

## 8. VACUUM and ANALYZE

### MVCC (Multi-Version Concurrency Control)

Postgres never overwrites rows in-place. `UPDATE` creates a new row version (tuple); the old tuple becomes "dead". `DELETE` just marks the tuple dead.

**Without VACUUM:** dead tuples accumulate → table bloat → slower scans.

### VACUUM

```sql
VACUUM orders;           -- remove dead tuples, mark space reusable
VACUUM FULL orders;      -- rewrite table, reclaim disk space (takes AccessExclusiveLock!)
VACUUM ANALYZE orders;   -- vacuum + update stats
```

### Autovacuum tuning

```sql
-- Per-table override (preferred over global config)
ALTER TABLE orders SET (
    autovacuum_vacuum_scale_factor = 0.01,  -- trigger at 1% dead tuples (default 20%)
    autovacuum_analyze_scale_factor = 0.005,
    autovacuum_vacuum_cost_delay = 2        -- ms (lower = more aggressive)
);
```

For high-write tables (events, logs), reduce `autovacuum_vacuum_scale_factor` significantly — otherwise bloat builds up before autovacuum fires.

### Bloat check

```sql
SELECT relname, n_dead_tup, n_live_tup,
       round(n_dead_tup::numeric / nullif(n_live_tup + n_dead_tup, 0) * 100, 1) AS dead_pct
FROM pg_stat_user_tables
ORDER BY n_dead_tup DESC;
```

---

## 9. Partitioning

Splits one logical table into multiple physical partitions. Postgres prunes irrelevant partitions at query time.

### Range partitioning (most common — time-series data)

```sql
CREATE TABLE events (
    id          BIGSERIAL,
    occurred_at TIMESTAMPTZ NOT NULL,
    payload     JSONB
) PARTITION BY RANGE (occurred_at);

CREATE TABLE events_2024_q1 PARTITION OF events
    FOR VALUES FROM ('2024-01-01') TO ('2024-04-01');

CREATE TABLE events_2024_q2 PARTITION OF events
    FOR VALUES FROM ('2024-04-01') TO ('2024-07-01');

-- Index applies to all partitions
CREATE INDEX ON events(occurred_at);
```

```sql
-- Planner prunes: only scans events_2024_q1
SELECT * FROM events WHERE occurred_at BETWEEN '2024-02-01' AND '2024-03-01';
```

### List partitioning

```sql
CREATE TABLE orders (
    id     BIGSERIAL,
    region TEXT NOT NULL,
    ...
) PARTITION BY LIST (region);

CREATE TABLE orders_us PARTITION OF orders FOR VALUES IN ('us-east', 'us-west');
CREATE TABLE orders_eu PARTITION OF orders FOR VALUES IN ('eu-west', 'eu-central');
```

### Hash partitioning (horizontal sharding within one DB)

```sql
CREATE TABLE users (
    id UUID NOT NULL,
    ...
) PARTITION BY HASH (id);

CREATE TABLE users_0 PARTITION OF users FOR VALUES WITH (MODULUS 4, REMAINDER 0);
CREATE TABLE users_1 PARTITION OF users FOR VALUES WITH (MODULUS 4, REMAINDER 1);
CREATE TABLE users_2 PARTITION OF users FOR VALUES WITH (MODULUS 4, REMAINDER 2);
CREATE TABLE users_3 PARTITION OF users FOR VALUES WITH (MODULUS 4, REMAINDER 3);
```

### When to partition

- Table > ~100 GB and queries always filter on the partition key.
- You need to drop old data fast (`DROP TABLE events_2023_q1` is instant vs `DELETE`).
- Archiving: move cold partitions to cheaper storage.

---

## 10. Sharding Strategies with Go

Sharding distributes data across multiple database servers.

### Application-level sharding

```go
type ShardRouter struct {
    shards []*sql.DB
}

func (r *ShardRouter) ShardFor(tenantID string) *sql.DB {
    // Consistent hash to pick shard
    h := fnv.New32a()
    h.Write([]byte(tenantID))
    idx := int(h.Sum32()) % len(r.shards)
    return r.shards[idx]
}

func (r *ShardRouter) GetOrders(ctx context.Context, tenantID string) ([]*Order, error) {
    db := r.ShardFor(tenantID)
    rows, err := db.QueryContext(ctx, "SELECT * FROM orders WHERE tenant_id = $1", tenantID)
    // ...
}
```

### Cross-shard queries

Avoid them when possible. If needed:
1. Query each shard in parallel → merge results in Go.
2. Use a scatter-gather pattern with `errgroup`.

```go
func (r *ShardRouter) SearchAllShards(ctx context.Context, query string) ([]*Order, error) {
    g, ctx := errgroup.WithContext(ctx)
    results := make([][]*Order, len(r.shards))

    for i, db := range r.shards {
        i, db := i, db
        g.Go(func() error {
            rows, err := db.QueryContext(ctx, "SELECT * FROM orders WHERE ...", query)
            if err != nil {
                return err
            }
            results[i] = scanOrders(rows)
            return nil
        })
    }
    if err := g.Wait(); err != nil {
        return nil, err
    }
    return mergeOrders(results...), nil
}
```

---

## 11. Read Replicas in Go

Route read queries to replicas, writes to primary.

```go
type DBCluster struct {
    primary  *sql.DB
    replicas []*sql.DB
    rr       atomic.Uint64 // round-robin counter
}

func (c *DBCluster) Primary() *sql.DB { return c.primary }

func (c *DBCluster) Replica() *sql.DB {
    if len(c.replicas) == 0 {
        return c.primary // fallback
    }
    idx := c.rr.Add(1) % uint64(len(c.replicas))
    return c.replicas[idx]
}

// Usage
func (r *OrderRepo) Create(ctx context.Context, o *Order) error {
    _, err := r.db.Primary().ExecContext(ctx, "INSERT INTO orders ...", ...)
    return err
}

func (r *OrderRepo) List(ctx context.Context, userID string) ([]*Order, error) {
    rows, err := r.db.Replica().QueryContext(ctx, "SELECT ... FROM orders WHERE user_id = $1", userID)
    // ...
}
```

**Replication lag:** Replicas can be slightly behind. For reads-after-writes:
```go
// Option 1: read from primary for 1s after write
// Option 2: use synchronous_commit = on for critical tables
// Option 3: read your own write token (send LSN to client, replica catches up)
```

---

## 12. Quick Reference Cheatsheet

### Index decision tree

```
Query has equality on column?
  → B-tree or Hash (Hash if equality-only)

Query has range / ORDER BY?
  → B-tree

Query uses JSONB containment @> ?
  → GIN

Query uses full-text search @@?
  → GIN

Query is geo nearest-neighbour?
  → GiST

Table is huge, append-only, range scans on sequential column?
  → BRIN

Multiple equality columns + one range?
  → Composite B-tree (equalities first, range last)

Query only needs subset of columns?
  → INCLUDE (covering index)

Query only touches subset of rows?
  → Partial index (WHERE clause)
```

### Connection pool quick reference

```go
db.SetMaxOpenConns(25)           // total connections (tuned to DB max_connections)
db.SetMaxIdleConns(10)           // keep N warm
db.SetConnMaxLifetime(5*time.Minute)  // recycle old connections
db.SetConnMaxIdleTime(2*time.Minute)  // close long-idle connections
```

### Key SQL anti-patterns

| Anti-pattern | Fix |
|---|---|
| `SELECT *` | Select only needed columns |
| `OFFSET n` on large tables | Keyset pagination |
| Function on indexed column in WHERE | Expression index |
| N+1 loop queries | JOIN or IN clause |
| `COUNT(*) > 0` | `EXISTS` |
| Wrong column order in composite index | Equality columns first |
| `LIKE '%suffix'` | Full-text search (GIN) |
| Missing `ANALYZE` after bulk load | Run `ANALYZE tablename` |

### EXPLAIN quick scan

1. Look for `Seq Scan` on large tables → add index.
2. Compare `rows` estimate vs `actual rows` — big difference → run `ANALYZE`.
3. Look for `Rows Removed by Filter` — high number → wrong index or missing index.
4. `Index Only Scan` = best case (no heap fetch).
5. Startup cost matters for `LIMIT` queries — nested loop + index often wins over hash join.
