# Multi-Tenancy Patterns

> **SDE-2** — Critical for SaaS products. Comes up in system design rounds for any B2B/platform role.

---

## The Three Models

```
┌─────────────────────────────────────────────────────────────────┐
│  SILO (dedicated infra)  │  SCHEMA-PER-TENANT  │  ROW-LEVEL    │
│                          │                     │  ISOLATION    │
│  Tenant A → DB-A         │  DB                 │  DB           │
│  Tenant B → DB-B         │  ├── schema_a        │  ├── users    │
│  Tenant C → DB-C         │  ├── schema_b        │  │   tenant_id│
│                          │  └── schema_c        │  └── orders   │
│  Full isolation          │  Medium isolation   │   tenant_id   │
│  High cost               │  Medium cost        │  Low cost     │
└─────────────────────────────────────────────────────────────────┘
```

---

## Trade-Off Table

| Dimension | Silo | Schema-per-tenant | Row-level isolation |
|-----------|------|-------------------|---------------------|
| **Data isolation** | ✅ Complete | ✅ Strong | ⚠️ Logical only |
| **Noisy neighbor** | ✅ None | ⚠️ Shared DB server | ❌ Shared tables |
| **Cost** | ❌ High (N × infra) | Medium | ✅ Low |
| **Ops complexity** | ❌ High (N deploys) | Medium | ✅ Low |
| **Schema migrations** | ❌ N migrations | ❌ N migrations | ✅ One migration |
| **Blast radius** | ✅ One tenant | ✅ One schema | ❌ Bug affects all |
| **Compliance (GDPR/HIPAA)** | ✅ Easiest (data at rest per tenant) | ✅ Good | ⚠️ Need RLS + audit |
| **Cross-tenant queries** | ❌ Hard | ❌ Hard | ✅ Easy |
| **Tenant onboarding** | ❌ Provision infra | Medium (create schema) | ✅ Insert row |
| **Max tenants** | ~100s | ~1000s | Unlimited |

---

## Model 1: Silo (Dedicated Infra)

Each tenant gets their own database, and often their own service instance.

**Use when:**
- Enterprise customers demand contractual data isolation
- Compliance requires (HIPAA Business Associate Agreements, FedRAMP)
- Tenants have wildly different load profiles
- You can charge a premium that justifies the cost

**Examples:** GitHub Enterprise Server, Salesforce Government Cloud, AWS GovCloud

```go
// Tenant-aware connection factory
type TenantDB struct {
    mu    sync.RWMutex
    conns map[string]*sql.DB // tenantID → dedicated DB
}

func (t *TenantDB) Get(tenantID string) (*sql.DB, error) {
    t.mu.RLock()
    db, ok := t.conns[tenantID]
    t.mu.RUnlock()
    if ok {
        return db, nil
    }

    // Lazy connect — look up DSN from config store
    dsn, err := t.lookupDSN(tenantID)
    if err != nil {
        return nil, fmt.Errorf("tenant %s not found: %w", tenantID, err)
    }
    db, err = sql.Open("postgres", dsn)
    if err != nil {
        return nil, err
    }
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)

    t.mu.Lock()
    t.conns[tenantID] = db
    t.mu.Unlock()
    return db, nil
}
```

---

## Model 2: Schema-per-Tenant

One PostgreSQL cluster, one database, but each tenant has their own schema.
`SET search_path = tenant_abc` routes all queries to the right tables.

**Use when:**
- You want isolation without the cost of N databases
- Tenants number in the hundreds to low thousands
- You need per-tenant customisation (extra columns, different indexes)

**Examples:** Notion (early), many mid-market SaaS products

```sql
-- Create tenant schema on onboarding
CREATE SCHEMA tenant_abc;
SET search_path = tenant_abc;

CREATE TABLE users (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL
);
-- Each tenant gets their own copy of every table
```

```go
// Middleware: inject schema into context, set search_path per query
func (r *Repository) withTenant(ctx context.Context, db *sql.DB) (*sql.Conn, error) {
    tenantID := TenantFromContext(ctx)
    if tenantID == "" {
        return nil, errors.New("missing tenant in context")
    }

    // Get a dedicated connection and set the search path
    conn, err := db.Conn(ctx)
    if err != nil {
        return nil, err
    }
    schema := "tenant_" + tenantID // sanitize tenantID first!
    if _, err := conn.ExecContext(ctx, "SET search_path = "+pq.QuoteIdentifier(schema)); err != nil {
        conn.Close()
        return nil, fmt.Errorf("set search_path: %w", err)
    }
    return conn, nil
}
```

