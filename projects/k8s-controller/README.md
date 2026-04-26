# Kubernetes Controller (Greeting Operator)

A minimal Kubernetes operator built with `controller-runtime`. It watches a custom `Greeting` resource and creates a ConfigMap containing the greeting message.

---

## Architecture

```mermaid
graph TD
    USER[kubectl apply -f greeting.yaml] --> API[Kubernetes API Server]
    API -->|watch event ADDED/MODIFIED| CTRL[Greeting Controller\ncontroller-runtime]
    CTRL -->|Reconcile| FETCH[Fetch Greeting CR]
    FETCH --> CHECK{ConfigMap exists?}
    CHECK -->|No| CREATE[Create ConfigMap\ngreeting-config]
    CHECK -->|Yes, message changed| UPDATE[Update ConfigMap]
    CHECK -->|Yes, in sync| NOOP[no-op]
    CREATE & UPDATE --> STATUS[Update Greeting.status.ready = true]
    STATUS --> API
```

## Reconciliation Loop

```mermaid
sequenceDiagram
    participant K as Kubernetes API
    participant C as Controller
    participant CM as ConfigMap

    K->>C: Reconcile(name: "hello-gopher")
    C->>K: Get Greeting "hello-gopher"
    K-->>C: Greeting{message: "Hello!"}
    C->>K: Get ConfigMap "greeting-hello-gopher"
    K-->>C: NotFound
    C->>K: Create ConfigMap{data: {message: "Hello!"}}
    K-->>C: Created
    C->>K: Update Greeting.status.ready = true
    Note over C: Returns — will be called again on next change
```

## Concepts

- **CRD** — extends the Kubernetes API with your own resource types (`config/crd/greeting.yaml`)
- **Custom Resource** — an instance of your CRD (`config/samples/greeting_sample.yaml`)
- **Reconciliation Loop** — called on every create/update/delete; must be idempotent
- **OwnerReference** — links the ConfigMap to the Greeting so K8s auto-deletes it when the Greeting is deleted

## Prerequisites

```shell
brew install kind
kind create cluster --name tour-of-go
```

## How to Run

```shell
kubectl apply -f config/crd/
make run
kubectl apply -f config/samples/
kubectl get greetings
kubectl get configmaps | grep greeting
```

## Key Files

```
api/v1/greeting_types.go                    # CRD Go types (Spec, Status)
internal/controller/greeting_controller.go  # The reconciliation loop
config/crd/greeting.yaml                    # CRD YAML manifest
config/samples/greeting_sample.yaml         # Sample CR to apply
```
