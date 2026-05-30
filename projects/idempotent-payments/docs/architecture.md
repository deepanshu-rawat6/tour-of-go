# Architecture

## Overview

Payment API that prevents double-charges using idempotency keys and `SELECT ... FOR UPDATE` for account-level locking.

## Components

```
cmd/main.go                        → HTTP server setup
internal/middleware/idempotency.go → Idempotency key middleware (cache responses)
internal/handler/payment.go        → Payment HTTP handler
internal/service/ledger.go         → Business logic (validate, transfer)
internal/repository/ledger.go      → PostgreSQL repository (transactions)
migrations/                        → SQL schema (accounts, payments, idempotency_keys)
```

## Request Flow

1. Client sends `POST /payments` with `Idempotency-Key` header
2. Middleware checks if key exists in DB → if yes, replay cached response
3. If new key, handler validates and calls LedgerService
4. LedgerService opens a transaction:
   - `SELECT ... FOR UPDATE` on source account (row-level lock)
   - Debit source, credit destination, insert payment record
   - `COMMIT`
5. Middleware stores response against the idempotency key
6. Background goroutine cleans up expired keys (TTL)

## Key Concepts

- **Idempotency key**: Client-generated UUID ensuring safe retries
- **SELECT FOR UPDATE**: Pessimistic row lock preventing concurrent overdrafts
- **Response caching**: Replays exact response on retry (status code + body)
- **Hexagonal architecture**: Ports (interfaces) + adapters (implementations)