### Migration Management (Schema-per-Tenant)

```go
// Run migrations on ALL tenant schemas
func MigrateAll(db *sql.DB, migrationsDir string) error {
    rows, err := db.Query(`SELECT schema_name FROM information_schema.schemata
                           WHERE schema_name LIKE 'tenant_%'`)
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var schema string
        rows.Scan(&schema)

        m, err := migrate.New("file://"+migrationsDir,
            fmt.Sprintf("postgres://...?search_path=%s", schema))
        if err != nil {
            return fmt.Errorf("migrate %s: %w", schema, err)
        }
        if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
            return fmt.Errorf("migrate %s: %w", schema, err)
        }
    }
    return nil
}
```

---

## Model 3: Row-Level Isolation

All tenants share the same tables. Every table has a `tenant_id` column.
This is the most common model for high-scale SaaS.

**Use when:**
- Tens of thousands of tenants
- You need cross-tenant analytics
- Operational simplicity is paramount
- You can enforce isolation in the application layer (+ optionally DB RLS)

**Examples:** Slack, Linear, most modern SaaS

### Database Schema

```sql
-- Every table has tenant_id — always in the WHERE clause
CREATE TABLE users (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    email      TEXT        NOT NULL,
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT uq_tenant_email UNIQUE (tenant_id, email)
);

-- Composite index: tenant_id first so queries always filter by tenant
CREATE INDEX idx_users_tenant ON users (tenant_id, created_at DESC);

CREATE TABLE orders (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL REFERENCES tenants(id),
    user_id    UUID        NOT NULL,
    status     TEXT        NOT NULL,
    total      NUMERIC(12,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_orders_tenant ON orders (tenant_id, status, created_at DESC);
```

### Go: Context-Propagated Tenant ID

```go
// context key — unexported to prevent collisions
type contextKey string
const tenantKey contextKey = "tenant_id"

// Set tenant in context (called by middleware)
func WithTenant(ctx context.Context, tenantID string) context.Context {
    return context.WithValue(ctx, tenantKey, tenantID)
}

// Extract tenant from context (used by repositories)
func TenantFromContext(ctx context.Context) (string, error) {
    id, ok := ctx.Value(tenantKey).(string)
    if !ok || id == "" {
        return "", errors.New("tenant_id missing from context")
    }
    return id, nil
}
```

### Go: Tenant Middleware

```go
// HTTP middleware: extract tenant from JWT sub-claim, subdomain, or X-Tenant-ID header
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var tenantID string

        // Strategy 1: from JWT claims (most common)
        if claims, ok := r.Context().Value("jwt_claims").(*JWTClaims); ok {
            tenantID = claims.TenantID
        }

        // Strategy 2: from subdomain (acme.myapp.com → "acme")
        if tenantID == "" {
            host := r.Host
            if idx := strings.Index(host, "."); idx > 0 {
                tenantID = host[:idx]
            }
        }

        // Strategy 3: from explicit header (internal services)
        if tenantID == "" {
            tenantID = r.Header.Get("X-Tenant-ID")
        }

        if tenantID == "" {
            http.Error(w, "missing tenant", http.StatusUnauthorized)
            return
        }

        ctx := WithTenant(r.Context(), tenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Go: Repository Layer — Always Inject tenant_id

```go
type UserRepository struct {
    db *sql.DB
}

// ALWAYS include tenant_id in WHERE — never query without it
func (r *UserRepository) FindByID(ctx context.Context, userID string) (*User, error) {
    tenantID, err := TenantFromContext(ctx)
    if err != nil {
        return nil, err
    }

    var u User
    err = r.db.QueryRowContext(ctx,
        `SELECT id, tenant_id, email, name FROM users
         WHERE tenant_id = $1 AND id = $2`,  // tenant_id first (index prefix)
        tenantID, userID,
    ).Scan(&u.ID, &u.TenantID, &u.Email, &u.Name)

    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrNotFound
    }
    return &u, err
}

func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*User, error) {
    tenantID, err := TenantFromContext(ctx)
    if err != nil {
        return nil, err
    }

    rows, err := r.db.QueryContext(ctx,
        `SELECT id, tenant_id, email, name FROM users
         WHERE tenant_id = $1
         ORDER BY created_at DESC
         LIMIT $2 OFFSET $3`,
        tenantID, limit, offset,
    )
    // ...
}
```

### PostgreSQL Row-Level Security (RLS)

RLS adds a **database-level enforcement** layer on top of the application check.
Even if a bug in the app omits `tenant_id = $1`, Postgres blocks the query.

```sql
-- Enable RLS on the table
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY; -- applies to table owner too

