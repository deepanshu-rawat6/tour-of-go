// Package store defines the backing store port interface.
package store

import "context"

// Store is the port for a persistent/remote backing store.
type Store interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, val string) error
	Delete(ctx context.Context, key string) error
}
