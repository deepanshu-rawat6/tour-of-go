package domain

import (
	"fmt"
	"time"
)

// Job is the core entity. It maps to the jfc_job table.
// Payload and ConcurrencyControl are stored as JSON in the DB.
type Job struct {
	ID                 int64             `db:"id"`
	JobName            string            `db:"job_name"`
	Tenant             int               `db:"tenant"`
	Priority           int               `db:"priority"` // lower = higher priority
	Status             JobStatus         `db:"status"`
	Payload            map[string]any    `db:"payload"`
	ConcurrencyControl map[string]string `db:"concurrency_control"`
	ExecutionCount     int               `db:"execution_count"`
	LastFailureReason  string            `db:"last_failure_reason"`
	SubmitTime         time.Time         `db:"submit_time"`
	LastUpdated        time.Time         `db:"last_updated"`
}

// Transition attempts to move the job to the next status.
// Returns an error if the transition is invalid.
func (j *Job) Transition(next JobStatus) error {
	if err := j.Status.Validate(next); err != nil {
		return fmt.Errorf("job %d: %w", j.ID, err)
	}
	j.Status = next
	if next.IsFinal() {
		j.ExecutionCount++
	}
	return nil
}

// JobProjection is a lightweight read-only view used by the scheduling algorithm.
// Avoids loading the full payload (which can be large) during scheduling.
// This is the Go equivalent of Java's JobProjectionDTO.
type JobProjection struct {
	ID                 int64             `db:"id"`
	JobName            string            `db:"job_name"`
	Tenant             int               `db:"tenant"`
	Priority           int               `db:"priority"`
	Status             JobStatus         `db:"status"`
	ConcurrencyControl map[string]string `db:"concurrency_control"`
}
