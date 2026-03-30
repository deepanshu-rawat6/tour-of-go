// Package ports defines the interfaces (ports) that the core domain depends on.
// Adapters in internal/adapters/ implement these interfaces.
// This is the hexagonal architecture boundary — the domain never imports adapters.
package ports

import (
	"context"
	"time"

	"tour_of_go/projects/distributed-scheduler/internal/domain"
)

// JobRepository is the persistence port for Job entities.
type JobRepository interface {
	Save(ctx context.Context, job *domain.Job) error
	FindByID(ctx context.Context, id int64) (*domain.Job, error)
	FindByIDs(ctx context.Context, ids []int64) ([]*domain.Job, error)
	// FindSchedulable returns jobs in WAITING or RETRY status for a given job name.
	FindSchedulable(ctx context.Context, jobName string, limit int) ([]*domain.Job, error)
	// GetProjections returns lightweight projections for PUBLISHED and PROCESSING jobs.
	GetActiveProjections(ctx context.Context) ([]*domain.JobProjection, error)
	MarkPublished(ctx context.Context, ids []int64) error
	MarkFailed(ctx context.Context, id int64, reason string) error
	UpdateStatus(ctx context.Context, id int64, status domain.JobStatus) error
	// GetStuckPublished returns jobs in PUBLISHED status older than the given threshold.
	GetStuckPublished(ctx context.Context, olderThan time.Duration) ([]*domain.Job, error)
	// GetProcessingJobs returns all jobs currently in PROCESSING status.
	GetProcessingJobs(ctx context.Context) ([]*domain.Job, error)
}

// JobConfigRepository is the persistence port for JobConfig.
type JobConfigRepository interface {
	GetByJobName(ctx context.Context, jobName string) (*domain.JobConfig, error)
	List(ctx context.Context) ([]*domain.JobConfig, error)
}

// ConcurrencyManager is the port for the in-memory concurrency tracking system.
// The implementation holds all concurrency state in memory for O(1) access.
type ConcurrencyManager interface {
	// EvaluateJobPublish atomically checks and increments concurrency counters.
	// Returns true if the job was approved (all rules satisfied) and counters incremented.
	// Returns false if any rule is violated — counters are NOT modified (all-or-nothing).
	EvaluateJobPublish(rules map[string]int, job *domain.JobProjection) bool

	// HandleFinishedJob decrements counters for a completed job.
	// Counters are floored at 0 to prevent negative values.
	HandleFinishedJob(job *domain.Job)

	// GetVacantSpots returns available capacity per rule key for a given job.
	// Used by the scheduling algorithm to find schedulable candidates.
	GetVacantSpots(rules map[string]int, job *domain.JobProjection) map[string]int

	// GetGlobalAvailability returns the available capacity for the $jobName rule.
	GetGlobalAvailability(jobName string) int

	// PopulateFromDB rebuilds the entire pool from current active jobs.
	// Called on startup and by the periodic ConcurrencyRefresher cron.
	PopulateFromDB(jobs []*domain.JobProjection, rules map[string]map[string]int)
}

// SearchService is the port for the embedded job search index (Bleve/Lucene equivalent).
type SearchService interface {
	// FindWaitingJobs returns schedulable jobs for a job name, excluding exhausted combinations.
	FindWaitingJobs(ctx context.Context, jobName string, excludeCombinations []map[string]string, limit int) ([]*domain.JobProjection, error)
	// FindDistinctJobNames returns all job names that have schedulable jobs.
	FindDistinctJobNames(ctx context.Context) ([]string, error)
	// Index adds or updates a job in the search index.
	Index(job *domain.Job) error
	// Remove deletes a job from the search index.
	Remove(jobID int64) error
	// MassIndex rebuilds the entire index from the provided jobs.
	MassIndex(jobs []*domain.Job) error
}

// Destination is the port for job execution backends (SQS, in-memory, etc.).
type Destination interface {
	// Publish sends a single job to the execution backend.
	Publish(ctx context.Context, job *domain.Job) error
	// BatchPublish sends multiple jobs. Returns a map of jobID → error for failures.
	BatchPublish(ctx context.Context, jobs []*domain.Job) map[int64]error
	// SupportsBatch reports whether this destination supports batch publishing.
	SupportsBatch() bool
	// Type returns the destination type identifier.
	Type() domain.DestinationType
}

// LeaseService is the port for distributed single-active-instance coordination.
type LeaseService interface {
	// Acquire attempts to acquire the global scheduler lease.
	// Returns true if acquired, false if another instance holds it.
	Acquire(ctx context.Context, holderID string) (bool, error)
	// Release releases the lease held by holderID.
	Release(ctx context.Context, holderID string) error
	// StartRefresh begins the background lease refresh goroutine.
	// The goroutine stops when ctx is cancelled.
	StartRefresh(ctx context.Context, holderID string)
}

// HeartbeatService is the port for zombie job detection via Redis TTL keys.
type HeartbeatService interface {
	// StartHeartbeat registers a job as running and begins TTL refresh.
	StartHeartbeat(jobID int64)
	// StopHeartbeat removes a job from the running set.
	StopHeartbeat(jobID int64)
	// MarkAllAlive refreshes TTL keys for all registered running jobs.
	MarkAllAlive(ctx context.Context) error
	// DetectZombies finds PROCESSING jobs whose heartbeat TTL has expired.
	// Returns job names of affected jobs (for rescheduling).
	DetectZombies(ctx context.Context) ([]string, error)
}
