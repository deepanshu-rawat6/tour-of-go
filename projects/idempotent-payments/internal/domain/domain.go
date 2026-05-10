package domain

import "time"

const (
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

type Account struct {
	ID      int64
	Name    string
	Balance float64
}

type Payment struct {
	ID             string
	FromAccountID  int64
	ToAccountID    int64
	Amount         float64
	IdempotencyKey string
	Status         string
	CreatedAt      time.Time
}

type IdempotencyRecord struct {
	Key        string
	StatusCode int
	Headers    map[string]string
	Body       []byte
	CreatedAt  time.Time
	ExpiresAt  time.Time
}
