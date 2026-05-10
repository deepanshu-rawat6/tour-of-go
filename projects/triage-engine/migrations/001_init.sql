CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE investigation_states (
    ticket_id  TEXT PRIMARY KEY,
    state      JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE runbook_chunks (
    id        TEXT PRIMARY KEY,
    content   TEXT NOT NULL,
    embedding vector(1536) NOT NULL
);

CREATE INDEX ON runbook_chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 10);
