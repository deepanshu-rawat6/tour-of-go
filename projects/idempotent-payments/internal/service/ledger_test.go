package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"tour_of_go/projects/idempotent-payments/internal/domain"
	"tour_of_go/projects/idempotent-payments/internal/service"
)

type mockDB struct {
	called bool
	err    error
}

func (m *mockDB) GetAccount(_ context.Context, _ int64) (*domain.Account, error) {
	return nil, nil
}

func (m *mockDB) TransferTx(_ context.Context, _, _ int64, _ float64, key string) (*domain.Payment, error) {
	m.called = true
	if m.err != nil {
		return nil, m.err
	}
	return &domain.Payment{
		ID: "uuid-1", FromAccountID: 1, ToAccountID: 2,
		Amount: 50.0, IdempotencyKey: key,
		Status: domain.StatusCompleted, CreatedAt: time.Now(),
	}, nil
}

func TestProcessPayment_HappyPath(t *testing.T) {
	db := &mockDB{}
	svc := service.NewLedgerService(db)
	p, err := svc.ProcessPayment(context.Background(), 1, 2, 50.0, "key-1")
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != domain.StatusCompleted {
		t.Fatalf("want completed, got %s", p.Status)
	}
	if !db.called {
		t.Fatal("expected TransferTx to be called")
	}
}

func TestProcessPayment_InvalidAmount(t *testing.T) {
	db := &mockDB{}
	svc := service.NewLedgerService(db)
	for _, amt := range []float64{0, -1, -100} {
		_, err := svc.ProcessPayment(context.Background(), 1, 2, amt, "key-1")
		if err == nil {
			t.Fatalf("expected error for amount %v", amt)
		}
		if db.called {
			t.Fatal("TransferTx must not be called on invalid input")
		}
	}
}

func TestProcessPayment_SameAccount(t *testing.T) {
	db := &mockDB{}
	svc := service.NewLedgerService(db)
	_, err := svc.ProcessPayment(context.Background(), 1, 1, 50.0, "key-1")
	if err == nil {
		t.Fatal("expected error for same from/to account")
	}
	if db.called {
		t.Fatal("TransferTx must not be called on invalid input")
	}
}

func TestProcessPayment_DBError(t *testing.T) {
	db := &mockDB{err: errors.New("insufficient funds")}
	svc := service.NewLedgerService(db)
	_, err := svc.ProcessPayment(context.Background(), 1, 2, 50.0, "key-1")
	if err == nil {
		t.Fatal("expected error from DB")
	}
}
