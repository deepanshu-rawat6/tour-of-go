# 12-consistent-hash: Deep Dive

## Why Consistent Hashing?

With naive modulo hashing (`hash(key) % N`), adding or removing one node remaps nearly all keys. This kills any caching layer — every cache miss on a fleet restart.

Consistent hashing remaps only `~1/N` of keys when a node joins or leaves.

## The Ring

All possible hash values form a ring (0 → 2³²-1 for crc32). Nodes are placed at positions on the ring. A key maps to the **first node clockwise from its hash position**.

```mermaid
graph TD
    K1["key:user:42\nhash → 850M\n→ cache-2 (900M)"]
    K2["key:session:7\nhash → 100M\n→ cache-1 (200M)"]
    K3["key:order:99\nhash → 3.5B\n→ cache-1 (wraps around)"]

    RING["Ring: 0 ───────── 200M(cache-1) ──── 900M(cache-2) ──── 2.5B(cache-3) ──── 4.2B → 0"]
    RING --> K1
    RING --> K2
    RING --> K3
```

## Virtual Nodes

One physical node → 150 virtual nodes, each at a different ring position. Prevents hot spots when physical nodes are unevenly distributed.

```mermaid
graph LR
    P1[cache-1] --> V1A[cache-1#0\npos: 123M]
    P1 --> V1B[cache-1#1\npos: 789M]
    P1 --> V1C[cache-1#2\npos: 2.1B]
    P2[cache-2] --> V2A[cache-2#0\npos: 456M]
    P2 --> V2B[cache-2#1\npos: 1.5B]
    P2 --> V2C[cache-2#2\npos: 3.8B]
```

Without virtual nodes: uneven ring gaps → one node gets 60% of keys, another gets 5%.
With 150 virtual nodes: each physical node handles `~1/N` of keyspace regardless of position.

## Lookup: Binary Search

```mermaid
flowchart LR
    KEY["key: user:42"] --> HASH["crc32(key) → pos"]
    HASH --> BS["binary search\nsorted []uint32 positions"]
    BS --> IDX["first position ≥ hash\n(wrap to 0 if none)"]
    IDX --> VNODE["virtual node at that position"]
    VNODE --> PHYS["physical node\n(strip #N suffix)"]
```

```go
func (r *Ring) Get(key string) string {
    h := r.hash(key)
    idx := sort.Search(len(r.positions), func(i int) bool {
        return r.positions[i] >= h
    })
    if idx == len(r.positions) {
        idx = 0 // wrap around
    }
    return r.nodeOf[r.positions[idx]] // strip virtual node suffix → physical
}
```

`sort.Search` is O(log N) where N = total virtual nodes. For 10 physical nodes × 150 vnodes = 1500 entries, that's ~11 comparisons.

## Node Join and Leave

```mermaid
flowchart TD
    ADD["AddNode(cache-4)"] --> PLACE["place 150 virtual nodes\nat crc32(cache-4#0..149)"]
    PLACE --> REBUCKET["only keys that now land\nbetween new positions\n→ remap to cache-4"]
    REBUCKET --> FRAC["~1/N keys moved\nall others unchanged"]

    REMOVE["RemoveNode(cache-2)"] --> DELETE["delete cache-2's\n150 positions from ring"]
    DELETE --> REPOINT["keys that were on cache-2\n→ next clockwise node"]
    REPOINT --> FRAC2["~1/N keys moved\nall others unchanged"]
```

**Why 1/N?** Each virtual node covers 1/(total vnodes) of the ring. Removing N physical nodes × 150 vnodes means those ring segments now point to the next clockwise node — only the keys in those segments move.

## Real-World Uses

| System | What it hashes | Why consistent hash |
|--------|---------------|-------------------|
| Memcached clients | cache key → node | Minimize cache misses on scale |
| Cassandra | partition key → replica | Even data distribution |
| Chord DHT | file content hash → peer | Decentralized lookup |
| Nginx upstream | URL → backend | Sticky sessions without state |

## Comparison: Modulo vs Consistent

```
Cluster: 3 nodes → add 1 node (now 4)

Modulo hash(key) % N:
  key "abc" → hash 1000 → 1000%3=1 → node-1
  after:       hash 1000 → 1000%4=0 → node-0  ← moved!
  ~75% of all keys move on a single node addition

Consistent hash:
  only keys in the ring segments now owned by the new node move
  ~25% of keys move (1/4)
```
