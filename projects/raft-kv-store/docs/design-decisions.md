# Design Decisions — raft-kv-store

## 1. gRPC for inter-node, HTTP for client API

**Decision:** Raft RPCs (`AppendEntries`, `RequestVote`) use gRPC with protobuf. The KV API uses plain HTTP.

**Why:** gRPC gives strongly-typed, code-generated RPC stubs with efficient binary encoding — exactly right for the high-frequency, low-latency RPCs between Raft nodes (heartbeats every 50ms). HTTP is simpler for the client-facing API where human readability and curl-friendliness matter more than efficiency.

---

## 2. Randomized election timeouts (150–300ms)

**Decision:** Each node picks a random election timeout in the 150–300ms range using `math/rand`.

**Why:** This is the core mechanism the Raft paper uses to prevent split votes. If all nodes had the same timeout, they'd all become candidates simultaneously and split votes indefinitely. Randomization ensures one node almost always times out first and wins the election before others start. The 150–300ms range is the range recommended in the original Raft paper.

---

## 3. WAL + in-memory log (both)

**Decision:** Entries are stored in both an append-only WAL file (for durability) and an in-memory `[]LogEntry` slice (for fast access).

**Why:** The WAL ensures entries survive process restarts — without it, a crashed node loses all uncommitted entries and can't rejoin the cluster correctly. The in-memory slice is needed for fast log access during replication (finding `prevLogTerm`, building `AppendEntries` payloads). The two are kept in sync: every `Append` writes to both.

---

## 4. Sentinel entry at log index 0

**Decision:** The log is initialized with a sentinel entry `{Term: 0, Index: 0}` at position 0.

**Why:** Raft's `prevLogIndex`/`prevLogTerm` consistency check needs to handle the case where a follower has an empty log. Without a sentinel, the leader would need special-case code for `prevLogIndex == 0`. The sentinel means `termOf(0) == 0` always, and the consistency check works uniformly for all entries including the first.

---

## 5. `stepDownCh` and `heartbeatCh` channels for role transitions

**Decision:** Role transitions are signalled via buffered channels (`stepDownCh`, `heartbeatCh`) rather than direct function calls or shared state polling.

**Why:** The Raft main loop (`runFollower`, `runCandidate`, `runLeader`) runs in a goroutine and blocks on `select`. Channels let concurrent goroutines (e.g., an incoming gRPC handler) signal the main loop without holding the mutex. Buffered channels (size 1) prevent the sender from blocking if the main loop is busy.

---

## 6. Follower returns 307 redirect (not proxy)

**Decision:** When a follower receives a write, it returns HTTP 307 Temporary Redirect with the leader's HTTP address. It does not proxy the request.

**Why:** Proxying adds complexity (the follower must act as an HTTP client, handle timeouts, propagate errors). A redirect is simpler, stateless, and puts the retry logic in the client where it belongs. HTTP clients like curl (`-L` flag) and most HTTP libraries follow redirects automatically. The 307 (not 301) preserves the HTTP method on redirect.

---

## 7. `Propose` blocks until commit or timeout

**Decision:** `Propose` appends the entry, triggers replication, then polls `commitCh` until the entry is committed (up to 5 seconds).

**Why:** The HTTP handler needs to return a response only after the write is durable (committed to a majority). A fire-and-forget approach would return 200 before the write is safe, violating linearizability. The 5-second timeout prevents the HTTP handler from hanging indefinitely if the cluster loses quorum.

---

## 8. `sync.Mutex` over channels for Raft state

**Decision:** All Raft state (`currentTerm`, `votedFor`, `log`, `role`, etc.) is protected by a single `sync.Mutex` rather than using channels or an actor model.

**Why:** Raft state is accessed from multiple goroutines (gRPC handlers, election goroutine, replication goroutines). A mutex is the simplest correct approach. The Raft paper itself describes the algorithm in terms of shared state with atomic updates. An actor model would require serializing all state access through a channel, adding latency to every RPC handler.

---

## 9. Transport interface decouples Raft from gRPC

**Decision:** The `raft` package defines a `Transport` interface (`RequestVote`, `AppendEntries`). The gRPC client implements this interface. The Raft core never imports the transport package.

**Why:** This makes the Raft core independently testable — election and replication tests use a `mockTransport` that returns controlled responses without any network I/O. It also means the transport layer could be swapped (e.g., for raw TCP) without touching the Raft logic.

---

## 10. Static cluster membership (no dynamic reconfiguration)

**Decision:** Cluster membership is fixed at startup via the YAML config. There is no `AddNode`/`RemoveNode` operation.

**Why:** Dynamic membership changes (Raft joint consensus) are one of the most complex parts of the Raft paper and a common source of bugs. For a learning implementation focused on the core mechanics, static membership is the right tradeoff. The cluster supports any odd number of nodes — just list them all in the config before starting.