-- Policy: a session can only see rows matching current_setting('app.tenant_id')
CREATE POLICY tenant_isolation ON users
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

-- Same for other tables
ALTER TABLE orders ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON orders
    USING (tenant_id = current_setting('app.tenant_id')::UUID);
```

```go
// Set the session variable at the start of each request (per-connection)
func (r *UserRepository) setTenantSession(ctx context.Context, conn *sql.Conn, tenantID string) error {
    _, err := conn.ExecContext(ctx,
        "SELECT set_config('app.tenant_id', $1, true)", // true = local to transaction
        tenantID,
    )
    return err
}

// Usage with RLS
func (r *UserRepository) FindByIDWithRLS(ctx context.Context, userID string) (*User, error) {
    tenantID, err := TenantFromContext(ctx)
    if err != nil {
        return nil, err
    }

    conn, err := r.db.Conn(ctx)
    if err != nil {
        return nil, err
    }
    defer conn.Close()

    if err := r.setTenantSession(ctx, conn, tenantID); err != nil {
        return nil, err
    }

    // RLS policy automatically filters — WHERE tenant_id = current_setting(...)
    // The app-level WHERE is still good practice (defense in depth)
    var u User
    err = conn.QueryRowContext(ctx,
        `SELECT id, tenant_id, email, name FROM users WHERE id = $1`,
        userID,
    ).Scan(&u.ID, &u.TenantID, &u.Email, &u.Name)
    // ...
    return &u, err
}
```

---

## Connection Pooling Per Tenant vs Shared Pool

### Shared Pool (Default — Recommended for row-level)

```go
// One pool for all tenants — simple, efficient
db, _ := sql.Open("postgres", "postgres://localhost/myapp")
db.SetMaxOpenConns(50)
db.SetMaxIdleConns(10)
// tenant_id injected per query via WHERE clause or set_config
```

### Per-Tenant Pool (For schema-per-tenant or silo)

```go
type PoolManager struct {
    mu    sync.RWMutex
    pools map[string]*pgxpool.Pool
}

func (pm *PoolManager) Get(ctx context.Context, tenantID string) (*pgxpool.Pool, error) {
    pm.mu.RLock()
    pool, ok := pm.pools[tenantID]
    pm.mu.RUnlock()
    if ok {
        return pool, nil
    }

    cfg, _ := pgxpool.ParseConfig(dsnForTenant(tenantID))
    cfg.MaxConns = 10 // per-tenant limit — prevent one tenant starving others
    cfg.MinConns = 2

    pool, err := pgxpool.NewWithConfig(ctx, cfg)
    if err != nil {
        return nil, err
    }

    pm.mu.Lock()
    pm.pools[tenantID] = pool
    pm.mu.Unlock()
    return pool, nil
}
```

---

## Noisy Neighbor Mitigation

```go
// Rate limiter per tenant — each tenant gets their own token bucket
type TenantRateLimiter struct {
    mu       sync.Mutex
    limiters map[string]*rate.Limiter
    rps      float64 // requests per second per tenant
    burst    int
}

func (trl *TenantRateLimiter) Allow(tenantID string) bool {
    trl.mu.Lock()
    l, ok := trl.limiters[tenantID]
    if !ok {
        l = rate.NewLimiter(rate.Limit(trl.rps), trl.burst)
        trl.limiters[tenantID] = l
    }
    trl.mu.Unlock()
    return l.Allow()
}

