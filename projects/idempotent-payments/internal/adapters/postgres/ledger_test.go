//go:build integration

package postgres_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"tour_of_go/projects/idempotent-payments/internal/adapters/postgres"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		dsn = "postgres://payments:payments@localhost:5432/payments"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM payments`)
		pool.Exec(context.Background(), `UPDATE accounts SET balance = CASE id WHEN 1 THEN 1000 WHEN 2 THEN 500 END WHERE id IN (1,2)`)
		pool.Close()
	})
	return pool
}

func TestTransferTx_Success(t *testing.T) {
	pool := testPool(t)
	repo := postgres.NewLedgerRepo(pool)
	ctx := context.Background()

	p, err := repo.TransferTx(ctx, 1, 2, 100.0, "test-key-success")
	if err != nil {
		t.Fatal(err)
	}
	if p.ID == "" {
		t.Fatal("expected payment ID")
	}

	alice, _ := repo.GetAccount(ctx, 1)
	bob, _ := repo.GetAccount(ctx, 2)
	if alice.Balance != 900.0 {
		t.Fatalf("alice: want 900, got %.4f", alice.Balance)
	}
	if bob.Balance != 600.0 {
		t.Fatalf("bob: want 600, got %.4f", bob.Balance)
	}
}

func TestTransferTx_InsufficientFunds(t *testing.T) {
	pool := testPool(t)
	repo := postgres.NewLedgerRepo(pool)
	ctx := context.Background()

	_, err := repo.TransferTx(ctx, 2, 1, 9999.0, "test-key-insuf")
	if err == nil {
		t.Fatal("expected insufficient funds error")
	}

	// Balances must be unchanged.
	bob, _ := repo.GetAccount(ctx, 2)
	if bob.Balance != 500.0 {
		t.Fatalf("bob balance should be unchanged, got %.4f", bob.Balance)
	}
}

func TestTransferTx_ConcurrentNoDrain(t *testing.T) {
	pool := testPool(t)
	repo := postgres.NewLedgerRepo(pool)
	ctx := context.Background()

	// Alice has 1000. Two goroutines each try to transfer 600 — only one should succeed.
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = repo.TransferTx(ctx, 1, 2, 600.0, fmt.Sprintf("concurrent-key-%d", idx))
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, e := range errs {
		if e == nil {
			successCount++
		}
	}
	if successCount != 1 {
		t.Fatalf("expected exactly 1 success, got %d", successCount)
	}

	alice, _ := repo.GetAccount(ctx, 1)
	if alice.Balance != 400.0 {
		t.Fatalf("alice: want 400, got %.4f", alice.Balance)
	}
}
