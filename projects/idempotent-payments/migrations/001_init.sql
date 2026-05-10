CREATE TABLE accounts (
    id      BIGSERIAL PRIMARY KEY,
    name    TEXT NOT NULL,
    balance NUMERIC(19,4) NOT NULL DEFAULT 0 CHECK (balance >= 0)
);

CREATE TABLE payments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_account_id BIGINT NOT NULL REFERENCES accounts(id),
    to_account_id   BIGINT NOT NULL REFERENCES accounts(id),
    amount          NUMERIC(19,4) NOT NULL,
    idempotency_key TEXT UNIQUE NOT NULL,
    status          TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE idempotency_keys (
    key         TEXT PRIMARY KEY,
    status_code INT NOT NULL,
    headers     JSONB NOT NULL DEFAULT '{}',
    body        BYTEA NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL
);

-- Seed accounts
INSERT INTO accounts (name, balance) VALUES ('Alice', 1000.00), ('Bob', 500.00);
