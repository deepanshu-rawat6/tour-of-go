# ADR-004: sqlx + Raw SQL over GORM

**Status:** Accepted  
**Date:** 2026-03-31

## Context

We need a database access layer for PostgreSQL.

## Decision

Use `jmoiron/sqlx` with raw SQL queries.

## Rationale

The Java codebase uses JPA/Hibernate which hides SQL behind annotations. This caused several issues in production:
- N+1 query problems discovered only in production
- Unexpected lazy loading causing `LazyInitializationException`
- Difficulty optimizing specific queries

In Go, GORM has similar issues. Raw SQL + sqlx gives us:

1. **Explicit queries** — every SQL statement is visible in the code
2. **Struct scanning** — sqlx maps rows to structs without reflection magic
3. **Batch operations** — `sqlx.In()` for `WHERE id IN (...)` queries
4. **No magic** — no hidden queries, no lazy loading, no proxy objects
5. **Performance** — direct `database/sql` with minimal overhead

## Trade-offs

- More verbose than GORM for simple CRUD
- No automatic migrations (we use SQL files in `migrations/`)
- Manual JSON marshalling for JSONB columns

## Connection Pooling

We configure `sql.DB` directly (equivalent to Hikari in Java):
```go
db.SetMaxOpenConns(20)   // Hikari: maximum-pool-size
db.SetMaxIdleConns(5)    // Hikari: minimum-idle
db.SetConnMaxLifetime(1 * time.Hour)
```
