package saga

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type StepStatus string

const (
	StatusPending      StepStatus = "pending"
	StatusSuccess      StepStatus = "success"
	StatusFailed       StepStatus = "failed"
	StatusCompensated  StepStatus = "compensated"
)

// Step defines a saga step with its action and compensating action.
type Step struct {
	Name       string
	Action     func(ctx context.Context, data map[string]any) error
	Compensate func(ctx context.Context, data map[string]any) error
}

// SagaLog records the execution state of a saga.
type SagaLog struct {
	ID        string
	Steps     []StepRecord
	Status    StepStatus
	StartedAt time.Time
	EndedAt   time.Time
}

type StepRecord struct {
	Name   string
	Status StepStatus
	Error  string
}

// Orchestrator executes saga steps and compensates on failure.
type Orchestrator struct {
	steps []Step
	logs  map[string]*SagaLog
}

func NewOrchestrator(steps []Step) *Orchestrator {
	return &Orchestrator{steps: steps, logs: make(map[string]*SagaLog)}
}

// Execute runs the saga forward. On failure, compensates completed steps in reverse.
func (o *Orchestrator) Execute(ctx context.Context, sagaID string, data map[string]any) error {
	log := &SagaLog{
		ID:        sagaID,
		Status:    StatusPending,
		StartedAt: time.Now(),
	}
	o.logs[sagaID] = log

	var completedSteps []int

	for i, step := range o.steps {
		slog.Info("executing step", "saga", sagaID, "step", step.Name)
		record := StepRecord{Name: step.Name, Status: StatusPending}

		if err := step.Action(ctx, data); err != nil {
			record.Status = StatusFailed
			record.Error = err.Error()
			log.Steps = append(log.Steps, record)
			slog.Error("step failed, compensating", "saga", sagaID, "step", step.Name, "err", err)

			// Compensate in reverse order
			o.compensate(ctx, sagaID, completedSteps, data)
			log.Status = StatusFailed
			log.EndedAt = time.Now()
			return fmt.Errorf("saga %s failed at step %s: %w", sagaID, step.Name, err)
		}

		record.Status = StatusSuccess
		log.Steps = append(log.Steps, record)
		completedSteps = append(completedSteps, i)
	}

	log.Status = StatusSuccess
	log.EndedAt = time.Now()
	slog.Info("saga completed", "saga", sagaID)
	return nil
}

func (o *Orchestrator) compensate(ctx context.Context, sagaID string, completed []int, data map[string]any) {
	for i := len(completed) - 1; i >= 0; i-- {
		step := o.steps[completed[i]]
		if step.Compensate == nil {
			continue
		}
		slog.Info("compensating step", "saga", sagaID, "step", step.Name)
		if err := step.Compensate(ctx, data); err != nil {
			slog.Error("compensation failed", "saga", sagaID, "step", step.Name, "err", err)
			// In production: alert + manual intervention queue
		}
	}
}

func (o *Orchestrator) GetLog(sagaID string) *SagaLog {
	return o.logs[sagaID]
}
