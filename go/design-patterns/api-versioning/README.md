# API Versioning

Strategies for evolving your API without breaking existing clients.

**The fundamental challenge:** clients are deployed independently of servers.
You cannot update all clients simultaneously, so you must support multiple
versions concurrently during a migration window.

---

## The Three Strategies

### Strategy 1: URL Path Versioning

```
GET /v1/users/42
GET /v2/users/42
```

**How it works:** the version is part of the URL path, making it visible,
bookmarkable, and cache-friendly.

**Trade-offs:**

| Pros | Cons |
|---|---|
| Obvious to developers | URLs are supposed to identify resources, not versions |
| Easy to route with any HTTP framework | Clients must hardcode the version |
| Browser/curl friendly | Search engines index multiple URLs for same resource |
| Simple to test with curl/Postman | Can't set version per-request without URL change |
| Industry standard (Stripe, GitHub, Twitter) | |

### Strategy 2: Header Versioning

```
GET /users/42
API-Version: 2
```

**How it works:** the version is supplied in a request header. The URL
stays stable; routing logic inspects the header.

**Trade-offs:**

| Pros | Cons |
|---|---|
| URLs represent resources cleanly | Less discoverable (need to read docs) |
| Version per-request without URL change | Not usable from browser address bar |
| Works well with SDK-based clients | Caching proxies may ignore the header |
| Used by Stripe (`Stripe-Version`) | Requires custom header handling |

### Strategy 3: Content-Type (Accept Header) Versioning

```
GET /users/42
Accept: application/vnd.myapi.v2+json
```

**How it works:** the client negotiates which representation it wants via
`Accept`. The server inspects the media type to determine the version.

**Trade-offs:**

| Pros | Cons |
|---|---|
| Fully RESTful (content negotiation) | Complex to implement |
| Same URL, same resource | Confusing for developers |
| Fine-grained: can version fields not endpoints | Testing requires correct headers |
| Used by GitHub API v3 (partially) | Overkill for most APIs |

---

## Trade-offs Summary Table

| | URL Path | Header | Content-Type |
|---|---|---|---|
| Discoverability | ✅ High | ❌ Low | ❌ Very Low |
| Cache-friendliness | ✅ Yes (URL = cache key) | ⚠️ Requires Vary header | ⚠️ Requires Vary header |
| REST purity | ❌ Low | ⚠️ Medium | ✅ High |
| Client complexity | ✅ Low | ⚠️ Medium | ❌ High |
| Industry adoption | ✅ Most common | ⚠️ Second | ❌ Rare |

**Recommendation for most teams:** Use **URL path versioning** (`/v1/`, `/v2/`).
It's the most familiar, simplest to route, and easiest to test.

---

## Go Implementation: URL Path Versioning

### With stdlib `net/http`

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
)

// extractVersion parses the version from a URL like /v1/users/42.
// Returns "v1" or "" if no version prefix found.
func extractVersion(path string) string {
    parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
    if len(parts) > 0 && strings.HasPrefix(parts[0], "v") {
        return parts[0] // "v1", "v2", etc.
    }
    return ""
}

// stripVersion removes the version prefix from a path.
// /v1/users/42 → /users/42
func stripVersion(path string) string {
    parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
    if len(parts) == 2 {
        return "/" + parts[1]
    }
    return "/"
}

// VersionedMux dispatches to different handlers based on URL version prefix.
type VersionedMux struct {
    handlers map[string]*http.ServeMux // "v1" → mux, "v2" → mux
}

func NewVersionedMux() *VersionedMux {
    return &VersionedMux{handlers: make(map[string]*http.ServeMux)}
}

// Register adds routes to a specific API version.
func (vm *VersionedMux) Register(version, pattern string, h http.HandlerFunc) {
    mux, ok := vm.handlers[version]
    if !ok {
        mux = http.NewServeMux()
        vm.handlers[version] = mux
    }
    mux.HandleFunc(pattern, h)
}

