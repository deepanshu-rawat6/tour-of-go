package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tour_of_go/projects/idempotent-payments/internal/domain"
)

type LedgerRepo struct {
	pool *pgxpool.Pool
}

func NewLedgerRepo(pool *pgxpool.Pool) *LedgerRepo {
	return &LedgerRepo{pool: pool}
}

func (r *LedgerRepo) GetAccount(ctx context.Context, id int64) (*domain.Account, error) {
	var a domain.Account
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, balance FROM accounts WHERE id = $1`, id,
	).Scan(&a.ID, &a.Name, &a.Balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("account %d not found", id)
	}
	return &a, err
}

func (r *LedgerRepo) TransferTx(ctx context.Context, fromID, toID int64, amount float64, idempotencyKey string) (*domain.Payment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Lock sender row to prevent concurrent overdrafts.
	var balance float64
	err = tx.QueryRow(ctx,
		`SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`, fromID,
	).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("account %d not found", fromID)
	}
	if err != nil {
		return nil, err
	}

	if balance < amount {
		return nil, fmt.Errorf("insufficient funds: have %.4f, need %.4f", balance, amount)
	}

	if _, err = tx.Exec(ctx,
		`UPDATE accounts SET balance = balance - $1 WHERE id = $2`, amount, fromID,
	); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx,
		`UPDATE accounts SET balance = balance + $1 WHERE id = $2`, amount, toID,
	); err != nil {
		return nil, err
	}

	p := &domain.Payment{
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Amount:         amount,
		IdempotencyKey: idempotencyKey,
		Status:         domain.StatusCompleted,
		CreatedAt:      time.Now(),
	}
	err = tx.QueryRow(ctx,
		`INSERT INTO payments (from_account_id, to_account_id, amount, idempotency_key, status)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		fromID, toID, amount, idempotencyKey, p.Status,
	).Scan(&p.ID, &p.CreatedAt)
	if err != nil {
		return nil, err
	}

	return p, tx.Commit(ctx)
}
