package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ── Functional Options ────────────────────────────────────────────────────────

type Server struct {
	addr    string
	port    int
	timeout time.Duration
}

type Option func(*Server)

func WithAddr(addr string) Option    { return func(s *Server) { s.addr = addr } }
func WithPort(port int) Option       { return func(s *Server) { s.port = port } }
func WithTimeout(d time.Duration) Option { return func(s *Server) { s.timeout = d } }

func NewServer(opts ...Option) *Server {
	s := &Server{addr: "localhost", port: 8080, timeout: 30 * time.Second}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ── Circuit Breaker ───────────────────────────────────────────────────────────

type State int

const (
	Closed   State = iota // normal
	Open                  // failing — block requests
	HalfOpen              // testing recovery
)

type CircuitBreaker struct {
	mu          sync.Mutex
	state       State
	failures    int
	threshold   int
	lastFailure time.Time
	retryAfter  time.Duration
}

func NewCircuitBreaker(threshold int, retryAfter time.Duration) *CircuitBreaker {
	return &CircuitBreaker{threshold: threshold, retryAfter: retryAfter}
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	if cb.state == Open && time.Since(cb.lastFailure) > cb.retryAfter {
		cb.state = HalfOpen
	}
	if cb.state == Open {
		cb.mu.Unlock()
		return errors.New("circuit open")
	}
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.threshold {
			cb.state = Open
		}
		return err
	}
	cb.failures = 0
	cb.state = Closed
	return nil
}

// ── Single-Flight ─────────────────────────────────────────────────────────────
// Ensures only one in-flight call per key — prevents cache stampede.

type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := &call{}
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	return c.val, c.err
}

func main() {
	fmt.Println("=== Functional Options ===")
	s1 := NewServer()
	fmt.Printf("default:  addr=%s port=%d timeout=%s\n", s1.addr, s1.port, s1.timeout)

	s2 := NewServer(WithAddr("0.0.0.0"), WithPort(9090), WithTimeout(5*time.Second))
	fmt.Printf("custom:   addr=%s port=%d timeout=%s\n", s2.addr, s2.port, s2.timeout)

	fmt.Println("\n=== Circuit Breaker ===")
	cb := NewCircuitBreaker(3, 50*time.Millisecond)
	callCount := 0
	failingFn := func() error {
		callCount++
		return errors.New("downstream error")
	}

	for i := 1; i <= 5; i++ {
		err := cb.Execute(failingFn)
		fmt.Printf("  call %d: %v\n", i, err)
	}
	fmt.Printf("  actual downstream calls made: %d (circuit blocked 2)\n", callCount)

	fmt.Println("\n=== Single-Flight ===")
	var g Group
	var wg sync.WaitGroup
	dbCalls := 0

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			val, _ := g.Do("user:42", func() (interface{}, error) {
				dbCalls++
				time.Sleep(10 * time.Millisecond) // simulate slow DB
				return "user data", nil
			})
			fmt.Printf("  goroutine %d got: %v\n", id, val)
		}(i)
	}
	wg.Wait()
	fmt.Printf("  DB was called %d time(s) for 5 concurrent requests\n", dbCalls)
}