// ServeHTTP implements http.Handler. Routes to the correct version mux.
func (vm *VersionedMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    version := extractVersion(r.URL.Path)
    mux, ok := vm.handlers[version]
    if !ok {
        http.Error(w, fmt.Sprintf("unknown API version: %s", version), http.StatusBadRequest)
        return
    }
    // Strip the version prefix before dispatching
    r2 := r.Clone(r.Context())
    r2.URL.Path = stripVersion(r.URL.Path)
    mux.ServeHTTP(w, r2)
}

// UserV1 is the v1 response shape.
type UserV1 struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

// UserV2 adds a new field (email) — additive change, safe.
// In real life you might also rename fields, change types, etc.
type UserV2 struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"` // new in v2
}

func getUserV1(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(UserV1{ID: 42, Name: "Alice"})
}

func getUserV2(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(UserV2{ID: 42, Name: "Alice", Email: "alice@example.com"})
}

func main() {
    vm := NewVersionedMux()
    vm.Register("v1", "/users/", getUserV1)
    vm.Register("v2", "/users/", getUserV2)

    http.ListenAndServe(":8080", vm)
    // GET /v1/users/42 → {"id":42,"name":"Alice"}
    // GET /v2/users/42 → {"id":42,"name":"Alice","email":"alice@example.com"}
}
```

### With chi router

```go
import "github.com/go-chi/chi/v5"

r := chi.NewRouter()

// Mount v1 routes under /v1 prefix
r.Route("/v1", func(r chi.Router) {
    r.Get("/users/{id}", getUserV1)
    r.Post("/users", createUserV1)
    r.Delete("/users/{id}", deleteUserV1)
})

// Mount v2 routes under /v2 prefix
r.Route("/v2", func(r chi.Router) {
    r.Get("/users/{id}", getUserV2)
    r.Post("/users", createUserV2)
    r.Delete("/users/{id}", deleteUserV2)
    r.Get("/users/{id}/profile", getProfileV2) // new endpoint in v2
})

http.ListenAndServe(":8080", r)
```

---

## Go Implementation: Header Versioning

```go
package main

import (
    "net/http"
    "strconv"
)

const defaultAPIVersion = 1

// versionFromHeader parses the "API-Version" header.
// Returns defaultAPIVersion if the header is absent or invalid.
func versionFromHeader(r *http.Request) int {
    h := r.Header.Get("API-Version")
    if h == "" {
        return defaultAPIVersion
    }
    v, err := strconv.Atoi(h)
    if err != nil || v < 1 {
        return defaultAPIVersion
    }
    return v
}

// VersionMiddleware injects the parsed version into the request context.
func VersionMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        version := versionFromHeader(r)

        // Validate the version is supported
        if version > 2 {
            http.Error(w, "unsupported API version", http.StatusBadRequest)
            return
        }

        // Advertise the version we're serving in the response
        w.Header().Set("API-Version", strconv.Itoa(version))

        // Inject into context for handlers to read
        ctx := withAPIVersion(r.Context(), version)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func getUser(w http.ResponseWriter, r *http.Request) {
    version := apiVersionFromCtx(r.Context())
    switch version {
    case 2:
        getUserV2(w, r)
    default:
        getUserV1(w, r)
    }
}

// Caching note:
// When using header versioning, set the Vary header so caches treat
// responses with different API-Version as distinct cache entries:
//   w.Header().Set("Vary", "API-Version")
```

---

## Go Implementation: Content-Type Versioning

```go
package main

import (
    "net/http"
    "strings"
)

