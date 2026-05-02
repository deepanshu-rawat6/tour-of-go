package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"tour_of_go/projects/cache-service/internal/store"
)

// Redis implements the store.Store interface backed by Redis.
type Redis struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedis(addr string, ttl time.Duration) *Redis {
	return &Redis{
		client: redis.NewClient(&redis.Options{Addr: addr}),
		ttl:    ttl,
	}
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", store.ErrNotFound
	}
	return val, err
}

func (r *Redis) Set(ctx context.Context, key, val string) error {
	return r.client.Set(ctx, key, val, r.ttl).Err()
}

func (r *Redis) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *Redis) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
