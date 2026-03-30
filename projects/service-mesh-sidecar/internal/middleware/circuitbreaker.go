package middleware

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker open")

type cbState int32

const (
	stateClosed   cbState = 0
	stateOpen     cbState = 1
	stateHalfOpen cbState = 2
)

// CircuitBreaker protects an upstream from cascading failures.
// Ported from more-internals/runnable/design-patterns/ with per-upstream support.
type CircuitBreaker struct {
	mu          sync.Mutex
	state       atomic.Int32
	failures    atomic.Int32
	threshold   int32
	lastFailure atomic.Int64
	retryAfter  time.Duration
}

func NewCircuitBreaker(threshold int, retryAfter time.Duration) *CircuitBreaker {
	return &CircuitBreaker{threshold: int32(threshold), retryAfter: retryAfter}
}

// State returns a human-readable state string for metrics.
func (cb *CircuitBreaker) State() string {
	switch cbState(cb.state.Load()) {
	case stateClosed:
		return "closed"
	case stateOpen:
		return "open"
	case stateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Allow returns nil if the request should proceed, ErrCircuitOpen if blocked.
func (cb *CircuitBreaker) Allow() error {
	state := cbState(cb.state.Load())
	if state == stateOpen {
		lastFail := time.Unix(0, cb.lastFailure.Load())
		if time.Since(lastFail) > cb.retryAfter {
			cb.state.CompareAndSwap(int32(stateOpen), int32(stateHalfOpen))
			return nil
		}
		return ErrCircuitOpen
	}
	return nil
}

// RecordSuccess resets the circuit to closed.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.failures.Store(0)
	cb.state.Store(int32(stateClosed))
}

// RecordFailure increments the failure counter and may open the circuit.
func (cb *CircuitBreaker) RecordFailure() {
	cb.failures.Add(1)
	cb.lastFailure.Store(time.Now().UnixNano())
	if cb.failures.Load() >= cb.threshold {
		cb.state.Store(int32(stateOpen))
	}
}
