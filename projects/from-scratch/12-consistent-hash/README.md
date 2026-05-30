# 12 — Consistent Hashing

Build a consistent hashing ring from scratch — the algorithm behind distributed caches (Memcached), databases (DynamoDB, Cassandra), and load balancers.

---

## Why Not Modulo?

```
// Naive: node = hash(key) % N
// Problem: when N changes (node dies), almost ALL keys remap
// With 4 nodes → 3 nodes: ~75% of keys move
```

Consistent hashing ensures only `K/N` keys move when a node is added/removed (K = total keys, N = total nodes).

---

## Architecture

```mermaid
graph TD
    subgraph Hash Ring 0..2^32
        direction LR
        N1[Node A\npos: 1200] 
        V1[Node A-vn1\npos: 3400]
        N2[Node B\npos: 5600]
        V2[Node B-vn1\npos: 7800]
        N3[Node C\npos: 9100]
        V3[Node C-vn1\npos: 11200]
    end

    K1[key: user:42\nhash: 2900] -->|clockwise| V1
    K2[key: session:7\nhash: 6000] -->|clockwise| V2
    K3[key: order:99\nhash: 9500] -->|clockwise| V3
```

```mermaid
sequenceDiagram
    participant Client
    participant Ring as Hash Ring
    participant Node as Target Node
    
    Client->>Ring: GetNode("user:42")
    Ring->>Ring: hash("user:42") → position 2900
    Ring->>Ring: binary search → next node at 3400 (A-vn1)
    Ring-->>Client: "NodeA"
    Client->>Node: GET user:42
    Node-->>Client: {data}
```

---

## Key Concepts

| Concept | Description |
|---------|-------------|
| Hash Ring | Circular space [0, 2^32) where nodes and keys are mapped |
| Virtual Nodes | Each physical node gets N positions on the ring for better distribution |
| Clockwise Lookup | A key maps to the first node found clockwise from its hash position |
| Minimal Disruption | Adding/removing a node only affects adjacent keys |

---

## Implementation

```go
type HashRing struct {
    nodes       map[uint32]string // hash position → node name
    sorted      []uint32          // sorted positions for binary search
    replicas    int               // virtual nodes per physical node
    mu          sync.RWMutex
}

func (r *HashRing) GetNode(key string) string {
    hash := crc32.ChecksumIEEE([]byte(key))
    idx := sort.Search(len(r.sorted), func(i int) bool {
        return r.sorted[i] >= hash
    })
    if idx >= len(r.sorted) {
        idx = 0 // wrap around
    }
    return r.nodes[r.sorted[idx]]
}
```

---

## Running

```bash
make build
make run
make test
```

---

## What You Learn

- Why distributed systems use consistent hashing over modulo
- Virtual nodes for load balancing
- Binary search on sorted ring
- Measuring key redistribution on topology changes
