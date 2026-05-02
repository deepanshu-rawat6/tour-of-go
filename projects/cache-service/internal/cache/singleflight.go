package cache

import (
	"context"

	"golang.org/x/sync/singleflight"
)

// Getter is the minimal interface needed by SingleflightCache.
type Getter interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, val string) error
	Delete(ctx context.Context, key string) error
}

// SingleflightCache wraps a Getter and ensures that concurrent cache misses
// for the same key result in only one backing-store fetch (prevents stampede).
type SingleflightCache struct {
	inner Getter
	group singleflight.Group
}

func NewSingleflightCache(inner Getter) *SingleflightCache {
	return &SingleflightCache{inner: inner}
}

func (s *SingleflightCache) Get(ctx context.Context, key string) (string, error) {
	v, err, _ := s.group.Do(key, func() (any, error) {
		return s.inner.Get(ctx, key)
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func (s *SingleflightCache) Set(ctx context.Context, key, val string) error {
	return s.inner.Set(ctx, key, val)
}

func (s *SingleflightCache) Delete(ctx context.Context, key string) error {
	return s.inner.Delete(ctx, key)
}
