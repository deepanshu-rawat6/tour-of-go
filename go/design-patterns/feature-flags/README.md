# Feature Flags

**Decouple deployment from release.** Feature flags let you merge and deploy code
that is "off" in production, then turn it on independently — without a new deploy.

---

## Why Feature Flags?

| Problem | Feature Flag Solution |
|---|---|
| Big-bang releases are risky | Dark-launch code behind a flag |
| Rollback requires re-deploy | Flip the flag off instantly |
| A/B testing needs infrastructure | Percentage-rollout flag |
| Emergency production incident | Kill switch disables the feature |
| Testing in prod without users seeing | Internal-user flag |

**The golden rule:** *flags are short-lived*. Every flag is tech debt. Create a
ticket to remove each flag within 30-90 days of going 100%.

---

## Pattern 1: Env-Var Flags (Simplest)

Use for **infrastructure toggles** — things that change between environments
(dev/staging/prod) and don't change at runtime.

```go
// flags/envflags.go
package flags

import "os"

// NewDBEnabled returns true if the new DB schema is active.
// Set NEW_DB_ENABLED=true in the environment to enable.
//
// This is the right tool for:
//   - Enabling debug endpoints in staging but not prod
//   - Switching between two infrastructure backends
//   - Feature toggles that follow deployment, not users
func NewDBEnabled() bool {
    return os.Getenv("NEW_DB_ENABLED") == "true"
}

// MaintenanceMode prevents all writes when true.
func MaintenanceMode() bool {
    return os.Getenv("MAINTENANCE_MODE") == "true"
}

// Usage in a handler:
//
//   func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
//       if flags.MaintenanceMode() {
//           http.Error(w, "service temporarily unavailable", http.StatusServiceUnavailable)
//           return
//       }
//       // ... normal logic
//   }
```

**Trade-offs:**
- ✅ Zero infrastructure required
- ✅ Works with any 12-factor app
- ❌ Requires re-deploy to change (not hot-reloadable without SIGHUP tricks)
- ❌ Not user-scoped, not percentage-based

---

## Pattern 2: In-Memory Flag Store with Hot Reload

Use for **application feature toggles** that need to change at runtime without
redeployment. Backed by a JSON config file or key-value store that is polled.

```go
// flags/store.go
package flags

import (
    "encoding/json"
    "log"
    "os"
    "sync"
    "time"
)

// FlagStore holds boolean feature flags and supports hot-reload via a
// background goroutine that re-reads a config file on an interval.
type FlagStore struct {
    mu      sync.RWMutex
    flags   map[string]bool
    path    string // path to JSON config file
}

// NewFlagStore loads flags from path and starts a background watcher.
// The watcher re-reads the file every interval and atomically swaps the map.
//
// Example config file (flags.json):
//   {
//     "new_checkout_flow": true,
//     "dark_mode_button": false,
//     "experimental_search": true
//   }
func NewFlagStore(path string, interval time.Duration) (*FlagStore, error) {
    fs := &FlagStore{path: path}
    if err := fs.load(); err != nil {
        return nil, err
    }
    go fs.watch(interval) // background hot-reload
    return fs, nil
}

// IsEnabled returns the current value of a flag.
// Returns false for unknown flags (safe default = off).
func (fs *FlagStore) IsEnabled(key string) bool {
    fs.mu.RLock()
    defer fs.mu.RUnlock()
    return fs.flags[key] // false if key doesn't exist
}

// Set overrides a flag in memory (useful for testing or emergency override).
func (fs *FlagStore) Set(key string, val bool) {
    fs.mu.Lock()
    defer fs.mu.Unlock()
    fs.flags[key] = val
}

func (fs *FlagStore) load() error {
    data, err := os.ReadFile(fs.path)
    if err != nil {
        return err
    }
    var m map[string]bool
    if err := json.Unmarshal(data, &m); err != nil {
        return err
    }
    fs.mu.Lock()
    fs.flags = m
    fs.mu.Unlock()
    return nil
}

func (fs *FlagStore) watch(interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    for range ticker.C {
        if err := fs.load(); err != nil {
            // Don't crash on reload failure — keep old flags active.
            // Log and alert; an operator can fix the config file.
            log.Printf("flag reload error: %v (keeping previous flags)", err)
        }
    }
}

// Usage:
//
//   store, err := flags.NewFlagStore("flags.json", 30*time.Second)
//   if err != nil {
//       log.Fatal(err)
//   }
//
//   if store.IsEnabled("new_checkout_flow") {
//       return newCheckout(w, r)
//   }
//   return legacyCheckout(w, r)
```

**Trade-offs:**
- ✅ Hot-reload without restart
- ✅ Zero external dependencies
- ❌ Config file must be on disk (or mounted volume in K8s)
- ❌ Not user-scoped
- ❌ All instances must agree (slight skew during reload window)

