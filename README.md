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

| # | Project | What you build | Key concepts |
|---|---------|---------------|--------------|
| 1 | [`grpc-service`](./projects/grpc-service/) | gRPC server + client | Protobuf, unary RPC, server streaming |
| 2 | [`otel-tracing`](./projects/otel-tracing/) | Distributed tracing across 2 HTTP services | OpenTelemetry, trace propagation, spans |
| 3 | [`k8s-controller`](./projects/k8s-controller/) | Kubernetes operator (CRD + controller) | controller-runtime, reconciliation loop, CRDs |
| 4 | [`distributed-scheduler`](./projects/distributed-scheduler/) | Production distributed job scheduler | Redis lease, concurrency manager, Bleve search, state machine |
| 5 | [`event-driven-pipeline`](./projects/event-driven-pipeline/) | Event processing pipeline | NATS JetStream, exactly-once, circuit breaker, DLQ |
| 6 | [`service-mesh-sidecar`](./projects/service-mesh-sidecar/) | TCP proxy sidecar | Connection pooling, token bucket, circuit breaking, Prometheus |
| 7 | [`realtime-dashboard`](./projects/realtime-dashboard/) | Live ops dashboard | HTMX, WebSocket, html/template, server-rendered UI |
| 8 | [`platform-console`](./projects/platform-console/) | K8s resource browser | html/template, Tailwind, SSE, client-go dynamic client |
| 9 | [`cli-tui`](./projects/cli-tui/) | Terminal dashboard | Bubble Tea, lipgloss, Elm architecture, TUI |
| 10 | [`aws-resource-reaper`](./projects/aws-resource-reaper/) | Concurrent FinOps CLI | AWS SDK v2, STS AssumeRole, errgroup + semaphore, log/slog |
| 11 | [`gocker`](./projects/gocker/) | Mini container runtime | Linux namespaces, OverlayFS, cgroups v1/v2, OCI pull, chroot |
| 12 | [`tf-drift-detector`](./projects/tf-drift-detector/) | Terraform drift detection daemon | errgroup, sync.Mutex, time.Ticker, stateful tracker, webhooks |
| 13 | [`raft-kv-store`](./projects/raft-kv-store/) | Distributed KV store via Raft | Leader election, log replication, WAL, gRPC, quorum commit |

---

## Project Architectures

### 1. grpc-service

```mermaid
sequenceDiagram
    participant C as Client
    participant S as gRPC Server
    C->>S: SayHello(name)
    S-->>C: HelloReply(message)
    C->>S: SayHelloStream(name)
    S-->>C: stream HelloReply × N
```

Protobuf-defined service with unary and server-streaming RPCs. The client demonstrates both call patterns.

---

### 2. otel-tracing

```mermaid
graph LR
    Client --> A[Service A\n:8080]
    A -->|HTTP + trace context| B[Service B\n:8081]
    A -->|OTLP| J[Jaeger\n:16686]
    B -->|OTLP| J
```

Two HTTP services propagate trace context via W3C `traceparent` headers. Both export spans to Jaeger via OTLP.

---

### 3. k8s-controller

```mermaid
graph TD
    User -->|kubectl apply| CRD[Greeting CRD\nkind: Greeting]
    CRD --> API[Kubernetes API Server]
    API -->|watch event| C[Greeting Controller\ncontroller-runtime]
    C -->|reconcile| CM[ConfigMap\ngreeting-config]
    C -->|update status| CRD
```

Custom Resource Definition + controller that reconciles `Greeting` objects into ConfigMaps. Demonstrates the operator pattern with controller-runtime.

---

### 4. distributed-scheduler

```mermaid
graph TD
    API[HTTP API] -->|submit job| S[Scheduler Service]
    S -->|acquire Redis lease| R[(Redis\nDistributed Lock)]
    R -->|leader only| W[Worker Pool\nsync.Mutex + goroutines]
    W -->|execute| D{Destination}
    D --> SQS[AWS SQS]
    D --> MEM[In-Memory]
    W -->|persist state| PG[(PostgreSQL\njob state machine)]
    W -->|heartbeat| R
    CRON[Cron Runner] -->|zombie detection| PG
    SEARCH[Bleve Search] -->|full-text index| PG
```

Production-grade job scheduler with Redis-based leader election, state machine (Pending → Running → Done/Failed), zombie detection, and full-text search.

---

### 5. event-driven-pipeline

```mermaid
graph LR
    P[Producer] -->|publish| N[NATS JetStream\nsubject: events.*]
    N -->|at-least-once| C[Consumer]
    C -->|idempotency check| R[(Redis\ndedup store)]
    R -->|new event| H[Pipeline Handler\ncircuit breaker]
    H -->|failed| DLQ[Dead Letter Queue]
    H -->|success| ACK[Ack to NATS]
```

Event processing pipeline with NATS JetStream for durable messaging, Redis-based idempotency (exactly-once semantics), circuit breaker, and DLQ for failed events.

