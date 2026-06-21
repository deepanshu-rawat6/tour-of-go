# Authentication & Authorization Deep Dive

OAuth2 flows, PKCE, session management, refresh token rotation, and JWT best practices.

---

## OAuth2 Flows

```mermaid
graph TD
    subgraph Which Flow?
        SPA[SPA / Mobile] --> PKCE[Authorization Code + PKCE]
        SERVER[Server-side App] --> AUTH_CODE[Authorization Code]
        M2M[Machine-to-Machine] --> CLIENT_CRED[Client Credentials]
        LEGACY[Legacy / Trusted] --> PASSWORD[Resource Owner Password<br>⚠️ avoid if possible]
    end
```

---

### Authorization Code + PKCE (Recommended for all clients)

```mermaid
sequenceDiagram
    participant User
    participant App as Client App
    participant Auth as Auth Server
    participant API as Resource Server
    
    App->>App: Generate code_verifier (random 43-128 chars)
    App->>App: code_challenge = SHA256(code_verifier)
    
    App->>Auth: GET /authorize?response_type=code&code_challenge=...&code_challenge_method=S256
    Auth->>User: Login page
    User->>Auth: Credentials
    Auth->>App: Redirect with ?code=abc123
    
    App->>Auth: POST /token {code, code_verifier, client_id}
    Auth->>Auth: Verify SHA256(code_verifier) == code_challenge
    Auth-->>App: {access_token, refresh_token, expires_in}
    
    App->>API: GET /resource (Authorization: Bearer access_token)
    API-->>App: Protected data
```

**Why PKCE?**
- Prevents authorization code interception attacks
- No client_secret needed (safe for SPAs/mobile)
- `code_verifier` proves the same client that started the flow is finishing it

---

### Client Credentials (M2M)

```mermaid
sequenceDiagram
    participant Service as Backend Service
    participant Auth as Auth Server
    participant API as Resource Server
    
    Service->>Auth: POST /token {grant_type=client_credentials, client_id, client_secret}
    Auth-->>Service: {access_token, expires_in}
    Service->>API: GET /internal/data (Bearer token)
```

No user involved — service authenticates itself.

---

## JWT Structure

```
eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ1c2VyLTEyMyIsInJvbGVzIjpbImFkbWluIl0sImV4cCI6MTcwMH0.signature
│── Header ──────────│── Payload ──────────────────────────────────────────────────│── Signature ──│
```

```go
// Claims
type Claims struct {
    jwt.RegisteredClaims
    UserID string   `json:"sub"`
    Roles  []string `json:"roles"`
}

// Issue
func IssueToken(userID string, roles []string, secret []byte) (string, error) {
    claims := Claims{
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "my-service",
        },
        UserID: userID,
        Roles:  roles,
    }
    return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}
```

### JWT Best Practices

| Practice | Why |
|----------|-----|
| Short expiry (15 min) | Limits damage if stolen |
| Use refresh tokens for longevity | Don't make access tokens long-lived |
| RS256 for multi-service | Verify without sharing secret |
| HS256 for single service | Simpler, faster |
| Never store sensitive data in payload | JWT is base64, not encrypted |
| Validate `iss`, `aud`, `exp` | Prevent token reuse across services |

---

## Refresh Token Rotation

```mermaid
sequenceDiagram
    participant Client
    participant Auth as Auth Server
    participant DB as Token Store
    
    Client->>Auth: POST /token/refresh {refresh_token: RT1}
    Auth->>DB: Lookup RT1 (valid, not revoked?)
    DB-->>Auth: Valid, family_id=F1
    Auth->>DB: Revoke RT1, Issue RT2 (same family F1)
    Auth-->>Client: {access_token: AT2, refresh_token: RT2}
    
    Note over Client,DB: If attacker replays RT1:
    Client->>Auth: POST /token/refresh {refresh_token: RT1}
    Auth->>DB: RT1 already revoked!
    Auth->>DB: Revoke ALL tokens in family F1
    Auth-->>Client: 401 (force re-login)
```

**Key**: Each refresh token is single-use. Reuse = compromise detected → revoke entire family.

---

## Session Management

```mermaid
graph LR
    subgraph Stateless JWT
        CLIENT1[Client] -->|Bearer token| SERVER1[Server]
        SERVER1 -->|verify signature| SERVER1
        Note1[No server state<br>Can't revoke easily]
    end
    
    subgraph Stateful Sessions
        CLIENT2[Client] -->|Cookie: session_id| SERVER2[Server]
        SERVER2 -->|lookup| REDIS[(Redis<br>session store)]
        Note2[Revocable<br>Server state required]
    end
```

| Aspect | JWT (Stateless) | Sessions (Stateful) |
|--------|----------------|-------------------|
| Revocation | Hard (need blocklist) | Easy (delete from store) |
| Scalability | No shared state | Need Redis/DB |
| Payload | Claims in token | Server-side only |
| Best for | APIs, microservices | Traditional web apps |

### Hybrid Approach (Recommended)

```
Access Token (JWT, 15 min) → stateless verification
Refresh Token (opaque, stored in DB) → revocable, rotated
```

---

## Go Implementation Patterns

```go
// Middleware: extract and validate JWT
func AuthMiddleware(secret []byte) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
            if token == "" {
                http.Error(w, "unauthorized", 401)
                return
            }

            claims := &Claims{}
            _, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
                return secret, nil
            })
            if err != nil {
                http.Error(w, "invalid token", 401)
                return
            }

            ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
            ctx = context.WithValue(ctx, rolesKey, claims.Roles)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// RBAC check
func RequireRole(role string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            roles := r.Context().Value(rolesKey).([]string)
            for _, r := range roles {
                if r == role {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            http.Error(w, "forbidden", 403)
        })
    }
}
```

---

## Security Checklist

- [ ] HTTPS everywhere (no tokens over HTTP)
- [ ] HttpOnly + Secure + SameSite cookies for refresh tokens
- [ ] CSRF protection for cookie-based auth
- [ ] Rate limit `/login` and `/token` endpoints
- [ ] Hash refresh tokens in DB (store SHA256, compare on use)
- [ ] Rotate signing keys periodically (JWKS endpoint)
- [ ] Log all auth events (login, refresh, revoke)
