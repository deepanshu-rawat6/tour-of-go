# Tour of Go

![go-mascot](./.img/go.png)

A hands-on Go learning journal — from language basics to production-grade platform engineering, distributed systems, FinOps tooling, systems programming, and infrastructure automation.

---

## Learning Path (Recommended Order)

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

---

## Running Topics

```shell
go run . packages
go run . concurrency worker-pool
go run . context timeout
go run .              # show help
```

---

## Projects

Standalone mini-projects in `projects/` — each is a separate Go module with its own README and docs.

See **[`projects/README.md`](./projects/README.md)** for the full project index, learning path diagram, and architecture diagrams for all 18 projects + the from-scratch series.

## Adding New Topics

```shell
mkdir mytopic
```

Create `mytopic/mytopic.go` with `Run()` and `RunExample(name string)` functions, then register in `main.go` with a `case "mytopic":` block.

---