---

### 6. service-mesh-sidecar

```mermaid
graph LR
    Client -->|TCP| S[Sidecar Proxy\n:8080]
    S --> RL[Token Bucket\nRate Limiter]
    RL --> CB[Circuit Breaker\nClosed/Open/Half-Open]
    CB -->|healthy| UP[Upstream Service\n:9090]
    CB -->|open| ERR[503 Error]
    S --> M[Prometheus Metrics\n:9091]
    S --> H[Health Checker\n/health]
```

TCP reverse proxy sidecar with token bucket rate limiting, circuit breaker (3 states), connection pooling, Prometheus metrics, and health checks.

---

### 7. realtime-dashboard

```mermaid
graph TD
    Browser -->|HTTP| H[HTTP Handler\nhtml/template]
    Browser -->|WebSocket| WS[WebSocket Hub]
    WS --> HUB[Hub\nsync.Mutex\nbroadcast channel]
    POLL[Scheduler Poller\ntime.Ticker] -->|job updates| HUB
    HUB -->|push| Browser
    Browser -->|HTMX swap| DOM[DOM Update\nno full reload]
```

Live ops dashboard for the distributed scheduler. WebSocket hub broadcasts job state changes to all connected browsers. HTMX swaps DOM fragments without a full page reload.

---

### 8. platform-console

```mermaid
graph TD
    Browser -->|HTTP GET /| H[Handler\nhtml/template + Tailwind]
    Browser -->|GET /watch SSE| W[K8s Watcher\nclient-go]
    W -->|watch.Event| SSE[SSE Stream\ntext/event-stream]
    SSE -->|push| Browser
    H --> K[K8s Dynamic Client\nclient-go]
    K -->|list Greetings| API[Kubernetes API Server]
```

Web console for browsing `Greeting` custom resources. Uses client-go's dynamic client for CRD listing and a Server-Sent Events stream for live updates.

---

### 9. cli-tui

```mermaid
graph TD
    Main[main.go] --> BT[Bubble Tea\nprogram.Start]
    BT --> M[Model\nElm Architecture]
    M -->|Init| CMD[tea.Cmd\nfetch jobs]
    CMD -->|HTTP| API[Scheduler API]
    API -->|jobs| MSG[tea.Msg]
    MSG -->|Update| M
    M -->|View| TUI[Terminal UI\nlipgloss styled]
```

Terminal dashboard for the distributed scheduler using Bubble Tea's Elm-inspired architecture (Model → Update → View). lipgloss handles styling and layout.

---

### 10. aws-resource-reaper

```mermaid
graph TD
    CLI[Cobra CLI\n--config --output --dry-run] --> CFG[YAML Config\naccounts + regions]
    CFG --> AUTH[Auth\nLoadDefaultConfig\nSTS AssumeRole]
    AUTH --> ENG[Discovery Engine\nerrgroup + semaphore]
    ENG --> S1[EBS Scanner]
    ENG --> S2[Elastic IP Scanner]
    ENG --> S3[EC2 Scanner]
    ENG --> S4[RDS Snapshot Scanner]
    ENG --> S5[ALB + CloudWatch Scanner]
    ENG --> S6[Security Group Scanner]
    S1 & S2 & S3 & S4 & S5 & S6 --> RULES[Rule Evaluator\nDelete / Recommend / Skip]
    RULES --> RPT[Reporter\ntext table or JSON]
    RPT --> EXEC{dry-run?}
    EXEC -->|false + confirm| DEL[Execute\nAWS delete APIs\nlog/slog audit]
```

Concurrent FinOps CLI. Runs on EC2/ECS with an instance profile, assumes roles into target accounts, scans 6 resource types across all regions in parallel, and reports or removes idle resources.

---

### 11. gocker

```mermaid
graph TD
    CLI[gocker pull / run / images / rmi] --> STORE[Image Store\n~/.gocker/images/name:tag/rootfs]
    CLI --> OCI[OCI Pull\nDocker Hub manifest\nlayer blobs → tar extract]
    OCI --> STORE
    STORE --> OVL[OverlayFS Mount\nlower=image read-only\nupper=container writable\nmerged=chroot target]
    OVL --> CG[cgroup Manager\nv1/v2 auto-detect\nmemory + cpu + pids]
    CG --> NS[Clone with Namespaces\nNEWPID NEWNS NEWUTS\nNEWIPC NEWNET]
    NS --> CHILD[Re-exec child\nchroot + mount /proc\nsyscall.Exec cmd]
    CHILD --> CLEANUP[defer cleanup\nunmount + delete cgroup\nremove container dir]
```

Mini Docker from scratch. Pulls real Alpine images via OCI spec, mounts OverlayFS for copy-on-write isolation, enforces resource limits via cgroups, and isolates processes with Linux namespaces.

---

### 12. tf-drift-detector

