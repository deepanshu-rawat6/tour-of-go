# High-Throughput & Database Design Patterns

Architectural patterns for scaling systems to handle massive loads and designing performant data layers.

## 1. Database Sharding (Horizontal Partitioning)
When a single database server can't handle the load, you split the data across multiple servers.

### Go Snippet (Simple Sharding)
```go
// getShard uses a 'Shard Key' to route the request to the correct DB instance.
func getShard(userID int) *sql.DB {
    // modulo sharding is common for simplicity.
    // If we have 10 DB instances, userID 15 goes to shardID 5.
    shardID := userID % len(shards)
    return shards[shardID]
}
```
- **How?** Use a "Shard Key" (e.g., `user_id % 10`) to determine which server holds the data.
- **Trade-off:** Joins across shards become very expensive and should be avoided.

## 2. CQRS (Command Query Responsibility Segregation)
Separate the "write" path from the "read" path.

### Go Snippet (Write vs Read DBs)
```go
type Service struct {
    // writeDB is the "Source of Truth" (e.g., MySQL Master).
    writeDB *sql.DB 
    // readDB is a fast "Read Replica" used to serve queries.
    readDB  *sql.DB 
}
```
- **Why?** Read-heavy applications shouldn't slow down because of a few heavy write operations.

## 3. Connection Pooling
Opening a database connection for every request is slow. Reuse warm connections to improve speed.

### Go Snippet (Tuning Pool in Go)
```go
db, _ := sql.Open("mysql", "dsn")

// 1. Limit the total number of open connections (Concurrency Control).
db.SetMaxOpenConns(100) 

// 2. Keep a number of idle connections 'warm' for immediate reuse.
db.SetMaxIdleConns(50) 

// 3. Set a lifetime for connections to rotate and avoid stale links.
db.SetConnMaxLifetime(time.Hour)
```

## 4. Write-Ahead Logging (WAL)
To ensure high throughput with durability, databases don't write to the main data file immediately. They append to a sequential log file first.

### Real-World Example: PostgreSQL & SQLite
- **Mechanism:** First, append the change to a sequential log file (WAL). Sequential writes are much faster than random disk access.
- **Recovery:** If the DB crashes, it replays the WAL to restore state.

## 5. Batching & Buffering
Instead of sending 1000 small requests to a DB or API, collect them in memory and send 1 large batch.

### Real-World Use Case: Kafka Producer
Kafka batches messages together (`linger.ms` and `batch.size`) before sending them over the network.

### Go Snippet (Ticker-Based Batcher)
```go
func batcher(items <-chan string, flushAt int, interval time.Duration) {
    var buffer []string
    ticker := time.NewTicker(interval)
    
    for {
        select {
        case item := <-items:
            // 1. Accumulate items into an in-memory buffer
            buffer = append(buffer, item)
            // 2. If the buffer is full, send the batch immediately.
            if len(buffer) >= flushAt {
                flush(buffer)
                buffer = nil
            }
        case <-ticker.C:
            // 3. Every X ms, flush the buffer even if it isn't full (Avoid latency).
            if len(buffer) > 0 {
                flush(buffer)
                buffer = nil
            }
        }
    }
}
```

## 6. Rate Limiting (Token Bucket)
Protect your high-throughput system from being "DDoS'ed" by its own clients.

### Real-World Use Case: AWS SQS & DynamoDB
AWS services use rate limiting to prevent one customer from taking down the entire data center's storage layer.

### Go Snippet (Using x/time/rate)
```go
// limiter refilling at 100 tokens per second with a burst capacity of 10.
limiter := rate.NewLimiter(rate.Every(time.Second/100), 10) 

func HandleRequest(r *Request) {
    // 1. Allow() takes a token. If the bucket is empty, it returns false.
    if !limiter.Allow() {
        fmt.Println("Rate limit exceeded")
        return // Reject with HTTP 429 (Too Many Requests)
    }
    // 2. Process the request normally
    process(r)
}
```

## 7. Cache-Aside Pattern
The application is responsible for managing the cache, keeping the hot data fast and the DB safe.

### Go Snippet (Cache Check)
```go
func getData(key string) (string, error) {
    // 1. ALWAYS check the Cache first.
    val, err := cache.Get(key)
    if err == nil {
        return val, nil // Cache Hit - Super fast!
    }
    
    // 2. If Cache Miss, fetch from the slower Database.
    val, err = db.Fetch(key) 
    if err == nil {
        // 3. CRITICAL: Put the data back into the cache so the next call is fast.
        cache.Set(key, val, time.Hour)
    }
    return val, err
}
```

## 8. LSM Trees vs B-Trees (Storage Engines)
- **B-Trees:** Optimized for fast reads (standard in Postgres/MySQL).
- **LSM Trees (Log-Structured Merge-Trees):** Optimized for massive write throughput (used in Cassandra, ScyllaDB, LevelDB).
