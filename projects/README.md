# Projects

Standalone mini-projects — each is a separate Go module with its own `README.md`, `docs/`, and `Makefile`.

---

## Learning Path

```mermaid
graph TD
    BASICS[Go Basics\npackages → concurrency → context] --> INTERNALS[more-internals/\nruntime · design patterns · system design]
    INTERNALS --> P1[grpc-service\nProtobuf + RPC]
    INTERNALS --> P2[otel-tracing\nDistributed tracing]
    INTERNALS --> P3[k8s-controller\nOperator pattern]
    P1 & P2 & P3 --> P4[distributed-scheduler\nRedis lease · state machine]
    P4 --> P5[event-driven-pipeline\nNATS · exactly-once · DLQ]
    P4 --> P6[service-mesh-sidecar\nTCP proxy · circuit breaker]
    P5 & P6 --> P7[realtime-dashboard\nWebSocket · HTMX]
    P5 & P6 --> P8[platform-console\nSSE · client-go]
    P7 & P8 --> P9[cli-tui\nBubble Tea · Elm]
    P3 --> P10[aws-resource-reaper\nerrgroup · FinOps]
    P10 --> P11[gocker\nnamespaces · cgroups · OCI]
    P10 --> P12[tf-drift-detector\ndrift detection · webhooks]
    P4 --> P13[raft-kv-store\nRaft · WAL · gRPC]
    P11 --> P14[xdp-firewall\neBPF · XDP · RESP]
    P3 --> P15[k8s-event-sink\ninformers · dedup · SQLite]
    INTERNALS --> P16[secure-api\nSOLID · TDD · JWT · mTLS]
    INTERNALS --> P17[cache-service\nLRU · cache-aside · Redis]
    INTERNALS --> P18[rabbitmq-worker\nAMQP · DLX · prefetch]
    INTERNALS --> FS[from-scratch/\nTCP → HTTP → WS → cache → URL shortener]
```

---

## Project Index

| # | Project | What you build | Key concepts |
|---|---------|---------------|--------------|
| 1 | [`grpc-service`](./grpc-service/) | gRPC server + client | Protobuf, unary RPC, server streaming |
| 2 | [`otel-tracing`](./otel-tracing/) | Distributed tracing across 2 HTTP services | OpenTelemetry, trace propagation, spans |
| 3 | [`k8s-controller`](./k8s-controller/) | Kubernetes operator (CRD + controller) | controller-runtime, reconciliation loop, CRDs |
| 4 | [`distributed-scheduler`](./distributed-scheduler/) | Production distributed job scheduler | Redis lease, concurrency manager, Bleve search, state machine |
| 5 | [`event-driven-pipeline`](./event-driven-pipeline/) | Event processing pipeline | NATS JetStream, exactly-once, circuit breaker, DLQ |
| 6 | [`service-mesh-sidecar`](./service-mesh-sidecar/) | TCP proxy sidecar | Connection pooling, token bucket, circuit breaking, Prometheus |
| 7 | [`realtime-dashboard`](./realtime-dashboard/) | Live ops dashboard | HTMX, WebSocket, html/template, server-rendered UI |
| 8 | [`platform-console`](./platform-console/) | K8s resource browser | html/template, Tailwind, SSE, client-go dynamic client |
| 9 | [`cli-tui`](./cli-tui/) | Terminal dashboard | Bubble Tea, lipgloss, Elm architecture, TUI |
| 10 | [`aws-resource-reaper`](./aws-resource-reaper/) | Concurrent FinOps CLI | AWS SDK v2, STS AssumeRole, errgroup + semaphore, log/slog |
| 11 | [`gocker`](./gocker/) | Mini container runtime | Linux namespaces, OverlayFS, cgroups v1/v2, OCI pull, chroot |
| 12 | [`tf-drift-detector`](./tf-drift-detector/) | Terraform drift detection daemon | errgroup, sync.Mutex, time.Ticker, stateful tracker, webhooks |
| 13 | [`raft-kv-store`](./raft-kv-store/) | Distributed KV store via Raft | Leader election, log replication, WAL, gRPC, quorum commit |
| 14 | [`xdp-firewall`](./xdp-firewall/) | Kernel-level XDP packet filter | eBPF LPM trie, XDP_DROP, PERCPU_ARRAY, hexagonal architecture |
| 15 | [`k8s-event-sink`](./k8s-event-sink/) | Kubernetes event vacuum daemon | SharedIndexInformer, leaky bucket dedup, SQLite, Bleve, Slack |
| 16 | [`secure-api`](./secure-api/) | JWT + OAuth2 + mTLS HTTP API | SOLID principles, TDD, immutable value objects, JWT, bcrypt, mTLS |
| 17 | [`cache-service`](./cache-service/) | In-memory + Redis caching layer | LRU eviction, TTL reaper, cache-aside, write-through, singleflight |
| 18 | [`rabbitmq-worker`](./rabbitmq-worker/) | RabbitMQ task worker system | AMQP, durable queues, DLX, prefetch/QoS, manual ack, graceful shutdown |

