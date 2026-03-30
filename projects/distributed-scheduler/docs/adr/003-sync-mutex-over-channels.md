# ADR-003: sync.Mutex + atomic.Int64 over Channels for Concurrency Pool

**Status:** Accepted  
**Date:** 2026-03-31

## Context

The concurrency pool (`map[string]*atomic.Int64`) is shared state accessed by:
- Multiple scheduling goroutines (read: GetVacantSpots)
- The publisher goroutine (write: EvaluateJobPublish)
- Finished job handlers (write: HandleFinishedJob)
- The ConcurrencyRefresher cron (write: PopulateFromDB)

## Decision

Use `sync.RWMutex` for the pool map + `atomic.Int64` per counter.

## Why Not Channels?

The Go proverb "don't communicate by sharing memory; share memory by communicating" applies to **communication between goroutines**, not to **shared state with multiple readers and writers**.

The concurrency pool is fundamentally shared state:
- Multiple goroutines need to read it simultaneously (GetVacantSpots)
- Writes must be atomic across multiple keys (EvaluateJobPublish must check ALL rules before incrementing ANY counter)

A channel-based approach would require:
- A single "pool manager" goroutine serializing all access
- Request/response channels for every read
- This would be slower and more complex than a mutex

## Why RWMutex over plain Mutex?

`GetVacantSpots` is called for every candidate job during the scheduling algorithm — potentially hundreds of times per scheduling cycle. Using `RWMutex` allows concurrent reads, which is the common case.

## The All-or-Nothing Invariant

`EvaluateJobPublish` must hold a **write lock** for the entire check-and-increment operation:

```go
// WRONG: check with read lock, increment with write lock
// → race condition: two goroutines both see capacity available, both increment
rlock(); check(); runlock()
lock(); increment(); unlock()

// CORRECT: check AND increment under single write lock
lock()
if allRulesSatisfied() { incrementAll() }
unlock()
```

This mirrors the Java `synchronized (concurrencyPool)` block exactly.
