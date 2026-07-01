// pooling.go — database/sql connection pool configuration and patterns.
//
// The database/sql package manages a pool of connections automatically.
// Configuring it correctly is critical: too few connections cause request
// queuing; too many overwhelm the database (PostgreSQL default max_connections
// is 100; each connection uses ~5-10 MB RAM on the DB server).
//
// ─────────────────────────────────────────────────────────────────────────────
// Little's Law refresher
// ─────────────────────────────────────────────────────────────────────────────
//
// L = λ · W
//   L = average number of requests in the system (connections in use)
//   λ = arrival rate (requests per second)
//   W = average service time per request (seconds)
//
// Example: 500 req/s, average DB query takes 20ms (0.02s):
//   L = 500 * 0.02 = 10 connections in use at steady state.
//   Add safety margin × 2 → MaxOpenConns = 20 (or 25 for bursts).
//
// Rule of thumb: MaxOpen = (cpu_cores × 2) + effective_spindle_count
// For a 2-vCPU cloud instance: MaxOpen ≈ 5–10 per app instance.
//
// ─────────────────────────────────────────────────────────────────────────────
// pgx pool equivalent (shown as code comment — pgx is not a dependency here)
// ─────────────────────────────────────────────────────────────────────────────
//
// import "github.com/jackc/pgx/v5/pgxpool"
//
//   cfg, _ := pgxpool.ParseConfig(dsn)
//   cfg.MaxConns            = 25               // like SetMaxOpenConns
//   cfg.MinConns            = 5                // keep warm connections ready
//   cfg.MaxConnLifetime     = 5 * time.Minute  // like SetConnMaxLifetime
//   cfg.MaxConnIdleTime     = 10 * time.Minute // like SetConnMaxIdleTime
//   cfg.HealthCheckPeriod   = 30 * time.Second // background ping
//   pool, _ := pgxpool.NewWithConfig(ctx, cfg)
//
// pgxpool is generally preferred over database/sql + pgx for PostgreSQL because
// it supports pgx's native types, pipeline mode, and copy protocol.
package main

