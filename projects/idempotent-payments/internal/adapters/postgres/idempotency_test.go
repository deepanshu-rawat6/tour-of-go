//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"tour_of_go/projects/idempotent-payments/internal/adapters/postgres"
	"tour_of_go/projects/idempotent-payments/internal/domain"
)

func TestIdempotencyRepo_SaveAndGet(t *testing.T) {
	pool := testPool(t)
	repo := postgres.NewIdempotencyRepo(pool)
	ctx := context.Background()

	rec := &domain.IdempotencyRecord{
		Key:        "idem-key-1",
		StatusCode: 201,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       []byte(`{"id":"uuid-1"}`),
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}

	if err := repo.Save(ctx, rec); err != nil {
		t.Fatal(err)
	}

	got, err := repo.Get(ctx, "idem-key-1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected record, got nil")
	}
	if got.StatusCode != 201 {
		t.Fatalf("want 201, got %d", got.StatusCode)
	}
	if string(got.Body) != `{"id":"uuid-1"}` {
		t.Fatalf("body mismatch: %s", got.Body)
	}
}

func TestIdempotencyRepo_ExpiredNotReturned(t *testing.T) {
	pool := testPool(t)
	repo := postgres.NewIdempotencyRepo(pool)
	ctx := context.Background()

	rec := &domain.IdempotencyRecord{
		Key:        "idem-key-expired",
		StatusCode: 200,
		Headers:    map[string]string{},
		Body:       []byte(`{}`),
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		ExpiresAt:  time.Now().Add(-1 * time.Hour), // already expired
	}
	if err := repo.Save(ctx, rec); err != nil {
		t.Fatal(err)
	}

	got, err := repo.Get(ctx, "idem-key-expired")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected nil for expired record")
	}
}

func TestIdempotencyRepo_DuplicateSaveNoError(t *testing.T) {
	pool := testPool(t)
	repo := postgres.NewIdempotencyRepo(pool)
	ctx := context.Background()

	rec := &domain.IdempotencyRecord{
		Key:        "idem-key-dup",
		StatusCode: 201,
		Headers:    map[string]string{},
		Body:       []byte(`{"id":"uuid-2"}`),
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	if err := repo.Save(ctx, rec); err != nil {
		t.Fatal(err)
	}
	// Second save must not error (ON CONFLICT DO NOTHING).
	if err := repo.Save(ctx, rec); err != nil {
		t.Fatalf("duplicate save should not error: %v", err)
	}
}

func TestIdempotencyRepo_DeleteExpired(t *testing.T) {
	pool := testPool(t)
	repo := postgres.NewIdempotencyRepo(pool)
	ctx := context.Background()

	// Insert one expired record directly.
	rec := &domain.IdempotencyRecord{
		Key:        "idem-key-del",
		StatusCode: 200,
		Headers:    map[string]string{},
		Body:       []byte(`{}`),
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		ExpiresAt:  time.Now().Add(-1 * time.Hour),
	}
	if err := repo.Save(ctx, rec); err != nil {
		t.Fatal(err)
	}

	n, err := repo.DeleteExpired(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("expected at least 1 deleted, got %d", n)
	}
}
