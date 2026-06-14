package concurrency

import (
	"fmt"
	"sync"
)

type SafeCounter struct {
	mu    sync.Mutex
	count int
}

func (c *SafeCounter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
}

func (c *SafeCounter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

// Cache demonstrates RWMutex — multiple readers, exclusive writer
type Cache struct {
	mu   sync.RWMutex
	data map[string]string
}

func (c *Cache) Set(k, v string) {
	c.mu.Lock() // exclusive write lock
	defer c.mu.Unlock()
	c.data[k] = v
}

func (c *Cache) Get(k string) (string, bool) {
	c.mu.RLock() // shared read lock — multiple goroutines can read simultaneously
	defer c.mu.RUnlock()
	v, ok := c.data[k]
	return v, ok
}

func mutexExample() {
	fmt.Println("Mutex (sync.Mutex):")

	counter := &SafeCounter{}
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Inc()
		}()
	}
	wg.Wait()
	fmt.Println("  100 concurrent increments, final count:", counter.Value())

	fmt.Println("\n  RWMutex — concurrent reads, exclusive writes:")
	cache := &Cache{data: make(map[string]string)}
	cache.Set("key", "value")
	v, ok := cache.Get("key")
	fmt.Printf("  cache.Get(\"key\") = %q, found=%v\n", v, ok)
}
