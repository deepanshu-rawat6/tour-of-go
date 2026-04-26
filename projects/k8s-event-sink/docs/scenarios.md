# Use Cases & Scenarios

## 1. OOMKill Storm During Node Memory Pressure

**Scenario:** A node runs out of memory. Kubernetes starts evicting pods, generating hundreds of OOMKill events per minute. Without deduplication, your Slack channel gets 500 identical alerts.

**How k8s-event-sink helps:**
- First OOMKill on each pod is forwarded immediately (you know within seconds)
- Subsequent OOMKills on the same pod are suppressed for 5 minutes
- When the window expires: "OOMKilled on api-pod (suppressed 47 similar events in window)"
- One alert per pod, not 500

```bash
curl "localhost:9090/events?severity=critical&namespace=production"
```

---

## 2. CrashLoopBackOff Investigation

**Scenario:** A deployment is crash-looping. You want to search for all events mentioning "connection refused" to find the root cause.

```bash
curl "localhost:9090/search?q=connection+refused"
# Returns all events with "connection refused" in the message
# Even events from 3 days ago that etcd already deleted
```

Bleve's inverted index makes this instant — no SQL LIKE queries.

---

## 3. Long-Term Audit Trail for Compliance

**Scenario:** Your compliance team needs a 90-day history of all pod evictions and OOMKills for a SOC 2 audit. Kubernetes only keeps events for 1 hour.

k8s-event-sink persists every critical event to SQLite on a PersistentVolume. Query by time range:

```bash
curl "localhost:9090/events?severity=critical"
# Returns all critical events since the daemon started
```

---

## 4. Multi-Namespace Monitoring with RBAC Scoping

**Scenario:** Your security team won't grant ClusterRole to a background daemon. You need to monitor only `production` and `staging` namespaces.

```yaml
# config.yaml
namespaces:
  - production
  - staging
```

The daemon spins up one dedicated informer per namespace, each with its own RBAC Role (not ClusterRole).

---

## 5. Suppressing Noisy Health Check Failures

**Scenario:** Your readiness probes fail briefly during deployments, generating hundreds of `Unhealthy` events. You want to ignore these but still catch real failures.

```yaml
ignore_reasons:
  - Unhealthy   # suppress during rolling deployments
severity:
  BackOff: critical  # but escalate back-off to critical
```

---

## 6. Prometheus Alerting Integration

**Scenario:** You want to trigger PagerDuty via Prometheus AlertManager when the drop rate spikes (indicating a pod storm).

```promql
# Alert when more than 100 events/minute are being deduplicated
rate(k8s_events_deduplicated_total[1m]) > 100
```

The `/metrics` endpoint exposes `k8s_events_received_total`, `k8s_events_dropped_total`, `k8s_events_deduplicated_total`, `k8s_events_forwarded_total`.
