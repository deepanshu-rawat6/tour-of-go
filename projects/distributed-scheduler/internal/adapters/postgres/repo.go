// Package postgres implements the JobRepository and JobConfigRepository ports
// using PostgreSQL via sqlx.
//
// Design decision (see docs/adr/004-sqlx-over-orm.md):
// We use raw SQL + sqlx instead of GORM because:
//   - SQL is explicit — no magic, no N+1 surprises
//   - sqlx adds struct scanning without hiding the query
//   - Easier to add DB-specific optimizations (e.g., batch upserts)
//   - The Java codebase used JPA which hid complexity; we want visibility
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"tour_of_go/projects/distributed-scheduler/internal/domain"
)

// JobRepo implements ports.JobRepository.
type JobRepo struct {
	db *sqlx.DB
}

func NewJobRepo(db *sqlx.DB) *JobRepo {
	return &JobRepo{db: db}
}

// jobRow is the DB-level representation. JSON fields are stored as []byte
// and unmarshalled into Go maps after scanning.
type jobRow struct {
	ID                 int64     `db:"id"`
	JobName            string    `db:"job_name"`
	Tenant             int       `db:"tenant"`
	Priority           int       `db:"priority"`
	Status             string    `db:"status"`
	Payload            []byte    `db:"payload"`
	ConcurrencyControl []byte    `db:"concurrency_control"`
	ExecutionCount     int       `db:"execution_count"`
	LastFailureReason  string    `db:"last_failure_reason"`
	SubmitTime         time.Time `db:"submit_time"`
	LastUpdated        time.Time `db:"last_updated"`
}

func (r jobRow) toDomain() (*domain.Job, error) {
	var payload map[string]any
	if err := json.Unmarshal(r.Payload, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	var cc map[string]string
	if err := json.Unmarshal(r.ConcurrencyControl, &cc); err != nil {
		return nil, fmt.Errorf("unmarshal concurrency_control: %w", err)
	}
	return &domain.Job{
		ID:                 r.ID,
		JobName:            r.JobName,
		Tenant:             r.Tenant,
		Priority:           r.Priority,
		Status:             domain.JobStatus(r.Status),
		Payload:            payload,
		ConcurrencyControl: cc,
		ExecutionCount:     r.ExecutionCount,
		LastFailureReason:  r.LastFailureReason,
		SubmitTime:         r.SubmitTime,
		LastUpdated:        r.LastUpdated,
	}, nil
}

func (r *JobRepo) Save(ctx context.Context, job *domain.Job) error {
	payload, _ := json.Marshal(job.Payload)
	cc, _ := json.Marshal(job.ConcurrencyControl)

	if job.ID == 0 {
		return r.db.QueryRowContext(ctx, `
			INSERT INTO jobs (job_name, tenant, priority, status, payload, concurrency_control)
			VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
			job.JobName, job.Tenant, job.Priority, string(job.Status), payload, cc,
		).Scan(&job.ID)
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE jobs SET status=$1, execution_count=$2, last_failure_reason=$3, last_updated=NOW()
		WHERE id=$4`,
		string(job.Status), job.ExecutionCount, job.LastFailureReason, job.ID,
	)
	return err
}

func (r *JobRepo) FindByID(ctx context.Context, id int64) (*domain.Job, error) {
	var row jobRow
	err := r.db.GetContext(ctx, &row, `SELECT * FROM jobs WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	return row.toDomain()
}

func (r *JobRepo) FindByIDs(ctx context.Context, ids []int64) ([]*domain.Job, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query, args, err := sqlx.In(`SELECT * FROM jobs WHERE id IN (?)`, ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var rows []jobRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	jobs := make([]*domain.Job, 0, len(rows))
	for _, row := range rows {
		j, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (r *JobRepo) FindSchedulable(ctx context.Context, jobName string, limit int) ([]*domain.Job, error) {
	var rows []jobRow
	err := r.db.SelectContext(ctx, &rows, `
		SELECT * FROM jobs
		WHERE job_name=$1 AND status IN ('WAITING','RETRY')
		ORDER BY priority ASC, submit_time ASC
		LIMIT $2`, jobName, limit)
	if err != nil {
		return nil, err
	}
	jobs := make([]*domain.Job, 0, len(rows))
	for _, row := range rows {
		j, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (r *JobRepo) GetActiveProjections(ctx context.Context) ([]*domain.JobProjection, error) {
	type projRow struct {
		ID                 int64  `db:"id"`
		JobName            string `db:"job_name"`
		Tenant             int    `db:"tenant"`
		Priority           int    `db:"priority"`
		Status             string `db:"status"`
		ConcurrencyControl []byte `db:"concurrency_control"`
	}
	var rows []projRow
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, job_name, tenant, priority, status, concurrency_control
		FROM jobs WHERE status IN ('PUBLISHED','PROCESSING')`)
	if err != nil {
		return nil, err
	}
	projs := make([]*domain.JobProjection, 0, len(rows))
	for _, row := range rows {
		var cc map[string]string
		json.Unmarshal(row.ConcurrencyControl, &cc)
		projs = append(projs, &domain.JobProjection{
			ID:                 row.ID,
			JobName:            row.JobName,
			Tenant:             row.Tenant,
			Priority:           row.Priority,
			Status:             domain.JobStatus(row.Status),
			ConcurrencyControl: cc,
		})
	}
	return projs, nil
}

func (r *JobRepo) MarkPublished(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids)+1)
	args[0] = "PUBLISHED"
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}
	_, err := r.db.ExecContext(ctx,
		fmt.Sprintf(`UPDATE jobs SET status=$1, last_updated=NOW() WHERE id IN (%s)`,
			strings.Join(placeholders, ",")),
		args...)
	return err
}

