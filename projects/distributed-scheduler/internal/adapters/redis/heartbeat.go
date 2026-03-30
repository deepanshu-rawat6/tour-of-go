package redis

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"tour_of_go/projects/distributed-scheduler/internal/ports"
)

const (
	heartbeatPrefix = "scheduler:heartbeat:"
	heartbeatTTL    = 2 * time.Minute
)

// HeartbeatService implements ports.HeartbeatService.
//
// Each running job gets a Redis key with a 2-minute TTL.
// The MarkAllAlive cron refreshes these keys every minute.
// The DetectZombies cron checks for PROCESSING jobs whose key has expired.
//
// In-memory runningJobIDs set (sync.Map) mirrors the Java ConcurrentHashMap.newKeySet().
// Only jobs explicitly registered via StartHeartbeat are refreshed — not all PROCESSING
// jobs from DB. This prevents refreshing jobs that this instance didn't start.
type HeartbeatService struct {
	client        *redis.Client
	runningJobIDs sync.Map // map[int64]struct{}
	jobRepo       ports.JobRepository
	log           *slog.Logger
}

func NewHeartbeatService(client *redis.Client, jobRepo ports.JobRepository) *HeartbeatService {
	return &HeartbeatService{client: client, jobRepo: jobRepo, log: slog.Default()}
}

func (s *HeartbeatService) StartHeartbeat(jobID int64) {
	s.runningJobIDs.Store(jobID, struct{}{})
}

func (s *HeartbeatService) StopHeartbeat(jobID int64) {
	// Refresh one last time to reduce race condition window before removal
	ctx := context.Background()
	s.client.Set(ctx, heartbeatKey(jobID), true, heartbeatTTL)
	s.runningJobIDs.Delete(jobID)
}

// MarkAllAlive refreshes TTL keys for all registered running jobs.
// Called by the HeartbeatMarker cron every minute.
func (s *HeartbeatService) MarkAllAlive(ctx context.Context) error {
	var errs []error
	s.runningJobIDs.Range(func(key, _ any) bool {
		jobID := key.(int64)
		if err := s.client.Set(ctx, heartbeatKey(jobID), true, heartbeatTTL).Err(); err != nil {
			errs = append(errs, fmt.Errorf("job %d: %w", jobID, err))
		}
		return true
	})
	if len(errs) > 0 {
		return fmt.Errorf("heartbeat errors: %v", errs)
	}
	return nil
}

// DetectZombies finds PROCESSING jobs whose heartbeat key has expired.
// Returns the job names of affected jobs so the caller can trigger rescheduling.
func (s *HeartbeatService) DetectZombies(ctx context.Context) ([]string, error) {
	jobs, err := s.jobRepo.GetProcessingJobs(ctx)
	if err != nil {
		return nil, err
	}

	var affected []string
	for _, job := range jobs {
		// Only check jobs not updated in the last 2 minutes (grace period)
		if time.Since(job.LastUpdated) < 2*time.Minute {
			continue
		}
		exists, err := s.client.Exists(ctx, heartbeatKey(job.ID)).Result()
		if err != nil {
			s.log.Error("heartbeat check failed", "jobID", job.ID, "error", err)
			continue
		}
		if exists == 0 {
			// Heartbeat expired — mark as failed
			s.log.Warn("zombie detected", "jobID", job.ID, "jobName", job.JobName)
			if err := s.jobRepo.MarkFailed(ctx, job.ID, "heartbeat expired"); err != nil {
				s.log.Error("failed to mark zombie job", "jobID", job.ID, "error", err)
				continue
			}
			affected = append(affected, job.JobName)
		}
	}
	return affected, nil
}

func heartbeatKey(jobID int64) string {
	return fmt.Sprintf("%s%d", heartbeatPrefix, jobID)
}
