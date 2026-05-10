# 11 — Edge API Gateway

A Netflix Zuul-style Edge API Gateway in Go. Single public entry point for a microservice cluster — handles request tracing, JWT authentication, rate limiting, and context-aware reverse proxying.

**Key concepts:** `go-chi/chi` middleware composability, JWT validation (HS256), `httputil.ReverseProxy` Director override, `context.Context` for identity propagation, `replace` directive for module composition.

## Architecture

```
Client
  │
  ▼
chi Router
  ├─ Global: RequestID (UUID inject) + Logger + Recoverer
  │
  ├─ GET  /health          → 200 {"status":"ok"}
  ├─ POST /login           → issues JWT
  │
  ├─ /api/v1/*  ──── Auth middleware (validates JWT)
  │              ──── RateLimit (100 req/s, token bucket from project 04)
  │              ├─ /api/v1/users/*   → proxy → user-service:8081
  │              └─ /api/v1/billing/* → proxy → billing-service:8082
  │
  └─ /admin/*   ──── Auth middleware
                ──── RequireRole("admin")
                └─ GET /admin/routes → list upstream map

Proxy Director (per forwarded request):
  - Strip Authorization header (JWT never reaches downstream)
  - Inject X-Internal-User-Id  (from context, set by Auth middleware)
  - Inject X-Internal-User-Role
  - Propagate X-Request-ID
```

## Quick Start

```bash
# Terminal 1 — mock upstream (echoes received headers)
make run-upstream          # listens on :8081

# Terminal 2 — gateway
make run                   # listens on :8080
```

## Demo Walkthrough

```bash
# 1. Get a JWT
TOKEN=$(curl -s -X POST localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","role":"user"}' | jq -r .token)

# 2. Hit a protected route — gateway proxies to mock upstream
curl -s localhost:8080/api/v1/users/profile \
  -H "Authorization: Bearer $TOKEN"
# Mock upstream terminal shows:
#   X-Internal-User-Id: alice
#   X-Internal-User-Role: user
#   X-Request-ID: <uuid>
#   Authorization: (empty — stripped by gateway)

# 3. Admin route with wrong role → 403
curl -s localhost:8080/admin/routes \
  -H "Authorization: Bearer $TOKEN"

# 4. Get an admin token
ADMIN=$(curl -s -X POST localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"bob","role":"admin"}' | jq -r .token)

curl -s localhost:8080/admin/routes \
  -H "Authorization: Bearer $ADMIN"
# Returns: {"users":"http://localhost:8081","billing":"http://localhost:8082"}

# 5. No token → 401
curl -s localhost:8080/api/v1/users/profile
```

## Testing

```bash
make test
```

## Module Composition

This project imports `04-rate-limiter` directly via a `replace` directive in `go.mod`:

```
replace tour_of_go/projects/from-scratch/04-rate-limiter => ../04-rate-limiter
```

The gateway depends on the `ratelimit.Limiter` interface — it never knows it's using a token bucket. Swapping to a sliding window requires changing one line in `main.go`.

## Key Design Decisions

- **Authenticator struct** — holds the HMAC secret as `[]byte` in memory. The middleware never reads from disk or env on each request.
- **Director override** — the proxy strips `Authorization` before forwarding. Downstream services can fully trust `X-Internal-User-Id` because the gateway already did the cryptographic verification.
- **chi route groups** — rate limiting applies only to `/api/v1`, not `/health` or `/login`. `RequireRole` applies only to `/admin`. No nested `if` chains in handlers.
- **replace directive** — demonstrates Go module composition without publishing. The rate limiter is a reusable library, not copy-pasted code.
