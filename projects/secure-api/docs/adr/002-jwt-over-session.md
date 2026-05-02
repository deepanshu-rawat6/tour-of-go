# ADR-002: JWT over Server-Side Sessions

**Status:** Accepted

## Decision

Use stateless JWT tokens instead of server-side sessions.

## Rationale

| Concern | Server-Side Sessions | JWT |
|---|---|---|
| Scalability | Requires shared session store (Redis/DB) across instances | Stateless — any instance validates without coordination |
| Revocation | Instant (delete from store) | Requires token blacklist or short expiry |
| Payload | Opaque ID — server fetches data on each request | Self-contained — claims embedded in token |
| Microservices | Session store becomes a shared dependency | Each service validates independently |
| Mobile/SPA | Cookie-based, CSRF concerns | Bearer token in Authorization header |

## Consequences

- Tokens cannot be revoked before expiry without a blacklist (accepted trade-off for this learning project).
- Secret rotation invalidates all outstanding tokens — use short expiry (1h) in production.
- For this project, the 1-hour expiry and HMAC-SHA256 signing are sufficient to demonstrate the pattern.
