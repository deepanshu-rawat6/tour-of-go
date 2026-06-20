# Saga Orchestrator

Distributed transactions across multiple services using the Saga pattern — both choreography and orchestration approaches implemented in Go.

## Docs

- [`docs/deep-dive.md`](./docs/deep-dive.md) — choreography vs orchestration trade-off, saga log durability, idempotent compensations, forward vs backward recovery, 2PC comparison

---

## Why Sagas?

In a monolith, a single DB transaction guarantees ACID. In microservices, each service owns its database — you can't use distributed transactions (2PC is slow and fragile). Sagas break a transaction into local transactions with compensating actions.

---

## Choreography vs Orchestration

```mermaid
graph TD
    subgraph Choreography [Choreography — Event-Driven]
        O1[Order Service] -->|OrderCreated| P1[Payment Service]
        P1 -->|PaymentCharged| I1[Inventory Service]
        I1 -->|InventoryReserved| S1[Shipping Service]
        S1 -->|ShipmentScheduled| O1
        
        I1 -->|InventoryFailed| P1
        P1 -->|PaymentRefunded| O1
    end
```

```mermaid
graph TD
    subgraph Orchestration [Orchestration — Central Coordinator]
        ORCH[Saga Orchestrator] -->|1. CreateOrder| O2[Order Service]
        ORCH -->|2. ChargePayment| P2[Payment Service]
        ORCH -->|3. ReserveInventory| I2[Inventory Service]
        ORCH -->|4. ScheduleShipping| S2[Shipping Service]
        
        I2 -->|failure| ORCH
        ORCH -->|compensate: RefundPayment| P2
        ORCH -->|compensate: CancelOrder| O2
    end
```

| Aspect | Choreography | Orchestration |
|--------|-------------|---------------|
| Coupling | Loose (events) | Central coordinator |
| Visibility | Hard to trace | Clear flow in one place |
| Complexity | Grows with services | Grows with steps |
| Best for | Simple flows (2-3 steps) | Complex flows (4+ steps) |

---

## Saga State Machine

```mermaid
stateDiagram-v2
    [*] --> Started
    Started --> OrderCreated: CreateOrder
    OrderCreated --> PaymentCharged: ChargePayment
    PaymentCharged --> InventoryReserved: ReserveInventory
    InventoryReserved --> Completed: ScheduleShipping
    
    OrderCreated --> CompensatingPayment: ChargePayment failed
    PaymentCharged --> CompensatingOrder: ReserveInventory failed
    InventoryReserved --> CompensatingInventory: ScheduleShipping failed
    
    CompensatingInventory --> CompensatingPayment: RefundPayment
    CompensatingPayment --> CompensatingOrder: CancelOrder
    CompensatingOrder --> Failed
    
    Completed --> [*]
    Failed --> [*]
```

---

## Architecture

```mermaid
graph TD
    API[HTTP API\nPOST /orders] --> ORCH[SagaOrchestrator]
    ORCH --> SM[State Machine\nstep tracking]
    SM --> LOG[Saga Log\nin-memory or DB]
    
    ORCH --> S1[OrderService.Create]
    ORCH --> S2[PaymentService.Charge]
    ORCH --> S3[InventoryService.Reserve]
    ORCH --> S4[ShippingService.Schedule]
    
    S2 -->|fail| COMP[Compensator]
    COMP --> C1[OrderService.Cancel]
    COMP --> C2[PaymentService.Refund]
    COMP --> C3[InventoryService.Release]
```

---

## Running

```bash
make run
# POST http://localhost:8080/orders with JSON body
# Watch saga execution and compensation in logs
```

## Key Concepts

- **Saga Step**: A local transaction + its compensating action
- **Compensating Transaction**: Undo a committed step (not rollback — semantic undo)
- **Saga Log**: Persists saga state for crash recovery
- **Idempotency**: Each step must be idempotent (safe to retry)
