# Background Jobs

Patterns for running work asynchronously in Go: debounce, throttle, worker pools,
delayed jobs, recurring jobs, and graceful shutdown.

---

## When NOT to Use In-Process Jobs

Before building anything here, consider whether in-process is appropriate:

| Situation | Better approach |
|---|---|
| Work must survive crashes/restarts | Use a persistent queue (Redis Streams, SQS, Postgres) |
| Jobs take > 30 seconds | Use a distributed worker (Temporal, Asynq, River) |
| Work spans multiple services | Use a message broker (Kafka, RabbitMQ) |
| You need retries with backoff | Use a job queue with built-in retry (Asynq, River) |
| Jobs must not run concurrently across instances | Use distributed locking + queue |

**In-process jobs are great for:** cache warming, metrics flushing, async
notifications, debouncing user input, periodic cleanup, best-effort background work.

---

## Pattern 1: Debounce

**Collapse rapid events into one.** The function fires only after N milliseconds
of silence (no new calls). Classic use: search-as-you-type, auto-save.

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

// Debounce returns a function that delays calling fn until after wait has
// elapsed since the last invocation. If called again before wait elapses,
// the timer resets.
//
// The returned function is safe to call from multiple goroutines.
//
// Example: search-as-you-type — only hit the API after the user stops typing
// for 300ms, preventing a request on every keystroke.
func Debounce(fn func(), wait time.Duration) func() {
    var (
        mu    sync.Mutex
        timer *time.Timer
    )
    return func() {
        mu.Lock()
        defer mu.Unlock()

        // Cancel any pending invocation and start fresh
        if timer != nil {
            timer.Stop()
        }
        timer = time.AfterFunc(wait, fn)
    }
}

func main() {
    calls := 0
    search := Debounce(func() {
        calls++
        fmt.Printf("API call #%d\n", calls)
    }, 300*time.Millisecond)

    // Simulate rapid keystrokes — only the last one fires the API
    for i := 0; i < 10; i++ {
        search()
        time.Sleep(50 * time.Millisecond) // faster than debounce window
    }
    time.Sleep(500 * time.Millisecond) // wait for the debounce to fire
    fmt.Printf("Total API calls: %d (expected 1)\n", calls)
}
```

**Debounce with argument:** if you need to pass the "latest value":

```go
// DebounceArg is a generic debounce that carries the most-recent argument.
func DebounceArg[T any](fn func(T), wait time.Duration) func(T) {
    var (
        mu    sync.Mutex
        timer *time.Timer
    )
    return func(arg T) {
        mu.Lock()
        defer mu.Unlock()
        if timer != nil {
            timer.Stop()
        }
        timer = time.AfterFunc(wait, func() { fn(arg) })
    }
}

// Usage:
//   save := DebounceArg(saveToDatabase, 500*time.Millisecond)
//   save(draft)  // called on every keystroke
```

---

## Pattern 2: Throttle

**Max N executions per time window.** Unlike debounce (waits for quiet), throttle
guarantees a minimum gap between executions regardless of how often you call it.

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

// Throttle returns a function that executes fn at most once per interval.
// Calls during the cooldown are dropped. The returned function is goroutine-safe.
//
// Use case: rate-limiting API calls, preventing double-click form submissions.
func Throttle(fn func(), interval time.Duration) func() {
    var (
        mu      sync.Mutex
        lastRun time.Time
    )
    return func() {
        mu.Lock()
        defer mu.Unlock()

        if time.Since(lastRun) < interval {
            return // drop this call — too soon
        }
        lastRun = time.Now()
        go fn() // run in background to avoid blocking the caller
    }
}

// ThrottleWithLast is a throttle that also fires on the LAST call after the
// window. This combines throttle + debounce: you get immediate response AND
// the final state after rapid calls settle.
func ThrottleWithLast(fn func(), interval time.Duration) func() {
    var (
        mu      sync.Mutex
        lastRun time.Time
        pending *time.Timer
    )
    return func() {
        mu.Lock()
        defer mu.Unlock()

        // Reset pending trailing call
        if pending != nil {
            pending.Stop()
        }
        // Schedule trailing call unconditionally
        pending = time.AfterFunc(interval, func() {
            mu.Lock()
            lastRun = time.Now()
            mu.Unlock()
            fn()
        })

        // Leading edge: fire immediately if outside the window
        if time.Since(lastRun) >= interval {
            lastRun = time.Now()
            go fn()
        }
    }
}

func main() {
    calls := 0
    limited := Throttle(func() { calls++ }, 100*time.Millisecond)

    // Fire 20 times in 50ms (all within the 100ms window)
    for i := 0; i < 20; i++ {
        limited()
        time.Sleep(5 * time.Millisecond)
    }
    time.Sleep(200 * time.Millisecond)
    fmt.Printf("Throttled calls: %d (not 20)\n", calls)
}
```

