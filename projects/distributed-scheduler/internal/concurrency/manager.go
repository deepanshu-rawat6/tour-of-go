package concurrency

import (
	"sync"
	"sync/atomic"

	"tour_of_go/projects/distributed-scheduler/internal/domain"
)

// Manager implements ports.ConcurrencyManager using an in-memory pool.
//
// Design decisions (see docs/adr/003-sync-mutex-over-channels.md):
//   - sync.RWMutex for the pool map: read-heavy (GetVacantSpots called per candidate),
//     write-only on publish/finish. RWMutex allows concurrent reads.
//   - atomic.Int64 per key: lock-free increment/decrement within the write lock.
//   - EvaluateJobPublish uses a full write lock for all-or-nothing atomicity.
//     We cannot use per-key locks because we need to check ALL rules before
//     incrementing ANY counter — a partial increment would corrupt state.
type Manager struct {
	mu   sync.RWMutex
	pool map[string]*atomic.Int64

	// configFn retrieves concurrency rules for a job name.
	// Injected to avoid circular dependency with the config service.
	configFn func(jobName string) map[string]int
}

// NewManager creates a Manager. configFn is called to look up rules per job name.
func NewManager(configFn func(jobName string) map[string]int) *Manager {
	return &Manager{
		pool:     make(map[string]*atomic.Int64),
		configFn: configFn,
	}
}

// EvaluateJobPublish atomically checks all concurrency rules and increments counters
// if — and only if — all rules are satisfied. This is the double-check pattern:
// the algorithm pre-screens candidates, but this is the authoritative gate.
func (m *Manager) EvaluateJobPublish(rules map[string]int, job *domain.JobProjection) bool {
	concurrencyMap := GenerateConcurrencyMap(rules, job)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Phase 1: check all rules
	for key, limit := range concurrencyMap {
		counter := m.getOrCreate(key)
		if counter.Load() >= int64(limit) {
			return false
		}
	}

	// Phase 2: all rules satisfied — increment atomically
	for key := range concurrencyMap {
		m.getOrCreate(key).Add(1)
	}
	return true
}

// HandleFinishedJob decrements counters for a completed job.
// Floors at 0 to handle edge cases (e.g., double-decrement on crash recovery).
func (m *Manager) HandleFinishedJob(job *domain.Job) {
	rules := m.configFn(job.JobName)
	proj := &domain.JobProjection{
		JobName:            job.JobName,
		Tenant:             job.Tenant,
		ConcurrencyControl: job.ConcurrencyControl,
	}
	concurrencyMap := GenerateConcurrencyMap(rules, proj)

	m.mu.Lock()
	defer m.mu.Unlock()

	for key := range concurrencyMap {
		if counter, ok := m.pool[key]; ok {
			// Atomically decrement, floor at 0
			for {
				old := counter.Load()
				if old <= 0 {
					break
				}
				if counter.CompareAndSwap(old, old-1) {
					break
				}
			}
		}
	}
}

// GetVacantSpots returns available capacity per rule template for a given job.
// Uses a read lock — multiple goroutines can call this concurrently.
func (m *Manager) GetVacantSpots(rules map[string]int, job *domain.JobProjection) map[string]int {
	templateToKey := GenerateKeys(rules, job)

	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]int, len(rules))
	for template, limit := range rules {
		key := templateToKey[template]
		var current int64
		if counter, ok := m.pool[key]; ok {
			current = counter.Load()
		}
		vacant := int64(limit) - current
		if vacant < 0 {
			vacant = 0
		}
		result[template] = int(vacant)
	}
	return result
}

// GetGlobalAvailability returns available capacity for the $jobName rule.
func (m *Manager) GetGlobalAvailability(jobName string) int {
	rules := m.configFn(jobName)
	dummy := &domain.JobProjection{JobName: jobName, Tenant: 1}
	spots := m.GetVacantSpots(rules, dummy)
	return spots["$jobName"]
}

// PopulateFromDB rebuilds the entire pool from current active jobs.
// Called on startup and by the ConcurrencyRefresher cron.
func (m *Manager) PopulateFromDB(jobs []*domain.JobProjection, rules map[string]map[string]int) {
	newPool := make(map[string]*atomic.Int64)

	for _, job := range jobs {
		jobRules, ok := rules[job.JobName]
		if !ok {
			continue
		}
		for key := range GenerateConcurrencyMap(jobRules, job) {
			if _, exists := newPool[key]; !exists {
				newPool[key] = &atomic.Int64{}
			}
			newPool[key].Add(1)
		}
	}

	m.mu.Lock()
	m.pool = newPool
	m.mu.Unlock()
}

// Snapshot returns a copy of the current pool state (for the /concurrency/keys endpoint).
func (m *Manager) Snapshot() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snap := make(map[string]int64, len(m.pool))
	for k, v := range m.pool {
		snap[k] = v.Load()
	}
	return snap
}

// getOrCreate returns the counter for key, creating it if absent.
// Must be called with m.mu write lock held.
func (m *Manager) getOrCreate(key string) *atomic.Int64 {
	if c, ok := m.pool[key]; ok {
		return c
	}
	c := &atomic.Int64{}
	m.pool[key] = c
	return c
}
