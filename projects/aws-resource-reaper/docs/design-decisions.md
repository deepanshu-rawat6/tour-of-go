# Design Decisions — aws-resource-reaper

Why the implementation is structured the way it is.

---

## 1. Default credential chain over static credentials

**Decision:** Use `config.LoadDefaultConfig` with no explicit credentials. The config file contains only account IDs and role ARNs — no keys.

**Why:** The tool is designed to run on EC2/ECS with an instance profile. `LoadDefaultConfig` resolves credentials automatically from IMDS → env vars → `~/.aws` in that order, so the same binary works on EC2 (IMDS), in CI (env vars), and locally (`~/.aws`) without any code changes. Storing credentials in config files is a security anti-pattern.

---

## 2. STS AssumeRole per account, cached

**Decision:** Call `sts:AssumeRole` once per account and cache the resulting `aws.Config`. All region scans for that account reuse the same assumed credentials.

**Why:** Each `AssumeRole` call has latency and counts against STS API rate limits. With N accounts × M regions, calling it M times per account would be wasteful. The credentials returned by AssumeRole are valid for 1 hour by default — far longer than a typical scan run.

---

## 3. `errgroup` + semaphore for concurrency, not a worker pool

**Decision:** Use `golang.org/x/sync/errgroup` with a `semaphore.Weighted` rather than a fixed goroutine worker pool.

**Why:** A worker pool requires pre-allocating goroutines and a job queue. `errgroup` + semaphore achieves the same bounded concurrency with less code: each `account × region` pair acquires a semaphore slot, runs, and releases it. The `errgroup` context cancellation means the first error automatically stops all in-flight work — something a worker pool requires explicit signalling to achieve.

---

## 4. Scanner interface per resource type

**Decision:** Each resource type is a separate struct implementing `Scanner`. The engine calls all scanners for every `account × region` pair.

**Why:** This makes adding a new resource type a single-file change with no modifications to the engine. It also makes each scanner independently testable with a mock client. The alternative — one large function with all resource types — would be harder to test and extend.

---

## 5. Mock interfaces for AWS clients in tests

**Decision:** Define narrow interfaces (`ec2API`, `rdsAPI`, `elbv2API`, `cwAPI`) that each scanner uses, rather than depending directly on the concrete AWS SDK client types.

**Why:** The AWS SDK clients cannot be instantiated without real credentials. Narrow interfaces let us inject mock implementations in tests, making the entire scanner suite testable without any AWS account. This is the standard Go approach to dependency injection — accept interfaces, not concrete types.

---

## 6. Rule evaluator as a separate layer from scanners

**Decision:** Scanners return raw `Resource` structs. A separate `Evaluator` applies `Rule` implementations to produce `Finding` structs with actions and savings estimates.

**Why:** Separation of concerns. Scanners answer "what exists?" — they make AWS API calls and return facts. Rules answer "what should we do about it?" — they are pure functions with no I/O. This makes rules trivially unit-testable (no mocks needed) and means you can add new rules without touching scanner code.

---

## 7. `Recommend` action for Graviton — never executed

**Decision:** EC2 instances flagged for Graviton migration get `ActionRecommend`, which the executor explicitly skips.

**Why:** Terminating a running EC2 instance is irreversible and high-impact. Graviton migration requires application testing, AMI changes, and planned maintenance — it cannot be automated safely. The tool surfaces the opportunity; a human makes the decision. This is the correct FinOps pattern: automate deletion of clearly idle resources, recommend (not automate) architectural changes.

---

## 8. `text/tabwriter` and `log/slog` — no external libraries

**Decision:** Use stdlib `text/tabwriter` for table output and `log/slog` (Go 1.21+) for structured execution logging. No third-party logging or table libraries.

**Why:** The tool already has a significant dependency footprint from the AWS SDK. Adding `zerolog`, `zap`, or a table library for what amounts to a few dozen lines of formatting code is not justified. `slog` produces structured JSON logs natively, which is what you want for audit trails in production. `tabwriter` produces aligned output with zero dependencies.

---

## 9. Dry-run on by default, `--dry-run=false` to execute

**Decision:** The default mode is always dry-run. Live execution requires an explicit `--dry-run=false` flag AND an interactive confirmation prompt.

**Why:** This tool can delete production resources across many accounts. A mistyped command or misconfigured CI job should never cause accidental deletions. Two explicit gates (flag + prompt) mean you have to actively choose to cause destruction. The confirmation prompt also shows the full plan before any API call is made.

---

## 10. Independent `go.mod` per project

**Decision:** `projects/aws-resource-reaper/` has its own `go.mod` with no imports from other projects in this repo.

**Why:** Each project in this repo is a standalone learning artifact. Shared dependencies between projects would create coupling — a version bump in one project could break another. Independent modules also mean `go mod tidy` and `go test ./...` work correctly within each project directory without needing workspace files.