```mermaid
graph TD
    CLI[detect run / daemon] --> CFG[YAML Config\nbackend + alerts + ignore rules]
    CFG --> STATE[State Loader\nS3 or local tfstate]
    STATE --> POLL[Live Poller\nerrgroup fan-out]
    POLL --> C1[EC2 Comparator]
    POLL --> C2[S3 Comparator]
    POLL --> C3[SG Comparator]
    POLL --> C4[RDS Comparator]
    POLL --> C5[IAM Comparator]
    POLL --> C6[Lambda Comparator]
    C1 & C2 & C3 & C4 & C5 & C6 --> DIFF[Diff Engine\nhardcoded + config ignore\ntype coercion]
    DIFF --> TRK[Drift Tracker\nsync.Mutex\nnew vs known vs resolved]
    TRK --> ALERT[Alert Pipeline]
    ALERT --> SL[Slack Webhook]
    ALERT --> DC[Discord Webhook]
    ALERT --> OUT[stdout JSON]
    DAEMON[time.Ticker\nsignal.NotifyContext] --> STATE
    DAEMON -->|SIGTERM| PERSIST[Persist drift state\nto file]
```

Daemon that compares Terraform state against live AWS infrastructure. Stateful tracking alerts only on new/resolved drift. Hardcoded + config-driven false-positive suppression.

---

### 13. raft-kv-store

```mermaid
graph TD
    Client[HTTP Client] --> HTTP[HTTP API\nGET/PUT/DELETE /kv\nGET /status]
    HTTP -->|write on follower| REDIR[307 Redirect → Leader]
    HTTP -->|write on leader| RAFT[Raft Node]
    RAFT --> WAL[Write-Ahead Log\nappend + fsync]
    WAL --> GRPC[gRPC AppendEntries\nto all followers]
    GRPC --> F1[Follower 1]
    GRPC --> F2[Follower 2]
    GRPC --> FN[Follower N]
    F1 & F2 & FN -->|ack| RAFT
    RAFT -->|quorum| COMMIT[Commit → in-memory KV]
    COMMIT --> Client

    subgraph Election
        TIMER[Randomized timeout\n150-300ms] -->|fires| CAND[Candidate\nRequestVote RPCs]
        CAND -->|majority| LEAD[Leader\nheartbeat ticker]
    end
```

Distributed KV store implementing the Raft consensus algorithm from scratch. Leader election with randomized timeouts, log replication with quorum-based commits, WAL durability, and HTTP REST API.

---

### 14. xdp-firewall

```mermaid
graph TD
    CLI[xdp-fw start\n--interface eth0] --> LOAD[Load BPF ELF\nAttach XDP to NIC]
    LOAD --> RECONCILE[Reconcile\nrules.json vs kernel map]
    RECONCILE --> DAEMON[Go Daemon\nHTTP API + Metrics Poller]

    subgraph Kernel Space
        NIC[NIC Driver] --> XDP[XDP Program]
        XDP --> LPM[LPM Trie\nBPF_MAP_TYPE_LPM_TRIE]
        LPM -->|match| DROP[XDP_DROP]
        LPM -->|no match| PASS[XDP_PASS]
        XDP --> CNTR[PERCPU_ARRAY\ncounters]
    end

    DAEMON -->|POST /api/blacklist| ENGINE[ThreatEngine\ncore domain]
    ENGINE -->|Insert CIDR| LPM
    ENGINE -->|Save| FILE[rules.json]
    DAEMON -->|poll| CNTR
    CNTR -->|aggregate| PROM[Prometheus /metrics]
```

Kernel-level XDP firewall. Drops packets from blacklisted CIDRs at the NIC driver level using an eBPF LPM trie — before the Linux networking stack allocates memory. Hexagonal architecture with HTTP admin API, atomic file persistence, and Prometheus metrics.

---

### 15. k8s-event-sink

```mermaid
graph TD
    INF[K8s SharedIndexInformer\nwatch v1.Event] --> PROC[EventProcessor]
    PROC --> FILTER[Filter\nNormal → drop\nWarning → classify]
    FILTER --> DEDUP[Dedup Engine\nleaky bucket\nnamespace+pod+reason]
    DEDUP -->|first occurrence| FWD[Forward immediately]
    DEDUP -->|window expiry| SUMMARY[Summary alert\nsuppressed N events]
    FWD & SUMMARY --> SQLITE[(SQLite\ntime-series queries)]
    FWD & SUMMARY --> BLEVE[(Bleve\nfull-text search)]
    FWD & SUMMARY --> SLACK[Slack Webhook]
```

Kubernetes event vacuum daemon. Streams cluster events via informers, deduplicates with leaky bucket (first occurrence forwarded immediately, summary on window expiry), classifies severity, persists to embedded SQLite + Bleve. Single binary, zero external dependencies.

## Adding New Topics

```shell
mkdir mytopic
```

Create `mytopic/mytopic.go` with `Run()` and `RunExample(name string)` functions, then register in `main.go` with a `case "mytopic":` block.

---

