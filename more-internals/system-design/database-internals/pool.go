package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PoolConfig shows production connection pool settings.
type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// RecommendedPool returns production-ready pool settings.
// Rule of thumb: MaxOpen = (core_count * 2) + effective_spindle_count
// For cloud Postgres (2 vCPU): ~5-10 connections per instance.
func RecommendedPool() PoolConfig {
	return PoolConfig{
		MaxOpenConns:    25,              // don't exceed DB max_connections / app_instances
		MaxIdleConns:    5,               // keep warm connections ready
		ConnMaxLifetime: 30 * time.Minute, // rotate to pick up DNS/config changes
		ConnMaxIdleTime: 5 * time.Minute,  // release idle connections
	}
}

// ConfigurePool applies settings to a *sql.DB.
func ConfigurePool(db *sql.DB, cfg PoolConfig) {
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
}

// HealthCheck verifies the pool can acquire a connection.
func HealthCheck(ctx context.Context, db *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

func demoPool() {
	fmt.Println("\n=== Connection Pool Config ===")
	cfg := RecommendedPool()
	fmt.Printf("  MaxOpenConns:    %d\n", cfg.MaxOpenConns)
	fmt.Printf("  MaxIdleConns:    %d\n", cfg.MaxIdleConns)
	fmt.Printf("  ConnMaxLifetime: %s\n", cfg.ConnMaxLifetime)
	fmt.Printf("  ConnMaxIdleTime: %s\n", cfg.ConnMaxIdleTime)
	fmt.Println("\n  // Usage:")
	fmt.Println("  // db, _ := sql.Open(\"pgx\", dsn)")
	fmt.Println("  // ConfigurePool(db, RecommendedPool())")
	fmt.Println("  // Always use context with timeout for queries:")
	fmt.Println("  // rows, err := db.QueryContext(ctx, query, args...)")
}
