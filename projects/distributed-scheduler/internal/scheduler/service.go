package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"tour_of_go/projects/distributed-scheduler/internal/domain"
	"tour_of_go/projects/distributed-scheduler/internal/ports"
)

// Service implements the scheduling orchestration layer.
// It is the Go equivalent of Java's JobSchedulingService.
//
// Design decision (see docs/adr/006-worker-pool-design.md):
// A bounded goroutine pool (semaphore pattern) replaces Java's ThreadPoolExecutor.
// The pool size is configurable. Deduplication uses sync.Map instead of
// ConcurrentHashMap.newKeySet() — same semantics, idiomatic Go.
type Service struct {
	algorithm   *Algorithm
	publisher   *Publisher
	configRepo  ports.JobConfigRepository
	sem         chan struct{}       // bounded semaphore: limits concurrent scheduling goroutines
	pending     sync.Map           // map[string]struct{}: job names currently being scheduled
	lastAttempt sync.Map           // map[string]time.Time: last scheduling attempt per job name
	log         *slog.Logger
}

func NewService(algorithm *Algorithm, publisher *Publisher, configRepo ports.JobConfigRepository, poolSize int) *Service {
	return &Service{
		algorithm:  algorithm,
		publisher:  publisher,
		configRepo: configRepo,
		sem:        make(chan struct{}, poolSize),
		log:        slog.Default(),
	}
}

// TriggerScheduling submits a scheduling task for jobName if one isn't already running.
// Returns true if a task was submitted, false if already pending (deduplicated).
func (s *Service) TriggerScheduling(ctx context.Context, jobName string) bool {
	// Deduplication: if already pending, skip
	if _, loaded := s.pending.LoadOrStore(jobName, struct{}{}); loaded {
		s.log.Debug("scheduling already pending, skipping", "jobName", jobName)
		return false
	}

	go func() {
		// Acquire semaphore slot (blocks if pool is full — backpressure)
		s.sem <- struct{}{}
		defer func() {
			<-s.sem
			s.pending.Delete(jobName)
			s.lastAttempt.Store(jobName, time.Now())
		}()

		if err := s.executeScheduling(ctx, jobName); err != nil {
			s.log.Error("scheduling failed", "jobName", jobName, "error", err)
		}
	}()

	return true
}

func (s *Service) executeScheduling(ctx context.Context, jobName string) error {
	config, err := s.configRepo.GetByJobName(ctx, jobName)
	if err != nil {
		return err
	}

	jobs, err := s.algorithm.FillVacantSpots(ctx, jobName, config.ConcurrencyRules)
	if err != nil {
		return err
	}

	return s.publisher.PublishJobs(ctx, config, jobs)
}

// IsPending reports whether scheduling is currently running for jobName.
func (s *Service) IsPending(jobName string) bool {
	_, ok := s.pending.Load(jobName)
	return ok
}

// LastAttempt returns the time of the last scheduling attempt for jobName.
func (s *Service) LastAttempt(jobName string) (time.Time, bool) {
	v, ok := s.lastAttempt.Load(jobName)
	if !ok {
		return time.Time{}, false
	}
	return v.(time.Time), true
}

// Publisher handles the final concurrency double-check and job dispatch.
// Equivalent to Java's JobScheduler.
type Publisher struct {
	concurrency ports.ConcurrencyManager
	jobRepo     ports.JobRepository
	destinations map[domain.DestinationType]ports.Destination
	log         *slog.Logger
}

func NewPublisher(concurrency ports.ConcurrencyManager, jobRepo ports.JobRepository, destinations map[domain.DestinationType]ports.Destination) *Publisher {
	return &Publisher{
		concurrency:  concurrency,
		jobRepo:      jobRepo,
		destinations: destinations,
		log:          slog.Default(),
	}
}

// PublishJobs performs the final concurrency check and dispatches jobs to their destination.
func (p *Publisher) PublishJobs(ctx context.Context, config *domain.JobConfig, projections []*domain.JobProjection) error {
	if len(projections) == 0 {
		return nil
	}

	dest, ok := p.destinations[config.Destination.Type]
	if !ok {
		p.log.Error("unknown destination type", "type", config.Destination.Type)
		return nil
	}

	// Phase 1: final concurrency double-check (atomic gate)
	var eligible []*domain.JobProjection
	for _, proj := range projections {
		if p.concurrency.EvaluateJobPublish(config.ConcurrencyRules, proj) {
			eligible = append(eligible, proj)
		}
	}

	if len(eligible) == 0 {
		return nil
	}

	// Phase 2: load full job entities (N+1 avoided with batch fetch)
	ids := make([]int64, len(eligible))
	for i, proj := range eligible {
		ids[i] = proj.ID
	}
	jobs, err := p.jobRepo.FindByIDs(ctx, ids)
	if err != nil {
		return err
	}

	// Phase 3: mark as PUBLISHED in DB
	if err := p.jobRepo.MarkPublished(ctx, ids); err != nil {
		return err
	}

	// Phase 4: dispatch to destination
	if dest.SupportsBatch() {
		failures := dest.BatchPublish(ctx, jobs)
		for jobID, err := range failures {
			p.log.Error("batch publish failed", "jobID", jobID, "error", err)
			p.jobRepo.MarkFailed(ctx, jobID, err.Error())
		}
	} else {
		for _, job := range jobs {
			if err := dest.Publish(ctx, job); err != nil {
				p.log.Error("publish failed", "jobID", job.ID, "error", err)
				p.jobRepo.MarkFailed(ctx, job.ID, err.Error())
			}
		}
	}

	p.log.Info("published jobs", "jobName", config.JobName, "count", len(eligible))
	return nil
}
