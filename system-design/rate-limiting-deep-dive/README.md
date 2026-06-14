# Rate Limiting Implementations in Go

Rate limiting is a critical defense mechanism for high-throughput applications. Here are the four primary techniques implemented in Go.

## 1. Fixed Window Counter
The simplest form. It resets the counter at the start of every time window (e.g., every minute).

### Go Implementation (Fixed Window)
```go
type FixedWindow struct {
	mu     sync.Mutex
	// counts maps the window ID (e.g., current minute) to its count.
	counts map[int64]int
	limit  int
}

func (fw *FixedWindow) Allow(id string) bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// 1. Determine the 'Current Window ID' (e.g., current unix minute).
	now := time.Now().Unix() / 60
	
	// 2. If the count for this window has reached the limit, reject the request.
	if fw.counts[now] >= fw.limit {
		return false
	}

	// 3. Otherwise, increment the count for the window and allow the request.
	fw.counts[now]++
	return true
}
```

## 2. Token Bucket (Standard Choice)
Allows for a specific "burst" size while maintaining a steady long-term rate.

### Go Implementation (Token Bucket)
```go
type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	rate       float64 // The number of tokens added per second.
	burst      int     // The maximum number of tokens the bucket can hold.
	lastRefill time.Time
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// 1. Calculate how many tokens to add based on time passed since last check.
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.rate

	// 2. Ensure tokens do not exceed the burst capacity.
	if tb.tokens > float64(tb.burst) {
		tb.tokens = float64(tb.burst)
	}
	tb.lastRefill = now

	// 3. If there's at least one token available, consume it and allow the request.
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	// 4. No tokens left, reject the request.
	return false
}
```

## 3. Leaky Bucket (Traffic Shaping)
Think of a bucket with a hole in the bottom. No matter how much water you pour in, it leaks at a **constant rate**, smoothing out the traffic perfectly.

### Go Implementation (Leaky Bucket)
```go
type LeakyBucket struct {
	// A buffered channel acts as the "Bucket".
	queue chan struct{}
}

func NewLeakyBucket(rate int) *LeakyBucket {
	lb := &LeakyBucket{
		queue: make(chan struct{}, rate),
	}
	// The "Leak": a background goroutine that empties the bucket at a steady rate.
	go func() {
		// Calculate the interval based on the desired leak rate.
		ticker := time.NewTicker(time.Second / time.Duration(rate))
		for range ticker.C {
			select {
			case <-lb.queue:
				// Leak one item from the bucket
			default:
				// Bucket is empty, nothing to leak
			}
		}
	}()
	return lb
}

func (lb *LeakyBucket) Allow() bool {
	select {
	case lb.queue <- struct{}{}:
		// Successfully added a request to the bucket.
		return true 
	default:
		// Bucket is full! (Overflow) - Reject the request.
		return false 
	}
}
```

## 4. Sliding Window Log
The most accurate (but most memory-intensive) method. It stores a timestamp for every single request.

### Go Implementation (Sliding Window Log)
```go
type SlidingWindowLog struct {
	mu     sync.Mutex
	// logs holds a slice of timestamps for all recent requests.
	logs   []time.Time
	limit  int
	window time.Duration
}

func (sw *SlidingWindowLog) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	// Define the boundary of our current sliding window.
	boundary := now.Add(-sw.window)

	// 1. CLEANUP: Filter out any logs that fall outside the current window.
	filtered := sw.logs[:0]
	for _, t := range sw.logs {
		if t.After(boundary) {
			filtered = append(filtered, t)
		}
	}
	sw.logs = filtered

	// 2. CHECK LIMIT: If current logs are at the limit, reject.
	if len(sw.logs) >= sw.limit {
		return false
	}

	// 3. ADD LOG: If under the limit, record this request and allow.
	sw.logs = append(sw.logs, now)
	return true
}
```
