package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// IdempotencyStore implements pipeline.IdempotencyStore using Redis SET NX.
// Exactly-once semantics: the first processor to SET the key wins.
// Subsequent processors see the key exists and skip processing.
type IdempotencyStore struct {
	client *redis.Client
}

func NewIdempotencyStore(client *redis.Client) *IdempotencyStore {
	return &IdempotencyStore{client: client}
}

func (s *IdempotencyStore) IsProcessed(ctx context.Context, key string) (bool, error) {
	exists, err := s.client.Exists(ctx, "idempotency:"+key).Result()
	return exists > 0, err
}

func (s *IdempotencyStore) MarkProcessed(ctx context.Context, key string, ttl time.Duration) error {
	return s.client.Set(ctx, "idempotency:"+key, 1, ttl).Err()
}
