package domain

import "testing"

func TestJobStatusTransitions(t *testing.T) {
	tests := []struct {
		from    JobStatus
		to      JobStatus
		allowed bool
	}{
		{StatusWaiting, StatusPublished, true},
		{StatusWaiting, StatusCancelled, true},
		{StatusWaiting, StatusProcessing, false}, // must go through PUBLISHED
		{StatusPublished, StatusProcessing, true},
		{StatusPublished, StatusFailedByJFC, true},
		{StatusPublished, StatusSuccessful, false},
		{StatusProcessing, StatusSuccessful, true},
		{StatusProcessing, StatusFailed, true},
		{StatusProcessing, StatusFailedByJFC, true},
		{StatusProcessing, StatusWaiting, false},
		{StatusSuccessful, StatusRetry, true},
		{StatusSuccessful, StatusWaiting, false},
		{StatusFailed, StatusRetry, true},
		{StatusRetry, StatusPublished, true},
		{StatusRetry, StatusCancelled, true},
		{StatusCancelled, StatusRetry, true},
		{StatusCancelled, StatusPublished, false},
	}

	for _, tt := range tests {
		got := tt.from.CanTransitionTo(tt.to)
		if got != tt.allowed {
			t.Errorf("%s → %s: expected allowed=%v, got %v", tt.from, tt.to, tt.allowed, got)
		}
	}
}

func TestJobTransition(t *testing.T) {
	job := &Job{ID: 1, Status: StatusWaiting}

	if err := job.Transition(StatusPublished); err != nil {
		t.Fatalf("expected valid transition, got: %v", err)
	}
	if job.Status != StatusPublished {
		t.Errorf("expected PUBLISHED, got %s", job.Status)
	}

	// Invalid transition
	if err := job.Transition(StatusWaiting); err == nil {
		t.Error("expected error for invalid transition PUBLISHED → WAITING")
	}
}

func TestExecutionCountIncrement(t *testing.T) {
	job := &Job{ID: 1, Status: StatusProcessing, ExecutionCount: 0}
	_ = job.Transition(StatusSuccessful)
	if job.ExecutionCount != 1 {
		t.Errorf("expected ExecutionCount=1, got %d", job.ExecutionCount)
	}
}

func TestIsSchedulable(t *testing.T) {
	if !StatusWaiting.IsSchedulable() || !StatusRetry.IsSchedulable() {
		t.Error("WAITING and RETRY should be schedulable")
	}
	if StatusPublished.IsSchedulable() || StatusProcessing.IsSchedulable() {
		t.Error("PUBLISHED and PROCESSING should not be schedulable")
	}
}
