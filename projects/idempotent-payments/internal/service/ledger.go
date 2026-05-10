package service

import (
	"context"
	"errors"

	"tour_of_go/projects/idempotent-payments/internal/domain"
	"tour_of_go/projects/idempotent-payments/internal/ports"
)

type LedgerService struct {
	db ports.LedgerDB
}

func NewLedgerService(db ports.LedgerDB) *LedgerService {
	return &LedgerService{db: db}
}

func (s *LedgerService) ProcessPayment(ctx context.Context, fromID, toID int64, amount float64, idempotencyKey string) (*domain.Payment, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	if fromID == toID {
		return nil, errors.New("from and to accounts must differ")
	}
	return s.db.TransferTx(ctx, fromID, toID, amount, idempotencyKey)
}
