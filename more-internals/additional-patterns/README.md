# Additional Engineering Patterns in Go

Further patterns to help you write testable, decoupled, and maintainable Go code.

## 1. Dependency Injection (Constructor Injection)
Instead of a struct creating its own dependencies (like a database connection), you "inject" them via the constructor. This is the #1 way to make Go code testable.

### Real-World Use Case: Unit Testing
If your `Service` directly connects to a **real MySQL database**, your tests will be slow and flaky. By using **Dependency Injection**, you can pass a `MockDatabase` during testing. This makes your tests run in milliseconds and ensures they don't need a real database to pass.

### Example
```go
// Database is an interface, allowing for different implementations (Real vs Mock).
type Database interface {
    Query(q string) string
}

type Service struct {
    // db is "injected" into this struct. The service doesn't know or care
    // if it's a real database or a mock.
    db Database 
}

// NewService is the "Constructor" where we inject the dependency.
func NewService(db Database) *Service {
    return &Service{db: db}
}

// In production: s := NewService(RealMySQLDatabase{})
// In tests:      s := NewService(MockDatabase{})
```

## 2. Observer Pattern (Pub/Sub)
Useful when multiple components need to react to a single event. In Go, this is often implemented using a slice of channels.

### Real-World Use Case: Video Upload System
When a user uploads a video to **YouTube**, the `UploadService` notifies the `NotificationService` (to tell subscribers), the `TranscoderService` (to start resizing), and the `AnalyticsService` (to log the event).

### Example
```go
type Subject struct {
    // observers is a slice of channels. Each channel represents a subscriber.
    observers []chan string
}

// Subscribe allows a new component to listen for events.
func (s *Subject) Subscribe() chan string {
    ch := make(chan string)
    s.observers = append(s.observers, ch)
    return ch
}

// Notify sends the data to all registered subscribers.
func (s *Subject) Notify(data string) {
    for _, ch := range s.observers {
        // Send the event to each subscriber channel
        ch <- data
    }
}
```

## 3. Factory Pattern
Provides an interface for creating objects, allowing the program to decide which class to instantiate at runtime.

### Real-World Use Case: Logging Environments
You might want to log to the **Console** during local development but to a **Cloud Provider** (like AWS CloudWatch) in production. A Factory decides which logger to provide based on the environment variable.

### Example
```go
type Logger interface {
    Log(msg string)
}

// LoggerFactory returns a specific implementation of Logger based on the environment.
func LoggerFactory(env string) Logger {
    if env == "prod" {
        // Returns a logger optimized for cloud environments
        return &CloudLogger{}
    }
    // Returns a simple console logger for development
    return &ConsoleLogger{}
}
```

## 4. Adapter Pattern
Allows incompatible interfaces to work together. Useful when you want to use a third-party library that doesn't match your internal interface.

### Real-World Use Case: Third-Party API Integration
Your system expects a `Process()` method, but a legacy library you MUST use only provides a `Run()` method. The Adapter "wraps" the legacy library to make it compatible.

### Example
```go
type InternalAPI interface {
    Process()
}

type LegacyAPI struct{} // The "Incompatible" library that only has Run()
func (l *LegacyAPI) Run() { /* ... */ }

// Adapter "adapts" LegacyAPI to match the InternalAPI interface.
type Adapter struct {
    legacy *LegacyAPI
}

func (a *Adapter) Process() {
    // The adapter maps the internal call 'Process()' to the legacy call 'Run()'
    a.legacy.Run()
}
```

## 5. Singleton (using sync.Once)
Ensures a piece of code (like initializing a config) only runs once globally, providing a single point of access.

### Real-World Use Case: Global Configuration or Database Pool
You want to ensure your application only creates **one** database connection pool, no matter how many times it's called.

### Example
```go
var (
    instance *Database
    once     sync.Once
)

// GetDB is a thread-safe way to retrieve the single database instance.
func GetDB() *Database {
    // once.Do guarantees that the inner function runs EXACTLY once,
    // even if 1000 goroutines call GetDB() simultaneously.
    once.Do(func() {
        fmt.Println("Initializing Database Pool...")
        instance = initDB() 
    })
    return instance
}
```
