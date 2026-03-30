// Package domain contains the core event types.
package domain

import "time"

// EventStatus tracks the processing lifecycle of an event.
type EventStatus string

const (
	EventPending    EventStatus = "PENDING"
	EventProcessed  EventStatus = "PROCESSED"
	EventFailed     EventStatus = "FAILED"
	EventDLQ        EventStatus = "DLQ" // moved to dead letter queue
)

// Event is the core unit of work in the pipeline.
type Event struct {
	ID             string         `json:"id"`
	Type           string         `json:"type"`
	Payload        map[string]any `json:"payload"`
	IdempotencyKey string         `json:"idempotencyKey"` // for exactly-once processing
	TraceID        string         `json:"traceId"`        // OTel trace propagation
	RetryCount     int            `json:"retryCount"`
	Status         EventStatus    `json:"status"`
	CreatedAt      time.Time      `json:"createdAt"`
}
