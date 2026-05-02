package cache

import (
	"context"
	"time"

	"tour_of_go/projects/cache-service/internal/store"
)

// WriteThrough writes to both the cache and the backing store atomically.
// Reads always hit the cache (no miss on recently written keys).
type WriteThrough struct {
	lru   *LRU
	store store.Store
	ttl   time.Duration
}

func NewWriteThrough(lru *LRU, s store.Store, ttl time.Duration) *WriteThrough {
	return &WriteThrough{lru: lru, store: s, ttl: ttl}
}

func (c *WriteThrough) Get(_ context.Context, key string) (string, error) {
	if v, ok := c.lru.Get(key); ok {
		return v.(string), nil
	}
	return "", store.ErrNotFound
}

// Set writes to the store first, then updates the cache.
func (c *WriteThrough) Set(ctx context.Context, key, val string) error {
	if err := c.store.Set(ctx, key, val); err != nil {
		return err
	}
	c.lru.Set(key, val, c.ttl)
	return nil
}

func (c *WriteThrough) Delete(ctx context.Context, key string) error {
	c.lru.Delete(key)
	return c.store.Delete(ctx, key)
}