---

## Pattern 3: Simple In-Process Worker Pool

Distribute work across N concurrent workers. Classic producer-consumer pattern
using channels as the job queue.

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

// Job represents a unit of work.
type Job struct {
    ID      int
    Payload string
}

// Result holds the outcome of a job.
type Result struct {
    JobID  int
    Output string
    Err    error
}

// WorkerPool processes jobs concurrently with a fixed number of workers.
// The pool reads from jobs channel and sends results to results channel.
//
// Pattern:
//   1. Create buffered channels to decouple producer and workers.
//   2. Start N worker goroutines; each reads from jobs until channel closes.
//   3. Use a WaitGroup to know when all workers are done.
//   4. Close results channel after all workers finish (signals consumer).
func WorkerPool(workers int, jobs <-chan Job, results chan<- Result) {
    var wg sync.WaitGroup

    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            for job := range jobs { // exits when jobs channel is closed
                // Simulate work
                time.Sleep(10 * time.Millisecond)
                results <- Result{
                    JobID:  job.ID,
                    Output: fmt.Sprintf("worker %d processed: %s", workerID, job.Payload),
                }
            }
        }(i)
    }

    // Close results when all workers are done so the consumer can range over it.
    go func() {
        wg.Wait()
        close(results)
    }()
}

func main() {
    const numJobs = 20
    const numWorkers = 5

    jobs := make(chan Job, numJobs)     // buffered: producer doesn't block
    results := make(chan Result, numJobs)

    WorkerPool(numWorkers, jobs, results)

    // Producer: submit all jobs
    for i := 1; i <= numJobs; i++ {
        jobs <- Job{ID: i, Payload: fmt.Sprintf("task-%d", i)}
    }
    close(jobs) // signal: no more jobs

    // Consumer: collect all results
    for r := range results {
        fmt.Printf("result %d: %s\n", r.JobID, r.Output)
    }
}
```

---

## Pattern 4: Delayed Jobs

Execute a function after a specified delay. `time.AfterFunc` is the stdlib
primitive; it runs the function in a new goroutine after the duration.

```go
package main

import (
    "fmt"
    "time"
)

// DelayedJob schedules fn to run after delay.
// Returns a cancel function to abort before execution.
//
// Use cases:
//   - Send "you left items in your cart" email 24h after abandonment
//   - Expire a temporary token after 15 minutes
//   - Retry a failed operation after exponential backoff
func DelayedJob(delay time.Duration, fn func()) (cancel func()) {
    timer := time.AfterFunc(delay, fn)
    return func() { timer.Stop() }
}

// ExponentialBackoff retries fn up to maxAttempts times with exponential delay.
// Delay sequence: base, base*2, base*4, base*8, ...
// Adds jitter to prevent the thundering herd: multiple services don't all
// retry at the same millisecond.
func ExponentialBackoff(maxAttempts int, base time.Duration, fn func() error) {
    for attempt := 0; attempt < maxAttempts; attempt++ {
        if err := fn(); err == nil {
            return // success
        }
        if attempt == maxAttempts-1 {
            fmt.Println("max retries reached")
            return
        }
        delay := base * (1 << attempt) // 100ms, 200ms, 400ms, 800ms, ...
        fmt.Printf("attempt %d failed, retrying in %v\n", attempt+1, delay)
        time.Sleep(delay)
    }
}

func main() {
    fmt.Println("scheduling delayed job...")
    cancel := DelayedJob(200*time.Millisecond, func() {
        fmt.Println("delayed job executed!")
    })

    // Cancel before it fires (demonstrates cancellation)
    _ = cancel
    // cancel() would prevent execution

    time.Sleep(500 * time.Millisecond) // wait for the job to run

    // Exponential backoff retry
    attempts := 0
    ExponentialBackoff(4, 50*time.Millisecond, func() error {
        attempts++
        if attempts < 3 {
            return fmt.Errorf("not ready yet")
        }
        fmt.Println("operation succeeded on attempt", attempts)
        return nil
    })
}
```

---

## Pattern 5: Recurring Jobs (Cron-like with ticker)

Use `time.NewTicker` for periodic work. Always respect context cancellation
for clean shutdown.

```go
package main

