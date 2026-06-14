# Runnable Advanced Snippets

Executable Go programs that bring the `go/` theory to life.
Each directory is a standalone `main` package — run it directly with `go run`.

## How to Run

```shell
# From the repo root:
go run ./go/runnable/concurrency-patterns/
go run ./go/runnable/design-patterns/
go run ./go/runnable/system-design/
```

## What Each One Demonstrates

| Directory | Patterns | Related Theory |
|-----------|----------|----------------|
| `concurrency-patterns/` | Pipeline, Fan-out/Fan-in | [Concurrency Patterns README](../design-patterns/concurrency-patterns/README.md) |
| `design-patterns/` | Functional Options, Circuit Breaker, Single-Flight | [Patterns README](../design-patterns/patterns/README.md), [Industry Patterns README](../design-patterns/industry-patterns/README.md) |
| `system-design/` | Token Bucket, Sliding Window rate limiter | [Rate Limiting README](../system-design/rate-limiting-deep-dive/README.md) |

## Tip

After running each snippet, read the corresponding README for the theory behind it.
The code is intentionally minimal — focus on the pattern, not the boilerplate.
