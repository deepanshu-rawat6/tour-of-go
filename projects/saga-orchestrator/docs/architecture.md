# Architecture

## Overview

The Saga Orchestrator implements the orchestration variant of the Saga pattern for distributed transactions across microservices.

## Components

```
cmd/main.go                    → Entry point, defines saga steps, runs scenarios
internal/saga/orchestrator.go  → Generic saga engine (step execution + compensation)
internal/services/services.go  → Mock microservice operations
```

## Flow

1. Client submits a multi-step operation (e.g., create order)
2. Orchestrator executes steps sequentially, tracking state
3. On failure at step N, compensates steps N-1 → 1 in reverse
4. Saga log records outcome for crash recovery

## Key Design Decisions

- **Interface-based steps**: Each step is an `Action` + `Compensate` function pair
- **Context propagation**: All steps receive context for cancellation/timeout
- **Data bag pattern**: Shared `map[string]any` passes data between steps
- **Idempotent compensation**: Compensating actions are safe to retry
