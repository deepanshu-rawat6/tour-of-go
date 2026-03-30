// Package redis implements the LeaseService and HeartbeatService ports using Redis.
//
// Design decision (see docs/adr/005-redis-lease-over-etcd.md):
// Redis SET NX (set-if-not-exists) with TTL is sufficient for single-active-instance
// coordination. etcd would add operational complexity without benefit here.
package redis

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	leaseTTL        = 2 * time.Minute
	leaseRefreshInterval = 60 * time.Second
	leaseKey        = "scheduler:global_lock"
)

// LeaseService implements ports.LeaseService using Redis SET NX.
type LeaseService struct {
	client *redis.Client
	log    *slog.Logger
}

func NewLeaseService(client *redis.Client) *LeaseService {
	return &LeaseService{client: client, log: slog.Default()}
}

// Acquire attempts to set the lease key with NX (only if not exists) and a 2-min TTL.
// Returns true if this instance acquired the lease.
func (s *LeaseService) Acquire(ctx context.Context, holderID string) (bool, error) {
	ok, err := s.client.SetNX(ctx, leaseKey, holderID, leaseTTL).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// Release deletes the lease key only if this instance holds it.
// Uses a Lua script for atomic check-and-delete.
func (s *LeaseService) Release(ctx context.Context, holderID string) error {
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		end
		return 0
	`)
	return script.Run(ctx, s.client, []string{leaseKey}, holderID).Err()
}

// StartRefresh starts a background goroutine that refreshes the lease TTL every 60s.
// Stops when ctx is cancelled. This prevents the lease from expiring while the
// scheduler is running normally.
func (s *LeaseService) StartRefresh(ctx context.Context, holderID string) {
	go func() {
		ticker := time.NewTicker(leaseRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				s.log.Info("lease refresh stopped")
				return
			case <-ticker.C:
				if err := s.client.Set(ctx, leaseKey, holderID, leaseTTL).Err(); err != nil {
					s.log.Error("failed to refresh lease", "error", err)
				}
			}
		}
	}()
}
