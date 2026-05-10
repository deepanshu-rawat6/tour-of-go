package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tour_of_go/projects/idempotent-payments/internal/domain"
	"tour_of_go/projects/idempotent-payments/internal/handler"
	"tour_of_go/projects/idempotent-payments/internal/service"
)

// mockLedgerDB for constructing a LedgerService in tests.
type mockLedgerDB struct {
	err error
}

func (m *mockLedgerDB) GetAccount(_ context.Context, _ int64) (*domain.Account, error) {
	return nil, nil
}

func (m *mockLedgerDB) TransferTx(_ context.Context, _, _ int64, amount float64, key string) (*domain.Payment, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &domain.Payment{
		ID: "uuid-1", FromAccountID: 1, ToAccountID: 2,
		Amount: amount, IdempotencyKey: key,
		Status: domain.StatusCompleted, CreatedAt: time.Now(),
	}, nil
}

func newRequest(t *testing.T, body map[string]any) *http.Request {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-key")
	return req
}

func TestPayment_Success(t *testing.T) {
	svc := service.NewLedgerService(&mockLedgerDB{})
	h := handler.Payment(svc)

	req := newRequest(t, map[string]any{"from_account_id": 1, "to_account_id": 2, "amount": 50.0})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp handler.PaymentResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Status != domain.StatusCompleted {
		t.Fatalf("want completed, got %s", resp.Status)
	}
}

func TestPayment_InvalidBody(t *testing.T) {
	svc := service.NewLedgerService(&mockLedgerDB{})
	h := handler.Payment(svc)

	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader([]byte(`not-json`)))
	req.Header.Set("Idempotency-Key", "test-key")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestPayment_ZeroAmount(t *testing.T) {
	svc := service.NewLedgerService(&mockLedgerDB{})
	h := handler.Payment(svc)

	req := newRequest(t, map[string]any{"from_account_id": 1, "to_account_id": 2, "amount": 0})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestPayment_InsufficientFunds(t *testing.T) {
	svc := service.NewLedgerService(&mockLedgerDB{err: errors.New("insufficient funds: have 10, need 500")})
	h := handler.Payment(svc)

	req := newRequest(t, map[string]any{"from_account_id": 1, "to_account_id": 2, "amount": 500.0})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("want 402, got %d", rr.Code)
	}
}
