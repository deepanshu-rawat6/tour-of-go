# Kubernetes Controller (Greeting Operator)

A minimal Kubernetes operator built with `controller-runtime`. It watches a custom `Greeting` resource and creates a ConfigMap containing the greeting message.

## Concepts

- **CRD (Custom Resource Definition)**: Extends the Kubernetes API with your own resource types (`config/crd/greeting.yaml`)
- **Custom Resource (CR)**: An instance of your CRD (`config/samples/greeting_sample.yaml`)
- **Controller**: A reconciliation loop that watches CRs and drives the cluster toward desired state
- **Reconciliation Loop**: Called on every create/update/delete of a watched resource — must be idempotent
- **OwnerReference**: Links the ConfigMap to the Greeting so K8s auto-deletes it when the Greeting is deleted

## Prerequisites

```shell
# Install kind (local K8s cluster)
brew install kind

# Create a cluster
kind create cluster --name tour-of-go
```

## How to Run

```shell
# 1. Install the CRD into your cluster
kubectl apply -f config/crd/

# 2. Run the controller locally (uses your current kubeconfig)
make run

# 3. In another terminal, apply a sample Greeting
kubectl apply -f config/samples/

# 4. Watch the controller create a ConfigMap
kubectl get configmaps
kubectl describe configmap greeting-hello-gopher

# 5. Check the Greeting status
kubectl get greetings
```

## Expected Output

Controller logs:
```
INFO  Reconciling Greeting  {"name": "hello-gopher", "namespace": "default"}
INFO  Creating ConfigMap    {"configmap": "greeting-hello-gopher"}
INFO  Reconciliation complete {"configmap": "greeting-hello-gopher"}
```

```shell
$ kubectl get greetings
NAME            MESSAGE                                    READY
hello-gopher    Hello from my first Kubernetes Operator!  true

$ kubectl get configmaps | grep greeting
greeting-hello-gopher   1      5s
```

## Key Files

```
api/v1/greeting_types.go              # CRD Go types (Spec, Status)
api/v1/groupversion_info.go           # Group/Version registration
api/v1/zz_generated.deepcopy.go       # Generated DeepCopy methods
internal/controller/greeting_controller.go  # The reconciliation loop
main.go                               # Manager setup and startup
config/crd/greeting.yaml              # CRD YAML manifest
config/samples/greeting_sample.yaml   # Sample CR to apply
Makefile                              # generate, manifests, run, docker-build
```

## Reconciliation Loop Explained

```
Greeting CR created/updated
        ↓
Reconcile() called
        ↓
Fetch Greeting from API server
        ↓
Does ConfigMap exist?
  NO  → Create ConfigMap with greeting.spec.message
  YES → Message changed? → Update ConfigMap
        ↓
Update Greeting.status.ready = true
        ↓
Return (will be called again on next change)
```

## What to Learn Next

- Add a `Finalizer` to run cleanup logic before deletion
- Add validation webhooks with `kubebuilder:webhook`
- Watch a secondary resource (e.g., reconcile when a Pod changes)
- See [Platform Ops README](../../more-internals/system-design/platform-ops/README.md) for the theory
