# Event-Sourced Ledger

A financial ledger built with Event Sourcing + CQRS — every state change is an immutable event, and read models are projections rebuilt from the event stream.

## Docs

- [`docs/deep-dive.md`](./docs/deep-dive.md) — hydration loop, optimistic concurrency with version checks, CQRS projection pattern, snapshot pattern, pitfalls, when to use event sourcing

---

## Event Sourcing vs Traditional CRUD

```mermaid
graph LR
    subgraph CRUD
        REQ1[Transfer $100] --> DB1[(accounts table<br>balance = balance - 100)]
        Note1[State is mutable<br>history is lost]
    end
    
    subgraph Event Sourcing
        REQ2[Transfer $100] --> ES[(Event Store<br>append-only log)]
        ES --> E1[AccountOpened t=1]
        ES --> E2[MoneyDeposited $500 t=2]
        ES --> E3[MoneyTransferred $100 t=3]
        ES --> PROJ[Projection rebuilds current state]
    end
```

---

## CQRS (Command Query Responsibility Segregation)

```mermaid
graph TD
    CMD[Command<br>Deposit / Withdraw / Transfer] --> AGG[Aggregate<br>Account]
    AGG -->|validate + emit| ES[(Event Store<br>append-only)]
    
    ES -->|subscribe| P1[Balance Projection<br>read-optimized view]
    ES -->|subscribe| P2[Transaction History<br>read-optimized view]
    ES -->|subscribe| P3[Audit Log<br>compliance]
    
    QUERY[Query<br>GetBalance / GetHistory] --> P1
    QUERY --> P2
```

| Side | Responsibility | Optimized for |
|------|---------------|---------------|
| Command | Validate + append events | Write consistency |
| Query | Read from projections | Read performance |

---

## Event Store Design

```mermaid
sequenceDiagram
    participant Cmd as Command Handler
    participant Agg as Account Aggregate
    participant ES as Event Store
    participant Proj as Projections
    
    Cmd->>Agg: Deposit($100, account-1)
    Agg->>Agg: Validate (account exists, amount > 0)
    Agg->>ES: Append(MoneyDeposited{Amount:100, AccountID:"account-1"})
    ES->>ES: Store with version check (optimistic concurrency)
    ES-->>Proj: Notify new event
    Proj->>Proj: Update balance view: +$100
```

---

## Architecture

```mermaid
graph TD
    HTTP[HTTP API] --> CH[Command Handler]
    CH --> AGG[Account Aggregate<br>validation + business rules]
    AGG --> ES[(Event Store<br>in-memory / append-only)]
    
    ES --> BP[Balance Projection<br>map account→balance]
    ES --> HP[History Projection<br>map account→events]
    
    HTTP --> QH[Query Handler]
    QH --> BP
    QH --> HP
```

---

## Key Concepts

- **Event**: Immutable fact that happened (past tense: `MoneyDeposited`, `AccountOpened`)
- **Aggregate**: Domain object that validates commands and emits events
- **Event Store**: Append-only log with optimistic concurrency (version checks)
- **Projection**: Read model rebuilt by replaying events
- **Snapshot**: Periodic aggregate state capture to avoid replaying all events

---

## Running

```bash
make run
# Runs demo: opens accounts, deposits, transfers, shows projections
```
