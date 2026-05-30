# API Design

REST best practices, versioning strategies, pagination patterns, error responses, and OpenAPI documentation for Go services.

---

## REST Resource Design

```mermaid
graph LR
    subgraph Resources
        USERS[/users] --> USER[/users/:id]
        USER --> ORDERS[/users/:id/orders]
        ORDERS --> ORDER[/users/:id/orders/:oid]
    end
    
    subgraph HTTP Methods
        GET[GET → Read]
        POST[POST → Create]
        PUT[PUT → Full Replace]
        PATCH[PATCH → Partial Update]
        DELETE[DELETE → Remove]
    end
```

| Method | Endpoint | Idempotent | Response |
|--------|----------|-----------|----------|
| GET | `/users` | Yes | 200 + list |
| GET | `/users/123` | Yes | 200 or 404 |
| POST | `/users` | No | 201 + Location header |
| PUT | `/users/123` | Yes | 200 or 204 |
| PATCH | `/users/123` | No | 200 |
| DELETE | `/users/123` | Yes | 204 |

---

## Versioning Strategies

```mermaid
graph TD
    subgraph URL Path
        V1[/api/v1/users]
        V2[/api/v2/users]
    end
    
    subgraph Header
        H1[Accept: application/vnd.api.v1+json]
        H2[Accept: application/vnd.api.v2+json]
    end
    
    subgraph Query Param
        Q1[/users?version=1]
    end
```

| Strategy | Pros | Cons | Use when |
|----------|------|------|----------|
| URL path (`/v1/`) | Obvious, cacheable | URL pollution | Public APIs |
| Header | Clean URLs | Hidden, harder to test | Internal APIs |
| Query param | Easy to add | Not RESTful | Quick iteration |

**Recommended for most Go services**: URL path versioning.

```go
mux := http.NewServeMux()
mux.Handle("/api/v1/", http.StripPrefix("/api/v1", v1Router))
mux.Handle("/api/v2/", http.StripPrefix("/api/v2", v2Router))
```

---

## Pagination

### Offset-Based (Simple)

```
GET /users?page=2&per_page=20
```

```go
type PaginatedResponse struct {
    Data       []User `json:"data"`
    Page       int    `json:"page"`
    PerPage    int    `json:"per_page"`
    TotalCount int    `json:"total_count"`
    TotalPages int    `json:"total_pages"`
}
```

**Problem**: Offset pagination breaks with concurrent inserts (skipped/duplicate items).

### Cursor-Based (Scalable)

```
GET /users?cursor=eyJpZCI6MTAwfQ&limit=20
```

```go
type CursorResponse struct {
    Data       []User `json:"data"`
    NextCursor string `json:"next_cursor,omitempty"` // base64(last_id)
    HasMore    bool   `json:"has_more"`
}

// SQL: WHERE id > $cursor ORDER BY id LIMIT $limit+1
func (r *Repo) ListAfter(ctx context.Context, cursor string, limit int) ([]User, string, error) {
    rows, _ := r.db.QueryContext(ctx,
        `SELECT id, name FROM users WHERE id > $1 ORDER BY id LIMIT $2`,
        cursor, limit+1, // fetch one extra to detect "has_more"
    )
    // ...
    if len(users) > limit {
        next := users[limit-1].ID
        return users[:limit], next, nil
    }
    return users, "", nil
}
```

| Aspect | Offset | Cursor |
|--------|--------|--------|
| Consistency | Breaks with inserts | Stable |
| Performance | O(offset) skip | O(1) seek |
| Random access | Yes (page=5) | No (sequential only) |
| Use case | Admin panels | Feeds, infinite scroll |

---

## Error Responses (RFC 7807)

```go
type ProblemDetail struct {
    Type     string `json:"type"`               // URI identifying error type
    Title    string `json:"title"`              // human-readable summary
    Status   int    `json:"status"`             // HTTP status code
    Detail   string `json:"detail,omitempty"`   // specific explanation
    Instance string `json:"instance,omitempty"` // URI of this occurrence
}

func writeError(w http.ResponseWriter, status int, title, detail string) {
    w.Header().Set("Content-Type", "application/problem+json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(ProblemDetail{
        Type:   "https://api.example.com/errors/" + http.StatusText(status),
        Title:  title,
        Status: status,
        Detail: detail,
    })
}
```

```json
{
  "type": "https://api.example.com/errors/insufficient-funds",
  "title": "Insufficient Funds",
  "status": 422,
  "detail": "Account abc-123 has $50, but $100 was requested",
  "instance": "/payments/pay-456"
}
```

---

## Request Validation

```go
type CreateUserRequest struct {
    Email string `json:"email" validate:"required,email"`
    Name  string `json:"name" validate:"required,min=2,max=100"`
    Age   int    `json:"age" validate:"gte=0,lte=150"`
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, 400, "Invalid JSON", err.Error())
        return
    }
    if err := validate.Struct(req); err != nil {
        writeError(w, 422, "Validation Failed", err.Error())
        return
    }
    // ...
}
```

---

## OpenAPI / Swagger

```yaml
openapi: "3.0.3"
info:
  title: User Service
  version: "1.0.0"
paths:
  /users:
    get:
      summary: List users
      parameters:
        - name: cursor
          in: query
          schema: { type: string }
        - name: limit
          in: query
          schema: { type: integer, default: 20, maximum: 100 }
      responses:
        "200":
          description: Paginated user list
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/UserList"
    post:
      summary: Create user
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/CreateUserRequest"
      responses:
        "201":
          description: Created
          headers:
            Location:
              schema: { type: string }
```

**Go tools**: `swaggo/swag` (generate from comments) or `oapi-codegen` (generate Go from OpenAPI spec).

---

## Rate Limiting Headers

```
HTTP/1.1 200 OK
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640000000

HTTP/1.1 429 Too Many Requests
Retry-After: 30
```
