package ports_test

import (
	"context"

	"tour_of_go/projects/idempotent-payments/internal/domain"
	"tour_of_go/projects/idempotent-payments/internal/ports"
)

// Compile-time interface satisfaction checks.
var _ ports.LedgerDB = (*mockLedgerDB)(nil)
var _ ports.IdempotencyStore = (*mockIdempotencyStore)(nil)

type mockLedgerDB struct{}

func (m *mockLedgerDB) GetAccount(_ context.Context, _ int64) (*domain.Account, error) {
	return nil, nil
}
func (m *mockLedgerDB) TransferTx(_ context.Context, _, _ int64, _ float64, _ string) (*domain.Payment, error) {
	return nil, nil
}

type mockIdempotencyStore struct{}

func (m *mockIdempotencyStore) Get(_ context.Context, _ string) (*domain.IdempotencyRecord, error) {
	return nil, nil
}
func (m *mockIdempotencyStore) Save(_ context.Context, _ *domain.IdempotencyRecord) error {
	return nil
}
func (m *mockIdempotencyStore) DeleteExpired(_ context.Context) (int64, error) { return 0, nil }
