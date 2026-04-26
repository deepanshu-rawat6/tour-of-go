package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/tour-of-go/k8s-event-sink/internal/core"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS events (
	id         TEXT PRIMARY KEY,
	namespace  TEXT NOT NULL,
	pod        TEXT NOT NULL,
	reason     TEXT NOT NULL,
	message    TEXT NOT NULL,
	type       TEXT NOT NULL,
	severity   TEXT NOT NULL,
	count      INTEGER NOT NULL DEFAULT 1,
	first_seen DATETIME NOT NULL,
	last_seen  DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_namespace ON events(namespace);
CREATE INDEX IF NOT EXISTS idx_events_severity  ON events(severity);
CREATE INDEX IF NOT EXISTS idx_events_last_seen ON events(last_seen);
`

// Store implements core.StoragePort using SQLite.
type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("creating schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Save(ctx context.Context, e core.Event) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO events
		(id, namespace, pod, reason, message, type, severity, count, first_seen, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Namespace, e.Pod, e.Reason, e.Message,
		e.Type, e.Severity, e.Count,
		e.FirstSeen.UTC().Format(time.RFC3339),
		e.LastSeen.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) Query(ctx context.Context, f core.QueryFilter) ([]core.Event, error) {
	q := `SELECT id, namespace, pod, reason, message, type, severity, count, first_seen, last_seen
	      FROM events WHERE 1=1`
	var args []interface{}

	if f.Namespace != "" {
		q += " AND namespace = ?"
		args = append(args, f.Namespace)
	}
	if f.Pod != "" {
		q += " AND pod = ?"
		args = append(args, f.Pod)
	}
	if f.Severity != "" {
		q += " AND severity = ?"
		args = append(args, f.Severity)
	}
	if !f.Since.IsZero() {
		q += " AND last_seen >= ?"
		args = append(args, f.Since.UTC().Format(time.RFC3339))
	}
	if !f.Until.IsZero() {
		q += " AND last_seen <= ?"
		args = append(args, f.Until.UTC().Format(time.RFC3339))
	}
	q += " ORDER BY last_seen DESC"
	if f.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", f.Limit)
	}

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []core.Event
	for rows.Next() {
		var e core.Event
		var firstSeen, lastSeen string
		if err := rows.Scan(&e.ID, &e.Namespace, &e.Pod, &e.Reason, &e.Message,
			&e.Type, &e.Severity, &e.Count, &firstSeen, &lastSeen); err != nil {
			return nil, err
		}
		e.FirstSeen, _ = time.Parse(time.RFC3339, firstSeen)
		e.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *Store) Close() error { return s.db.Close() }
