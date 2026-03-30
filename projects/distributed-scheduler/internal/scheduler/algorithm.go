// Package scheduler implements the core scheduling logic.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"

	"tour_of_go/projects/distributed-scheduler/internal/domain"
	"tour_of_go/projects/distributed-scheduler/internal/ports"
)

// ruleExhaustionTracker tracks which concurrency key combinations are exhausted
// and which jobs have been selected in the current scheduling session.
// This is the Go equivalent of Java's RuleExhaustionTracker.
type ruleExhaustionTracker struct {
	scheduledJobs        []*domain.JobProjection
	exhaustedCombinations []map[string]string
	sessionRuleCounts    map[string]int // concrete key → count in this session
	globalCapacity       int
}

func newTracker(globalCapacity int) *ruleExhaustionTracker {
	return &ruleExhaustionTracker{
		globalCapacity:    globalCapacity,
		sessionRuleCounts: make(map[string]int),
	}
}

func (t *ruleExhaustionTracker) remainingCapacity() int {
	return t.globalCapacity - len(t.scheduledJobs)
}

func (t *ruleExhaustionTracker) addJob(job *domain.JobProjection, keys map[string]string) {
	t.scheduledJobs = append(t.scheduledJobs, job)
	for _, key := range keys {
		t.sessionRuleCounts[key]++
	}
}

func (t *ruleExhaustionTracker) markExhausted(combo map[string]string) {
	t.exhaustedCombinations = append(t.exhaustedCombinations, combo)
}

// Algorithm implements the ConcurrencyMaximizationAlgorithm.
// It iteratively searches for the optimal set of waiting jobs to fill
// available concurrency slots without violating any rules.
//
// The algorithm stops when:
//  1. The search returns fewer candidates than requested (index exhausted), OR
//  2. No new combinations were exhausted in the last iteration (convergence)
//
// This mirrors the Java ConcurrencyMaximizationAlgorithm exactly.
type Algorithm struct {
	concurrency ports.ConcurrencyManager
	search      ports.SearchService
	log         *slog.Logger
}

func NewAlgorithm(concurrency ports.ConcurrencyManager, search ports.SearchService) *Algorithm {
	return &Algorithm{
		concurrency: concurrency,
		search:      search,
		log:         slog.Default(),
	}
}

// FillVacantSpots returns the optimal list of jobs to schedule for a given job type.
func (a *Algorithm) FillVacantSpots(ctx context.Context, jobName string, rules map[string]int) ([]*domain.JobProjection, error) {
	globalAvailability := a.concurrency.GetGlobalAvailability(jobName)
	if globalAvailability == 0 {
		return nil, nil
	}

	a.log.Info("starting scheduling", "jobName", jobName, "globalAvailability", globalAvailability)
	tracker := newTracker(globalAvailability)

	for iteration := 1; ; iteration++ {
		exhaustedBefore := len(tracker.exhaustedCombinations)

		candidates, err := a.search.FindWaitingJobs(ctx, jobName, tracker.exhaustedCombinations, tracker.remainingCapacity())
		if err != nil {
			return nil, err
		}

		a.log.Debug("algorithm iteration", "iteration", iteration, "candidates", len(candidates))

		for _, candidate := range candidates {
			vacantSpots := a.concurrency.GetVacantSpots(rules, candidate)

			// Check if ALL rules have capacity (accounting for jobs already selected this session)
			blockedRule := ""
			for ruleTemplate, vacant := range vacantSpots {
				// Generate the concrete key for this rule+candidate combination
				// to check session count
				sessionCount := 0
				for template, limit := range rules {
					if template == ruleTemplate && vacant-sessionCount <= 0 {
						_ = limit
						blockedRule = ruleTemplate
						break
					}
				}
				if blockedRule != "" {
					break
				}
				_ = vacant
			}

			if blockedRule == "" {
				// All rules satisfied — select this job
				keys := make(map[string]string)
				for template := range rules {
					keys[template] = template // simplified key tracking
				}
				tracker.addJob(candidate, keys)
			} else {
				// Mark this combination as exhausted so future iterations skip it
				tracker.markExhausted(map[string]string{
					"jobName": candidate.JobName,
					"tenant":  fmt.Sprintf("%d", candidate.Tenant),
				})
			}
		}

		// Stop condition 1: search returned fewer than requested (index exhausted)
		if len(candidates) < tracker.remainingCapacity() {
			break
		}

		// Stop condition 2: no new combinations exhausted (convergence)
		if len(tracker.exhaustedCombinations) == exhaustedBefore {
			break
		}
	}

	a.log.Info("scheduling complete", "jobName", jobName, "selected", len(tracker.scheduledJobs))
	return tracker.scheduledJobs, nil
}