### From Scratch Series

| # | Project | What you build | Key concepts |
|---|---------|---------------|--------------|
| FS-01 | [`from-scratch/01-tcp-server`](./from-scratch/01-tcp-server/) | Raw TCP echo server | `net.Listener`, goroutine-per-conn, `io.Copy` |
| FS-02 | [`from-scratch/02-http-server`](./from-scratch/02-http-server/) | HTTP/1.1 parser on TCP + stdlib | Request line parsing, routing, response writing |
| FS-03 | [`from-scratch/03-websocket-chat`](./from-scratch/03-websocket-chat/) | Multi-room WebSocket chat | Hub pattern, broadcast, room isolation |
| FS-04 | [`from-scratch/04-rate-limiter`](./from-scratch/04-rate-limiter/) | All 4 rate limiting algorithms | Token bucket, leaky bucket, fixed window, sliding window |
| FS-05 | [`from-scratch/05-load-balancer`](./from-scratch/05-load-balancer/) | L7 reverse proxy | Round-robin, least-connections, health checks |
| FS-06 | [`from-scratch/06-message-queue`](./from-scratch/06-message-queue/) | In-memory pub/sub + TCP server | Broker, topics, fan-out, custom text protocol |
| FS-07 | [`from-scratch/07-distributed-cache`](./from-scratch/07-distributed-cache/) | Redis-compatible KV store | RESP protocol, TTL eviction, `redis-cli` compatible |
| FS-08 | [`from-scratch/08-log-aggregator`](./from-scratch/08-log-aggregator/) | Log tail → ship → aggregate → query | File tailer, TCP shipper, in-memory store, HTTP search |
| FS-09 | [`from-scratch/09-task-scheduler`](./from-scratch/09-task-scheduler/) | Cron-like task scheduler | Cron parser, tick loop, HTTP API |
| FS-10 | [`from-scratch/10-url-shortener`](./from-scratch/10-url-shortener/) | URL shortener (capstone) | Integrates FS-04 + FS-07 + FS-06 + FS-09 |

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

---

### 16. secure-api

```mermaid
graph TD
    Client -->|POST /oauth2/token| TH[Token Handler]
    TH --> UA[UserAuthenticator\nport — ISP]
    UA --> US[UserStore\nbcrypt]
    TH --> TI[TokenIssuer\nport — ISP]
    TI --> JWT[JWTAdapter\nHS256]
    JWT -->|Bearer token| Client

    Client -->|GET /me\nAuthorization: Bearer| AM[Auth Middleware\nOCP — chain]
    AM --> TV[TokenValidator\nport — ISP]
    TV --> JWT
    AM -->|Claims in context\nimmutable value object| MH[Me Handler\nSRP]
    MH -->|user_id + roles| Client
```

JWT + OAuth2 password grant + optional mTLS HTTP API. Built with SOLID principles, TDD (table-driven tests), and immutable value objects.

---

### 17. cache-service

```mermaid
graph LR
    Client -->|GET PUT DELETE /cache/:key| H[HTTP Handler\nhit/miss stats]
    H --> SF[SingleflightCache\nstampede prevention]
    SF --> CA[CacheAside\nlazy loading]
    CA -->|hit| LRU[LRU Cache\ncontainer/list + map\nO1 evict + TTL reaper]
    CA -->|miss| ST[Backing Store\nMemory or Redis]
    ST -->|populate| LRU
    H -.->|CACHE_BACKEND=redis| R[Redis\ngo-redis/v9]
```

In-memory + Redis caching layer. Hand-rolled LRU cache with O(1) eviction and background TTL reaper. Three strategies: cache-aside, write-through, and singleflight stampede prevention.

---

### 18. rabbitmq-worker

```mermaid
graph LR
    P[Producer\ncmd/producer] -->|persistent JSON| EX[tasks\ndirect exchange]
    EX --> Q[tasks.queue\ndurable + DLX binding]
    Q -->|prefetch=5\nautoAck=false| W1[Worker 1]
    Q --> W2[Worker 2]
    Q --> W3[Worker 3]
    W1 -->|Ack| DONE[done]
    W2 -->|Nack requeue| Q
    W3 -->|Nack no-requeue| DLX[tasks.dlx\nfanout]
    DLX --> DLQ[tasks.dlq\ndead letters]
```

RabbitMQ task worker system. Durable exchange, DLX for failed messages after 3 retries, QoS prefetch for backpressure, graceful SIGTERM shutdown.
