# Go for Platform Ops & SRE

Platform Engineering requires a deep understanding of how Go interacts with the OS, the network, and orchestration layers like Kubernetes.

## 1. Deep Go Internals (Interview Depth)
To debug high-scale platforms, you must understand the "magic" under the hood:
- **The Scheduler (G-M-P Model):** Understand how **G**oroutines are mapped to **M**achine threads via **P**rocessors.
- **Escape Analysis:** Know when a variable stays on the **Stack** vs. moving to the **Heap**.
- **Garbage Collection (STW):** Understand **Stop-The-World (STW)** pauses and how `GOGC` and `GOMEMLIMIT` affect platform stability.

## 2. The Context Power-User
In Platform Ops, **nothing should run without a timeout**.

### Go Snippet (Context Timeout)
```go
// Create a background context with a 5-second timeout.
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

// CRITICAL: Always defer cancel() to release resources even if the task 
// finishes before the timeout.
defer cancel()

// Create a request tied to the context. If the request takes longer than 
// 5 seconds, it will automatically be cancelled by the system.
req, _ := http.NewRequestWithContext(ctx, "GET", "http://slow-service", nil)
```

## 3. Kubernetes & Client-Go Patterns
If you are building platform tools, you must know the **Operator Pattern**.

### Go Snippet (Using Informers/Listers)
```go
// podLister.List fetches the list of pods from a LOCAL cache (Informer).
// This is significantly faster and less resource-intensive than 
// calling the K8s API server every time.
pods, err := podLister.List(labels.Everything())
if err != nil {
    return err
}

// Proceed to process the cached list of pods.
```

## 4. System Programming in Go
Platforms interact with the Linux kernel through system calls and signal handling.

### Go Snippet (Graceful Signal Handling)
```go
// Create a buffered channel to receive OS signals.
sigChan := make(chan os.Signal, 1)

// Notify our channel of SIGTERM and SIGINT (Interruption).
signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

// Blocks here until a signal is received (e.g., K8s sends SIGTERM).
<-sigChan 

fmt.Println("Graceful shutdown initiated...")
cleanup() // Flush logs, close DB connections, etc.
```

## 5. Observability (The "Golden Signals")
Your Go code isn't production-ready until it exports metrics for Prometheus.

### Go Snippet (Prometheus Counter)
```go
// promauto.NewCounter creates and registers a global counter metric.
var jobsProcessed = promauto.NewCounter(prometheus.CounterOpts{
    Name: "jobs_processed_total",
    Help: "The total number of processed jobs",
})

func process() {
    // Increment the metric every time a job is completed.
    jobsProcessed.Inc()
}
```

## 6. Networking & Protocols
- **gRPC vs. REST:** gRPC is the standard for internal platform communication.
- **TCP/UDP Low-level:** Using the `net` package to write health-check agents.
- **Keep-Alives & Timeouts:** Configuring `http.Transport` to reuse connections.