---

## Pattern 3: Remote Flag Service

Use for **production-grade feature management** with user targeting, percentage
rollouts, and analytics. Services: **LaunchDarkly**, **Unleash**, **Flagsmith**
(self-hosted), **AWS AppConfig**, **Firebase Remote Config**.

```go
// flags/remote.go
package flags

import (
    "context"
    "sync/atomic"
    "time"
)

// RemoteFlags is an abstraction over any remote flag service.
// Define this interface in your app and inject it — never depend on the
// SDK directly in business logic. This makes testing trivial (see below).
type RemoteFlags interface {
    // BoolFlag returns the value of a boolean flag for a given user context.
    // The FlagContext carries user ID, attributes for targeting rules.
    BoolFlag(ctx context.Context, key string, fc FlagContext, defaultVal bool) bool

    // StringFlag returns one of several string variants (multivariate flag).
    StringFlag(ctx context.Context, key string, fc FlagContext, defaultVal string) string
}

// FlagContext carries attributes used for targeting rules:
// "show to users in EU", "show to beta users", etc.
type FlagContext struct {
    UserID     string
    Email      string
    Country    string
    Plan       string // "free", "pro", "enterprise"
    Attributes map[string]string
}

// CachingFlags wraps a RemoteFlags provider and caches the last-known
// values. If the remote service is unreachable, serve stale cached values
// rather than crashing. This is the graceful degradation pattern.
type CachingFlags struct {
    provider RemoteFlags
    cache    atomic.Pointer[map[string]bool] // lock-free read
    ttl      time.Duration
}

// BoolFlag returns the flag value from cache (or provider if cache is cold).
func (c *CachingFlags) BoolFlag(ctx context.Context, key string, fc FlagContext, def bool) bool {
    if m := c.cache.Load(); m != nil {
        if val, ok := (*m)[key]; ok {
            return val
        }
    }
    // Cache miss or cold start — call provider (may fail during outage)
    return c.provider.BoolFlag(ctx, key, fc, def)
}

// LaunchDarkly pattern (pseudo-code showing the integration shape):
//
//   import ld "gopkg.in/launchdarkly/go-server-sdk.v6"
//
//   client, _ := ld.MakeClient(sdkKey, 5*time.Second)
//
//   user := ldcontext.NewBuilder("user-123").
//       Name("Alice").
//       SetString("plan", "pro").
//       Build()
//
//   enabled, _ := client.BoolVariation("new-checkout", user, false)
//
//   // Multivariate: "control", "variant-a", "variant-b"
//   variant, _ := client.StringVariation("search-algorithm", user, "control")

// Unleash pattern (pseudo-code):
//
//   import unleash "github.com/Unleash/unleash-client-go/v3"
//
//   unleash.Initialize(
//       unleash.WithUrl("https://unleash.internal/api"),
//       unleash.WithAppName("payment-service"),
//   )
//
//   enabled := unleash.IsEnabled("new-checkout-flow",
//       unleash.WithContext(unleash.Context{UserId: "user-123"}),
//   )
```

---

## Graceful Degradation

If the flag service goes down, your app should continue working with
**safe defaults**, not crash or block.

```go
// Graceful degradation: always supply a sensible default.
//
// Rule: the default should be the CONSERVATIVE path (the existing behaviour).
// A flag for "new checkout" should default to FALSE (use old checkout).
// Never default to a behaviour that could corrupt data.

func (h *Handler) Checkout(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    fc := FlagContext{UserID: userID(r), Plan: plan(r)}

    // If flag service is down, IsEnabled returns the default (false).
    // Old checkout continues to work — degraded but not broken.
    if h.flags.BoolFlag(ctx, "new-checkout-flow", fc, false) {
        h.newCheckout(w, r)
        return
    }
    h.legacyCheckout(w, r)
}
```

---

## Kill Switches

A kill switch is a flag that **disables** a feature immediately in an incident.
Unlike a regular feature flag (enable new code), a kill switch **disables
existing code**.

```go
// Kill switch example: the payment processor is returning 5xx.
// Flip PAYMENT_KILL_SWITCH=true to reject new payment requests immediately.
//
// Naming convention: *_KILL_SWITCH or *_DISABLED makes intent obvious.

func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
    if h.flags.IsEnabled("payment_kill_switch") {
        // Return 503 with Retry-After header so clients back off gracefully.
        w.Header().Set("Retry-After", "120")
        http.Error(w, "payment service temporarily disabled", http.StatusServiceUnavailable)
        return
    }
    // ... normal payment logic
}

// Kill switch checklist:
//   - Can an on-call engineer flip it in < 60 seconds?
//   - Does it return 503 (not 500) so load balancers route away?
//   - Does it emit an alert/metric when activated?
//   - Is there a runbook entry for it?
```