import (
    "context"
    "fmt"
    "time"
)

// RecurringJob runs fn on every tick until ctx is cancelled.
// It is safe to use with graceful shutdown: cancel the context and the
// goroutine exits cleanly on the next select iteration.
//
// IMPORTANT: if fn takes longer than interval, the next tick is NOT skipped —
// it queues up. Use a non-blocking send or a "running" flag to prevent overlap.
func RecurringJob(ctx context.Context, interval time.Duration, fn func(ctx context.Context)) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            fmt.Println("recurring job: context cancelled, stopping")
            return
        case t := <-ticker.C:
            fmt.Printf("tick at %v\n", t.Format(time.TimeOnly))
            fn(ctx)
        }
    }
}

// NonOverlappingJob prevents concurrent runs: if fn is still running when
// the next tick fires, skip that tick.
func NonOverlappingJob(ctx context.Context, interval time.Duration, fn func()) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    running := make(chan struct{}, 1) // semaphore: 1 slot = at most 1 concurrent run

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            select {
            case running <- struct{}{}: // acquired slot
                go func() {
                    defer func() { <-running }() // release slot when done
                    fn()
                }()
            default:
                // Previous run still in progress — skip this tick
                fmt.Println("skipping tick: previous run still active")
            }
        }
    }
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
    defer cancel()

    go RecurringJob(ctx, 100*time.Millisecond, func(ctx context.Context) {
        fmt.Println("  → doing periodic work")
    })

    <-ctx.Done()
    time.Sleep(50*time.Millisecond) // let the goroutine print its exit message
}
```

---

## Pattern 6: Persistent Job Queue Patterns

For jobs that must survive crashes, use an external store. Popular choices:

| Tool | Backed by | Use When |
|---|---|---|
| **Asynq** | Redis | High-throughput, simple retry, priorities |
| **River** | PostgreSQL | You already have Postgres; ACID guarantees |
| **Temporal** | MySQL/Cassandra | Complex workflows, human approval steps |
| **SQS + Lambda** | AWS | Serverless, managed, pay-per-message |
| **BullMQ** | Redis | Node.js ecosystem; Go can produce/consume |

**Transactional outbox pattern** — for reliable "enqueue on database write":

```go
// The outbox pattern ensures a job is enqueued if and only if the
// database transaction commits. No lost jobs, no double-enqueues.
//
// 1. Within a transaction:
//    - Write your domain change (e.g., orders INSERT)
//    - Write a row to outbox table (job type + payload)
//    - COMMIT
//
// 2. A background poller reads unprocessed outbox rows and pushes to queue.
// 3. On success: mark outbox row as processed.
//
// This decouples your app from the queue — if the queue is down,
// the transaction still succeeds and jobs accumulate in the outbox.

// BEGIN TRANSACTION
//   INSERT INTO orders (user_id, total) VALUES ($1, $2) RETURNING id
//   INSERT INTO outbox (job_type, payload, created_at)
//          VALUES ('send_confirmation_email', '{"order_id": 42}', NOW())
// COMMIT
```

---

## Pattern 7: Dead-Letter / Retry with Backoff

```go
package main

import (
    "errors"
    "fmt"
    "time"
)

// RetryConfig controls retry behaviour.
type RetryConfig struct {
    MaxAttempts int
    InitialWait time.Duration
    MaxWait     time.Duration
    Multiplier  float64
}

var DefaultRetry = RetryConfig{
    MaxAttempts: 5,
    InitialWait: 100 * time.Millisecond,
    MaxWait:     30 * time.Second,
    Multiplier:  2.0,
}

// ErrMaxRetries is returned when all attempts are exhausted.
var ErrMaxRetries = errors.New("max retries exceeded")

