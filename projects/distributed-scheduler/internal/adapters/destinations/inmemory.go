// Package destinations implements the Destination port for different execution backends.
package destinations

import (
	"context"
	"fmt"
	"log/slog"

	"tour_of_go/projects/distributed-scheduler/internal/domain"
)

// InMemoryDestination is a buffered channel-based destination for local dev and testing.
// Jobs are placed on the channel and can be consumed by test workers.
type InMemoryDestination struct {
	queue chan *domain.Job
	log   *slog.Logger
}

func NewInMemoryDestination(bufferSize int) *InMemoryDestination {
	return &InMemoryDestination{
		queue: make(chan *domain.Job, bufferSize),
		log:   slog.Default(),
	}
}

func (d *InMemoryDestination) Publish(_ context.Context, job *domain.Job) error {
	select {
	case d.queue <- job:
		d.log.Debug("job queued", "jobID", job.ID, "jobName", job.JobName)
		return nil
	default:
		return fmt.Errorf("in-memory queue full (capacity %d)", cap(d.queue))
	}
}

func (d *InMemoryDestination) BatchPublish(ctx context.Context, jobs []*domain.Job) map[int64]error {
	failures := make(map[int64]error)
	for _, job := range jobs {
		if err := d.Publish(ctx, job); err != nil {
			failures[job.ID] = err
		}
	}
	return failures
}

func (d *InMemoryDestination) SupportsBatch() bool              { return true }
func (d *InMemoryDestination) Type() domain.DestinationType     { return domain.DestinationInMemory }
func (d *InMemoryDestination) Queue() <-chan *domain.Job         { return d.queue }