func (r *JobRepo) MarkFailed(ctx context.Context, id int64, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE jobs SET status='FAILED_BY_JFC', last_failure_reason=$1, last_updated=NOW() WHERE id=$2`,
		reason, id)
	return err
}

func (r *JobRepo) UpdateStatus(ctx context.Context, id int64, status domain.JobStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE jobs SET status=$1, last_updated=NOW() WHERE id=$2`, string(status), id)
	return err
}

func (r *JobRepo) GetStuckPublished(ctx context.Context, olderThan time.Duration) ([]*domain.Job, error) {
	threshold := time.Now().Add(-olderThan)
	var rows []jobRow
	err := r.db.SelectContext(ctx, &rows,
		`SELECT * FROM jobs WHERE status='PUBLISHED' AND last_updated < $1`, threshold)
	if err != nil {
		return nil, err
	}
	jobs := make([]*domain.Job, 0, len(rows))
	for _, row := range rows {
		j, _ := row.toDomain()
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (r *JobRepo) GetProcessingJobs(ctx context.Context) ([]*domain.Job, error) {
	var rows []jobRow
	err := r.db.SelectContext(ctx, &rows, `SELECT * FROM jobs WHERE status='PROCESSING'`)
	if err != nil {
		return nil, err
	}
	jobs := make([]*domain.Job, 0, len(rows))
	for _, row := range rows {
		j, _ := row.toDomain()
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// ConfigRepo implements ports.JobConfigRepository.
type ConfigRepo struct {
	db *sqlx.DB
}

func NewConfigRepo(db *sqlx.DB) *ConfigRepo {
	return &ConfigRepo{db: db}
}

type configRow struct {
	JobName          string `db:"job_name"`
	ConcurrencyRules []byte `db:"concurrency_rules"`
	DestinationType  string `db:"destination_type"`
	DestinationConfig []byte `db:"destination_config"`
}

func (r *ConfigRepo) GetByJobName(ctx context.Context, jobName string) (*domain.JobConfig, error) {
	var row configRow
	err := r.db.GetContext(ctx, &row, `SELECT * FROM job_configs WHERE job_name=$1`, jobName)
	if err != nil {
		return nil, err
	}
	return rowToConfig(row)
}

func (r *ConfigRepo) List(ctx context.Context) ([]*domain.JobConfig, error) {
	var rows []configRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT * FROM job_configs`); err != nil {
		return nil, err
	}
	configs := make([]*domain.JobConfig, 0, len(rows))
	for _, row := range rows {
		c, err := rowToConfig(row)
		if err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, nil
}

func rowToConfig(row configRow) (*domain.JobConfig, error) {
	var rules map[string]int
	if err := json.Unmarshal(row.ConcurrencyRules, &rules); err != nil {
		return nil, err
	}
	return &domain.JobConfig{
		JobName:          row.JobName,
		ConcurrencyRules: rules,
		Destination: domain.DestinationInfo{
			Type: domain.DestinationType(row.DestinationType),
		},
	}, nil
}
