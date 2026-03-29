# Distributed Job-Scheduler Design

Design principles for building a robust, distributed job execution system.

## 1. Idempotency (Exactly-Once Semantics)
In a distributed system, network failures cause retries. Each job MUST be idempotent, meaning running it twice results in the same outcome as running it once.

### Real-World Use Case: Stripe Payment API
When a client sends a request to charge a customer, they include an `Idempotency-Key` header. If the network drops and they retry with the same key, Stripe recognizes it and doesn't charge the customer twice.

### Go Snippet (The "Database-First" Check)
```go
// ProcessJob handles a job in an idempotent way using a database check.
func (s *Worker) ProcessJob(jobID string) error {
    // 1. Check if the jobID has already been marked as 'processed' in the database.
    // This must be a thread-safe or atomic operation.
    processed, err := s.db.HasProcessed(jobID)
    if err != nil {
        return err // Database error, retry later
    }
    
    // 2. If it's already done, return successfully without re-doing the work.
    if processed {
        fmt.Printf("Job %s already completed, skipping.\n", jobID)
        return nil 
    }

    // 3. Perform the actual work (e.g., charge a customer, send an email)
    if err := s.performTask(jobID); err != nil {
        return err
    }
    
    // 4. Mark as processed so future retries are ignored.
    return s.db.MarkAsProcessed(jobID)
}
```

## 2. Distributed Consensus (Leasing)
When multiple schedulers exist, they must agree on who is the "leader" or who owns a job to avoid double execution.

### Real-World Use Case: Kubernetes Leader Election
Multiple `kube-scheduler` instances run, but only one is "active." They use a Lease object in Etcd; the one that holds the lease is the leader.

### Go Snippet (Using Redis/Etcd Lease)
```go
// GrabJob attempts to acquire a timed "lease" on a specific job.
func (w *Worker) GrabJob(ctx context.Context, jobID string) bool {
    // We use Redis SET with NX (Set if Not eXists) and EX (Expires in 30s).
    // This ensures only one worker can 'own' the job at any given time.
    success, err := w.redis.SetNX(ctx, "lock:"+jobID, "worker_1", 30*time.Second).Result()
    if err != nil {
        return false // Error communicating with Redis
    }
    
    // If success is true, this worker now holds the lease and can process the job.
    return success
}
```

## 3. Graceful Worker Shutdown (Context)
When a worker node is being replaced, it should not kill running jobs immediately.

### Go Snippet (SIGTERM Handling)
```go
// Start initiates the worker loop and listens for shutdown signals.
func (w *Worker) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            // ctx.Done() is closed when the parent context is cancelled (e.g., SIGTERM).
            fmt.Println("Shutting down worker gracefully...")
            w.cleanup() // Finish current tasks, close connections
            return
        default:
            // Continue processing the next available job
            w.runOneJob()
        }
    }
}
```

## 4. Dead Letter Queues (DLQ)
Jobs that fail repeatedly after retries should not block the system or enter an infinite loop.

### Go Snippet (Move to DLQ)
```go
// ProcessWithRetry attempts to run a job and moves it to a DLQ after MaxRetries.
func (w *Worker) ProcessWithRetry(job Job) {
    if err := runJob(job); err != nil {
        // If the job has failed too many times, move it to the 'Dead Letter' storage.
        if job.RetryCount >= MaxRetries {
            fmt.Printf("Job %s failed %d times. Moving to DLQ.\n", job.ID, MaxRetries)
            w.moveToDLQ(job) 
        } else {
            // Increment retry count and put back into the queue with a delay (Backoff).
            job.RetryCount++
            w.requeue(job)
        }
    }
}
```

## 5. Backpressure and Throttling
A scheduler should not overwhelm its workers or downstream databases.

### Go Snippet (Semaphore Limiting)
```go
// Limit the number of concurrent jobs to 10 using a Weighted Semaphore.
var sem = semaphore.NewWeighted(10)

func (w *Worker) HandleJob(ctx context.Context, job Job) {
    // Acquire(ctx, 1) blocks if 10 jobs are already running.
    // This provides 'Backpressure' to the job source.
    if err := sem.Acquire(ctx, 1); err != nil {
        fmt.Println("Failed to acquire semaphore:", err)
        return 
    }
    
    // Release MUST be called to allow the next job to start.
    defer sem.Release(1)
    
    w.execute(job)
}
```
