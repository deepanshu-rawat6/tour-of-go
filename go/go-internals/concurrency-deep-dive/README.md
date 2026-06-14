# In-Depth Concurrency & Orchestration

In platform engineering, concurrency is not just about "doing things at once"; it's about **orchestration**, **resource safety**, and **error propagation**.

## 1. The Error Group (errgroup.Group)
Standard `sync.WaitGroup` is great, but it doesn't handle errors. In a high-throughput system, if one sub-task fails, you usually want to cancel all others and return that error.

### Real-World Use Case: Parallel API Calls
Imagine a "Dashboard" page that needs to call 5 different microservices (User, Billing, Posts, Friends, Notifications). If any of these calls fail, the whole page might as well fail. **Errgroup** triggers all 5 calls in parallel and returns the first error encountered, while also cancelling the other 4 calls immediately to save resources.

### Go Snippet (Error Group)
```go
// WithContext returns a new Group and an associated Context derived from ctx.
// The derived Context is cancelled the first time a function passed to Go returns a non-nil error.
g, ctx := errgroup.WithContext(mainCtx)

for _, url := range urls {
    url := url // avoid closure capture issues
    // g.Go runs the function in a new goroutine
    g.Go(func() error {
        // Pass the derived ctx to the function so it can be cancelled
        return fetchURL(ctx, url) 
    })
}

// Wait blocks until all function calls from the Go method have returned, 
// then returns the first non-nil error (if any).
if err := g.Wait(); err != nil {
    return err 
}
```

## 2. Weighted Semaphores (Concurrency Limiting)
A worker pool is a "fixed" limit. A semaphore allows for "weighted" limits—useful when different tasks consume different amounts of "capacity" (e.g., CPU vs Memory).

### Go Snippet (Semaphore)
```go
// Create a semaphore with a total weight of 10.
var sem = semaphore.NewWeighted(10)

func handle(ctx context.Context) {
    // Acquire weight of 1. Blocks until weight is available or ctx is cancelled.
    if err := sem.Acquire(ctx, 1); err != nil {
        return
    }
    // Release MUST be called to return weight to the semaphore.
    defer sem.Release(1)
    
    // Perform task...
}
```

## 3. Resource Recycling (sync.Pool)
In high-throughput Go applications (like a log agent), the biggest bottleneck is **GC Pressure** from allocating temporary buffers.

### Real-World Use Case: Uber's `zap` logger
The famous **Zap Logger** uses `sync.Pool` extensively. Every time you log a message, it doesn't create a new string or buffer. It pulls one from a pool, writes to it, and puts it back. This is why Zap is significantly faster and uses less memory.

### Go Snippet (sync.Pool)
```go
var bufPool = sync.Pool{
    // New defines a function to create an item when the pool is empty.
    New: func() interface{} { return new(bytes.Buffer) },
}

func Log(msg string) {
    // Get retrieves an item from the pool.
    b := bufPool.Get().(*bytes.Buffer)
    
    // CRITICAL: Always reset the state of the object before use!
    b.Reset() 
    
    // Put returns the item to the pool for reuse.
    defer bufPool.Put(b)
    
    b.WriteString(msg)
    os.Stdout.Write(b.Bytes())
}
```

## 4. The Signal/Broadcast Pattern (sync.Cond)
Channels are great for "handing off" data. `sync.Cond` is better for "notifying" multiple goroutines that a specific state has changed.

### Go Snippet (sync.Cond)
```go
// Create a new condition variable with a Locker (usually a Mutex).
c := sync.NewCond(&sync.Mutex{})

// In workers:
c.L.Lock()
for !condition {
    // Wait atomically unlocks c.L and suspends execution of the goroutine.
    // After resuming, Wait locks c.L before returning.
    c.Wait()
}
c.L.Unlock()

// In leader:
// Broadcast wakes ALL goroutines waiting on c.
c.Broadcast() 
```

## 5. Load Shedding & Admission Control
A robust platform tool must know when to say **"NO"** instead of crashing.

### Go Snippet (Select Default)
```go
select {
case workQueue <- task:
    // Task accepted into the buffered channel
case <-ctx.Done():
    return ctx.Err()
default:
    // If the channel is full, the 'default' case executes immediately.
    // This is 'Load Shedding' - dropping work to protect the system.
    metrics.DroppedTasks.Inc()
    return errors.New("overloaded")
}
```

## 6. The "Done" Channel Pattern (Graceful Shutdown)
In SRE/Ops, your agent will be restarted often (K8s Rolling Updates). You must handle `SIGTERM` to finish current work.

### Go Snippet (Quit Channel)
```go
// done is a receive-only channel used to signal shutdown.
func worker(done <-chan struct{}) {
    for {
        select {
        case <-done:
            // Stop the worker when the channel is closed or a signal is sent.
            fmt.Println("Worker stopping...")
            return
        default:
            // Continue performing work
            doWork()
        }
    }
}
```

## 7. Dynamic Worker Scaling
Instead of a fixed `const Workers = 10`, use a goroutine that monitors the length of your `workQueue` to scale workers up or down dynamically based on load.
- **Logic:** 
    - If `len(queue) > threshold`, start a new worker goroutine.
    - If `len(queue) == 0` for X seconds, have the worker exit.
- **Benefit:** Saves memory on quiet nodes while scaling up on busy ones.
