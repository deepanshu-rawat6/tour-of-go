# Architecture

## Overview

Event-sourced financial ledger implementing CQRS (Command Query Responsibility Segregation).

## Components

```
cmd/main.go                         → Demo: open accounts, deposit, transfer, show projections
internal/eventstore/store.go        → Append-only event store with optimistic concurrency
internal/ledger/account.go          → Account aggregate (validates commands, emits events)
internal/projection/balance.go      → Balance projection (read model rebuilt from events)
```

## Event Flow

1. Command arrives (Deposit, Transfer)
2. Aggregate hydrates from event store (replay events → current state)
3. Aggregate validates business rules (sufficient funds?)
4. New event appended with version check (optimistic concurrency)
5. Projections rebuild by replaying all events

## Key Design Decisions

- **Append-only store**: Events are immutable facts, never updated or deleted
- **Optimistic concurrency**: Version check on append prevents conflicting writes
- **Hydration**: Aggregate state rebuilt from events (no mutable DB row)
- **Projections**: Read models are disposable — can be rebuilt from scratch at any time
