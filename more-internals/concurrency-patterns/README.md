# Advanced Concurrency Patterns

Go's concurrency primitives (channels and goroutines) allow for powerful orchestration patterns.

## 1. Pipeline Pattern
A pipeline is a series of stages connected by channels.

### Real-World Use Case: Image Processing
In an app like **Instagram**, an image upload might go through several stages:
1. Resize (Stage 1)
2. Apply Filter (Stage 2)
3. Generate Thumbnail (Stage 3)
4. Upload to S3 (Stage 4)
Using a pipeline allows Stage 1 to start processing the next image while Stage 2 is still filtering the first one.

### Example
```go
// gen transforms a slice of integers into a receive-only channel.
func gen(nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        for _, n := range nums {
            // Send each number to the 'out' channel
            out <- n
        }
        // Close the channel so the 'range' loop in the next stage knows when to stop.
        close(out)
    }()
    return out
}

// sq receives numbers from a channel, squares them, and sends them to a new channel.
func sq(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        for n := range in {
            // Square each incoming number and send to 'out'
            out <- n * n
        }
        close(out)
    }()
    return out
}

// Usage in consumer:
// for n := range sq(sq(gen(2, 3))) { 
//     fmt.Println(n) // Result: 16, 81
// }
```

## 2. Fan-out, Fan-in
Multiplexing multiple inputs into one.

### Real-World Use Case: Log Aggregation (Search)
Imagine searching for a "User ID" across **100 different log files**.
- **Fan-out:** 10 goroutines each take a batch of files to search in parallel.
- **Fan-in:** All 10 send their "matches" back to a single `results` channel.
This turns a slow, linear search into a fast, parallel operation.

### Fan-in Example
```go
// merge (Fan-in) takes a variable number of input channels and returns a single merged channel.
func merge(cs ...<-chan int) <-chan int {
    var wg sync.WaitGroup
    out := make(chan int)

    // output is a function that copies values from one input channel to the merged channel.
    output := func(c <-chan int) {
        for n := range c {
            out <- n
        }
        // Signal that one of the input channels is done.
        wg.Done()
    }

    // Launch a goroutine for each input channel.
    wg.Add(len(cs))
    for _, c := range cs {
        go output(c)
    }

    // Start a background goroutine to close the 'out' channel when ALL inputs are finished.
    go func() {
        wg.Wait()
        close(out)
    }()
    
    return out
}
```
