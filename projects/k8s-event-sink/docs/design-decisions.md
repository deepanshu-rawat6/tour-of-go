# Design Decisions — k8s-event-sink

## 1. Leaky bucket over fixed time window

**Decision:** Use a leaky bucket where the first event passes immediately and subsequent events are suppressed until the window expires, rather than a fixed window that batches all events.

**Why:** During an incident, you need to know immediately when the first OOMKill happens — not 5 minutes later when the window closes. The leaky bucket gives you instant notification on first occurrence and a summary on window expiry. A fixed window would delay the first alert by up to the full window duration.

---

## 2. SQLite + Bleve over PostgreSQL

**Decision:** Use embedded SQLite (structured queries) and Bleve (full-text search) instead of an external PostgreSQL database.

**Why:** A Kubernetes daemon with a hard dependency on an external database is fragile — if Postgres is down, the event sink stops working. SQLite and Bleve run embedded in the Go binary, storing data on a PersistentVolume. The daemon is self-contained: one container image, one PV, zero external services. `modernc.org/sqlite` is pure Go (no CGo), so the binary cross-compiles cleanly.

---

## 3. modernc.org/sqlite over mattn/go-sqlite3

**Decision:** Use `modernc.org/sqlite` (pure Go transpilation of SQLite) instead of `mattn/go-sqlite3` (CGo wrapper).

**Why:** CGo breaks cross-compilation. `mattn/go-sqlite3` requires a C compiler and the SQLite C source at build time. `modernc.org/sqlite` is a pure Go port — the same binary builds on macOS and runs on Alpine Linux in a container without any C toolchain. This is the same reasoning as choosing `cilium/ebpf` over `libbpfgo` in the xdp-firewall project.

---

## 4. Hardcoded severity defaults + config overrides

**Decision:** Ship a built-in severity map for common K8s event reasons, with YAML config overrides.

**Why:** Without defaults, every user must configure the same obvious mappings (OOMKilled → critical, CrashLoopBackOff → critical). With defaults only, users can't suppress noisy events specific to their environment. The combination means the tool works correctly out of the box and adapts to any cluster's noise profile via a ConfigMap.

---

## 5. One informer per namespace, not one global informer

**Decision:** When `namespaces: ["ns1", "ns2"]` is configured, spin up one `SharedIndexInformer` per namespace rather than a single cluster-wide informer with client-side filtering.

**Why:** A cluster-wide informer requires `ClusterRole` — broad permissions that security teams often block. Per-namespace informers work with standard `Role` + `RoleBinding` scoped to specific namespaces. The RBAC blast radius is minimal. When `namespaces: ["*"]` is configured, a single cluster-wide informer is used for efficiency.

---

## 6. Bucket key = namespace:pod:reason

**Decision:** The dedup bucket key is `namespace + pod + reason`, not just `reason` or `pod`.

**Why:** `reason` alone would deduplicate OOMKills across all pods — you'd miss that 10 different pods are all OOMKilling simultaneously (which is a node-level problem). `pod` alone would deduplicate all events for a pod regardless of reason. The three-part key correctly identifies "this specific failure mode on this specific pod in this namespace" as one logical event stream.

---

## 7. Filter before dedup, not after

**Decision:** The `Filter` runs before the `DedupEngine`. Dropped events never enter the bucket map.

**Why:** If Normal events entered the dedup engine, they'd create buckets that consume memory and never expire (since Normal events are dropped before forwarding). Filtering first means the dedup engine only sees events that will actually be forwarded — the bucket map stays small and the memory footprint is bounded by the number of active Warning/Critical events.

---

## 8. Summary event on window expiry, not a separate notification

**Decision:** The summary ("suppressed 47 similar events") is emitted as a regular `Event` through the same storage + alerter pipeline, not as a separate notification type.

**Why:** This means summaries are automatically persisted to SQLite, indexed in Bleve, and sent to all configured alerters — the same as any other event. No special-casing needed in the storage or alerter adapters. The `Count` field on the `Event` struct carries the suppressed count.

---

## 9. Flush dedup buckets on shutdown

**Decision:** On SIGTERM, call `dedup.Flush()` before exiting to emit pending summaries.

**Why:** Without flushing, events suppressed in open buckets are silently lost on shutdown. If the daemon is restarted during an incident, the summary "OOMKill seen 47 times" would never be sent. Flushing on shutdown ensures every suppressed event is accounted for, even if the window hasn't expired yet.

---

## 10. EventProcessor as the central coordinator

**Decision:** The `EventProcessor` struct owns the filter → dedup → storage + alerter pipeline. The informer calls `processor.Process()`, not individual components.

**Why:** This is the hexagonal architecture's "application service" pattern. The informer (inbound adapter) doesn't know about storage or alerting — it just calls `Process()`. The storage and alerter adapters don't know about filtering or deduplication — they just implement ports. The `EventProcessor` is the only place that knows the full pipeline, making it easy to test with mock adapters and easy to change the pipeline order without touching adapters.
