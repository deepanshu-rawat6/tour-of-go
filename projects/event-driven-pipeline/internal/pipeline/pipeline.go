// Package pipeline implements the event processing pipeline with:
//   - Exactly-once semantics via Redis idempotency keys
//   - Circuit breaker on downstream calls
//   - Backpressure via bounded channel
//   - Retry with exponential backoff
//   - Dead letter queue after max retries
package pipeline

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"tour_of_go/projects/event-driven-pipeline/internal/domain"
)

const (
	maxRetries    = 3
	dlqSubject    = "events.dlq"
)

// IdempotencyStore checks and records processed event IDs.
type IdempotencyStore interface {
	// IsProcessed returns true if the idempotency key has already been processed.
	IsProcessed(ctx context.Context, key string) (bool, error)
	// MarkProcessed records the key as processed with a TTL.
	MarkProcessed(ctx context.Context, key string, ttl time.Duration) error
}

// EventHandler processes a single event. Implementations call downstream services.
type EventHandler interface {
	Handle(ctx context.Context, event *domain.Event) error
}

// DLQPublisher sends failed events to the dead letter queue.
type DLQPublisher interface {
	Publish(ctx context.Context, event *domain.Event) error
}

// circuitBreakerState tracks circuit breaker state.
type circuitBreakerState int32

const (
	cbClosed   circuitBreakerState = 0
	cbOpen     circuitBreakerState = 1
	cbHalfOpen circuitBreakerState = 2
)

// CircuitBreaker protects downstream calls from cascading failures.
// See more-internals/runnable/design-patterns/ for the pattern explanation.
type CircuitBreaker struct {
	state       atomic.Int32
	failures    atomic.Int32
	threshold   int32
	lastFailure atomic.Int64 // unix nano
	retryAfter  time.Duration
}

func NewCircuitBreaker(threshold int, retryAfter time.Duration) *CircuitBreaker {
	cb := &CircuitBreaker{threshold: int32(threshold), retryAfter: retryAfter}
	return cb
}

var ErrCircuitOpen = errors.New("circuit breaker open")

func (cb *CircuitBreaker) Execute(fn func() error) error {
	state := circuitBreakerState(cb.state.Load())

	if state == cbOpen {
		lastFail := time.Unix(0, cb.lastFailure.Load())
		if time.Since(lastFail) > cb.retryAfter {
			cb.state.CompareAndSwap(int32(cbOpen), int32(cbHalfOpen))
		} else {
			return ErrCircuitOpen
		}
	}

	err := fn()
	if err != nil {
		cb.failures.Add(1)
		cb.lastFailure.Store(time.Now().UnixNano())
		if cb.failures.Load() >= cb.threshold {
			cb.state.Store(int32(cbOpen))
		}
		return err
	}

	cb.failures.Store(0)
	cb.state.Store(int32(cbClosed))
	return nil
}

// Processor is the core pipeline stage.
// It wraps an EventHandler with idempotency, circuit breaking, retry, and DLQ.
type Processor struct {
	handler     EventHandler
	idempotency IdempotencyStore
	dlq         DLQPublisher
	cb          *CircuitBreaker
	// buffer is the backpressure channel — bounded to prevent memory exhaustion
	buffer      chan *domain.Event
	log         *slog.Logger
	wg          sync.WaitGroup
}

func NewProcessor(handler EventHandler, idempotency IdempotencyStore, dlq DLQPublisher, bufferSize int) *Processor {
	return &Processor{
		handler:     handler,
		idempotency: idempotency,
		dlq:         dlq,
		cb:          NewCircuitBreaker(5, 30*time.Second),
		buffer:      make(chan *domain.Event, bufferSize),
		log:         slog.Default(),
	}
}

// Submit adds an event to the processing buffer.
// Returns an error if the buffer is full (backpressure signal to the consumer).
func (p *Processor) Submit(event *domain.Event) error {
	select {
	case p.buffer <- event:
		return nil
	default:
		return errors.New("processor buffer full — backpressure")
	}
}

// Start launches worker goroutines that drain the buffer.
func (p *Processor) Start(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-p.buffer:
					if !ok {
						return
					}
					p.process(ctx, event)
				}
			}
		}()
	}
}

// Wait blocks until all workers finish (call after context cancellation).
func (p *Processor) Wait() { p.wg.Wait() }

func (p *Processor) process(ctx context.Context, event *domain.Event) {
	// Step 1: Idempotency check — skip if already processed
	processed, err := p.idempotency.IsProcessed(ctx, event.IdempotencyKey)
	if err != nil {
		p.log.Error("idempotency check failed", "eventID", event.ID, "error", err)
	}
	if processed {
		p.log.Debug("duplicate event skipped", "eventID", event.ID, "key", event.IdempotencyKey)
		return
	}

	// Step 2: Process with circuit breaker + retry
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 100ms, 200ms, 400ms
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 100 * time.Millisecond
			time.Sleep(backoff)
		}

		lastErr = p.cb.Execute(func() error {
			return p.handler.Handle(ctx, event)
		})

		if lastErr == nil {
			// Success — mark as processed
			p.idempotency.MarkProcessed(ctx, event.IdempotencyKey, 24*time.Hour)
			p.log.Info("event processed", "eventID", event.ID, "attempt", attempt+1)
			return
		}

		if errors.Is(lastErr, ErrCircuitOpen) {
			p.log.Warn("circuit open, sending to DLQ", "eventID", event.ID)
			break
		}

		p.log.Warn("event processing failed, retrying", "eventID", event.ID, "attempt", attempt+1, "error", lastErr)
	}

	// Step 3: Max retries exceeded or circuit open → DLQ
	event.Status = domain.EventDLQ
	event.RetryCount = maxRetries
	if err := p.dlq.Publish(ctx, event); err != nil {
		p.log.Error("DLQ publish failed", "eventID", event.ID, "error", err)
	} else {
		p.log.Warn("event sent to DLQ", "eventID", event.ID, "lastError", lastErr)
	}
}
