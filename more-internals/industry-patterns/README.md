# Industry Design Patterns in Go

These patterns are standard in professional Go codebases, particularly in microservices and web APIs.

## 1. Middleware (Decorator) Pattern
Extremely common in web development for cross-cutting concerns like logging, authentication, and rate limiting. It wraps a handler to add functionality.

### Real-World Use Case: HTTP Request Logging
Before processing a user's request, you want to log the request method and path. Middleware allows you to do this for ALL endpoints without repeating code.

### Example
```go
// Handler defines the function signature for a standard request.
type Handler func(string)

// LoggingMiddleware wraps an existing Handler to add logging functionality.
func LoggingMiddleware(next Handler) Handler {
    return func(s string) {
        // 1. Perform 'Pre-processing' logic
        fmt.Println("Logging: Received request for", s)
        
        // 2. Call the 'Actual' handler (the one being wrapped)
        next(s)
        
        // 3. Perform 'Post-processing' logic (optional)
        fmt.Println("Logging: Request finished.")
    }
}
```

## 2. Strategy Pattern
Instead of complex `switch` statements, use interfaces to define a family of algorithms. This makes the code open for extension but closed for modification.

### Real-World Use Case: Multiple Payment Methods
A checkout system that supports **Credit Card**, **PayPal**, and **Bitcoin**. Instead of a giant `if-else` block, you use the Strategy pattern to select the payment method at runtime.

### Example
```go
// PaymentStrategy defines the contract for any payment method.
type PaymentStrategy interface {
    Pay(amount float64)
}

// CreditCard implements PaymentStrategy.
type CreditCard struct{}
func (c *CreditCard) Pay(amount float64) { fmt.Println("Paid via Card") }

// PayPal implements PaymentStrategy.
type PayPal struct{}
func (p *PayPal) Pay(amount float64) { fmt.Println("Paid via PayPal") }

type Checkout struct {
    // strategy can be ANY type that implements PaymentStrategy.
    strategy PaymentStrategy
}

func (c *Checkout) Process(amount float64) {
    // We don't care HOW the payment happens; we just call the Pay() method.
    c.strategy.Pay(amount)
}
```

## 3. Circuit Breaker Pattern
Used in distributed systems to prevent a failing service from causing a cascading failure. If a downstream service is down, the "circuit" trips, and subsequent calls fail fast.

### Real-World Use Case: External API Calls
If the **Google Maps API** you depend on is failing, the Circuit Breaker stops your app from waiting 30 seconds for a timeout on every request, allowing your system to stay responsive.

### Go Implementation
```go
type State int

const (
    Closed State = iota // Normal operation
    Open                // Failures exceeded; stop requests
    HalfOpen            // Testing for recovery
)

type CircuitBreaker struct {
    mu           sync.Mutex
    state        State
    failures     int
    threshold    int // Max failures before opening the circuit
    retryTimeout time.Duration
    lastFailure  time.Time
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
    cb.mu.Lock()
    
    // 1. If Open, check if enough time has passed to try again (Half-Open).
    if cb.state == Open && time.Since(cb.lastFailure) > cb.retryTimeout {
        cb.state = HalfOpen
    }

    // 2. If STILL Open, block the request immediately to avoid waiting for timeouts.
    if cb.state == Open {
        cb.mu.Unlock()
        return errors.New("circuit is open; request blocked")
    }
    cb.mu.Unlock()

    // 3. Perform the actual call to the downstream service.
    err := fn()

    cb.mu.Lock()
    defer cb.mu.Unlock()

    if err != nil {
        cb.failures++
        cb.lastFailure = time.Now()
        // 4. Trip the circuit if the failure threshold is reached.
        if cb.failures >= cb.threshold {
            cb.state = Open
        }
        return err
    }

    // 5. Success! Reset the circuit to normal operation.
    cb.failures = 0
    cb.state = Closed
    return nil
}
```

## 4. Single-Flight Pattern
Prevents "cache stampede" by ensuring that only one execution is in flight for a given key.

### Real-World Use Case: Heavy Database Query
If **1000 users** simultaneously request the same "Top 10 Products" list, Single-Flight ensures only **ONE** database query is executed. The other 999 users "wait" for that single result and share it.

### Example (using `golang.org/x/sync/singleflight`)
```go
var g singleflight.Group

func getData(key string) (interface{}, error) {
    // Do() ensures that only one call to the anonymous function 
    // is "in-flight" for the given 'key'.
    v, err, shared := g.Do(key, func() (interface{}, error) {
        // This only runs ONCE for multiple simultaneous calls
        return db.FetchFromHeavySource(key) 
    })
    
    // shared will be true for the 999 users who didn't trigger the fetch.
    return v, err
}
```
