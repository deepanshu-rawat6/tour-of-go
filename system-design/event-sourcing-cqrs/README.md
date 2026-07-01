# Event Sourcing & CQRS

> Level: SDE-2 | Patterns: Event Sourcing, CQRS, Sagas, Projections

---

## Table of Contents

1. [Event Sourcing Fundamentals](#1-event-sourcing-fundamentals)
2. [vs CRUD](#2-vs-crud)
3. [Building Blocks: Aggregate, Event, Command](#3-building-blocks-aggregate-event-command)
4. [Event Store](#4-event-store)
5. [Snapshots](#5-snapshots)
6. [CQRS](#6-cqrs)
7. [Projections](#7-projections)
8. [Eventual Consistency](#8-eventual-consistency)
9. [Sagas with Event Sourcing](#9-sagas-with-event-sourcing)
10. [Go Code: Full Example](#10-go-code-full-example)
11. [Trade-offs Table](#11-trade-offs-table)
12. [Common Mistakes](#12-common-mistakes)
13. [Event Versioning](#13-event-versioning)

---

## 1. Event Sourcing Fundamentals

Instead of storing the *current state* of an entity, you store the *sequence of events* that led to that state. The current state is always derived by replaying events.

```
Traditional DB row:   { id: "o1", status: "shipped", total: 99.99 }

Event Sourced store:
  t=1  OrderCreated   { order_id: "o1", user_id: "u1", total: 99.99 }
  t=2  PaymentTaken   { order_id: "o1", amount: 99.99, method: "card" }
  t=3  OrderShipped   { order_id: "o1", tracking: "TRK123" }
```

**Core invariants:**
- Events are **immutable** — you never update or delete them.
- The event log is **append-only**.
- Current state = `fold(initialState, events)`.
- Events are named in **past tense** (`OrderShipped`, not `ShipOrder`).

---

## 2. vs CRUD

| Dimension | CRUD | Event Sourcing |
|---|---|---|
| Storage | Current state only | Full history of changes |
| Auditability | Requires separate audit log | Built-in — events are the log |
| Temporal queries | Hard ("what was the state on Jan 1?") | Natural — replay up to a timestamp |
| Debugging | See current state only | Replay and inspect any past state |
| Write model | `UPDATE orders SET status='shipped'` | Append `OrderShipped` event |
| Complexity | Low | High — projections, snapshotting, versioning |
| Schema migration | Alter table | Upcast old events to new schema |
| Concurrency | Optimistic/pessimistic locks | Optimistic concurrency on version number |

**When CRUD is better:** Simple CRUD apps, small teams, no audit requirements, tight deadlines.

**When Event Sourcing wins:** Complex domain logic, audit trails required by compliance, temporal queries, multiple read models needed, event-driven microservices.

---

## 3. Building Blocks: Aggregate, Event, Command

### Command

An intent to change state. May be rejected.

```go
type CreateOrderCommand struct {
    OrderID string
    UserID  string
    Items   []OrderItem
}

type ShipOrderCommand struct {
    OrderID    string
    TrackingNo string
}
```

### Event

A fact that happened. Cannot be rejected (already happened).

```go
type Event interface {
    AggregateID() string
    EventType()   string
    OccurredAt()  time.Time
}

type OrderCreated struct {
    ID        string
    UserID    string
    Items     []OrderItem
    CreatedAt time.Time
}

func (e OrderCreated) AggregateID() string { return e.ID }
func (e OrderCreated) EventType()   string { return "OrderCreated" }
func (e OrderCreated) OccurredAt()  time.Time { return e.CreatedAt }
```

### Aggregate

An entity that enforces business rules, handles commands, and emits events.

```go
type OrderAggregate struct {
    ID      string
    Status  string
    Items   []OrderItem
    Total   float64
    Version int // for optimistic concurrency

    uncommitted []Event // events not yet persisted
}

func (o *OrderAggregate) Handle(cmd CreateOrderCommand) error {
    if len(cmd.Items) == 0 {
        return errors.New("order must have at least one item")
    }
    o.apply(OrderCreated{
        ID:        cmd.OrderID,
        UserID:    cmd.UserID,
        Items:     cmd.Items,
        CreatedAt: time.Now(),
    })
    return nil
}

func (o *OrderAggregate) Handle(cmd ShipOrderCommand) error {
    if o.Status != "confirmed" {
        return fmt.Errorf("cannot ship order in status %s", o.Status)
    }
    o.apply(OrderShipped{
        OrderID:    o.ID,
        TrackingNo: cmd.TrackingNo,
        ShippedAt:  time.Now(),
    })
    return nil
}

// apply mutates state from event — called both on command and replay
func (o *OrderAggregate) apply(e Event) {
    switch ev := e.(type) {
    case OrderCreated:
        o.ID     = ev.ID
        o.Items  = ev.Items
        o.Status = "pending"
        o.Total  = calculateTotal(ev.Items)
    case OrderShipped:
        o.Status = "shipped"
    }
    o.Version++
    o.uncommitted = append(o.uncommitted, e)
}

func (o *OrderAggregate) Uncommitted() []Event { return o.uncommitted }
func (o *OrderAggregate) ClearUncommitted()    { o.uncommitted = nil }
```

---

## 4. Event Store

The append-only log of all domain events.

### Interface

```go
type EventStore interface {
    // Append events; expectedVersion enables optimistic concurrency
    // Pass -1 to skip version check
    Append(ctx context.Context, aggregateID string, events []Event, expectedVersion int) error

    // Load all events for an aggregate
    Load(ctx context.Context, aggregateID string) ([]Event, error)

    // Load events after a version (for snapshot + tail pattern)
    LoadFrom(ctx context.Context, aggregateID string, fromVersion int) ([]Event, error)
}
```

### PostgreSQL implementation sketch

```sql
CREATE TABLE event_store (
    id             BIGSERIAL PRIMARY KEY,
    aggregate_id   TEXT        NOT NULL,
    aggregate_type TEXT        NOT NULL,
    version        INT         NOT NULL,
    event_type     TEXT        NOT NULL,
    payload        JSONB       NOT NULL,
    metadata       JSONB,
    occurred_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Optimistic concurrency: no two events for same aggregate+version
    UNIQUE (aggregate_id, version)
);

CREATE INDEX idx_event_store_aggregate ON event_store(aggregate_id, version);
```

```go
type pgEventStore struct{ db *sql.DB }

func (s *pgEventStore) Append(ctx context.Context, aggregateID string, events []Event, expectedVersion int) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Check current version (optimistic concurrency)
    if expectedVersion >= 0 {
        var current int
        err = tx.QueryRowContext(ctx,
            `SELECT COALESCE(MAX(version), -1) FROM event_store WHERE aggregate_id = $1`,
            aggregateID,
        ).Scan(&current)
        if err != nil {
            return err
        }
        if current != expectedVersion {
            return fmt.Errorf("concurrency conflict: expected version %d, got %d", expectedVersion, current)
        }
    }

    for i, e := range events {
        payload, _ := json.Marshal(e)
        _, err = tx.ExecContext(ctx, `
            INSERT INTO event_store(aggregate_id, aggregate_type, version, event_type, payload, occurred_at)
            VALUES ($1, $2, $3, $4, $5, $6)`,
            aggregateID,
            aggregateTypeOf(e),
            expectedVersion+1+i,
            e.EventType(),
            payload,
            e.OccurredAt(),
        )
        if err != nil {
            return err
        }
    }
    return tx.Commit()
}

func (s *pgEventStore) Load(ctx context.Context, aggregateID string) ([]Event, error) {
    rows, err := s.db.QueryContext(ctx,
        `SELECT event_type, payload FROM event_store WHERE aggregate_id = $1 ORDER BY version`,
        aggregateID,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    return deserializeEvents(rows)
}
```

---

## 5. Snapshots

Replaying 10,000 events on every load is expensive. Snapshots cache aggregate state periodically.

### When to snapshot

- Every N events (e.g., every 50 or 100).
- When aggregate version exceeds a threshold.
- On a schedule for rarely-updated but frequently-read aggregates.

### Snapshot + tail pattern

```go
type SnapshotStore interface {
    Save(ctx context.Context, aggregateID string, version int, state []byte) error
    Load(ctx context.Context, aggregateID string) (version int, state []byte, err error)
}

func LoadAggregate(ctx context.Context, id string, es EventStore, ss SnapshotStore) (*OrderAggregate, error) {
    agg := &OrderAggregate{}

    // Try snapshot first
    snapVersion, snapState, err := ss.Load(ctx, id)
    if err == nil {
        json.Unmarshal(snapState, agg)
        agg.Version = snapVersion
    }

    // Load only events after snapshot version
    events, err := es.LoadFrom(ctx, id, agg.Version)
    if err != nil {
        return nil, err
    }
    for _, e := range events {
        agg.apply(e)
    }

    // Optionally save new snapshot if many events loaded
    if len(events) > 50 {
        state, _ := json.Marshal(agg)
        ss.Save(ctx, id, agg.Version, state)
    }

    return agg, nil
}
```

---

## 6. CQRS

**Command Query Responsibility Segregation** — split your model into:

- **Write model:** handles commands, enforces business rules, writes to event store.
- **Read model:** optimised for queries, built by projecting events into read-friendly structures.

```
         ┌─────────────┐     Command      ┌──────────────────┐
 Client ─►│  API Layer  ├─────────────────►│ Command Handler  │
         └─────────────┘                  └────────┬─────────┘
                                                   │ events
                                          ┌────────▼─────────┐
                                          │   Event Store    │
                                          └────────┬─────────┘
                                                   │ events
                                          ┌────────▼─────────┐
                                          │    Projector     │
                                          └────────┬─────────┘
                                                   │ writes
                                          ┌────────▼─────────┐
         ┌─────────────┐      Query       │   Read Store     │
 Client ─►│  API Layer  ├─────────────────► (Postgres/Redis) │
         └─────────────┘                  └──────────────────┘
```

### Why separate?

| Reason | Explanation |
|---|---|
| Write contention | Aggregates lock each other during writes; reads don't need that contention |
| Query complexity | Read models can be denormalized — exactly what the UI needs |
| Independent scaling | Read side can be replicated/scaled independently of write side |
| Different storage | Writes go to event store; reads can use Redis, Elasticsearch, etc. |

### Command Handler

```go
type CreateOrderHandler struct {
    store     EventStore
    snapshots SnapshotStore
}

func (h *CreateOrderHandler) Handle(ctx context.Context, cmd CreateOrderCommand) error {
    agg, err := LoadAggregate(ctx, cmd.OrderID, h.store, h.snapshots)
    if err != nil {
        return err
    }
    if err := agg.Handle(cmd); err != nil {
        return err
    }
    return h.store.Append(ctx, cmd.OrderID, agg.Uncommitted(), agg.Version-len(agg.Uncommitted()))
}
```

### Read Model (Query Side)

```go
// Denormalized view — exactly what the orders list page needs
type OrderSummary struct {
    ID         string
    UserName   string // denormalized from users
    Status     string
    Total      float64
    ItemCount  int
    CreatedAt  time.Time
}

type OrderQueryService struct {
    db *sql.DB // read-optimized store
}

func (s *OrderQueryService) ListByUser(ctx context.Context, userID string) ([]*OrderSummary, error) {
    rows, err := s.db.QueryContext(ctx,
        `SELECT id, user_name, status, total, item_count, created_at
         FROM order_summaries WHERE user_id = $1 ORDER BY created_at DESC`,
        userID,
    )
    // ...
}
```

---

## 7. Projections

Projections consume events and build read models. They must be **idempotent** — replaying an event twice must not produce wrong state.

```go
type Projector interface {
    Project(ctx context.Context, event Event) error
}

type OrderSummaryProjector struct {
    db *sql.DB
}

func (p *OrderSummaryProjector) Project(ctx context.Context, e Event) error {
    switch ev := e.(type) {
    case OrderCreated:
        _, err := p.db.ExecContext(ctx, `
            INSERT INTO order_summaries(id, user_id, status, total, item_count, created_at)
            VALUES ($1, $2, 'pending', $3, $4, $5)
            ON CONFLICT (id) DO NOTHING`, // idempotency: ignore if already projected
            ev.ID, ev.UserID, ev.Total, len(ev.Items), ev.CreatedAt,
        )
        return err

    case OrderShipped:
        _, err := p.db.ExecContext(ctx,
            `UPDATE order_summaries SET status = 'shipped' WHERE id = $1`,
            ev.OrderID,
        )
        return err
    }
    return nil
}
```

### Rebuilding projections

Projections can always be rebuilt from scratch by replaying all events. This is a superpower — you can add a new read model, backfill it, then cut over.

```go
func RebuildProjection(ctx context.Context, es EventStore, projector Projector, allAggregateIDs []string) error {
    for _, id := range allAggregateIDs {
        events, err := es.Load(ctx, id)
        if err != nil {
            return fmt.Errorf("load %s: %w", id, err)
        }
        for _, e := range events {
            if err := projector.Project(ctx, e); err != nil {
                return fmt.Errorf("project event %s/%s: %w", id, e.EventType(), err)
            }
        }
    }
    return nil
}
```

---

## 8. Eventual Consistency

The write model (event store) and read models (projections) are eventually consistent. There's a lag between appending an event and the projection being updated.

### Communicating lag to clients

**Option 1: Polling**
```
POST /orders → 202 Accepted + { "order_id": "o1" }
Client polls GET /orders/o1 until status != "pending"
```

**Option 2: Long-polling**
```
GET /orders/o1?wait=true  (blocks until status changes or timeout)
```

**Option 3: Server-Sent Events / WebSocket**
```
Client subscribes to /events/orders/o1
Server pushes OrderShipped event when it happens
```

**Option 4: Optimistic UI**
Client assumes success immediately and updates local state.

**Interview answer:** In CQRS with event sourcing, reads are eventually consistent. For reads-after-writes that need strong consistency, either read from the write model directly (if acceptable) or use synchronous projection updates in the same transaction (which sacrifices some separation).

---

## 9. Sagas with Event Sourcing

A **Process Manager** (or Saga) is itself an aggregate that reacts to events from multiple aggregates and coordinates multi-step workflows.

```go
type OrderProcessManager struct {
    OrderID     string
    State       string // "awaiting_payment" | "awaiting_shipment" | "complete" | "compensating"
    Version     int
    uncommitted []Event
}

func (pm *OrderProcessManager) On(e Event) []Command {
    switch ev := e.(type) {
    case OrderCreated:
        pm.State = "awaiting_payment"
        return []Command{ChargePaymentCommand{OrderID: ev.ID, Amount: ev.Total}}

    case PaymentSucceeded:
        pm.State = "awaiting_shipment"
        return []Command{BookShipmentCommand{OrderID: ev.OrderID}}

    case PaymentFailed:
        pm.State = "compensating"
        return []Command{CancelOrderCommand{OrderID: ev.OrderID}}

    case ShipmentBooked:
        pm.State = "complete"
        return nil
    }
    return nil
}
```

---

## 10. Go Code: Full Example

```go
// ---- domain/order.go ----

package domain

import (
    "errors"
    "time"
)

// Events
type OrderCreated struct {
    ID        string      `json:"id"`
    UserID    string      `json:"user_id"`
    Items     []OrderItem `json:"items"`
    Total     float64     `json:"total"`
    CreatedAt time.Time   `json:"created_at"`
}
func (e OrderCreated) AggregateID() string  { return e.ID }
func (e OrderCreated) EventType() string    { return "OrderCreated" }
func (e OrderCreated) OccurredAt() time.Time { return e.CreatedAt }

type OrderShipped struct {
    OrderID    string    `json:"order_id"`
    TrackingNo string    `json:"tracking_no"`
    ShippedAt  time.Time `json:"shipped_at"`
}
func (e OrderShipped) AggregateID() string  { return e.OrderID }
func (e OrderShipped) EventType() string    { return "OrderShipped" }
func (e OrderShipped) OccurredAt() time.Time { return e.ShippedAt }

// Aggregate
type Order struct {
    ID          string
    UserID      string
    Status      string
    Total       float64
    Items       []OrderItem
    Version     int
    uncommitted []Event
}

func NewOrder() *Order { return &Order{} }

func (o *Order) Create(id, userID string, items []OrderItem) error {
    if len(items) == 0 {
        return errors.New("at least one item required")
    }
    total := 0.0
    for _, it := range items {
        total += it.Price * float64(it.Quantity)
    }
    o.raise(OrderCreated{ID: id, UserID: userID, Items: items, Total: total, CreatedAt: time.Now()})
    return nil
}

func (o *Order) Ship(trackingNo string) error {
    if o.Status != "confirmed" {
        return fmt.Errorf("cannot ship order in status %q", o.Status)
    }
    o.raise(OrderShipped{OrderID: o.ID, TrackingNo: trackingNo, ShippedAt: time.Now()})
    return nil
}

func (o *Order) raise(e Event) {
    o.apply(e)
    o.uncommitted = append(o.uncommitted, e)
}

func (o *Order) apply(e Event) {
    switch ev := e.(type) {
    case OrderCreated:
        o.ID, o.UserID, o.Items, o.Total, o.Status = ev.ID, ev.UserID, ev.Items, ev.Total, "pending"
    case OrderShipped:
        o.Status = "shipped"
    }
    o.Version++
}

func (o *Order) Rehydrate(events []Event) {
    for _, e := range events {
        o.apply(e)
    }
}

func (o *Order) Uncommitted() []Event  { return o.uncommitted }
func (o *Order) ClearUncommitted()     { o.uncommitted = nil }
```

---

## 11. Trade-offs Table

| Factor | Event Sourcing | Traditional CRUD |
|---|---|---|
| Audit log | Free | Extra work |
| Temporal queries | Natural | Hard |
| Debug production | Replay events | Add logging |
| New read models | Replay and build | Write migration |
| Schema evolution | Upcasters for old events | ALTER TABLE |
| Operational complexity | High | Low |
| Team learning curve | High | Low |
| Write throughput | High (append-only) | Medium |
| Storage cost | Higher (full history) | Lower |
| Consistency | Eventual (CQRS) | Strong (default) |

---

## 12. Common Mistakes

### Too many fine-grained events

```
BAD:  UserFirstNameChanged, UserLastNameChanged, UserEmailChanged
GOOD: UserProfileUpdated { first_name, last_name, email }
```

Think domain events, not field-level diffs.

### Leaking infrastructure into events

```
BAD:  OrderDatabaseRowInserted
GOOD: OrderPlaced
```

### Not handling idempotency in projections

Always use `ON CONFLICT DO NOTHING` or check-then-update with a processed event ID.

### Forgetting about event ordering

Events within one aggregate are ordered by version. Across aggregates, there is no global order — use wall-clock time with caution; use logical clocks (Lamport, vector clocks) if ordering matters.

---

## 13. Event Versioning

Events are immutable, but schemas evolve. Strategies:

### Weak schema (add-only)

Only add new optional fields to events. Consumers that don't know about new fields ignore them.

```go
// v1
type OrderCreated struct {
    ID    string `json:"id"`
    Total float64 `json:"total"`
}

// v2: added discount_code — old events just have empty string
type OrderCreated struct {
    ID           string  `json:"id"`
    Total        float64 `json:"total"`
    DiscountCode string  `json:"discount_code,omitempty"` // new field
}
```

### Versioned event types

```go
type OrderCreatedV1 struct { ... }
type OrderCreatedV2 struct { ... } // breaking change: renamed a field
```

Store `event_type = "OrderCreated.v2"` in the event store.

### Upcaster pattern

Transform old events to new format at read time:

```go
type Upcaster interface {
    Upcast(raw json.RawMessage) (Event, error)
}

type OrderCreatedUpcaster struct{}

func (u *OrderCreatedUpcaster) Upcast(raw json.RawMessage) (Event, error) {
    // Try v2 first
    var v2 OrderCreatedV2
    if err := json.Unmarshal(raw, &v2); err == nil && v2.SomeV2Field != "" {
        return v2, nil
    }
    // Fall back to v1 → convert to v2
    var v1 OrderCreatedV1
    json.Unmarshal(raw, &v1)
    return OrderCreatedV2{
        ID:          v1.ID,
        Total:       v1.Total,
        SomeV2Field: "default_value",
    }, nil
}
```

Upcasters run in the `EventStore.Load()` path — the aggregate always sees the latest event shape.

### Golden rule

**Never change the meaning of an existing field.** Add new fields. Deprecate old ones by ignoring them. Only create a new event type (v2) when a change is truly breaking.
