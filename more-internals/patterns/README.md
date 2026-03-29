# Go Design Patterns

Commonly used patterns in Go that leverage its unique features like interfaces and concurrency.

## 1. Functional Options Pattern
Functional Options is an idiomatic Go pattern used to configure complex objects. Instead of having multiple constructors (`NewServer`, `NewServerWithAddr`, etc.) or a massive configuration struct where most fields are zero-values, you use functions that modify the object.

### Why use it? (Context)
- **Clean API:** The user only provides the options they care about.
- **Backward Compatibility:** You can add new options in the future without breaking existing code.
- **Default Values:** It's easy to set sensible defaults and override them.

### Real-World Use Case: HTTP Server or Database Client Configuration
When creating an HTTP server, you might want to configure its port, timeout, and logging level. Instead of a constructor with 10 arguments, you use `WithPort(8080)`, `WithTimeout(30s)`, etc.

### Example
```go
type Server struct {
    Addr string
    Port int
}

// Option is a function type that takes a pointer to our Server.
type Option func(*Server)

// WithAddr is a 'Functional Option' that returns a function to set the Addr.
func WithAddr(addr string) Option {
    return func(s *Server) { s.Addr = addr }
}

// NewServer initializes a Server with defaults and applies any provided options.
func NewServer(opts ...Option) *Server {
    // 1. Set default values
    s := &Server{Addr: "localhost", Port: 8080} 

    // 2. Apply each option function to our server instance
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```


## 2. Generator Pattern (Channels)
The Generator pattern is a powerful way to produce a stream of data lazily. Instead of calculating a massive slice and returning it all at once (which consumes memory), a Generator returns a **receive-only channel**. The data is produced in a background goroutine and "sent" to the consumer only when they are ready to read it.

### Why use it? (Context)
- **Memory Efficiency:** You don't need to store the entire dataset in memory.
- **Lazy Evaluation:** You only calculate the next value when the consumer asks for it.
- **Concurrency:** It naturally separates the "production" of data from the "consumption."

### Real-World Use Case: Log Streaming or API Pagination
Imagine you are fetching **1 million logs** from a database. Instead of loading them into a slice `[]Log`, you return a `<-chan Log`. As your UI or processing engine reads from the channel, the generator fetches the next batch from the DB.

### Example: Fibonacci Generator
```go
// fibGenerator returns a receive-only channel (<-chan) of Fibonacci numbers.
// This allows the caller to iterate over the results as they are produced.
func fibGenerator(n int) <-chan int {
    // 1. Create an unbuffered channel to communicate between goroutines
    c := make(chan int)

    // 2. Start a background goroutine to produce data
    go func() {
        a, b := 0, 1
        for i := 0; i < n; i++ {
            // 3. This send operation blocks until a consumer reads from the channel
            c <- a 
            a, b = b, a+b
        }
        // 4. CRITICAL: Close the channel so the 'range' loop in the consumer knows when to stop
        close(c)
    }()

    // 5. Return the channel immediately; the producer runs in the background
    return c
}

// Usage in consumer:
// for num := range fibGenerator(10) {
//     fmt.Println(num) // Reads one by one, memory stays low
// }
```


## 3. Worker Pool Pattern
A Worker Pool is a pattern that manages a fixed number of goroutines to perform a large number of tasks. This is essential for controlling concurrency and resource usage.

### Why use it? (Context)
- **Concurrency Control:** Limits the number of goroutines to prevent overloading memory or CPU.
- **Resource Re-use:** Reuses a set of goroutines instead of spawning a new one for every task.
- **Backpressure:** Forces the producer to wait if all workers are busy.

### Real-World Use Case: Image Processing or Log Ingestion
If you need to process **10,000 images**, you don't spawn 10,000 goroutines (which could crash the machine). Instead, you spawn **10 workers** that each take an image from a "jobs" channel, process it, and send the result to a "results" channel.

### Example
```go
// worker takes tasks from the 'jobs' channel and sends results to the 'results' channel.
// We use directional channels: <-chan for read-only, chan<- for write-only.
func worker(id int, jobs <-chan int, results chan<- int) {
    // 1. Each worker iterates over the jobs channel until it is closed.
    for j := range jobs {
        // 2. Perform the actual task (in this case, just doubling a number)
        fmt.Printf("worker %d processing job %d\n", id, j)
        results <- j * 2
    }
    // 3. When 'jobs' is closed, the range loop exits and the worker finishes.
}

// In the main function:
// 1. Create channels for jobs and results
// 2. Spawn 3-5 workers in background
// 3. Send 100 jobs to the channel
// 4. Close the jobs channel
```

