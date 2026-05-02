// Package cache provides an in-memory LRU cache with per-entry TTL eviction.
package cache

import (
	"container/list"
	"sync"
	"time"
)

type entry struct {
	key       string
	value     any
	expiresAt time.Time
}

// LRU is a thread-safe, capacity-bounded LRU cache with per-entry TTL.
// Eviction policy: least-recently-used when capacity is exceeded, or TTL expiry.
type LRU struct {
	mu       sync.Mutex
	cap      int
	items    map[string]*list.Element
	order    *list.List // front = most recently used
	stopOnce sync.Once
	stop     chan struct{}
}

// NewLRU creates an LRU cache with the given capacity and starts a background
// reaper that removes expired entries every second.
func NewLRU(capacity int) *LRU {
	c := &LRU{
		cap:   capacity,
		items: make(map[string]*list.Element, capacity),
		order: list.New(),
		stop:  make(chan struct{}),
	}
	go c.reap()
	return c
}

// Set inserts or updates a key. A zero ttl means the entry never expires.
func (c *LRU) Set(key string, val any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	exp := time.Time{} // zero = no expiry
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}

	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		el.Value.(*entry).value = val
		el.Value.(*entry).expiresAt = exp
		return
	}

	// Evict LRU entry if at capacity
	if c.order.Len() >= c.cap {
		c.evictOldest()
	}

	el := c.order.PushFront(&entry{key: key, value: val, expiresAt: exp})
	c.items[key] = el
}

// Get retrieves a value. Returns (nil, false) on miss or expiry.
func (c *LRU) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	e := el.Value.(*entry)
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		c.remove(el)
		return nil, false
	}
	c.order.MoveToFront(el)
	return e.value, true
}

// Delete removes a key from the cache.
func (c *LRU) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.remove(el)
	}
}

// Len returns the number of entries currently in the cache.
func (c *LRU) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

// Close stops the background reaper goroutine.
func (c *LRU) Close() {
	c.stopOnce.Do(func() { close(c.stop) })
}

// evictOldest removes the least-recently-used entry. Caller must hold mu.
func (c *LRU) evictOldest() {
	if el := c.order.Back(); el != nil {
		c.remove(el)
	}
}

// remove deletes an element from both the list and the map. Caller must hold mu.
func (c *LRU) remove(el *list.Element) {
	c.order.Remove(el)
	delete(c.items, el.Value.(*entry).key)
}

// reap runs every second and removes expired entries.
func (c *LRU) reap() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for el := c.order.Back(); el != nil; {
				prev := el.Prev()
				e := el.Value.(*entry)
				if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
					c.remove(el)
				}
				el = prev
			}
			c.mu.Unlock()
		case <-c.stop:
			return
		}
	}
}
