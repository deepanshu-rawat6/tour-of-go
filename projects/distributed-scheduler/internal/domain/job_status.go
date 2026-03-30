// Package domain contains the core business entities and rules.
// No external dependencies — pure Go types.
package domain

import "fmt"

// JobStatus represents the lifecycle state of a job.
// The state machine enforces valid transitions — invalid transitions return an error.
//
// State diagram:
//
//	WAITING ──► PUBLISHED ──► PROCESSING ──► SUCCESSFUL ──► RETRY
//	   │             │              │                          │
//	   ▼             ▼              ▼                          ▼
//	CANCELLED   FAILED_BY_JFC   FAILED ──────────────────► RETRY
//	   │                        FAILED_BY_JFC               │
//	   └────────────────────────────────────────────────────┘
type JobStatus string

const (
	StatusWaiting     JobStatus = "WAITING"
	StatusPublished   JobStatus = "PUBLISHED"
	StatusProcessing  JobStatus = "PROCESSING"
	StatusSuccessful  JobStatus = "SUCCESSFUL"
	StatusFailed      JobStatus = "FAILED"
	StatusFailedByJFC JobStatus = "FAILED_BY_JFC"
	StatusRetry       JobStatus = "RETRY"
	StatusCancelled   JobStatus = "CANCELLED"
)

// allowedTransitions defines the valid next states for each status.
// This is the single source of truth for the state machine.
var allowedTransitions = map[JobStatus][]JobStatus{
	StatusWaiting:     {StatusPublished, StatusCancelled},
	StatusPublished:   {StatusProcessing, StatusFailedByJFC},
	StatusProcessing:  {StatusSuccessful, StatusFailed, StatusFailedByJFC},
	StatusSuccessful:  {StatusRetry},
	StatusFailed:      {StatusRetry},
	StatusFailedByJFC: {StatusSuccessful, StatusFailed, StatusRetry},
	StatusRetry:       {StatusPublished, StatusCancelled},
	StatusCancelled:   {StatusRetry},
}

// CanTransitionTo reports whether transitioning from s to next is valid.
func (s JobStatus) CanTransitionTo(next JobStatus) bool {
	for _, allowed := range allowedTransitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}

// AllowedTransitions returns the valid next states from s.
func (s JobStatus) AllowedTransitions() []JobStatus {
	return allowedTransitions[s]
}

// IsSchedulable reports whether a job in this status can be picked up by the algorithm.
func (s JobStatus) IsSchedulable() bool {
	return s == StatusWaiting || s == StatusRetry
}

// IsFinal reports whether this is a terminal state (no further transitions except explicit retry).
func (s JobStatus) IsFinal() bool {
	return s == StatusSuccessful || s == StatusFailed || s == StatusCancelled || s == StatusFailedByJFC
}

// IsActive reports whether a job in this status holds a concurrency slot.
func (s JobStatus) IsActive() bool {
	return s == StatusPublished || s == StatusProcessing
}

// Validate returns an error if transitioning from current to next is not allowed.
func (s JobStatus) Validate(next JobStatus) error {
	if !s.CanTransitionTo(next) {
		return fmt.Errorf("invalid transition: %s → %s", s, next)
	}
	return nil
}
