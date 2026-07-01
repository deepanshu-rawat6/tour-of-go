# Saga Pattern

> Level: SDE-2 | Patterns: Distributed Transactions, Choreography, Orchestration

---

## Table of Contents

1. [What Sagas Are](#1-what-sagas-are)
2. [Choreography vs Orchestration](#2-choreography-vs-orchestration)
3. [Compensating Transactions](#3-compensating-transactions)
4. [Example: E-Commerce Order Saga](#4-example-e-commerce-order-saga)
5. [Isolation Issues](#5-isolation-issues)
6. [Idempotency](#6-idempotency)
7. [Saga State Machine](#7-saga-state-machine)
8. [Go Implementation](#8-go-implementation)
9. [Sagas vs Other Approaches](#9-sagas-vs-other-approaches)
10. [Real-World Examples](#10-real-world-examples)
11. [Integration with Event Sourcing](#11-integration-with-event-sourcing)
12. [Testing Sagas](#12-testing-sagas)

---

## 1. What Sagas Are

A **saga** is a sequence of local transactions that together implement a long-running distributed operation, without using a distributed two-phase commit (2PC).

Each step in a saga:
1. Executes a local transaction (within one service).
2. Publishes an event or sends a command to trigger the next step.
3. Has a corresponding **compensating transaction** that undoes the effect if a later step fails.

```
Saga = [T1, T2, T3, ..., Tn]
      where each Ti has a compensating transaction Ci

Happy path:  T1 → T2 → T3 → done
Failure at T3: T3 fails → C2 → C1 → saga failed (rolled back)
```

**Why not 2PC?** 2PC requires all participants to lock resources until all agree to commit. In microservices this means:
- Cross-service locks → tight coupling and latency.
- Coordinator becomes a single point of failure.
- Services must expose a prepare/commit API.

Sagas trade **ACID isolation** for **availability** — you get ACD (Atomicity, Consistency, Durability) but not Isolation.

---

## 2. Choreography vs Orchestration

### Choreography (decentralized)

Each service publishes domain events. Other services listen and react — there is no central coordinator.

```
OrderService          InventoryService       PaymentService         ShippingService
     │                       │                      │                      │
     │──OrderPlaced──────────►│                      │                      │
     │                       │──InventoryReserved───►│                      │
     │                       │                      │──PaymentCharged──────►│
     │                       │                      │                      │──ShipmentBooked
     │◄──────────────────────────────────────────────────────────────────────
```

**Compensations are also events:**
```
PaymentFailed ──► InventoryService: InventoryReleased
```

### Orchestration (centralized)

A dedicated **Saga Orchestrator** (process manager) sends commands to each service and receives replies.

```
                     ┌─────────────────────────────────┐
                     │         OrderOrchestrator        │
                     └─┬─────────────┬─────────────────┘
                       │             │
          ┌────────────▼──┐   ┌──────▼────────┐   ┌──────────────────┐
          │InventoryService│   │PaymentService │   │  ShippingService │
          └───────────────┘   └───────────────┘   └──────────────────┘
```

1. Orchestrator sends `ReserveInventory` command → receives `InventoryReserved` or `InventoryReservationFailed`.
2. On success: sends `ChargePayment` → receives `PaymentCharged` or `PaymentFailed`.
3. On failure at any step: orchestrator sends compensating commands in reverse.

### Trade-off table

| Dimension | Choreography | Orchestration |
|---|---|---|
| Coupling | Loose (services only know events) | Tighter (orchestrator knows all steps) |
| Visibility | Hard — saga flow is implicit | Clear — one place defines the flow |
| Single point of failure | None | Orchestrator (mitigated by HA) |
| Cyclic dependencies | Risk — event chains can loop | Lower risk |
| Testing | Hard to trace end-to-end | Easy — test orchestrator in isolation |
| Complexity | Grows with saga length | Linear in the orchestrator |
| When to use | Simple sagas, event-driven teams | Complex flows, ops need visibility |

---

## 3. Compensating Transactions

A compensating transaction **semantically undoes** a completed step. It is not a rollback — the original transaction committed; the compensation creates a new transaction that reverses the business effect.

| Step | Forward transaction | Compensating transaction |
|---|---|---|
| 1 | `ReserveInventory(orderId, items)` | `ReleaseInventory(orderId, items)` |
| 2 | `ChargePayment(orderId, amount)` | `RefundPayment(orderId, amount)` |
| 3 | `BookShipment(orderId)` | `CancelShipment(orderId)` |

**Key insight:** Some steps cannot be compensated (e.g., sending an email, calling an external API). Those are called **pivots**. Design your saga so pivots happen last and earlier steps are all compensatable.

---

## 4. Example: E-Commerce Order Saga

### Steps

```
1. OrderService:     Create order (status=pending)
2. InventoryService: Reserve inventory
3. PaymentService:   Charge payment
4. ShippingService:  Book shipment
5. OrderService:     Confirm order (status=confirmed)
```

### Choreography version (Go pseudocode)

```go
// OrderService — publishes OrderCreated
func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderReq) (*Order, error) {
    order := &Order{ID: newID(), Status: "pending", Items: req.Items}
    if err := s.repo.Save(ctx, order); err != nil {
        return nil, err
    }
    s.bus.Publish(OrderCreated{OrderID: order.ID, Items: order.Items, Total: order.Total})
    return order, nil
}

// InventoryService — listens for OrderCreated
func (s *InventoryService) OnOrderCreated(ctx context.Context, e OrderCreated) {
    if err := s.reserve(ctx, e.OrderID, e.Items); err != nil {
        s.bus.Publish(InventoryReservationFailed{OrderID: e.OrderID, Reason: err.Error()})
        return
    }
    s.bus.Publish(InventoryReserved{OrderID: e.OrderID})
}

// PaymentService — listens for InventoryReserved
func (s *PaymentService) OnInventoryReserved(ctx context.Context, e InventoryReserved) {
    order, _ := s.orderClient.Get(ctx, e.OrderID)
    if err := s.charge(ctx, order.UserID, order.Total); err != nil {
        s.bus.Publish(PaymentFailed{OrderID: e.OrderID, Reason: err.Error()})
        return
    }
    s.bus.Publish(PaymentCharged{OrderID: e.OrderID})
}

// InventoryService — compensates on PaymentFailed
func (s *InventoryService) OnPaymentFailed(ctx context.Context, e PaymentFailed) {
    s.release(ctx, e.OrderID) // compensation
    s.bus.Publish(InventoryReleased{OrderID: e.OrderID})
}

// OrderService — compensates on InventoryReservationFailed
func (s *OrderService) OnInventoryReservationFailed(ctx context.Context, e InventoryReservationFailed) {
    s.repo.UpdateStatus(ctx, e.OrderID, "failed")
}
```

### Orchestration version (Go pseudocode)

```go
type OrderSaga struct {
    orderID string
    state   string
    store   SagaStore
}

func (s *OrderSaga) Execute(ctx context.Context, orchestrator *Orchestrator) error {
    // Step 1: Reserve inventory
    s.setState(ctx, "reserving_inventory")
    if err := orchestrator.Send(ctx, ReserveInventoryCmd{OrderID: s.orderID}); err != nil {
        return s.compensate(ctx, orchestrator, 0)
    }

    // Step 2: Charge payment
    s.setState(ctx, "charging_payment")
    if err := orchestrator.Send(ctx, ChargePaymentCmd{OrderID: s.orderID}); err != nil {
        return s.compensate(ctx, orchestrator, 1)
    }

    // Step 3: Book shipment
    s.setState(ctx, "booking_shipment")
    if err := orchestrator.Send(ctx, BookShipmentCmd{OrderID: s.orderID}); err != nil {
        return s.compensate(ctx, orchestrator, 2)
    }

    s.setState(ctx, "completed")
    return nil
}

func (s *OrderSaga) compensate(ctx context.Context, orchestrator *Orchestrator, failedStep int) error {
    // Execute compensations in reverse order up to failedStep
    compensations := []Command{
        CancelShipmentCmd{OrderID: s.orderID},
        RefundPaymentCmd{OrderID: s.orderID},
        ReleaseInventoryCmd{OrderID: s.orderID},
    }
    for i := failedStep - 1; i >= 0; i-- {
        orchestrator.Send(ctx, compensations[i])
    }
    s.setState(ctx, "failed")
    return ErrSagaFailed
}
```

---

## 5. Isolation Issues

Sagas do NOT provide isolation between saga steps. Between steps, the partial state is visible to other transactions.

### Dirty reads

A second saga reads intermediate state written by step 1 of the first saga, before step 2 completes.

```
Saga A: T1(reserve inventory) ──────────────── T2(charge payment) ──► done
                                    ↑
Saga B:                    reads inventory as "reserved" → wrong decision
```

### Write skew

Two sagas both read the same resource, both decide there's enough, both proceed — but combined they exceed the limit.

### Countermeasures

| Countermeasure | Mechanism |
|---|---|
| **Semantic locks** | Mark resource as "pending" during saga; other sagas see it and back off |
| **Commutative updates** | Design updates so order doesn't matter (increment/decrement instead of set) |
| **Pessimistic view** | Assume worst-case in reads: treat "pending" inventory as if already consumed |
| **Re-read before commit** | Re-read the resource in the last step before committing |
| **By-value vs by-reference** | Pass values (snapshot) not references so intermediate changes don't affect saga |

### Semantic lock example

```go
// Step 1: mark as locked
UPDATE inventory SET status = 'pending', locked_by = $sagaID WHERE product_id = $1 AND status = 'available'
// Step 2 on success: release lock, decrement
UPDATE inventory SET status = 'available', quantity = quantity - $n, locked_by = NULL WHERE locked_by = $sagaID
// Compensation: release lock
UPDATE inventory SET status = 'available', locked_by = NULL WHERE locked_by = $sagaID
```

---

## 6. Idempotency

Every saga step must be **idempotent** — safe to execute multiple times (because messages can be delivered more than once).

```go
func (s *InventoryService) ReserveInventory(ctx context.Context, cmd ReserveInventoryCmd) error {
    // Idempotency key: if we already processed this saga step, skip
    exists, err := s.store.HasProcessed(ctx, cmd.SagaID, "ReserveInventory")
    if err != nil {
        return err
    }
    if exists {
        return nil // already done — safely skip
    }

    // Do the work
    if err := s.doReserve(ctx, cmd); err != nil {
        return err
    }

    // Mark as processed
    return s.store.MarkProcessed(ctx, cmd.SagaID, "ReserveInventory")
}
```

```sql
-- Idempotency table
CREATE TABLE saga_step_log (
    saga_id   TEXT NOT NULL,
    step_name TEXT NOT NULL,
    PRIMARY KEY (saga_id, step_name)
);
```

---

## 7. Saga State Machine

```
                   ┌──────────┐
     start ───────►│ pending  │
                   └────┬─────┘
                        │ inventory reserved
                   ┌────▼──────────────┐
                   │ inventory_reserved │
                   └────┬──────────────┘
                        │ payment charged
                   ┌────▼──────────────┐
                   │ payment_charged    │
                   └────┬──────────────┘
                        │ shipment booked
                   ┌────▼──────────────┐
                   │   completed        │◄── terminal
                   └───────────────────┘

Failure paths:
   pending ──────────────────────────────────────► failed (inventory failed — nothing to compensate)
   inventory_reserved ──► compensating ──► failed (payment failed → release inventory)
   payment_charged ──────► compensating ──► failed (shipment failed → refund + release)
```

```go
type SagaState string

const (
    SagaPending              SagaState = "pending"
    SagaInventoryReserved    SagaState = "inventory_reserved"
    SagaPaymentCharged       SagaState = "payment_charged"
    SagaCompleted            SagaState = "completed"
    SagaCompensating         SagaState = "compensating"
    SagaFailed               SagaState = "failed"
)
```

---

## 8. Go Implementation

### Interfaces

```go
// Step represents one unit of saga work + its compensation
type Step interface {
    Name()        string
    Execute(ctx context.Context, sagaCtx SagaContext) error
    Compensate(ctx context.Context, sagaCtx SagaContext) error
}

// SagaContext carries data shared between steps
type SagaContext map[string]interface{}

// SagaStore persists saga state durably
type SagaStore interface {
    Save(ctx context.Context, sagaID string, state SagaState, ctx SagaContext) error
    Load(ctx context.Context, sagaID string) (SagaState, SagaContext, error)
}

// Orchestrator drives execution
type Orchestrator struct {
    steps []Step
    store SagaStore
}

func NewOrchestrator(store SagaStore, steps ...Step) *Orchestrator {
    return &Orchestrator{steps: steps, store: store}
}

func (o *Orchestrator) Run(ctx context.Context, sagaID string, sagaCtx SagaContext) error {
    completed := 0
    for _, step := range o.steps {
        if err := step.Execute(ctx, sagaCtx); err != nil {
            // Save compensating state
            o.store.Save(ctx, sagaID, SagaCompensating, sagaCtx)
            // Compensate in reverse order
            for i := completed - 1; i >= 0; i-- {
                if cerr := o.steps[i].Compensate(ctx, sagaCtx); cerr != nil {
                    // Log and continue — best-effort compensation
                    log.Printf("compensation failed for step %s: %v", o.steps[i].Name(), cerr)
                }
            }
            o.store.Save(ctx, sagaID, SagaFailed, sagaCtx)
            return fmt.Errorf("saga failed at step %s: %w", step.Name(), err)
        }
        completed++
        o.store.Save(ctx, sagaID, SagaState(step.Name()+"_done"), sagaCtx)
    }
    o.store.Save(ctx, sagaID, SagaCompleted, sagaCtx)
    return nil
}
```

### Concrete steps

```go
type ReserveInventoryStep struct{ client InventoryClient }

func (s *ReserveInventoryStep) Name() string { return "reserve_inventory" }

func (s *ReserveInventoryStep) Execute(ctx context.Context, sc SagaContext) error {
    orderID := sc["order_id"].(string)
    items := sc["items"].([]OrderItem)
    return s.client.Reserve(ctx, orderID, items)
}

func (s *ReserveInventoryStep) Compensate(ctx context.Context, sc SagaContext) error {
    orderID := sc["order_id"].(string)
    return s.client.Release(ctx, orderID)
}

type ChargePaymentStep struct{ client PaymentClient }

func (s *ChargePaymentStep) Name() string { return "charge_payment" }

func (s *ChargePaymentStep) Execute(ctx context.Context, sc SagaContext) error {
    return s.client.Charge(ctx, sc["order_id"].(string), sc["amount"].(float64))
}

func (s *ChargePaymentStep) Compensate(ctx context.Context, sc SagaContext) error {
    return s.client.Refund(ctx, sc["order_id"].(string), sc["amount"].(float64))
}
```

### Wiring it together

```go
func CreateOrderSaga(inventorySvc InventoryClient, paymentSvc PaymentClient, shippingSvc ShippingClient, store SagaStore) *Orchestrator {
    return NewOrchestrator(store,
        &ReserveInventoryStep{client: inventorySvc},
        &ChargePaymentStep{client: paymentSvc},
        &BookShipmentStep{client: shippingSvc},
    )
}

// In the order handler:
sagaCtx := SagaContext{
    "order_id": orderID,
    "items":    order.Items,
    "amount":   order.Total,
}
if err := saga.Run(ctx, orderID, sagaCtx); err != nil {
    return nil, err
}
```

---

## 9. Sagas vs Other Approaches

| Approach | Consistency | Availability | Complexity | When to use |
|---|---|---|---|---|
| **2PC** | Strong (ACID) | Low (coordinator SPOF, locks) | Medium | Single DB or same vendor, latency-tolerant |
| **Saga** | Eventual (ACD) | High | Medium-High | Multi-service, long-running |
| **TCC (Try-Confirm-Cancel)** | Stronger than saga | Medium | High | Tight business consistency, financial |
| **Outbox Pattern** | At-least-once delivery | High | Low-Medium | Reliable event publishing, complement to saga |
| **Saga + Outbox** | Best of both | High | Higher | Production-grade distributed transactions |

### TCC overview

- **Try:** Reserve resources (tentative).
- **Confirm:** Commit if all services OK.
- **Cancel:** Release all reservations if any fail.

More like a 2PC but at the application layer. Each service exposes Try/Confirm/Cancel APIs.

### Outbox pattern (complement to sagas)

When publishing a saga event, write it to an **outbox table** in the same local transaction as the business data. A separate relay reads the outbox and publishes to the message bus.

```go
// In one transaction:
tx.Exec("INSERT INTO orders ...")
tx.Exec("INSERT INTO outbox(event_type, payload) VALUES ($1, $2)", "OrderCreated", payload)
tx.Commit()

// Relay (separate process):
for row := range outbox.Poll() {
    bus.Publish(row.EventType, row.Payload)
    outbox.MarkPublished(row.ID)
}
```

This guarantees **at-least-once** delivery without dual-write risk.

---

## 10. Real-World Examples

| System | Approach | Notes |
|---|---|---|
| **Uber Cadence / Temporal** | Orchestration | Durable workflow engine; replays function code on failures |
| **AWS Step Functions** | Orchestration | Managed state machine; each step = Lambda |
| **Apache Kafka + microservices** | Choreography | Events over Kafka topics; each service owns its consumer group |
| **Shopify** | Outbox + choreography | Transactional outbox + Kafka for order processing |
| **Netflix Conductor** | Orchestration | Open-source workflow orchestration engine |

### Temporal example sketch

```go
// Temporal workflow = orchestrated saga, fully durable
func OrderSagaWorkflow(ctx workflow.Context, input OrderInput) error {
    ao := workflow.ActivityOptions{StartToCloseTimeout: 10 * time.Second}
    ctx = workflow.WithActivityOptions(ctx, ao)

    var reserved bool
    if err := workflow.ExecuteActivity(ctx, ReserveInventoryActivity, input).Get(ctx, &reserved); err != nil {
        return err // Temporal auto-retries, then marks failed
    }

    var charged bool
    if err := workflow.ExecuteActivity(ctx, ChargePaymentActivity, input).Get(ctx, &charged); err != nil {
        workflow.ExecuteActivity(ctx, ReleaseInventoryActivity, input)
        return err
    }

    return workflow.ExecuteActivity(ctx, BookShipmentActivity, input).Get(ctx, nil)
}
```

---

## 11. Integration with Event Sourcing

When combining sagas with event sourcing:

1. The saga orchestrator is itself an event-sourced aggregate.
2. Saga state transitions are events stored in the event store.
3. Commands are sent via a command bus; responses come back as events.

```go
// Saga state stored as events
type SagaStarted      struct { SagaID string; Input OrderInput }
type StepCompleted    struct { SagaID string; StepName string }
type StepFailed       struct { SagaID string; StepName string; Err string }
type CompensationDone struct { SagaID string; StepName string }
type SagaCompleted    struct { SagaID string }
type SagaFailed       struct { SagaID string }
```

This gives you full observability: replay saga events to see exactly what happened, when, and why.

---

## 12. Testing Sagas

### Unit test each step independently

```go
func TestReserveInventoryStep_Execute_Success(t *testing.T) {
    mockClient := &mockInventoryClient{}
    mockClient.On("Reserve", "order-1", items).Return(nil)

    step := &ReserveInventoryStep{client: mockClient}
    err := step.Execute(context.Background(), SagaContext{"order_id": "order-1", "items": items})

    assert.NoError(t, err)
    mockClient.AssertExpectations(t)
}

func TestReserveInventoryStep_Compensate(t *testing.T) {
    mockClient := &mockInventoryClient{}
    mockClient.On("Release", "order-1").Return(nil)

    step := &ReserveInventoryStep{client: mockClient}
    err := step.Compensate(context.Background(), SagaContext{"order_id": "order-1"})

    assert.NoError(t, err)
}
```

### Test the happy path end-to-end

```go
func TestOrderSaga_HappyPath(t *testing.T) {
    inv  := newMockInventory().expectReserve("o1").succeed()
    pay  := newMockPayment().expectCharge("o1", 99.99).succeed()
    ship := newMockShipping().expectBook("o1").succeed()

    saga := CreateOrderSaga(inv, pay, ship, newInMemorySagaStore())
    err  := saga.Run(context.Background(), "o1", SagaContext{"order_id": "o1", "amount": 99.99})

    assert.NoError(t, err)
    inv.AssertReserved(t, "o1")
    pay.AssertCharged(t, "o1")
    ship.AssertBooked(t, "o1")
}
```

### Test the compensation path

```go
func TestOrderSaga_PaymentFails_CompensatesInventory(t *testing.T) {
    inv  := newMockInventory().expectReserve("o1").succeed().expectRelease("o1").succeed()
    pay  := newMockPayment().expectCharge("o1", 99.99).fail(errors.New("card declined"))
    ship := newMockShipping() // should never be called

    saga := CreateOrderSaga(inv, pay, ship, newInMemorySagaStore())
    err  := saga.Run(context.Background(), "o1", SagaContext{"order_id": "o1", "amount": 99.99})

    assert.Error(t, err)
    inv.AssertReleased(t, "o1")  // compensation ran
    ship.AssertNotCalled(t)
}
```

### Integration test with real message bus

Use a testcontainer (Kafka/Redis) to spin up a real broker, then publish and consume events end-to-end.

```go
func TestOrderSaga_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("integration test")
    }
    ctx := context.Background()
    broker := startTestKafka(t)
    // ... wire up full service graph against test broker
    // ... submit order, wait for completion event
    // ... assert final state
}
```
