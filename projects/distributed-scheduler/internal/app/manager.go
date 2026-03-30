// Package app implements the application lifecycle manager.
// Equivalent to Java's JobSchedulerManager + ApplicationListener<ApplicationReadyEvent>.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tour_of_go/projects/distributed-scheduler/internal/crons"
	"tour_of_go/projects/distributed-scheduler/internal/ports"
)

// Manager orchestrates the full application lifecycle:
//  1. Acquire Redis lease (single-active-instance)
//  2. Populate in-memory concurrency pool from DB
//  3. Mass-index search
//  4. Start all cron goroutines
//  5. Start HTTP server
//  6. On SIGTERM: stop crons → stop HTTP → release lease
type Manager struct {
	lease       ports.LeaseService
	concurrency ports.ConcurrencyManager
	search      ports.SearchService
	jobRepo     ports.JobRepository
	configRepo  ports.JobConfigRepository
	heartbeat   ports.HeartbeatService

	coldScheduler       *crons.ColdScheduler
	concurrencyRefresher *crons.ConcurrencyRefresher
	longPublishedChecker *crons.LongPublishedChecker
	heartbeatMarker     *crons.HeartbeatMarker
	zombieChecker       *crons.ZombieChecker

	httpServer *http.Server
	holderID   string
	log        *slog.Logger
}

func NewManager(
	lease ports.LeaseService,
	concurrency ports.ConcurrencyManager,
	search ports.SearchService,
	jobRepo ports.JobRepository,
	configRepo ports.JobConfigRepository,
	heartbeat ports.HeartbeatService,
	coldScheduler *crons.ColdScheduler,
	concurrencyRefresher *crons.ConcurrencyRefresher,
	longPublishedChecker *crons.LongPublishedChecker,
	heartbeatMarker *crons.HeartbeatMarker,
	zombieChecker *crons.ZombieChecker,
	httpServer *http.Server,
	holderID string,
) *Manager {
	return &Manager{
		lease:                lease,
		concurrency:          concurrency,
		search:               search,
		jobRepo:              jobRepo,
		configRepo:           configRepo,
		heartbeat:            heartbeat,
		coldScheduler:        coldScheduler,
		concurrencyRefresher: concurrencyRefresher,
		longPublishedChecker: longPublishedChecker,
		heartbeatMarker:      heartbeatMarker,
		zombieChecker:        zombieChecker,
		httpServer:           httpServer,
		holderID:             holderID,
		log:                  slog.Default(),
	}
}

// Run starts the scheduler and blocks until SIGTERM/SIGINT.
func (m *Manager) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Step 1: Acquire distributed lease
	acquired, err := m.lease.Acquire(ctx, m.holderID)
	if err != nil {
		return fmt.Errorf("lease acquire: %w", err)
	}
	if !acquired {
		return fmt.Errorf("another scheduler instance is running (lease held)")
	}
	m.log.Info("lease acquired", "holderID", m.holderID)
	m.lease.StartRefresh(ctx, m.holderID)

	// Step 2: Populate concurrency pool from DB
	projs, err := m.jobRepo.GetActiveProjections(ctx)
	if err != nil {
		return fmt.Errorf("get active projections: %w", err)
	}
	configs, err := m.configRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("get configs: %w", err)
	}
	rulesMap := make(map[string]map[string]int, len(configs))
	for _, cfg := range configs {
		rulesMap[cfg.JobName] = cfg.ConcurrencyRules
	}
	m.concurrency.PopulateFromDB(projs, rulesMap)
	m.log.Info("concurrency pool populated", "activeJobs", len(projs))

	// Step 3: Mass-index search
	allJobs, err := m.jobRepo.FindSchedulable(ctx, "", 100000)
	if err != nil {
		m.log.Error("mass index failed", "error", err)
	} else {
		m.search.MassIndex(allJobs)
		m.log.Info("search index built", "jobs", len(allJobs))
	}

	// Step 4: Start cron goroutines
	m.coldScheduler.Start(ctx)
	m.concurrencyRefresher.Start(ctx)
	m.longPublishedChecker.Start(ctx)
	m.heartbeatMarker.Start(ctx)
	m.zombieChecker.Start(ctx)
	m.log.Info("all crons started")

	// Step 5: Start HTTP server in background
	serverErr := make(chan error, 1)
	go func() {
		m.log.Info("HTTP server starting", "addr", m.httpServer.Addr)
		if err := m.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Step 6: Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-sigCh:
		m.log.Info("shutdown signal received", "signal", sig)
	case err := <-serverErr:
		m.log.Error("HTTP server error", "error", err)
	}

	// Graceful shutdown
	m.log.Info("starting graceful shutdown")
	cancel() // stops all cron goroutines

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := m.httpServer.Shutdown(shutdownCtx); err != nil {
		m.log.Error("HTTP shutdown error", "error", err)
	}

	if err := m.lease.Release(shutdownCtx, m.holderID); err != nil {
		m.log.Error("lease release error", "error", err)
	}

	m.log.Info("shutdown complete")
	return nil
}

// Hostname returns the machine hostname for use as the lease holder ID.
func Hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
