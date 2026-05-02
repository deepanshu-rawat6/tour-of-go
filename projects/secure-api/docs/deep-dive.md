# secure-api: Deep Dive

## SOLID Principles in Practice

### Single Responsibility Principle
Each package owns exactly one concern:
- `internal/domain` — value objects only, no I/O
- `internal/auth` — authentication logic only
- `internal/middleware` — HTTP interception only
- `internal/handler` — HTTP response shaping only
- `internal/config` — configuration loading only

### Open/Closed Principle
The middleware chain in `cmd/server/main.go` is a slice of `func(http.Handler) http.Handler`. Adding rate limiting, logging, or CORS requires appending to the slice — no existing code changes.

```go
mux.Handle("/me", middleware.Chain(
    http.HandlerFunc(handler.Me),
    middleware.Auth(jwtAdapter),
    // middleware.RateLimit(100),  ← add without touching Auth or Me
))
```

### Interface Segregation Principle
Each port interface has exactly one method. `handler.Token` only needs `UserAuthenticator` and `TokenIssuer` — it never sees `Validate`. This means you can swap the JWT library without touching the handler.

### Dependency Inversion Principle
`handler.Token` is constructed with interface values:
```go
func Token(authn ports.UserAuthenticator, issuer ports.TokenIssuer) http.HandlerFunc
```
The handler never imports `auth` or `jwt`. Wiring happens only in `cmd/server/main.go`.

---

## TDD Workflow

Every component in this project was written test-first:

1. **Red** — write a failing test describing the desired behaviour
2. **Green** — write the minimal code to make it pass
3. **Refactor** — clean up without breaking tests

Example: `TestJWTAdapter_Validate_TableDriven` was written before `jwt.go` existed. The test defined the contract (valid token → Claims, expired → ErrTokenExpired, wrong sig → ErrTokenInvalid), then the implementation was written to satisfy it.

---

## Immutable Value Objects

`Claims` and `Token` follow the immutable value object pattern:

```go
// No public fields. No setters. Construction only.
type Claims struct {
    userID    string
    roles     []string
    expiresAt time.Time
}

// Roles() returns a copy — callers cannot mutate internal state.
func (c Claims) Roles() []string {
    r := make([]string, len(c.roles))
    copy(r, c.roles)
    return r
}
```

This eliminates an entire class of bugs: a handler cannot accidentally modify the claims that were validated by the middleware.

---

## JWT Internals

Tokens are HMAC-SHA256 signed JWTs with three claims:
- `sub` — user ID
- `roles` — array of role strings
- `exp` / `iat` — expiry and issued-at Unix timestamps

The `JWTAdapter` implements both `TokenIssuer` and `TokenValidator` — two separate interfaces — demonstrating that one concrete type can satisfy multiple ISP-compliant interfaces.

---

## mTLS Flow

```
Client                    Server
  |                          |
  |--- ClientHello --------->|
  |<-- ServerHello + Cert ---|
  |--- ClientCert + Verify ->|  ← client proves identity
  |<-- Finished -------------|
  |=== Encrypted channel ====|
```

The server's `tls.Config` sets `ClientAuth: tls.RequireAndVerifyClientCert` and provides a `ClientCAs` pool containing the CA cert. Any client without a cert signed by that CA is rejected at the TLS handshake — before any HTTP code runs.
