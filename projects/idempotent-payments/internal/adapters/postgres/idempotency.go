package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tour_of_go/projects/idempotent-payments/internal/domain"
)

type IdempotencyRepo struct {
	pool *pgxpool.Pool
}

func NewIdempotencyRepo(pool *pgxpool.Pool) *IdempotencyRepo {
	return &IdempotencyRepo{pool: pool}
}

func (r *IdempotencyRepo) Get(ctx context.Context, key string) (*domain.IdempotencyRecord, error) {
	var rec domain.IdempotencyRecord
	var headersJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT key, status_code, headers, body, created_at, expires_at
		 FROM idempotency_keys WHERE key = $1 AND expires_at > NOW()`, key,
	).Scan(&rec.Key, &rec.StatusCode, &headersJSON, &rec.Body, &rec.CreatedAt, &rec.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(headersJSON, &rec.Headers); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *IdempotencyRepo) Save(ctx context.Context, rec *domain.IdempotencyRecord) error {
	headersJSON, err := json.Marshal(rec.Headers)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO idempotency_keys (key, status_code, headers, body, expires_at)
		 VALUES ($1, $2, $3, $4, $5) ON CONFLICT (key) DO NOTHING`,
		rec.Key, rec.StatusCode, headersJSON, rec.Body, rec.ExpiresAt,
	)
	return err
}

func (r *IdempotencyRepo) DeleteExpired(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM idempotency_keys WHERE expires_at <= NOW()`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
