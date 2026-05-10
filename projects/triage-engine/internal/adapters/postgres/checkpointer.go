package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tour_of_go/projects/triage-engine/internal/domain"
)

type Checkpointer struct {
	pool *pgxpool.Pool
}

func NewCheckpointer(pool *pgxpool.Pool) *Checkpointer {
	return &Checkpointer{pool: pool}
}

func (c *Checkpointer) Save(ctx context.Context, state *domain.InvestigationState) error {
	state.UpdatedAt = time.Now()
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = c.pool.Exec(ctx,
		`INSERT INTO investigation_states (ticket_id, state, updated_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (ticket_id) DO UPDATE SET state = $2, updated_at = NOW()`,
		state.TicketID, data,
	)
	return err
}

func (c *Checkpointer) Load(ctx context.Context, ticketID string) (*domain.InvestigationState, error) {
	var data []byte
	err := c.pool.QueryRow(ctx,
		`SELECT state FROM investigation_states WHERE ticket_id = $1`, ticketID,
	).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var state domain.InvestigationState
	return &state, json.Unmarshal(data, &state)
}
