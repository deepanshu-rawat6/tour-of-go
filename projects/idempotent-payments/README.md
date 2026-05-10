# Idempotent Transaction API

A mock payment processing API that prevents double-charges using idempotency keys.

**SDE 2 concepts demonstrated:** Idempotency keys · HTTP middleware · `SELECT ... FOR UPDATE` · ACID transactions · Hexagonal architecture

## Architecture

```
POST /payments + Idempotency-Key
        │
        ▼
IdempotencyMiddleware ──cache hit──► replay cached response
        │ cache miss
        ▼
  PaymentHandler
        │
        ▼
  LedgerService  (validates: amount > 0, from ≠ to)
        │
        ▼
  LedgerRepo.TransferTx
    BEGIN
    SELECT ... FOR UPDATE  ← prevents concurrent overdrafts
    UPDATE balance (debit)
    UPDATE balance (credit)
    INSERT payment
    COMMIT
        │
        ▼
  IdempotencyRepo.Save  ← stores full HTTP response
```

## Quick Start

```bash
make docker-up   # start postgres:16 on :5432
make run         # start server on :8080
```

## Demo: Double-Charge Prevention

```bash
# First call — processes payment, deducts 100 from Alice (id=1)
curl -s -X POST localhost:8080/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: checkout-abc-123" \
  -d '{"from_account_id":1,"to_account_id":2,"amount":100}'

# Second call — same key, returns identical response, NO second deduction
curl -s -X POST localhost:8080/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: checkout-abc-123" \
  -d '{"from_account_id":1,"to_account_id":2,"amount":100}'

# Verify Alice's balance was only reduced once
curl -s localhost:8080/accounts/1   # balance: 900
curl -s localhost:8080/accounts/2   # balance: 600
```

## Error Cases

```bash
# Missing Idempotency-Key → 400
curl -s -X POST localhost:8080/payments \
  -H "Content-Type: application/json" \
  -d '{"from_account_id":1,"to_account_id":2,"amount":100}'

# Insufficient funds → 402
curl -s -X POST localhost:8080/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: big-payment" \
  -d '{"from_account_id":2,"to_account_id":1,"amount":9999}'
```

## Testing

```bash
make test                # unit tests (no DB required)
make docker-up
make integration-test    # integration tests (requires postgres)
```

## Configuration

Set `CONFIG_PATH` env var to a YAML file:

```yaml
server:
  addr: ":8080"
database:
  dsn: "postgres://payments:payments@localhost:5432/payments"
idempotency:
  ttl: 24h
  cleanupInterval: 1h
```

## Key Design Decisions

- **`SELECT ... FOR UPDATE`** — row-level lock on the sender's account prevents two concurrent requests from both passing the balance check and both deducting.
- **`ON CONFLICT DO NOTHING`** on idempotency key insert — first writer wins in a race; the second request will find the key on its next `Get` call.
- **Full response caching** — the middleware stores the exact status code, headers, and body so retries receive a byte-for-byte identical response.
- **TTL cleanup goroutine** — expired keys are purged on a configurable interval (default 1h) to prevent unbounded table growth.
