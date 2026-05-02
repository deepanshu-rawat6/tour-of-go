# ADR-001: LRU over LFU Eviction Policy

**Status:** Accepted

## Decision

Use Least-Recently-Used (LRU) eviction instead of Least-Frequently-Used (LFU).

## Rationale

| Concern | LRU | LFU |
|---|---|---|
| Implementation complexity | O(1) with doubly-linked list + map | O(1) requires complex frequency buckets (e.g., Caffeine algorithm) |
| Temporal locality | Excellent — recent access = likely future access | Good for stable hot-set, poor for scan patterns |
| Cold start | Handles well — new entries compete on recency | Penalises new entries (low frequency) |
| Go stdlib | `container/list` is sufficient | No stdlib support; requires custom frequency tracking |
| Learning value | Demonstrates list + map composition clearly | Obscures the core caching concept |

## Consequences

- LRU can be suboptimal for workloads with a stable hot-set accessed infrequently (e.g., configuration data accessed once per minute). LFU would retain these better.
- For the typical web caching workload (recent = relevant), LRU is the industry standard (Redis, Memcached, CPU L1/L2 caches all default to LRU or approximations of it).