---

## Percentage Rollouts

Rollout to N% of users by hashing the user ID. The hash is deterministic:
the same user always gets the same experience within a rollout window.

```go
package flags

import (
    "crypto/sha256"
    "encoding/binary"
)

// InRollout returns true if userID falls in the first pct% of users.
// The hash is stable: user "alice" always maps to the same bucket.
//
// How it works:
//   1. SHA-256 hash of (flagName + ":" + userID) → 32 bytes
//   2. Take first 8 bytes as uint64 → value in [0, 2^64)
//   3. value % 100 → bucket in [0, 99]
//   4. bucket < pct → user is in the rollout
//
// The flagName is included in the hash so the same user gets different
// buckets for different flags (avoids correlated rollouts).
func InRollout(flagName, userID string, pct int) bool {
    if pct <= 0 {
        return false
    }
    if pct >= 100 {
        return true
    }
    h := sha256.Sum256([]byte(flagName + ":" + userID))
    bucket := int(binary.BigEndian.Uint64(h[:8]) % 100)
    return bucket < pct
}

// Usage:
//
//   // Gradually ramp from 0% → 10% → 50% → 100% over 2 weeks.
//   if flags.InRollout("new-search", userID, 10) {
//       return h.newSearch(w, r)
//   }
//   return h.legacySearch(w, r)
```

---

## Flag Types

| Type | Example Key | Values | Use Case |
|---|---|---|---|
| Boolean | `new_checkout` | true/false | Feature on/off |
| String | `search_algorithm` | "bm25", "neural", "hybrid" | A/B/n variants |
| Integer | `rate_limit_rps` | 100, 500, 1000 | Tunable thresholds |
| JSON | `feature_config` | `{"timeout": 5, "retries": 3}` | Complex config |

```go
// Multivariate string flag: A/B/C test three search algorithms.
variant := flags.StringFlag(ctx, "search_algorithm", fc, "bm25")
switch variant {
case "neural":
    return h.neuralSearch(query)
case "hybrid":
    return h.hybridSearch(query)
default: // "bm25" or any unknown value
    return h.bm25Search(query)
}
```

---

## Testing with Feature Flags

**Always define an interface** for your flag provider. Tests inject a mock
implementation that returns deterministic values, no network needed.

```go
// flags/interface.go
package flags

// Provider is the interface your business logic depends on.
// The real implementation calls LaunchDarkly/Unleash/your store.
// The test implementation is a simple map.
type Provider interface {
    IsEnabled(key string) bool
    StringFlag(key, defaultVal string) string
}

// StaticProvider is a test double that returns fixed values.
// Create it in your test setup, inject it, assert on side effects.
type StaticProvider struct {
    Booleans map[string]bool
    Strings  map[string]string
}

func (s *StaticProvider) IsEnabled(key string) bool {
    return s.Booleans[key]
}

func (s *StaticProvider) StringFlag(key, def string) string {
    if v, ok := s.Strings[key]; ok {
        return v
    }
    return def
}

// In your test:
//
//   func TestNewCheckout(t *testing.T) {
//       flags := &StaticProvider{
//           Booleans: map[string]bool{"new_checkout_flow": true},
//       }
//       h := NewHandler(flags)
//       // ... assert new checkout path is taken
//   }
//
//   func TestLegacyCheckout(t *testing.T) {
//       flags := &StaticProvider{} // all flags default to false
//       h := NewHandler(flags)
//       // ... assert legacy checkout path is taken
//   }
```

---

## Production Checklist

Before shipping a feature flag:

- [ ] **Default is safe**: the false/default case is the existing behaviour
- [ ] **Interface injected**: business logic uses `flags.Provider`, not SDK directly
- [ ] **Kill switch ready**: flag can be flipped by on-call in < 60 seconds
- [ ] **Observability**: emit a metric/log when the flag changes value for a user
- [ ] **Expiry ticket**: created JIRA/GitHub issue to remove the flag in 30-90 days
- [ ] **Graceful degradation**: app works correctly when flag service is unreachable
- [ ] **No permanent flags**: flags are not a config system; use env vars for that
- [ ] **Testing coverage**: tests for both `true` and `false` branches

---

## Real-World Examples

**GitHub Feature Flags**: GitHub uses an internal system called `Flipper` to
gate features behind user/org membership checks before public rollout.

**Facebook Gatekeeper**: Facebook's system supports compound conditions
("show to employees OR users in canary group") and instant rollback.

**Stripe**: Uses flags extensively to test payment processing changes on a
fraction of transactions before full rollout, with automatic rollback if
error rates spike.