// HTTP middleware
func TenantRateLimitMiddleware(limiter *TenantRateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            tenantID, _ := TenantFromContext(r.Context())
            if !limiter.Allow(tenantID) {
                w.Header().Set("Retry-After", "1")
                http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## Tenant Onboarding

```go
// Row-level: onboarding is just an INSERT
func OnboardTenant(ctx context.Context, db *sql.DB, name, plan string) (string, error) {
    var tenantID string
    err := db.QueryRowContext(ctx,
        `INSERT INTO tenants (name, plan, status, created_at)
         VALUES ($1, $2, 'active', NOW())
         RETURNING id`,
        name, plan,
    ).Scan(&tenantID)
    return tenantID, err
}

// Schema-per-tenant: onboarding requires DDL
func OnboardTenantSchema(ctx context.Context, db *sql.DB, tenantID string) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    schema := "tenant_" + sanitize(tenantID)
    if _, err := tx.ExecContext(ctx, "CREATE SCHEMA "+pq.QuoteIdentifier(schema)); err != nil {
        return fmt.Errorf("create schema: %w", err)
    }
    // Run migrations for the new schema...
    return tx.Commit()
}
```

---

## Data Isolation: GDPR / Compliance

```go
// GDPR Right to Erasure — delete all tenant data
func PurgeTenant(ctx context.Context, db *sql.DB, tenantID string) error {
    // Row-level: delete all rows belonging to the tenant
    // Order matters — respect FK constraints
    tables := []string{"order_items", "orders", "sessions", "users", "tenants"}

    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    for _, table := range tables {
        if _, err := tx.ExecContext(ctx,
            fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1", table),
            tenantID,
        ); err != nil {
            return fmt.Errorf("purge %s: %w", table, err)
        }
    }
    return tx.Commit()
}

// Schema-per-tenant: drop the whole schema
func PurgeTenantSchema(ctx context.Context, db *sql.DB, tenantID string) error {
    schema := "tenant_" + sanitize(tenantID)
    _, err := db.ExecContext(ctx,
        "DROP SCHEMA "+pq.QuoteIdentifier(schema)+" CASCADE",
    )
    return err
}
```

---

## Testing Multi-Tenant Code

```go
// Test helper: run test in an isolated tenant context
func withTestTenant(t *testing.T, db *sql.DB) (context.Context, string, func()) {
    t.Helper()
    tenantID := uuid.New().String()

    // Insert test tenant
    _, err := db.Exec(`INSERT INTO tenants (id, name) VALUES ($1, 'test-tenant')`, tenantID)
    require.NoError(t, err)

    ctx := WithTenant(context.Background(), tenantID)

    cleanup := func() {
        db.Exec(`DELETE FROM users WHERE tenant_id = $1`, tenantID)
        db.Exec(`DELETE FROM tenants WHERE id = $1`, tenantID)
    }
    return ctx, tenantID, cleanup
}

func TestUserRepository_IsolatesTenants(t *testing.T) {
    db := testDB(t)
    repo := &UserRepository{db: db}

    // Create two tenants
    ctx1, tenant1, cleanup1 := withTestTenant(t, db)
    ctx2, _, cleanup2 := withTestTenant(t, db)
    defer cleanup1()
    defer cleanup2()

    // Insert user in tenant 1
    userID := insertUser(t, db, tenant1, "alice@example.com")

    // Tenant 1 can see the user
    u, err := repo.FindByID(ctx1, userID)
    require.NoError(t, err)
    assert.Equal(t, "alice@example.com", u.Email)

    // Tenant 2 CANNOT see tenant 1's user
    _, err = repo.FindByID(ctx2, userID)
    assert.ErrorIs(t, err, ErrNotFound, "tenant isolation broken!")
}
```

---

## Real-World Examples

| Company | Model | Why |
|---------|-------|-----|
| **Salesforce** | Schema-per-tenant (Org) | Enterprise isolation, compliance |
| **Slack** | Row-level + sharding by workspace | Scale (millions of workspaces) |
| **GitHub Enterprise** | Silo (dedicated instances) | Enterprise contractual isolation |
| **Linear** | Row-level isolation | Simplicity at their scale |
| **Notion** | Started schema-per-tenant, migrated to row-level | Operational complexity |
| **Shopify** | Sharded row-level (by shop_id) | Extreme scale |

---

## When to Pick Each Model

```
Startup / SMB SaaS (< 10k tenants, no compliance requirements)
→ Row-level isolation. Simple, cheap, fast to build.

Mid-market SaaS (< 10k tenants, some enterprise customers asking for isolation)
→ Schema-per-tenant OR row-level + RLS. Gives isolation story without silo cost.

Enterprise SaaS / Regulated industry (HIPAA, FedRAMP, financial)
→ Silo model for enterprise tier. Row-level for SMB tier (tiered isolation).

Hyper-scale SaaS (millions of tenants, Slack-scale)
→ Row-level + horizontal sharding by tenant_id.
   Each shard is a Postgres cluster serving a range of tenant_ids.
```

---

## Common Mistakes

1. **Forgetting `tenant_id` in a WHERE clause** — query leaks data across tenants. Fix: RLS as a safety net + integration tests that verify isolation.
2. **Trusting `tenant_id` from the request body** — always extract from the authenticated JWT/session, never from user input.
3. **Composite index with `tenant_id` last** — put `tenant_id` first in every index so the query planner can use it as a prefix filter.
4. **Shared sequence/auto-increment IDs** — use UUIDs so tenant A can't enumerate tenant B's IDs by guessing integers.
5. **No rate limiting per tenant** — one runaway tenant can starve all others. Always rate-limit at the tenant level.
