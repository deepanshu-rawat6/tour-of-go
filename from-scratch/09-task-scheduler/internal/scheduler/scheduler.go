// Package scheduler runs tasks on a cron schedule.
package scheduler

import (
	"context"
	"sync"
	"time"

	"tour_of_go/projects/from-scratch/09-task-scheduler/internal/cron"
)

// Task is a scheduled job.
type Task struct {
	ID       string
	Name     string
	Expr     string
	schedule *cron.Schedule
	Fn       func()
	LastRun  time.Time
}

// Scheduler ticks every second and fires tasks whose cron expression matches.
type Scheduler struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

func New() *Scheduler {
	return &Scheduler{tasks: make(map[string]*Task)}
}

// Add registers a task. Returns error if the cron expression is invalid.
func (s *Scheduler) Add(id, name, expr string, fn func()) error {
	sched, err := cron.Parse(expr)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.tasks[id] = &Task{ID: id, Name: name, Expr: expr, schedule: sched, Fn: fn}
	s.mu.Unlock()
	return nil
}

// Remove deletes a task by ID.
func (s *Scheduler) Remove(id string) {
	s.mu.Lock()
	delete(s.tasks, id)
	s.mu.Unlock()
}

// List returns all registered tasks.
func (s *Scheduler) List() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tasks := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}

// Start runs the scheduler tick loop until ctx is cancelled.
func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.tick(now)
		}
	}
}

func (s *Scheduler) tick(now time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.tasks {
		if t.schedule.Match(now) {
			go t.Fn()
			t.LastRun = now
		}
	}
}
