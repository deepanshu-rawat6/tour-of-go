# Architecture

## Overview

Kubernetes operator implementing a custom controller for `Greeting` CRD using controller-runtime.

## Components

```
main.go                          → Manager setup, controller registration
api/v1alpha1/greeting_types.go   → CRD type definitions
internal/controller/greeting.go  → Reconciliation logic
config/                          → CRD YAML, RBAC, manager manifests
```

## Reconciliation Loop

1. User applies `Greeting` custom resource via kubectl
2. K8s API server stores the resource, fires a watch event
3. Controller receives event, enters `Reconcile()` function
4. Controller creates/updates a ConfigMap from the Greeting spec
5. Controller updates Greeting status with the result
6. If state drifts, controller re-reconciles (level-triggered)

## Key Concepts

- **CRD**: Custom Resource Definition — extends the K8s API
- **Controller**: Watches resources, reconciles desired vs actual state
- **Level-triggered**: Reconcile based on current state, not events (idempotent)
- **Owner references**: Garbage collection when parent resource is deleted
