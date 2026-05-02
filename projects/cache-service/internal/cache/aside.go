package cache

import (
	"context"
	"time"

	"tour_of_go/projects/cache-service/internal/store"
)

// CacheAside implements the cache-aside (lazy loading) pattern:
// read from cache → on miss, fetch from store → populate cache.
type CacheAside struct {
	lru   *LRU
	store store.Store
	ttl   time.Duration
}

func NewCacheAside(lru *LRU, s store.Store, ttl time.Duration) *CacheAside {
	return &CacheAside{lru: lru, store: s, ttl: ttl}
}

func (c *CacheAside) Get(ctx context.Context, key string) (string, error) {
	if v, ok := c.lru.Get(key); ok {
		return v.(string), nil
	}
	// Cache miss — fetch from backing store
	val, err := c.store.Get(ctx, key)
	if err != nil {
		return "", err
	}
	c.lru.Set(key, val, c.ttl)
	return val, nil
}

func (c *CacheAside) Set(ctx context.Context, key, val string) error {
	// Cache-aside: caller writes directly to store; cache is populated on next read.
	return c.store.Set(ctx, key, val)
}

func (c *CacheAside) Delete(ctx context.Context, key string) error {
	c.lru.Delete(key)
	return c.store.Delete(ctx, key)
}
