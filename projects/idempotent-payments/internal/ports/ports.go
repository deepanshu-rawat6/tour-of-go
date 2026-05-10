package ports

import (
	"context"

	"tour_of_go/projects/idempotent-payments/internal/domain"
)

// LedgerDB is the outbound port for account and payment persistence.
type LedgerDB interface {
	GetAccount(ctx context.Context, id int64) (*domain.Account, error)
	TransferTx(ctx context.Context, fromID, toID int64, amount float64, idempotencyKey string) (*domain.Payment, error)
}

// IdempotencyStore is the outbound port for idempotency key caching.
type IdempotencyStore interface {
	Get(ctx context.Context, key string) (*domain.IdempotencyRecord, error)
	Save(ctx context.Context, record *domain.IdempotencyRecord) error
	DeleteExpired(ctx context.Context) (int64, error)
}
