// Package crons implements all periodic background jobs.
//
// Design decision (see docs/adr/008-ticker-over-cron-library.md):
// All crons use time.Ticker + context.Context instead of a cron library.
// This gives us:
//   - Context-aware cancellation (graceful shutdown)
//   - No external dependency
//   - Explicit control over goroutine lifecycle
//   - Easy testing (inject a short interval)
package crons

import (
	"context"
	"log/slog"
	"time"

	"tour_of_go/projects/distributed-scheduler/internal/ports"
)

// ColdScheduler periodically finds stale WAITING/RETRY jobs and triggers scheduling.
// "Cold" because it handles jobs that missed their event-driven trigger.
// Equivalent to Java's ColdJobScheduler.
type ColdScheduler struct {
	search      ports.SearchService
	concurrency ports.ConcurrencyManager
	triggerFn   func(ctx context.Context, jobName string) bool
	interval    time.Duration
	staleAfter  time.Duration
	isPending   func(jobName string) bool
	lastAttempt func(jobName string) (time.Time, bool)
	log         *slog.Logger
}

func NewColdScheduler(
	search ports.SearchService,
	concurrency ports.ConcurrencyManager,
	triggerFn func(ctx context.Context, jobName string) bool,
	isPending func(string) bool,
	lastAttempt func(string) (time.Time, bool),
	interval, staleAfter time.Duration,
) *ColdScheduler {
	return &ColdScheduler{
		search:      search,
		concurrency: concurrency,
		triggerFn:   triggerFn,
		isPending:   isPending,
		lastAttempt: lastAttempt,
		interval:    interval,
		staleAfter:  staleAfter,
		log:         slog.Default(),
	}
}

func (c *ColdScheduler) Start(ctx context.Context) {
	go func() {
		// Run immediately on startup, then on interval
		c.run(ctx)
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				c.log.Info("cold scheduler stopped")
				return
			case <-ticker.C:
				c.run(ctx)
			}
		}
	}()
}

func (c *ColdScheduler) run(ctx context.Context) {
	jobNames, err := c.search.FindDistinctJobNames(ctx)
	if err != nil {
		c.log.Error("cold scheduler: find job names failed", "error", err)
		return
	}

	staleThreshold := time.Now().Add(-c.staleAfter)
	triggered := 0

	for _, jobName := range jobNames {
		if c.isPending(jobName) {
			continue
		}
		if last, ok := c.lastAttempt(jobName); ok && last.After(staleThreshold) {
			continue
		}
		if c.concurrency.GetGlobalAvailability(jobName) == 0 {
			continue
		}
		if c.triggerFn(ctx, jobName) {
			triggered++
		}
	}

	c.log.Info("cold scheduler run complete", "jobNames", len(jobNames), "triggered", triggered)
}

// ConcurrencyRefresher periodically rebuilds the in-memory concurrency pool from DB.
// Corrects any drift from missed events or crashes.
type ConcurrencyRefresher struct {
	jobRepo     ports.JobRepository
	configRepo  ports.JobConfigRepository
	concurrency ports.ConcurrencyManager
	interval    time.Duration
	log         *slog.Logger
}

func NewConcurrencyRefresher(jobRepo ports.JobRepository, configRepo ports.JobConfigRepository, concurrency ports.ConcurrencyManager, interval time.Duration) *ConcurrencyRefresher {
	return &ConcurrencyRefresher{jobRepo: jobRepo, configRepo: configRepo, concurrency: concurrency, interval: interval, log: slog.Default()}
}

func (r *ConcurrencyRefresher) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.refresh(ctx)
			}
		}
	}()
}

func (r *ConcurrencyRefresher) refresh(ctx context.Context) {
	projs, err := r.jobRepo.GetActiveProjections(ctx)
	if err != nil {
		r.log.Error("concurrency refresh: get projections failed", "error", err)
		return
	}
	configs, err := r.configRepo.List(ctx)
	if err != nil {
		r.log.Error("concurrency refresh: get configs failed", "error", err)
		return
	}
	rulesMap := make(map[string]map[string]int, len(configs))
	for _, cfg := range configs {
		rulesMap[cfg.JobName] = cfg.ConcurrencyRules
	}
	r.concurrency.PopulateFromDB(projs, rulesMap)
	r.log.Info("concurrency pool refreshed", "activeJobs", len(projs))
}

// LongPublishedChecker fails jobs stuck in PUBLISHED state beyond a threshold.
// Catches jobs that were sent to a destination but never picked up by a worker.
type LongPublishedChecker struct {
	jobRepo   ports.JobRepository
	threshold time.Duration
	interval  time.Duration
	log       *slog.Logger
}

func NewLongPublishedChecker(jobRepo ports.JobRepository, threshold, interval time.Duration) *LongPublishedChecker {
	return &LongPublishedChecker{jobRepo: jobRepo, threshold: threshold, interval: interval, log: slog.Default()}
}

func (c *LongPublishedChecker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.check(ctx)
			}
		}
	}()
}

func (c *LongPublishedChecker) check(ctx context.Context) {
	jobs, err := c.jobRepo.GetStuckPublished(ctx, c.threshold)
	if err != nil {
		c.log.Error("long published check failed", "error", err)
		return
	}
	for _, job := range jobs {
		c.log.Warn("failing stuck published job", "jobID", job.ID, "jobName", job.JobName)
		c.jobRepo.MarkFailed(ctx, job.ID, "stuck in PUBLISHED state")
	}
	if len(jobs) > 0 {
		c.log.Info("long published checker: failed stuck jobs", "count", len(jobs))
	}
}

// HeartbeatMarker refreshes TTL keys for all running jobs every minute.
type HeartbeatMarker struct {
	heartbeat ports.HeartbeatService
	interval  time.Duration
	log       *slog.Logger
}

func NewHeartbeatMarker(heartbeat ports.HeartbeatService, interval time.Duration) *HeartbeatMarker {
	return &HeartbeatMarker{heartbeat: heartbeat, interval: interval, log: slog.Default()}
}

func (m *HeartbeatMarker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := m.heartbeat.MarkAllAlive(ctx); err != nil {
					m.log.Error("heartbeat mark failed", "error", err)
				}
			}
		}
	}()
}

// ZombieChecker detects and fails jobs whose heartbeat has expired.
type ZombieChecker struct {
	heartbeat ports.HeartbeatService
	triggerFn func(ctx context.Context, jobName string) bool
	interval  time.Duration
	log       *slog.Logger
}

func NewZombieChecker(heartbeat ports.HeartbeatService, triggerFn func(ctx context.Context, jobName string) bool, interval time.Duration) *ZombieChecker {
	return &ZombieChecker{heartbeat: heartbeat, triggerFn: triggerFn, interval: interval, log: slog.Default()}
}

func (z *ZombieChecker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(z.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				affected, err := z.heartbeat.DetectZombies(ctx)
				if err != nil {
					z.log.Error("zombie check failed", "error", err)
					continue
				}
				for _, jobName := range affected {
					z.triggerFn(ctx, jobName)
				}
			}
		}
	}()
}