// WithRetry executes fn with exponential backoff. Returns the last error if
// all attempts fail. Permanent errors (non-retryable) should be wrapped with
// a sentinel type so callers can distinguish transient from permanent failures.
func WithRetry(cfg RetryConfig, fn func() error) error {
    wait := cfg.InitialWait

    for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
        err := fn()
        if err == nil {
            return nil // success
        }

        if attempt == cfg.MaxAttempts {
            return fmt.Errorf("%w: last error: %v", ErrMaxRetries, err)
        }

        fmt.Printf("attempt %d/%d failed: %v — retrying in %v\n",
            attempt, cfg.MaxAttempts, err, wait)
        time.Sleep(wait)

        // Exponential backoff with cap
        wait = time.Duration(float64(wait) * cfg.Multiplier)
        if wait > cfg.MaxWait {
            wait = cfg.MaxWait
        }
    }
    return ErrMaxRetries
}

// In a real dead-letter queue (DLQ) system:
//
//   1. On MaxRetries exceeded, write the failed job to a DLQ table/topic.
//   2. Alert the on-call team (PagerDuty/Slack).
//   3. Provide an admin endpoint to replay DLQ jobs after the bug is fixed.
//
//   // Asynq DLQ example (pseudo-code):
//   inspector := asynq.NewInspector(redisOpt)
//   tasks, _ := inspector.ListDeadTasks("default")
//   for _, task := range tasks {
//       inspector.RunTask("default", task.ID)  // replay
//   }
```

---

## Pattern 8: Graceful Shutdown of Background Workers

Stop background goroutines cleanly without losing in-flight work.

```go
package main

import (
    "context"
    "fmt"
    "os/signal"
    "sync"
    "syscall"
    "time"
)

// BackgroundWorker manages a set of background goroutines.
// All goroutines respect the context; when it's cancelled they finish
// their current unit of work and exit.
type BackgroundWorker struct {
    wg     sync.WaitGroup
    ctx    context.Context
    cancel context.CancelFunc
}

func NewBackgroundWorker() *BackgroundWorker {
    ctx, cancel := context.WithCancel(context.Background())
    return &BackgroundWorker{ctx: ctx, cancel: cancel}
}

// Go starts a goroutine tracked by the worker's WaitGroup.
// The goroutine receives the shared context; it should select on ctx.Done().
func (bw *BackgroundWorker) Go(fn func(ctx context.Context)) {
    bw.wg.Add(1)
    go func() {
        defer bw.wg.Done()
        fn(bw.ctx)
    }()
}

// Shutdown signals all goroutines to stop and waits for them to finish.
// timeout is the maximum wait time before giving up.
func (bw *BackgroundWorker) Shutdown(timeout time.Duration) {
    bw.cancel() // signal all goroutines to stop

    done := make(chan struct{})
    go func() {
        bw.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        fmt.Println("graceful shutdown complete")
    case <-time.After(timeout):
        fmt.Println("shutdown timeout: some goroutines still running")
    }
}

func main() {
    bw := NewBackgroundWorker()

    // Start two background jobs
    bw.Go(func(ctx context.Context) {
        ticker := time.NewTicker(50 * time.Millisecond)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                fmt.Println("metrics flusher: shutting down")
                return
            case <-ticker.C:
                fmt.Println("flushing metrics...")
            }
        }
    })

    bw.Go(func(ctx context.Context) {
        for {
            select {
            case <-ctx.Done():
                fmt.Println("cache warmer: shutting down")
                return
            case <-time.After(200 * time.Millisecond):
                fmt.Println("warming cache...")
            }
        }
    })

    // Wait for OS signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

    time.Sleep(300 * time.Millisecond) // let jobs run briefly
    fmt.Println("\nShutdown signal received...")
    bw.Shutdown(5 * time.Second)
}
```

---

## Choosing the Right Pattern

```
Is the work triggered by a user event?
  ├── yes, collapse rapid events? → Debounce
  ├── yes, limit rate?            → Throttle
  └── yes, run in background?    → WorkerPool (fire-and-forget)

Is the work scheduled?
  ├── once, in the future?       → DelayedJob (time.AfterFunc)
  └── repeatedly?                → RecurringJob (time.NewTicker)

Does the work survive restarts?
  ├── no (acceptable loss)       → In-process patterns above
  └── yes (must not lose)        → Persistent queue (Asynq, River, SQS)

Did it fail?
  └── retry needed?              → WithRetry (exponential backoff)
  └── gave up?                   → Dead-letter queue
```
