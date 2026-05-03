// Package taskscheduler exposes the scheduler for use by other modules.
package taskscheduler

import (
	"context"
	"tour_of_go/projects/from-scratch/09-task-scheduler/internal/scheduler"
)

// Scheduler wraps the internal scheduler.
type Scheduler = scheduler.Scheduler

// New creates a new Scheduler.
func New() *Scheduler { return scheduler.New() }

// Start runs the scheduler until ctx is cancelled.
func Start(s *Scheduler, ctx context.Context) { s.Start(ctx) }
