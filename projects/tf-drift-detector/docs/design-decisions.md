# Design Decisions — tf-drift-detector

## 1. Alert only on new and resolved drift, not every cycle

**Decision:** The `Tracker` compares current poll results against a known-drift map. Only new drift (first detection) and resolved drift (was drifted, now clean) trigger alerts.

**Why:** Without this, every polling cycle would re-alert on the same drift. A 5-minute interval with 10 drifted resources would generate 2,880 Slack messages per day for the same issues. Stateful tracking means the team gets one alert when drift appears and one when it's fixed — the right signal-to-noise ratio.

---

## 2. Hardcoded ignore list + config overrides (not config-only)

**Decision:** Ship a built-in ignore list per resource type for known AWS-computed fields, and allow users to add their own via `ignore_fields` in the config.

**Why:** If the ignore list were config-only, every user would need to discover and configure the same set of noisy fields (e.g., `aws_s3_bucket.arn`, `aws_lambda_function.last_modified`). These fields are universally noisy — AWS always returns them, Terraform never manages them. Hardcoding them as defaults means the tool works correctly out of the box. The config override handles org-specific cases.

---

## 3. Type coercion in the diff engine (`fmt.Sprintf("%v", ...)`)

**Decision:** Compare values by converting both to string via `fmt.Sprintf("%v", v)` rather than strict type equality.

**Why:** Terraform stores all attribute values as strings in the state file (e.g., `"128"` for memory_size). AWS SDK returns typed values (`int32(128)`). Without coercion, every numeric field would appear drifted even when the values are identical. `fmt.Sprintf` handles int32, int64, float64, bool, and string uniformly without a type switch.

---

## 4. `errgroup` + semaphore for concurrent polling

**Decision:** Use `golang.org/x/sync/errgroup` with a `semaphore.Weighted` to bound concurrent AWS API calls.

**Why:** A Terraform state file with 100 resources would require 100 sequential API calls without concurrency — potentially taking minutes. Fan-out with errgroup reduces this to `ceil(100 / concurrency)` rounds. The semaphore prevents hitting AWS API rate limits. The errgroup context cancellation means the first error stops all in-flight work cleanly.

---

## 5. `sync.Mutex` for the drift tracker, not channels

**Decision:** The `Tracker`'s known-drift map is protected by a `sync.Mutex` rather than using a channel-based actor pattern.

**Why:** The tracker is accessed from a single goroutine (the polling cycle) in the daemon. A mutex is simpler and more readable than an actor pattern for this access pattern. The mutex also makes the `KnownCount()` health check method straightforward to implement safely.

---

## 6. `signal.NotifyContext` for graceful shutdown

**Decision:** Use `signal.NotifyContext(ctx, SIGTERM, SIGINT)` to propagate shutdown through the entire call stack via context cancellation.

**Why:** Every AWS API call, S3 fetch, and webhook POST accepts a `context.Context`. When SIGTERM arrives, the context is cancelled, which causes all in-flight operations to return immediately with `context.Canceled`. This is cleaner than a separate done channel and ensures the daemon never hangs on shutdown waiting for a slow API call.

---

## 7. Persist drift state on shutdown, load on startup

**Decision:** The tracker writes its known-drift map to a JSON file on SIGTERM (via `defer`) and reads it back on startup.

**Why:** Without persistence, a daemon restart would re-alert on all existing drift as if it were new. This would cause alert fatigue after every deployment or restart. Persistence means the daemon resumes exactly where it left off. The state file is plain JSON — human-readable and easy to inspect or reset manually.

---

## 8. Missing live resource = drift, not error

**Decision:** If `FetchLive` returns an error (resource not found in AWS), the engine records it as drifted with `Fields: [{Path: "existence", Expected: "exists", Actual: "not found"}]` rather than propagating the error.

**Why:** A resource that exists in Terraform state but not in AWS is the most severe form of drift — it was deleted outside of Terraform. Treating it as an error would skip alerting on it. Treating it as drift surfaces it correctly in the alert pipeline.

---

## 9. Two run modes: `run` (one-shot) and `daemon` (ticker)

**Decision:** Separate `run` and `daemon` subcommands rather than a single command with a `--once` flag.

**Why:** The two modes have meaningfully different behavior: `run` has no tracker (alerts on all drift found), while `daemon` uses the tracker (alerts only on new/resolved). Separate subcommands make this distinction explicit in the CLI surface and help text. It also makes CI usage (`detect run`) clearly distinct from production deployment (`detect daemon`).

---

## 10. Flat `map[string]interface{}` for resource attributes

**Decision:** Both TF state attributes and live API responses are represented as `map[string]interface{}` rather than typed structs per resource.

**Why:** Typed structs would require a separate struct definition for each of the 6 resource types, plus marshaling/unmarshaling logic. The flat map approach means the diff engine is a single generic function that works for all resource types. The tradeoff — no compile-time field validation — is acceptable because the comparators are unit-tested with mock clients that verify the correct keys are returned.
