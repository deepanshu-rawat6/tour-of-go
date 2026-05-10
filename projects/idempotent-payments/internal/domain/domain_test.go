package domain_test

import (
	"testing"
	"time"

	"tour_of_go/projects/idempotent-payments/internal/domain"
)

func TestStatusConstants(t *testing.T) {
	if domain.StatusCompleted != "completed" {
		t.Fatalf("want completed, got %s", domain.StatusCompleted)
	}
	if domain.StatusFailed != "failed" {
		t.Fatalf("want failed, got %s", domain.StatusFailed)
	}
}

func TestStructInstantiation(t *testing.T) {
	a := domain.Account{ID: 1, Name: "Alice", Balance: 1000.0}
	if a.ID != 1 {
		t.Fatal("account ID mismatch")
	}

	p := domain.Payment{
		ID: "uuid-1", FromAccountID: 1, ToAccountID: 2,
		Amount: 50.0, IdempotencyKey: "key-1",
		Status: domain.StatusCompleted, CreatedAt: time.Now(),
	}
	if p.Status != domain.StatusCompleted {
		t.Fatal("payment status mismatch")
	}

	r := domain.IdempotencyRecord{
		Key: "key-1", StatusCode: 201,
		Headers:   map[string]string{"Content-Type": "application/json"},
		Body:      []byte(`{"id":"uuid-1"}`),
		CreatedAt: time.Now(), ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if r.StatusCode != 201 {
		t.Fatal("record status code mismatch")
	}
}
