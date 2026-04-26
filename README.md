# Tour of Go

![go-mascot](./.img/go.png)

A hands-on Go learning journal — from language basics to platform engineering, Kubernetes operators, FinOps tooling, systems programming, and infrastructure automation.

## Learning Path (Recommended Order)

Follow this sequence to go from Go basics to building production-grade platform tools:

```
1. packages              → Variables, functions, types, constants
2. flow_control_statements → For, if, switch, defer
3. more_types            → Pointers, structs, slices, maps, closures
4. methods               → Value/pointer receivers, fmt.Stringer
5. interfaces            → Implicit satisfaction, type assertions, embedding
6. error_handling        → Custom errors, wrapping (%w), panic/recover
7. generics              → Type parameters, constraints, generic types
8. concurrency           → Goroutines, channels, select, mutex, worker pool
9. context               → Cancellation, timeouts, request-scoped values
   ↓
more-internals/          → Deep dives: Go runtime, design patterns, system design
   ↓
projects/                → Runnable platform projects
```

## Advanced Guides & Internals

For deep-dives into the Go runtime, idiomatic design patterns, and system design for Platform Engineering, check out our [**Master Table of Contents**](./more-internals/README.md).

### 🟢 Phase 1: Go Internals
Master the runtime mechanics: `defer`, Memory Layout, `cgo`, and Plan9 Assembly.

### 🔵 Phase 2: Design Patterns
Idiomatic patterns: Functional Options, Plugin Architectures, and the Repository Pattern.

### 🔴 Phase 3: System Design & Platform Ops
Architecting for scale: eBPF, Gossip Protocols, Distributed Tracing, and K8s-native services.

## Running Topics

### Run all examples in a topic

```shell
go run . packages
go run . concurrency
go run . context
```

### Run a specific example

```shell
go run . packages basic
go run . interfaces type-assertions
go run . error_handling wrapping
go run . concurrency worker-pool
go run . context timeout
```

### Show help

```shell
go run .
```

## Building

```shell
go build .
./tour_of_go packages
./tour_of_go concurrency worker-pool
```

## Advanced Runnable Snippets

The `more-internals/runnable/` directory contains executable Go programs for advanced topics:

```shell
go run ./more-internals/runnable/concurrency-patterns/   # Pipeline, fan-out/fan-in
go run ./more-internals/runnable/design-patterns/        # Functional options, circuit breaker, singleflight
go run ./more-internals/runnable/system-design/          # Rate limiter simulation
```

## Projects

Standalone mini-projects in `projects/` — each is a separate Go module with its own README and docs:

| Project | What you build | Key concepts |
|---------|---------------|--------------|
| [`projects/grpc-service/`](./projects/grpc-service/) | gRPC server + client | Protobuf, unary RPC, server streaming |
| [`projects/otel-tracing/`](./projects/otel-tracing/) | Distributed tracing across 2 HTTP services | OpenTelemetry, trace propagation, spans |
| [`projects/k8s-controller/`](./projects/k8s-controller/) | Kubernetes operator (CRD + controller) | controller-runtime, reconciliation loop, CRDs |
| [`projects/distributed-scheduler/`](./projects/distributed-scheduler/) | Production distributed job scheduler | Redis lease, concurrency manager, Bleve search, state machine, zombie detection |
| [`projects/event-driven-pipeline/`](./projects/event-driven-pipeline/) | Event processing pipeline | NATS JetStream, exactly-once, circuit breaker, DLQ, OTel tracing |
| [`projects/service-mesh-sidecar/`](./projects/service-mesh-sidecar/) | TCP proxy sidecar | Connection pooling, token bucket, circuit breaking, health checks, Prometheus |
| [`projects/realtime-dashboard/`](./projects/realtime-dashboard/) | Live ops dashboard for job scheduler | HTMX, WebSocket, html/template, server-rendered UI |
| [`projects/platform-console/`](./projects/platform-console/) | K8s Greeting resource browser | html/template, Tailwind, SSE, client-go dynamic client |
| [`projects/cli-tui/`](./projects/cli-tui/) | Terminal dashboard for job scheduler | Bubble Tea, lipgloss, Elm architecture, TUI |
| [`projects/aws-resource-reaper/`](./projects/aws-resource-reaper/) | Concurrent FinOps CLI — scans and cleans idle AWS resources across accounts | AWS SDK v2, STS AssumeRole, errgroup + semaphore, Cobra CLI, log/slog |
| [`projects/gocker/`](./projects/gocker/) | Mini container runtime (mini Docker from scratch) | Linux namespaces, OverlayFS, cgroups v1/v2, OCI image pull, chroot, re-exec trick |
| [`projects/tf-drift-detector/`](./projects/tf-drift-detector/) | Terraform drift detector daemon — compares TF state vs live AWS infrastructure | errgroup fan-out, sync.Mutex, time.Ticker, signal.NotifyContext, stateful tracker, webhook alerting |
| [`projects/raft-kv-store/`](./projects/raft-kv-store/) | Distributed KV store using Raft consensus from scratch | Leader election, log replication, WAL, gRPC transport, HTTP API, quorum commit |

---

## Adding New Topics

### 1. Create a new directory and dispatcher

```shell
mkdir mytopic
```

Create `mytopic/mytopic.go`:

```go
package mytopic

import (
    "fmt"
    "os"
)

func Run() {
    fmt.Println("=== My Topic ===")
    myExample()
}

func RunExample(name string) {
    fmt.Printf("=== My Topic: %s ===\n\n", name)
    switch name {
    case "my-example":
        myExample()
    default:
        fmt.Printf("Unknown example: %s\n", name)
        os.Exit(1)
    }
}
```

### 2. Add example files

Create `mytopic/my-example.go`:

```go
package mytopic

import "fmt"

func myExample() {
    fmt.Println("My example output")
}
```

### 3. Register in main.go

Add the import and a `case "mytopic":` block in the switch statement.
