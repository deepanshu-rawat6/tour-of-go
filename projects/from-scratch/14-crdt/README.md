# 14 — CRDTs (Conflict-Free Replicated Data Types)

Data structures that can be replicated across nodes and merged without coordination — enabling eventual consistency without conflicts.

---

## Why CRDTs?

In distributed systems, you often face the CAP theorem trade-off. CRDTs let you choose **AP** (Available + Partition-tolerant) while guaranteeing eventual consistency — no coordination, no conflicts, no data loss.

```mermaid
graph TD
    subgraph Traditional Approach
        N1A[Node 1: counter=5] -->|conflict!| MERGE1[Which value wins?]
        N2A[Node 2: counter=7] -->|conflict!| MERGE1
    end
    
    subgraph CRDT Approach
        N1B[Node 1: +3 ops] -->|merge| MERGE2[Deterministic merge\nno conflicts ever]
        N2B[Node 2: +5 ops] -->|merge| MERGE2
        MERGE2 --> RESULT[Result: 8\nboth contributions preserved]
    end
```

---

## G-Counter (Grow-Only Counter)

Each node maintains its own counter. The global value is the sum of all nodes.

```mermaid
graph LR
    subgraph Node A
        A[A: {A:3, B:0, C:0}\nlocal value = 3]
    end
    subgraph Node B
        B[B: {A:0, B:5, C:0}\nlocal value = 5]
    end
    subgraph Node C
        C[C: {A:0, B:0, C:2}\nlocal value = 2]
    end
    
    A -->|merge| M[Merged: {A:3, B:5, C:2}\nglobal value = 10]
    B -->|merge| M
    C -->|merge| M
```

**Merge rule**: For each node ID, take the **max** of both values.

### Use Cases
- Like counters (Facebook likes)
- View counters
- Any monotonically increasing metric

---

## PN-Counter (Positive-Negative Counter)

Two G-Counters: one for increments, one for decrements. Value = P - N.

```mermaid
graph TD
    INC[Increment G-Counter\n{A:5, B:3}] --> VAL[Value = sum P - sum N\n= 8 - 2 = 6]
    DEC[Decrement G-Counter\n{A:1, B:1}] --> VAL
```

---

## LWW-Register (Last-Writer-Wins Register)

Each write carries a timestamp. On merge, the highest timestamp wins.

```mermaid
sequenceDiagram
    participant A as Node A
    participant B as Node B
    
    A->>A: Set("alice", t=100)
    B->>B: Set("bob", t=105)
    
    Note over A,B: Merge: compare timestamps
    A->>B: Sync {value:"alice", t:100}
    B->>A: Sync {value:"bob", t:105}
    
    Note over A: t:105 > t:100 → value = "bob"
    Note over B: t:105 > t:100 → value = "bob"
    Note over A,B: Both converge to "bob"
```

**Trade-off**: Concurrent writes → one is silently dropped (the older one).

---

## Running

```bash
make run    # demos G-Counter, PN-Counter, LWW-Register
make test
```