import (
	"context"
	"fmt"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// PoolingExample — demonstrates connection pool configuration.
// ─────────────────────────────────────────────────────────────────────────────

// PoolingExample walks through every pool setting in database/sql,
// explains the reasoning, and shows anti-patterns to avoid.
func PoolingExample() {
	fmt.Println("=== database/sql Connection Pool Configuration ===")
	fmt.Println()

	// ─────────────────────────────────────────────────────────────────────────
	// 1. sql.Open is LAZY — it does NOT open a connection.
	//    It only validates the driver name and DSN format.
	//    The actual connection is made on the first query.
	//
	//    Anti-pattern: calling sql.Open and assuming the database is reachable.
	//    Fix: always follow sql.Open with db.PingContext to fail fast.
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("--- 1. sql.Open is lazy ---")
	fmt.Println(`
  // sql.Open does NOT connect; it just creates the pool struct.
  db, err := sql.Open("pgx", "postgres://user:pass@localhost/mydb?sslmode=require")
  if err != nil {
      log.Fatalf("invalid DSN: %v", err)
  }
  defer db.Close()

  // PingContext forces an actual connection and fails fast on startup.
  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()
  if err := db.PingContext(ctx); err != nil {
      log.Fatalf("cannot reach database: %v", err)
  }
  log.Println("database connection verified")`)

	// ─────────────────────────────────────────────────────────────────────────
	// 2. The four pool settings and their rationale.
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n--- 2. The four pool settings ---")

	// We create a no-op sql.DB using the fake driver below for demonstration.
	// In production, replace "" with "pgx" or "postgres".
	showPoolConfig()

	// ─────────────────────────────────────────────────────────────────────────
	// 3. db.Stats() — observe pool health at runtime.
	//    Monitor these metrics in Prometheus/Datadog to catch issues early.
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n--- 3. db.Stats() — pool health metrics ---")
	showPoolStats()

	// ─────────────────────────────────────────────────────────────────────────
	// 4. Context-based query with timeout.
	//    NEVER call db.Query / db.Exec without a context in production.
	//    Without a context, a slow query holds a connection forever.
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n--- 4. Context-based query with timeout ---")
	showContextQuery()

	// ─────────────────────────────────────────────────────────────────────────
	// 5. Prepared statements and their interaction with the pool.
	//    A prepared statement in database/sql is NOT tied to one connection.
	//    sql.Stmt re-prepares on whichever connection it gets from the pool.
	//    This means the DB server sees the prepare + execute on each connection.
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n--- 5. Prepared statements and the pool ---")
	showPreparedStatements()

	// ─────────────────────────────────────────────────────────────────────────
	// 6. Anti-patterns
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n--- 6. Anti-patterns ---")
	showAntiPatterns()

	// ─────────────────────────────────────────────────────────────────────────
	// 7. Little's Law calculation
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n--- 7. Little's Law — sizing MaxOpenConns ---")
	littlesLawCalc(500, 20*time.Millisecond, 2.0)
}

// showPoolConfig prints the recommended pool configuration with reasoning.
func showPoolConfig() {
	fmt.Println("  Settings explained:")
	fmt.Println()
	fmt.Println(`  db.SetMaxOpenConns(25)
    // Maximum number of OPEN connections (in-use + idle) at any time.
    // If all 25 are busy, the 26th caller BLOCKS until one is returned.
    // Too high → DB OOM (each PG connection ~5-10 MB RAM).
    // Too low  → request queuing and high latency.
    // Start: min(DB_max_connections / num_app_instances, 25)`)
	fmt.Println()
	fmt.Println(`  db.SetMaxIdleConns(25)
    // Maximum number of IDLE (warm) connections kept in the pool.
    // Setting to same as MaxOpen means all connections stay warm.
    // Setting lower → connections closed after use → re-connect overhead.
    // Rule: SetMaxIdleConns == SetMaxOpenConns for steady-state workloads.`)
	fmt.Println()
	fmt.Println(`  db.SetConnMaxLifetime(5 * time.Minute)
    // Maximum age of any connection (even if idle).
    // Forces rotation so the app picks up:
    //   - DNS changes (cloud DB failover)
    //   - SSL cert rotation
    //   - Load balancer topology changes
    // Set to 0 = connections live forever (NOT recommended in cloud).`)
	fmt.Println()
	fmt.Println(`  db.SetConnMaxIdleTime(10 * time.Minute)
    // Maximum time a connection can sit IDLE in the pool.
    // If no query arrives within this window, the connection is closed.
    // Prevents resource leaks during low-traffic periods.
    // Must be < ConnMaxLifetime (otherwise idle ones get cut first anyway).`)
}

// showPoolStats demonstrates db.Stats() and what each field means.
func showPoolStats() {
	// db.Stats() returns sql.DBStats — a snapshot of the pool state.
	// In production, export these as Prometheus gauges.
	fmt.Println(`
  stats := db.Stats()

  Metric                  | What it means
  ─────────────────────── | ─────────────────────────────────────────────
  stats.MaxOpenConnections | cap you set with SetMaxOpenConns
  stats.OpenConnections    | current total open (in-use + idle)
  stats.InUse              | actively executing queries RIGHT NOW
  stats.Idle               | warm, available, waiting in pool
  stats.WaitCount          | times a caller had to wait for a free conn
  stats.WaitDuration       | total time callers spent waiting
  stats.MaxIdleClosed      | conns closed due to MaxIdleConns limit
  stats.MaxIdleTimeClosed  | conns closed due to ConnMaxIdleTime
  stats.MaxLifetimeClosed  | conns closed due to ConnMaxLifetime

  Alert rules:
    WaitCount   > 0  → pool might be undersized
    InUse / MaxOpen > 80%  → near saturation, consider scaling out
    MaxLifetimeClosed high → healthy rotation (expected)`)

	// Example exported metrics (pseudocode — requires prometheus SDK):
	fmt.Println(`
  // Prometheus metrics (example with prometheus/client_golang)
  //
  // func collectPoolMetrics(db *sql.DB) {
  //     s := db.Stats()
  //     dbOpenConnsGauge.Set(float64(s.OpenConnections))
  //     dbInUseGauge.Set(float64(s.InUse))
  //     dbIdleGauge.Set(float64(s.Idle))
  //     dbWaitCountGauge.Set(float64(s.WaitCount))
  // }`)
}

// showContextQuery shows the correct pattern for context-aware queries.
func showContextQuery() {
	fmt.Println(`
  // CORRECT: always pass a context with a deadline
  func getUser(ctx context.Context, db *sql.DB, userID int) (*User, error) {
      // Per-query timeout: prevents a single slow query from holding a conn
      ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
      defer cancel() // always cancel to free resources

      row := db.QueryRowContext(ctx,
          "SELECT id, name, email FROM users WHERE id = $1",
          userID,
      )

      var u User
      err := row.Scan(&u.ID, &u.Name, &u.Email)
      if errors.Is(err, sql.ErrNoRows) {
          return nil, ErrNotFound
      }
      return &u, err
  }

  // WRONG: no context — query can run forever and leak connections
  // row := db.QueryRow("SELECT ...")  ← avoid in production`)
}

// showPreparedStatements explains prepared statement behaviour with pool.
func showPreparedStatements() {
	fmt.Println(`
  // db.Prepare creates a sql.Stmt. The statement is NOT tied to one connection.
  // When you call stmt.QueryContext, database/sql:
  //   1. Picks an available connection from the pool.
  //   2. If that connection doesn't have the prepared statement, prepares it.
  //   3. Executes the query.
  //
  // This means: with N connections in the pool, the DB sees up to N PREPAREs
  // for the same statement. This is a known database/sql limitation.
  //
  // Recommendation:
  //   - For simple apps: use db.QueryContext with $1 placeholders (driver handles prep).
  //   - For high-frequency, parameterised queries: use pgx's prepared statement cache.
  //   - Avoid long-lived sql.Stmt across pool-scaling events.

  stmt, err := db.PrepareContext(ctx, "INSERT INTO events (type, data) VALUES ($1, $2)")
  if err != nil {
      return err
  }
  defer stmt.Close() // close when no longer needed — returns prep resources

  for _, event := range events {
      _, err := stmt.ExecContext(ctx, event.Type, event.Data)
      if err != nil {
          return fmt.Errorf("insert event %s: %w", event.Type, err)
      }
  }`)
}

// showAntiPatterns lists the most common connection pool mistakes.
func showAntiPatterns() {
	fmt.Println(`
  Anti-pattern 1: Setting MaxOpenConns too high
    db.SetMaxOpenConns(1000) // BAD
    // PostgreSQL default max_connections = 100
    // 1000 conns from one app = 10x oversubscription → DB OOM
    // Fix: MaxOpen <= DB_max_connections / number_of_app_instances

  Anti-pattern 2: Setting MaxIdleConns to 0
    db.SetMaxIdleConns(0) // BAD — disables the pool!
    // Every query creates + destroys a connection → 3-way TCP + TLS + auth
    // overhead on every request. ~10-100ms extra latency per query.
    // Fix: SetMaxIdleConns = SetMaxOpenConns

  Anti-pattern 3: Not closing rows
    rows, _ := db.QueryContext(ctx, "SELECT ...")
    // if you return early without rows.Close(), the connection leaks
    defer rows.Close() // ALWAYS defer rows.Close() right after QueryContext

  Anti-pattern 4: Not checking rows.Err()
    for rows.Next() { ... }
    if err := rows.Err(); err != nil { // MUST check this
        return err
    }

  Anti-pattern 5: Using sql.Open result directly without PingContext
    db, _ := sql.Open("pgx", dsn) // lazy — no actual connection yet
    db.Query("SELECT 1")          // might fail silently at startup
    // Fix: PingContext on startup to fail fast`)
}

// littlesLawCalc prints a sizing recommendation based on Little's Law.
func littlesLawCalc(rps float64, avgQueryTime time.Duration, safetyFactor float64) {
	// Little's Law: L = λ × W
	//   λ = requests per second (rps)
	//   W = average service time (seconds)
	//   L = average concurrency (connections in use)
	W := avgQueryTime.Seconds()
	L := rps * W
	recommended := int(L*safetyFactor) + 1

	fmt.Printf("  Input:  %.0f req/s, avg query = %v\n", rps, avgQueryTime)
	fmt.Printf("  L = λ × W = %.0f × %.3f = %.1f connections in use (steady state)\n",
		rps, W, L)
	fmt.Printf("  With %.1fx safety factor: recommend MaxOpenConns = %d\n",
		safetyFactor, recommended)
	fmt.Println()
	fmt.Println("  Remember:")
	fmt.Println("    - This is per app instance. Divide DB max_connections by instance count.")
	fmt.Println("    - For bursty traffic, multiply by peak_rps / avg_rps.")
	fmt.Println("    - Monitor db.Stats().WaitCount; if > 0, increase MaxOpenConns.")

	// Suppress unused import warning — context and time are used in comments
	_ = context.Background()
	_ = time.Second
}