// parseVersionFromAccept extracts the version from an Accept header like:
//   application/vnd.myapi.v2+json → "v2"
//   application/json              → "" (use default)
func parseVersionFromAccept(accept string) string {
    // Split on comma for multiple media types: "application/vnd.myapi.v2+json, */*"
    for _, part := range strings.Split(accept, ",") {
        part = strings.TrimSpace(part)
        // application/vnd.myapi.v2+json
        if strings.HasPrefix(part, "application/vnd.myapi.") {
            // Extract between "vnd.myapi." and "+"
            rest := strings.TrimPrefix(part, "application/vnd.myapi.")
            if idx := strings.Index(rest, "+"); idx > 0 {
                return rest[:idx] // "v2"
            }
        }
    }
    return "" // no version specified
}

func contentNegotiationHandler(w http.ResponseWriter, r *http.Request) {
    version := parseVersionFromAccept(r.Header.Get("Accept"))
    switch version {
    case "v2":
        // Respond with v2 format
        w.Header().Set("Content-Type", "application/vnd.myapi.v2+json")
        getUserV2(w, r)
    default:
        // Default to v1
        w.Header().Set("Content-Type", "application/vnd.myapi.v1+json")
        getUserV1(w, r)
    }
}
```

---

## Sunset Headers

When deprecating an API version, communicate it in response headers.
The `Deprecation` and `Sunset` headers are defined in RFC 8594.

```go
package main

import (
    "net/http"
    "time"
)

// SunsetMiddleware adds deprecation headers to versioned responses.
// Clients that observe these headers can update before the cutoff date.
func SunsetMiddleware(version string, sunsetDate time.Time) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // RFC 8594: Deprecation header marks the version as deprecated
            w.Header().Set("Deprecation", "true")

            // Sunset header tells clients the exact removal date (RFC 1123 format)
            w.Header().Set("Sunset", sunsetDate.UTC().Format(http.TimeFormat))

            // Link header points to migration docs
            w.Header().Set("Link",
                `<https://api.example.com/docs/migration/v2>; rel="successor-version"`)

            next.ServeHTTP(w, r)
        })
    }
}

// Usage in a chi router:
//
//   v1Sunset := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
//
//   r.Route("/v1", func(r chi.Router) {
//       r.Use(SunsetMiddleware("v1", v1Sunset))
//       r.Get("/users/{id}", getUserV1)
//   })
```

Example response headers clients receive on v1 endpoints:

```http
HTTP/1.1 200 OK
Content-Type: application/json
Deprecation: true
Sunset: Thu, 31 Dec 2025 00:00:00 GMT
Link: <https://api.example.com/docs/migration/v2>; rel="successor-version"
```

---

## Backwards Compatibility Rules

### ✅ Safe (Additive) Changes — No Version Bump Needed

```
Adding a new endpoint               GET /v1/orders/summary  (new)
Adding a new optional field         { "email": "..."  }    (new field)
Adding a new query parameter        ?include_archived=true  (optional)
Adding a new enum value             status: "processing"   (new value)
Relaxing validation                 field now accepts longer strings
Making a required field optional    was required, now optional
Adding a new HTTP header            X-Request-ID response header
```

### ❌ Breaking Changes — Require New Version

```
Renaming a field                    "user_id" → "userId"
Changing a field type               "age": 30 → "age": "30"
Removing a field                    deleting "legacy_id"
Changing error response shape       { "error": "..." } → { "errors": [...] }
Changing HTTP status codes          200 → 201 for creates
Making an optional field required   field becomes required
Removing an endpoint                DELETE /v1/users/{id} removed
Changing pagination behaviour       page-based → cursor-based
Changing sort order without notice  results in different order
```

---

## Version Routing with Version Registry

For larger APIs, centralise version routing logic:

```go
package api

import (
    "net/http"
    "sort"
)

// Route represents a single endpoint at a specific version.
type Route struct {
    Method  string
    Pattern string
    Version int
    Handler http.HandlerFunc
}

// Registry holds all routes across all versions.
// At startup, routes for each version include all lower-version routes
// that haven't been overridden (inheritance).
type Registry struct {
    routes []Route
}

func (reg *Registry) Register(version int, method, pattern string, h http.HandlerFunc) {
    reg.routes = append(reg.routes, Route{method, pattern, version, h})
}

// BuildMux creates an http.ServeMux for a specific version.
// A handler registered at v1 is available at v2 unless v2 overrides it.
func (reg *Registry) BuildMux(targetVersion int) *http.ServeMux {
    mux := http.NewServeMux()

    // Sort by version so higher versions override lower ones
    sort.Slice(reg.routes, func(i, j int) bool {
        return reg.routes[i].Version < reg.routes[j].Version
    })

    seen := map[string]http.HandlerFunc{}
    for _, r := range reg.routes {
        if r.Version > targetVersion {
            continue // not available in this version
        }
        key := r.Method + " " + r.Pattern
        seen[key] = r.Handler // higher version overwrites lower
    }

    for key, h := range seen {
        parts := splitMethodPattern(key)
        mux.HandleFunc(parts[1], func(w http.ResponseWriter, r *http.Request) {
            if r.Method != parts[0] {
                http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
                return
            }
            h(w, r)
        })
    }
    return mux
}

func splitMethodPattern(key string) [2]string {
    for i, c := range key {
        if c == ' ' {
            return [2]string{key[:i], key[i+1:]}
        }
    }
    return [2]string{"GET", key}
}
```

---

## Consumer-Driven Contract Testing

Before releasing a new version, verify you haven't broken existing clients
by running **Pact** (or similar) contract tests.

```
Client publishes a "contract" (pact file):
  "I call GET /v1/users/42 and expect { id: int, name: string }"

Server runs contract tests against the pact file:
  - If the contract passes → safe to deploy
  - If the contract fails  → breaking change detected before release

Tools: pact.io, dredd (API Blueprint), OpenAPI diff (oasdiff)

  go install github.com/Tufin/oasdiff@latest
  oasdiff diff api-v1.yaml api-v2.yaml --fail-on ERR
  # exits non-zero if breaking changes are detected (run in CI)
```

---

## Real-World Examples

### Stripe (Header Versioning)

```
Stripe-Version: 2023-10-16

Stripe dates their API versions by release date. Customers pin to the date
they integrated. Stripe maintains ALL versions forever (100+ versions since 2011).
Each customer's requests are transparently translated to their pinned version.
```

### GitHub (URL Path)

```
GET https://api.github.com/v3/users/octocat
Accept: application/vnd.github.v3+json

GitHub used both URL path (/v3) and Content-Type versioning simultaneously.
GitHub recently added /2022-11-28 date-based versions for some new APIs.
```

### Twilio (URL Path)

```
POST https://api.twilio.com/2010-04-01/Accounts/{AccountSid}/Messages

Twilio uses date-based URL versioning. The year-date in the path is the
API version. They've never needed to change it (stable since 2010).
Lesson: a stable, well-designed v1 is better than versioning for its own sake.
```

### Kubernetes (URL Path + Kind Version)

```
GET /apis/apps/v1/deployments       (stable)
GET /apis/apps/v1beta1/deployments  (beta — may change)
GET /apis/apps/v1alpha1/deployments (alpha — expect breaking changes)

Kubernetes uses semantic maturity levels in the URL: alpha, beta, stable.
This signals to users what SLA to expect before adopting a resource type.
```

---

## Checklist Before Releasing a New Version

- [ ] Document breaking changes in a CHANGELOG or migration guide
- [ ] Set `Deprecation` and `Sunset` headers on old version routes
- [ ] Update SDK client libraries (if you distribute them)
- [ ] Notify API consumers with at least 6 months' notice before sunset
- [ ] Run contract tests (Pact/oasdiff) in CI to catch unintentional breaks
- [ ] Ensure the old version returns helpful error messages pointing to new docs
- [ ] Keep the old version running at least through the sunset date
- [ ] Monitor old-version traffic to gauge migration progress
