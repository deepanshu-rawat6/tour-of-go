# SRE: Debugging & Recovery

---

## Debugging 5XX Errors — The SRE Way

When your application starts returning 5XX errors, there is a systematic investigation order. Jumping straight to application logs often wastes time — start from the outside (load balancer, pod status) and work inward.

### The Debugging Funnel

```mermaid
graph TD
    classDef blue fill:#3498db,stroke:#2980b9,color:#fff,rx:8
    classDef green fill:#2ecc71,stroke:#27ae60,color:#fff,rx:8
    classDef red fill:#e74c3c,stroke:#c0392b,color:#fff,rx:8
    classDef orange fill:#e67e22,stroke:#d35400,color:#fff,rx:8
    classDef purple fill:#9b59b6,stroke:#8e44ad,color:#fff,rx:8
    classDef teal fill:#1abc9c,stroke:#16a085,color:#fff,rx:8
    classDef dark fill:#2c3e50,stroke:#1a252f,color:#fff,rx:8
    classDef yellow fill:#f39c12,stroke:#d68910,color:#000,rx:8
    classDef k8s fill:#326ce5,stroke:#254ea8,color:#fff,rx:8
    classDef aws fill:#ff9900,stroke:#cc7a00,color:#000,rx:8
    ALERT["Alert: 5XX rate above threshold"]:::blue --> L1

    L1["Step 1: Check load balancer metrics first"]:::blue --> LB_CHECK{LB healthy?}
    LB_CHECK -->|"ALB 5XX but target group healthy"| LB_ISSUE["ALB issue: listener rules, health check config, or expired SSL cert"]:::blue
    LB_CHECK -->|"5XX from targets"| L2

    L2["Step 2: kubectl get pods -n ns -l app=name"]:::blue --> POD_CHECK{All pods Running?}
    POD_CHECK -->|CrashLoopBackOff| CRASH["App crashing on startup — kubectl logs pod --previous"]:::red
    POD_CHECK -->|OOMKilled| OOM["Memory limit exceeded — check limits, heap profile"]:::blue
    POD_CHECK -->|Pending| PENDING["Scheduling issue — kubectl describe pod, check Events"]:::blue
    POD_CHECK -->|"Running but 5XX"| L3

    L3["Step 3: kubectl describe pod name"]:::blue --> DESC_CHECK{Events clean?}
    DESC_CHECK -->|"Readiness probe failing"| PROBE["App unhealthy internally — check startup errors in logs"]:::green
    DESC_CHECK -->|"Resource pressure warnings"| RESOURCES["Node under pressure — kubectl top nodes / kubectl top pods"]:::blue
    DESC_CHECK -->|"Clean events"| L4

    L4["Step 4: kubectl logs pod -f --tail=200"]:::blue --> LOG_CHECK{Errors in logs?}
    LOG_CHECK -->|"DB connection errors"| DB_ISSUE["Database issue — check RDS, connection pool exhaustion"]:::teal
    LOG_CHECK -->|"Timeout errors to upstream"| UPSTREAM["Upstream dependency degraded — check circuit breaker metrics"]:::blue
    LOG_CHECK -->|"panic / nil deref"| PANIC["Application bug — capture goroutine dump, fix and redeploy"]:::green
    LOG_CHECK -->|"Clean logs"| L5

    L5["Step 5: kubectl top pods — check Prometheus/Grafana"]:::blue --> METRICS_CHECK{Resource exhaustion?}
    METRICS_CHECK -->|"CPU throttled"| CPU_ISSUE["CPU limit too low — throttled pod causes slow responses and timeouts"]:::blue
    METRICS_CHECK -->|"Memory near limit"| MEM_ISSUE["About to OOM — increase memory limit or fix leak"]:::blue
    METRICS_CHECK -->|"Normal resources"| L6

    L6["Step 6: kubectl exec pod -- curl upstream and nslookup service"]:::blue --> NET_CHECK{Connectivity OK?}
    NET_CHECK -->|"DNS fails"| DNS_ISSUE["CoreDNS issue — kubectl get pods -n kube-system -l k8s-app=kube-dns"]:::blue
    NET_CHECK -->|"Connection refused"| NET_POL["NetworkPolicy blocking — kubectl get networkpolicy -n ns"]:::red
    NET_CHECK -->|"Timeouts"| UPSTREAM2["Upstream too slow — check service latency p99"]:::teal
```

### kubectl Runbook

```bash
# --- Step 1: Pod overview ---
kubectl get pods -n <namespace> -l app=<name> -o wide
# Look for: STATUS (CrashLoopBackOff, OOMKilled, Pending), RESTARTS count, NODE assignment

# --- Step 2: Describe pod — most important first step ---
kubectl describe pod <pod-name> -n <namespace>
# Key sections to scan:
#   Events: at the bottom — "Back-off restarting failed container", "OOMKilled", "FailedScheduling"
#   Conditions: Ready=False, reason
#   Containers → State: Waiting/Running/Terminated, LastState: exit code

# Exit code reference:
#   137 = OOMKilled (128 + signal 9 SIGKILL)
#   1   = Application error / unhandled exception
#   2   = Misuse of shell command
#   143 = SIGTERM (graceful shutdown, 128 + signal 15)

# --- Step 3: Logs ---
kubectl logs <pod-name> -n <namespace> --tail=200
kubectl logs <pod-name> -n <namespace> --previous     # logs from last crashed container
kubectl logs <pod-name> -n <namespace> -c <container> # specific container in multi-container pod
kubectl logs -l app=<name> -n <namespace> --tail=50   # logs from ALL pods with this label

# --- Step 4: Resource consumption ---
kubectl top pods -n <namespace> --sort-by=memory
kubectl top nodes
kubectl describe node <node-name> | grep -A5 "Allocated resources"

# --- Step 5: Events (cluster-wide, sorted by time) ---
kubectl get events -n <namespace> --sort-by='.lastTimestamp'
kubectl get events -n <namespace> --field-selector reason=OOMKilling

# --- Step 6: Exec into pod for debugging ---
kubectl exec -it <pod-name> -n <namespace> -- sh
# Inside: curl, wget, nslookup, cat /proc/meminfo, env

# --- Step 7: Check endpoints (is service pointing to healthy pods?) ---
kubectl get endpoints <service-name> -n <namespace>
# If empty or missing IPs → pods not matching service selector, or pods not Ready

# --- Step 8: Port-forward to test pod directly (bypass LB/ingress) ---
kubectl port-forward pod/<pod-name> 8080:8080 -n <namespace>
curl -v localhost:8080/healthz
```

### Common 5XX Root Causes

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| `CrashLoopBackOff`, exit code 1 | App panics on startup — missing env var, bad config, failed DB migration | Check logs from previous container |
| `OOMKilled`, exit code 137 | Memory limit too low, or memory leak | Increase limit, profile heap, check for goroutine leaks |
| Readiness probe fails, pod not Ready | App takes too long to start, or /readyz endpoint broken | Tune `initialDelaySeconds`, fix readiness logic |
| All pods Running but 5XX | Upstream dependency down (DB, cache, external API) | Check dependency health, circuit breaker open? |
| 5XX only from some pods | Node-level issue (disk pressure, kernel bug) | `kubectl cordon <node>`, drain, investigate node |
| CPU throttling (`kubectl top` shows 100% but no OOM) | CPU limit too restrictive | Increase CPU limit, or remove limit entirely (requests only) |
| `Pending` pods | Insufficient cluster capacity, PodAffinity mismatch, PV stuck | Check events: `FailedScheduling`, check `kubectl describe pod` |

---

## Debugging 5XX Errors — EKS-Specific

When running on EKS, you have additional AWS-native tooling on top of the kubectl workflow above.

### AWS Load Balancer Controller — Target Group Health

The AWS Load Balancer Controller creates ALBs/NLBs in response to Ingress/Service objects. If pods are healthy but ALB returns 503, the target group may not have registered the pods yet (or the health check is misconfigured).

```bash
# Find the ALB created for your ingress
kubectl get ingress -n <namespace>
# ANNOTATION: kubernetes.io/ingress.class: alb shows it's managed by LBC

# Get the ALB ARN from the ingress status
kubectl describe ingress <name> -n <namespace>
# Look for: Address: <alb-dns-name>

# Check target group health via AWS CLI
aws elbv2 describe-target-health \
  --target-group-arn arn:aws:elasticloadbalancing:us-east-1:123:targetgroup/k8s-xxx/xxx \
  --region us-east-1
# Unhealthy targets show: State.Reason = "Target.FailedHealthChecks"

# Common cause: security group on the node/pod doesn't allow
# health check traffic from the ALB security group on the health check port
```

### CloudWatch Container Insights

When Container Insights is enabled (via `aws-node` add-on or ADOT), metrics flow to CloudWatch:

```bash
# Query pod OOM events via CloudWatch Logs Insights
# Log group: /aws/containerinsights/<cluster>/performance

fields @timestamp, PodName, reason
| filter Type = "Pod" and reason = "OOMKilling"
| sort @timestamp desc
| limit 50
```

```bash
# Application logs from pods (if using Fluent Bit DaemonSet)
# Log group: /aws/containerinsights/<cluster>/application

fields @timestamp, kubernetes.pod_name, log
| filter kubernetes.namespace_name = "production"
| filter log like /ERROR|PANIC|fatal/
| sort @timestamp desc
| limit 100
```

### EKS Control Plane Logs for Debugging

```bash
# Scheduler logs — why is my pod Pending?
aws logs filter-log-events \
  --log-group-name /aws/eks/my-cluster/cluster \
  --log-stream-name-prefix kube-scheduler \
  --filter-pattern '"my-pod-name"' \
  --region us-east-1

# API Server audit log — who deleted/modified a resource?
aws logs filter-log-events \
  --log-group-name /aws/eks/my-cluster/cluster \
  --log-stream-name-prefix kube-apiserver-audit \
  --filter-pattern '{ $.requestURI = "/apis/apps/v1/namespaces/prod/deployments/my-app" }' \
  --region us-east-1

# Authenticator logs — auth failures (403, unauthorized)
aws logs filter-log-events \
  --log-group-name /aws/eks/my-cluster/cluster \
  --log-stream-name-prefix authenticator \
  --filter-pattern '"error"' \
  --region us-east-1
```

### X-Ray / AWS Distro for OpenTelemetry (ADOT)

If your app instruments with OpenTelemetry and sends traces to X-Ray via ADOT Collector:

```bash
# X-Ray service map shows 5XX at which service hop
aws xray get-service-graph \
  --start-time $(date -u -v-1H +%s) \
  --end-time $(date -u +%s) \
  --region us-east-1

# Get traces with 5XX status
aws xray get-trace-summaries \
  --start-time $(date -u -v-1H +%s) \
  --end-time $(date -u +%s) \
  --filter-expression 'responsetime > 5 AND http.status = 500' \
  --region us-east-1
```

---

## OOM-Killed Recovery

### Why OOM Kill Happens

The Linux kernel's **OOM Killer** is invoked when a container exceeds its **memory limit** (cgroup memory limit set from `spec.containers[].resources.limits.memory`). The kernel sends `SIGKILL` (signal 9) to the process — not SIGTERM. There is no graceful shutdown. The process is immediately killed.

Kubernetes detects the exit code 137 (128 + 9) and records the reason as `OOMKilled` in the pod's `lastState`.

```
Containers:
  app:
    Last State: Terminated
      Reason:   OOMKilled
      Exit Code: 137
      Started:  Sat, 06 Jun 2026 10:00:00
      Finished: Sat, 06 Jun 2026 10:15:32
```

**Requests vs Limits for memory:**
- `requests.memory`: the amount the scheduler reserves on the node. Used for placement. Guaranteed to the container.
- `limits.memory`: the hard ceiling the kernel enforces. Exceeding this = OOMKill.
- Best practice: set requests = your p95 steady-state memory, limits = your p99.9 + buffer. Never set limits to 10x requests "just in case" — this causes node over-commitment and cascading OOM kills during memory pressure.

### Diagnosing the Memory Issue

```bash
# 1. Confirm OOM kill and see historical container memory usage
kubectl describe pod <pod-name>

# 2. Check current memory usage
kubectl top pods -n <namespace> --containers

# 3. If app is still running (not yet OOM killed), capture heap profile
# (requires pprof endpoint in the app)
kubectl port-forward pod/<pod-name> 6060:6060
go tool pprof http://localhost:6060/debug/pprof/heap
# In pprof: top20, list <func>, web (opens flame graph)

# 4. Check for goroutine leaks (goroutines hold stack memory)
curl http://localhost:6060/debug/pprof/goroutine?debug=2 | head -100
```

### Recovery: Singleton Pod

A **singleton** is a single-replica deployment — typically a controller, cron job, leader-elected worker, or stateful singleton service.

**The risk:** When OOM-killed, the pod restarts (kubelet's `restartPolicy: Always`). During the restart window (image pull + startup time), there is **zero capacity** serving requests. For a stateful singleton, in-flight operations are lost.

**Mitigation strategies:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-singleton
spec:
  replicas: 1  # singleton
  template:
    spec:
      containers:
        - name: app
          image: my-org/my-app:v1
          resources:
            requests:
              memory: "256Mi"   # what the scheduler reserves
              cpu: "100m"
            limits:
              memory: "512Mi"   # OOM if exceeded — set realistically
              cpu: "500m"

          # Give the process time to finish in-flight work before SIGKILL
          lifecycle:
            preStop:
              exec:
                command: ["/bin/sh", "-c", "sleep 5"]  # drain connections

          # Readiness probe prevents traffic during restart
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
            failureThreshold: 3

      # Give the pod up to 30s to shutdown gracefully after SIGTERM
      terminationGracePeriodSeconds: 30
```

**For a true singleton, also consider Vertical Pod Autoscaler (VPA)** to automatically right-size memory based on historical usage:

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: my-singleton-vpa
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-singleton
  updatePolicy:
    updateMode: "Auto"   # or "Off" to just see recommendations without applying
  resourcePolicy:
    containerPolicies:
      - containerName: app
        minAllowed:
          memory: "128Mi"
        maxAllowed:
          memory: "2Gi"
```

### Recovery: Distributed/Replicated Pod

A **distributed** workload runs `replicas: N > 1`. When one pod OOM-kills, others continue serving. The key is ensuring:
1. **Enough replicas** so one death doesn't cause capacity collapse
2. **PodDisruptionBudget** so rolling restarts/node drains don't kill too many at once
3. **Anti-affinity** so replicas aren't all on the same node

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-service
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1   # at most 1 pod down during update/restart
      maxSurge: 1
  template:
    spec:
      # Spread pods across nodes — don't put all replicas on same node
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchLabels:
                    app: my-service
                topologyKey: kubernetes.io/hostname

      containers:
        - name: app
          resources:
            requests:
              memory: "256Mi"
              cpu: "200m"
            limits:
              memory: "512Mi"
              cpu: "1000m"
---
# PodDisruptionBudget — prevent too many simultaneous disruptions
# (node drains, rolling deployments, voluntary disruptions)
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: my-service-pdb
spec:
  selector:
    matchLabels:
      app: my-service
  minAvailable: 2   # always keep at least 2 pods running
  # OR: maxUnavailable: 1
```

**With PDB in place:** When a node is drained (for upgrade, scaling down), the drain will block if removing a pod would violate `minAvailable`. The node drain waits until a replacement pod is healthy before proceeding. This prevents rolling OOM-kills from cascading into a full outage.

---

## Leader Election for Singleton Workloads

If you need exactly one active instance (not zero, not two), use Kubernetes **Lease-based leader election**. The Lease API is a lightweight lock stored in etcd.

**Use cases:** distributed job scheduler, CDC consumer, singleton reconciler, any process that must not run concurrently.

### How It Works

```mermaid
sequenceDiagram
    participant P1 as Pod 1 (candidate)
    participant P2 as Pod 2 (standby)
    participant K8s as K8s API Server (Lease object in etcd)

    P1->>K8s: Try to acquire Lease (create/update with holderIdentity=pod-1)
    K8s-->>P1: Lease acquired — pod-1 is leader
    P2->>K8s: Try to acquire Lease
    K8s-->>P2: Lease held by pod-1, renewDeadline not expired — not acquired

    loop Every leaseDuration/2
        P1->>K8s: Renew lease (update renewTime)
        K8s-->>P1: OK
    end

    Note over P1: Pod 1 OOM-killed / crashes
    Note over K8s: Lease expires (no renewal within leaseDuration)

    P2->>K8s: Try to acquire Lease — lease expired!
    K8s-->>P2: Lease acquired — pod-2 is now leader
    Note over P2: Pod 2 starts doing work
```

### Go Implementation with `client-go`

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

func main() {
	// Identity: use pod name so we can see who is leader
	id := os.Getenv("POD_NAME")
	if id == "" {
		id, _ = os.Hostname()
	}

	// In-cluster config (works when running inside a K8s pod)
	cfg, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	client := kubernetes.NewForConfigOrDie(cfg)

	// Lease lock — stored as a Lease object in the given namespace
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "my-app-leader",    // Lease object name
			Namespace: "default",
		},
		Client: client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id, // pod name — identifies who holds the lease
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock: lock,

		// How long a lease is valid without renewal
		// If the leader crashes, a new leader can only be elected after this duration
		LeaseDuration: 15 * time.Second,

		// How long the leader has to renew before losing leadership
		// Must be < LeaseDuration
		RenewDeadline: 10 * time.Second,

		// How often the leader tries to renew
		// Must be < RenewDeadline
		RetryPeriod: 2 * time.Second,

		// ReleaseOnCancel: release the lease gracefully when context is cancelled
		// (e.g., on SIGTERM). Without this, the lease holds for LeaseDuration
		// after crash, delaying failover.
		ReleaseOnCancel: true,

		Callbacks: leaderelection.LeaderCallbacks{
			// Called when this pod becomes the leader — start your work here
			OnStartedLeading: func(ctx context.Context) {
				fmt.Printf("[%s] became leader — starting work\n", id)
				runWork(ctx)
			},

			// Called when this pod loses leadership (lease expired, context cancelled)
			// Stop your work here — MUST return quickly
			OnStoppedLeading: func() {
				fmt.Printf("[%s] lost leadership — stopping work\n", id)
				// If work goroutine is running, the ctx passed to OnStartedLeading
				// is cancelled automatically by the leader election library
				os.Exit(0) // let kubelet restart the pod to re-compete
			},

			// Called when any pod acquires the lease (informational)
			OnNewLeader: func(identity string) {
				if identity == id {
					return // we already know we're leader from OnStartedLeading
				}
				fmt.Printf("[%s] new leader elected: %s\n", id, identity)
			},
		},
	})
}

// runWork is the actual singleton work. ctx is cancelled when leadership is lost.
func runWork(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("work stopped — leadership lost or context cancelled")
			return
		case t := <-ticker.C:
			fmt.Printf("doing singleton work at %s\n", t.Format(time.RFC3339))
			// e.g., process a job queue, run a reconciliation loop, etc.
		}
	}
}
```

### Required RBAC

The pod needs permission to create/get/update the `Lease` object:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-app
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: leader-election
  namespace: default
rules:
  - apiGroups: ["coordination.k8s.io"]
    resources: ["leases"]
    verbs: ["get", "create", "update"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: my-app-leader-election
  namespace: default
subjects:
  - kind: ServiceAccount
    name: my-app
roleRef:
  kind: Role
  name: leader-election
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 2   # both pods run, only one is leader at a time
  template:
    spec:
      serviceAccountName: my-app  # bind the SA with lease permissions
      containers:
        - name: app
          image: my-org/my-app:v1
          env:
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name  # inject pod name as identity
```

### Key Tuning Parameters

| Parameter | Default | Effect of too low | Effect of too high |
|-----------|---------|------------------|--------------------|
| `LeaseDuration` | 15s | Frequent leader churn under network blips | Slow failover after leader crash |
| `RenewDeadline` | 10s | Leader gives up leadership under transient API slowness | Leader holds on too long during real failures |
| `RetryPeriod` | 2s | High API Server load (many renewals) | Slow to detect that renewal is needed |
| `replicas` | 2+ | Single point of failure if leader pod is OOM-killed | Wasted resources |

**`ReleaseOnCancel: true`** is critical for fast failover on graceful shutdown. When the pod receives SIGTERM (rolling update, scale-down), it cancels the context, the leader election library releases the Lease immediately, and a new leader is elected within `RetryPeriod` — not `LeaseDuration`. Without this, failover waits the full 15 seconds.


---

## SRE Core Concepts

### SLI, SLO, Error Budget, SLA

```mermaid
graph LR
    classDef blue fill:#3498db,stroke:#2980b9,color:#fff,rx:8
    classDef green fill:#2ecc71,stroke:#27ae60,color:#fff,rx:8
    classDef red fill:#e74c3c,stroke:#c0392b,color:#fff,rx:8
    classDef orange fill:#e67e22,stroke:#d35400,color:#fff,rx:8
    classDef purple fill:#9b59b6,stroke:#8e44ad,color:#fff,rx:8
    classDef teal fill:#1abc9c,stroke:#16a085,color:#fff,rx:8
    classDef dark fill:#2c3e50,stroke:#1a252f,color:#fff,rx:8
    classDef yellow fill:#f39c12,stroke:#d68910,color:#000,rx:8
    classDef k8s fill:#326ce5,stroke:#254ea8,color:#fff,rx:8
    classDef aws fill:#ff9900,stroke:#cc7a00,color:#000,rx:8
    SLI["SLI: the measured signal"]:::blue --> SLO["SLO: internal target, e.g. 99.9% over 30 days"]:::blue
    SLO --> EB["Error Budget: 100% minus SLO = allowed failure"]:::green
    SLO --> SLA["SLA: external contract, SLO must be tighter than SLA"]:::blue
    EB --> BEHAVIOR["Budget remaining? Ship fast. Budget burned? Freeze and fix."]:::blue
```

**SLI (Service Level Indicator):**
The actual measured signal. What you observe.
- % of requests served successfully (status != 5xx)
- % of requests under 300ms latency
- Uptime percentage
- Error rate

**SLO (Service Level Objective):**
The internal target for the SLI. What you commit to internally.
- 99.9% of requests successful over a 30-day rolling window
- p99 latency under 500ms, measured over 1-hour windows
- 99.95% availability per quarter

**Error Budget:**
`100% - SLO = allowed failure`

| SLO | Budget | Monthly downtime allowed |
|-----|--------|--------------------------|
| 99% | 1% | ~7.2 hours |
| 99.9% | 0.1% | ~43 minutes |
| 99.95% | 0.05% | ~21.6 minutes |
| 99.99% | 0.01% | ~4.3 minutes |

The budget is a **resource**: burn it fast (bad incident) -> freeze risky changes, focus on reliability. Budget remaining -> ship features aggressively. This depersonalizes the argument: the budget decides, not opinion.

**How error budget changes behavior:** The team checks error budget weekly. If a bad incident burns 50% of the month's budget in one day, the next sprint freezes new features and focuses entirely on reliability.

**SLA (Service Level Agreement):**
External contract with customers. Breaching it means fines/credits. SLOs should always be tighter than SLAs (buffer for detecting breaches before customers do).

---

### MTTD, MTTR, MTTF

```mermaid
graph LR
    classDef blue fill:#3498db,stroke:#2980b9,color:#fff,rx:8
    classDef green fill:#2ecc71,stroke:#27ae60,color:#fff,rx:8
    classDef red fill:#e74c3c,stroke:#c0392b,color:#fff,rx:8
    classDef orange fill:#e67e22,stroke:#d35400,color:#fff,rx:8
    classDef purple fill:#9b59b6,stroke:#8e44ad,color:#fff,rx:8
    classDef teal fill:#1abc9c,stroke:#16a085,color:#fff,rx:8
    classDef dark fill:#2c3e50,stroke:#1a252f,color:#fff,rx:8
    classDef yellow fill:#f39c12,stroke:#d68910,color:#000,rx:8
    classDef k8s fill:#326ce5,stroke:#254ea8,color:#fff,rx:8
    classDef aws fill:#ff9900,stroke:#cc7a00,color:#000,rx:8
    INCIDENT_START["Incident starts (users affected)"]:::blue -->|"MTTD"| DETECTED["Team alerted"]:::blue
    DETECTED -->|"MTTR"| RESOLVED["Service restored"]:::teal
    RESOLVED -->|"MTTF"| NEXT["Next incident"]:::blue
```

**MTTD (Mean Time To Detect):**
Time from incident START to team KNOWS about it.

Improve by: SLO-based burn rate alerts, synthetic monitoring, anomaly detection, on-call coverage gaps analysis.

**MTTR (Mean Time To Recover):**
Time from detection to SERVICE RESTORED.

Improve by: runbooks (documented and tested), automated rollback, feature flags (instant kill switches), on-call drills, pre-approved playbooks for common failure modes.

**MTTF (Mean Time To Failure):**
Average time between incidents. Higher = more reliable.

Improve by: reducing toil, better testing, chaos engineering, capacity planning.

**Key insight:** MTTD and MTTR are separate problems.
- Fast detection + slow recovery = still bad (you know it's broken but can't fix it)
- Fast recovery + slow detection = users hurt for a long time before you knew

---

### Toil

**Toil** = manual, repetitive, automatable, reactive operational work that scales linearly with service growth and produces no enduring improvement.

**All traits must be true:**

| Trait | Question |
|-------|----------|
| Manual | No automation runs it |
| Repetitive | Done again and again |
| Automatable | Could be automated |
| Scales with growth | More users = more toil instances |
| No durable value | Does it again next time, leaves nothing behind |

**Examples:**
- Manually restarting a stuck service
- Hand-running database migrations
- Rotating certs by SSH-ing to 30 servers
- A daily release call with the same manual steps

**Why cap at ~50%:** If toil isn't capped, the team drowns in ops as the service grows and never builds anything permanent. SRE dedicates the freed time to automating the toil away.

---

### Incident Response Flow

```mermaid
graph TD
    classDef blue fill:#3498db,stroke:#2980b9,color:#fff,rx:8
    classDef green fill:#2ecc71,stroke:#27ae60,color:#fff,rx:8
    classDef red fill:#e74c3c,stroke:#c0392b,color:#fff,rx:8
    classDef orange fill:#e67e22,stroke:#d35400,color:#fff,rx:8
    classDef purple fill:#9b59b6,stroke:#8e44ad,color:#fff,rx:8
    classDef teal fill:#1abc9c,stroke:#16a085,color:#fff,rx:8
    classDef dark fill:#2c3e50,stroke:#1a252f,color:#fff,rx:8
    classDef yellow fill:#f39c12,stroke:#d68910,color:#000,rx:8
    classDef k8s fill:#326ce5,stroke:#254ea8,color:#fff,rx:8
    classDef aws fill:#ff9900,stroke:#cc7a00,color:#000,rx:8
    DETECT["1. DETECT: Alert fires on SLO burn rate. Goal: minimize MTTD"]:::blue --> TRIAGE
    TRIAGE["2. TRIAGE: Assess severity SEV1/2/3, assign Incident Commander, open war room"]:::blue --> MITIGATE
    MITIGATE["3. MITIGATE: Rollback, shift traffic, feature-flag off, shed load. Do NOT wait for root cause."]:::orange --> RECOVER
    RECOVER["4. RECOVER: Confirm service restored, metrics normal, communicate status"]:::teal --> POSTMORTEM
    POSTMORTEM["5. BLAMELESS POSTMORTEM: Timeline, contributing factors, 5 Whys, action items"]:::blue
```

**Key principle of step 3:** Do NOT wait for root cause analysis. Mitigate first, investigate after. Rollback the deploy, shed load, feature-flag off — worry about why later.

**Blameless postmortem questions:**
- What happened? (timeline)
- What allowed it to happen? (not who caused it — what system gap)
- How did we detect it? Could we have detected it faster?
- How did we mitigate? Could we have mitigated faster?
- What prevents this class of failure from recurring? (action items)

---

### Monitoring: RED Method and Alerting

**RED method for request-based services:**

| Signal | What it measures | Alert when |
|--------|-----------------|------------|
| **R**ate | Requests per second | Sudden drop (service down?) or spike (DDoS?) |
| **E**rrors | Error rate (5xx / total) | Exceeds error budget burn rate |
| **D**uration | Latency distribution (p50, p95, p99) | p99 exceeds SLO threshold |

**Rules for good alerts:**
1. Alert on **symptoms** (user impact), not causes (CPU high, memory high)
2. Every page must be **actionable** — if no human action needed, it is not a page
3. Use **SLO burn rate** alerts — page when consuming error budget faster than sustainable
4. Group and deduplicate — one incident = one page, not 30
5. Audit alerts quarterly — delete any that fire but nobody acts on

**SLO burn rate alert example (Prometheus):**

```promql
# Fires when error budget burns 2x faster than sustainable over 1h window
sum(rate(http_requests_total{status=~"5.."}[1h])) /
sum(rate(http_requests_total[1h]))
> (1 - 0.999) * 2     # 2x the error budget rate for 99.9% SLO
```

---

### Scaling: Horizontal vs Vertical

| | Horizontal (scale out) | Vertical (scale up) |
|--|----------------------|-------------------|
| How | Add more instances | Bigger instance (more CPU/RAM) |
| Works for | Stateless services, microservices | Any app, no code changes |
| Fails when | App has shared state hard to distribute | Hit the largest instance size |
| Downtime | Zero (add instances behind LB) | Possible for traditional infra (not K8s) |
| Tools | K8s HPA, ECS Autoscaling, ASG | Change instance type |

**Best practice:** Design stateless services (session in Redis, not in-process memory) so horizontal scaling works cleanly. Vertical scale is the emergency lever when you can't distribute state.

---

## CrashLoopBackOff: Debugging Runbook

`CrashLoopBackOff` means the container starts, crashes (exits non-zero), Kubernetes restarts it, it crashes again — and Kubernetes applies exponential backoff between restarts (10s → 20s → 40s → 80s → 160s → 5min cap). It will keep retrying indefinitely.

**It is not a single cause — it's a symptom.** The exit code tells you where to look first.

### Step 1: Get the exit code and last logs

```bash
# See restart count and status
kubectl get pod <pod> -n <ns>

# Last crash reason + exit code
kubectl describe pod <pod> -n <ns>
# Look for: Last State, Exit Code, Reason (OOMKilled / Error / Completed)

# Logs from the crashed container (not the current one)
kubectl logs <pod> -n <ns> --previous

# If multi-container pod
kubectl logs <pod> -n <ns> -c <container-name> --previous
```

### Step 2: Diagnose by exit code

| Exit Code | Meaning | Where to look |
|-----------|---------|---------------|
| `1` | App error / panic / unhandled exception | `--previous` logs — missing env var, failed DB connect, bad config |
| `2` | Misuse of shell / script error | Entrypoint/command misconfigured |
| `137` | `SIGKILL` — OOM killed by kernel | Memory limit too low, memory leak |
| `139` | Segfault | Corrupt binary, wrong arch (arm vs amd64) |
| `143` | `SIGTERM` not handled — graceful shutdown timeout | App ignores SIGTERM, preStop hook too short |
| `125` | Docker/container runtime error | Bad image, missing binary in container |
| `126` | Permission denied on entrypoint | File not executable, wrong user |
| `127` | Entrypoint binary not found | Typo in `command`, wrong base image |

### Step 3: Work through the cause tree

```
CrashLoopBackOff
├── Exit 137 (OOMKilled)
│   ├── memory.limit too low → increase resources.limits.memory
│   ├── memory leak → profile with pprof, check goroutine count
│   └── JVM heap not set → add -Xmx flag
│
├── Exit 1 (App panic/error)
│   ├── Missing env var → kubectl describe pod, check envFrom / env
│   ├── Can't connect to DB/Redis on startup → wrong SERVICE_NAME, wrong port
│   ├── Failed DB migration (runs in same container) → separate init container
│   ├── Config file not found → ConfigMap not mounted, wrong mountPath
│   └── Secret not found → Secret doesn't exist in namespace
│
├── Exit 127 (binary not found)
│   ├── Wrong command/args in Deployment spec
│   └── Multi-stage build forgot to copy the binary
│
├── Exit 1 but logs are empty
│   ├── App crashes before logger initializes → add stderr logging early
│   ├── Init container failing → kubectl logs <pod> -c <init-container>
│   └── readinessProbe killing pod before app fully starts
│
└── Liveness probe killing healthy pod
    ├── initialDelaySeconds too short → app not ready when probe fires
    ├── Probe endpoint wrong → 404 returns, pod killed
    └── timeoutSeconds too low → slow startup looks like failure
```

### Common contributors and fixes

#### 1. Missing / wrong environment variable

```bash
kubectl describe pod <pod> -n <ns> | grep -A5 "Environment"
# Or check what the app is actually seeing:
kubectl exec <pod> -n <ns> -- env | grep DB_

# Fix: check ConfigMap and Secret refs in Deployment
```

```yaml
# Common mistake: referencing a Secret that doesn't exist in this namespace
env:
- name: DB_PASSWORD
  valueFrom:
    secretKeyRef:
      name: db-secret      # does this Secret exist in the same namespace?
      key: password
```

```bash
kubectl get secret db-secret -n <ns>   # 404 here = pod will CrashLoop
```

#### 2. Can't connect to dependency on startup

App tries to connect to DB/Redis/external API during `init()` or startup, fails, panics.

```bash
# Test connectivity from inside a debug pod in same namespace
kubectl run debug --rm -it --image=busybox -n <ns> -- sh
wget -qO- http://my-service:5432   # or nc -zv my-service 5432
```

**Fix:** Add startup retry logic — don't fail fast on first connection attempt. Or use an init container to wait for the dependency:

```yaml
initContainers:
- name: wait-for-db
  image: busybox
  command: ['sh', '-c', 'until nc -z postgres-svc 5432; do echo waiting; sleep 2; done']
```

#### 3. OOMKilled — memory limit too low

```bash
kubectl describe pod <pod> | grep -A3 "Last State"
# Last State: Terminated  Reason: OOMKilled

# Check current memory usage before the crash
kubectl top pod <pod> -n <ns>

# Check limits
kubectl get pod <pod> -o jsonpath='{.spec.containers[0].resources}'
```

**Fix:**
```yaml
resources:
  requests:
    memory: "256Mi"
  limits:
    memory: "512Mi"   # increase this, or remove limit to use node memory
```

For Go apps: `GOGC` env var controls GC aggressiveness. `GOMEMLIMIT` (Go 1.19+) sets a soft memory ceiling before GC kicks in aggressively:

```yaml
env:
- name: GOMEMLIMIT
  value: "450MiB"   # slightly below the k8s limit
```

#### 4. Liveness probe killing the pod

Pod shows `CrashLoopBackOff` but logs look fine — the liveness probe is the killer.

```bash
kubectl describe pod <pod> | grep -A10 "Liveness"
# Events: Liveness probe failed: ... Killing container with id...
```

```yaml
# Fix: give app time to start before first probe
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 30    # wait 30s before first check
  periodSeconds: 10
  failureThreshold: 3        # need 3 consecutive failures before kill
  timeoutSeconds: 5          # give it 5s to respond
```

#### 5. Init container failing silently

```bash
# Init containers show separately
kubectl get pod <pod> -n <ns>
# Init:0/1 means init container hasn't completed

kubectl logs <pod> -n <ns> -c <init-container-name>
kubectl logs <pod> -n <ns> -c <init-container-name> --previous
```

#### 6. Wrong image / wrong arch

```bash
kubectl describe pod <pod> | grep -A5 "Events"
# exec format error → image built for wrong arch (amd64 image on arm node or vice versa)

# Fix: build multi-arch image
docker buildx build --platform linux/amd64,linux/arm64 -t my-org/app:v1 --push .
```

### Quick checklist

```
□ kubectl logs <pod> --previous          → read the actual crash message
□ kubectl describe pod <pod>             → exit code, events, probe config
□ kubectl get events -n <ns> --sort-by=.lastTimestamp  → cluster-level events
□ exit code 137?                         → OOMKilled, check memory limits
□ exit code 1, empty logs?               → init container? probe killing it?
□ env vars correct?                      → describe pod, check secret/configmap exists
□ can pod reach its dependencies?        → debug pod with nc/wget
□ liveness probe initialDelaySeconds?    → might be firing too early
□ init containers healthy?               → kubectl logs -c <init-container>
□ image right arch?                      → exec format error in events
```

---

## Pod Stuck in `Pending`

Pod is created but never scheduled — it's sitting in the scheduler queue with no node assigned.

```bash
kubectl describe pod <pod> -n <ns>
# Look at Events section at the bottom — FailedScheduling with reason
```

### Cause tree

| Event message | Root cause | Fix |
|--------------|-----------|-----|
| `0/3 nodes are available: 3 Insufficient cpu` | No node has enough CPU | Scale up node group, lower requests, or check if requests are over-specified |
| `0/3 nodes are available: 3 Insufficient memory` | No node has enough memory | Same as above |
| `0/3 nodes are available: 3 node(s) had taint... that the pod didn't tolerate` | Missing toleration | Add toleration for the taint |
| `0/3 nodes are available: 3 node(s) didn't match Pod's node affinity/selector` | nodeAffinity/nodeSelector mismatch | Check labels on nodes vs pod spec |
| `0/3 nodes are available: 3 node(s) had untolerated taint + 0 didn't match affinity` | Both issues at once | Fix both |
| `unbound immediate PersistentVolumeClaims` | PVC not bound to a PV | See PVC stuck section below |
| `didn't match pod anti-affinity rules` | Hard anti-affinity unsatisfiable | Not enough nodes, or switch to soft |

```bash
# Check actual node capacity vs allocatable
kubectl describe nodes | grep -A5 "Allocated resources"

# Check what's consuming resources
kubectl top nodes
kubectl top pods -A --sort-by=memory

# Check if cluster autoscaler is blocked
kubectl logs -n kube-system -l app=cluster-autoscaler | tail -50
```

---

## `ImagePullBackOff` / `ErrImagePull`

Container image can't be pulled. `ErrImagePull` is the first attempt; `ImagePullBackOff` is repeated failure with backoff.

```bash
kubectl describe pod <pod> -n <ns>
# Events: Failed to pull image "...", reason in the message
```

### Cause tree

| Error message | Cause | Fix |
|--------------|-------|-----|
| `unauthorized: authentication required` | No pull secret / wrong creds | Add `imagePullSecrets` to pod spec |
| `not found` / `manifest unknown` | Tag doesn't exist | `docker pull <image>` locally to verify |
| `no basic auth credentials` | ECR token expired (12hr TTL) | Refresh ECR token, or use IRSA + ECR pull-through |
| `exec format error` | Wrong arch (arm image on amd64 node) | Build multi-arch image |
| `connection refused` / timeout | Node can't reach registry | Check node's internet access / NAT GW / VPC endpoint for ECR |
| `ImagePullBackOff` on private registry | `imagePullSecrets` missing or wrong namespace | Secret must be in same namespace as pod |

```bash
# Test pulling manually on the node
kubectl debug node/<node-name> -it --image=busybox
# inside: docker pull <image> or crictl pull <image>

# Check if imagePullSecret exists in the right namespace
kubectl get secret regcred -n <ns>

# For ECR: check node IAM role has ecr:GetAuthorizationToken
aws iam simulate-principal-policy \
  --policy-source-arn <node-role-arn> \
  --action-names ecr:GetAuthorizationToken
```

---

## Pod Stuck in `Terminating`

`kubectl delete pod` was run but pod stays in Terminating state indefinitely.

```bash
kubectl describe pod <pod> -n <ns>
# Look for: finalizers, volumes not unmounting, preStop hook hanging
```

### Causes

**1. Finalizer not being removed**
```bash
kubectl get pod <pod> -n <ns> -o json | jq '.metadata.finalizers'
# If a controller crashed and never removed the finalizer, pod hangs

# Force remove finalizer (only if you're sure the controller is gone)
kubectl patch pod <pod> -n <ns> -p '{"metadata":{"finalizers":[]}}' --type=merge
```

**2. preStop hook hanging**
```yaml
# preStop has a hard deadline of terminationGracePeriodSeconds (default 30s)
# If hook runs longer, pod is force killed after the grace period
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 5"]  # must complete within grace period
```

**3. Volume not unmounting (PVC)**
```bash
kubectl describe pod <pod> | grep -A5 "Volumes"
# "Unable to unmount volumes" in events = storage driver issue

# Force delete as last resort (may leave volume in bad state)
kubectl delete pod <pod> -n <ns> --grace-period=0 --force
```

**4. Node is NotReady / unreachable**
```bash
# If node went offline, pods on it stay Terminating until node comes back or is deleted
kubectl get node <node>
kubectl delete node <node>   # removes the node object, pods get rescheduled
```

---

## Node `NotReady`

```bash
kubectl get nodes
# NAME        STATUS      ROLES    AGE
# node-1      NotReady    <none>   2d

kubectl describe node <node-name>
# Look at: Conditions, Events, kubelet logs
```

### Cause tree

```
Node NotReady
├── kubelet stopped
│   └── ssh to node: systemctl status kubelet
│       journalctl -u kubelet -n 100
│
├── Disk pressure (DiskPressure=True)
│   ├── Node full of logs/images → kubelet evicts pods
│   └── Fix: kubectl drain + increase disk, or add imagePrunner CronJob
│       docker system prune / crictl rmi --prune
│
├── Memory pressure (MemoryPressure=True)
│   ├── System processes consuming memory
│   └── Fix: kubectl drain + investigate, check for memory leak in DaemonSets
│
├── PID pressure (PIDPressure=True)
│   ├── Too many processes (fork bomb, runaway threads)
│   └── Fix: find the pod: kubectl top pods --sort-by=cpu
│
├── Network unreachable
│   ├── CNI plugin crashed → pods can't get IPs
│   └── kubectl logs -n kube-system -l k8s-app=aws-node (VPC CNI)
│       kubectl logs -n kube-system -l app=calico-node
│
└── Cloud provider issue (EKS)
    └── EC2 instance health check failing → terminate + replace node
        aws ec2 describe-instance-status --instance-id <id>
```

```bash
# Cordon + drain to safely remove workloads before investigating
kubectl cordon <node>
kubectl drain <node> --ignore-daemonsets --delete-emptydir-data

# Check node events
kubectl get events -n default --field-selector involvedObject.name=<node>
```

---

## High Latency / Slow Requests

Pod is Running, no crashes, but requests are slow.

### Step 1 — isolate the layer

```bash
# Is it all pods or specific ones?
kubectl top pods -n <ns>    # CPU throttled? Memory pressure?

# Check if requests hit a specific pod (check per-pod metrics in Grafana/Datadog)
# Is load balancer distributing unevenly?
```

### Step 2 — CPU throttling (most common invisible cause)

```bash
# Check throttling metrics
kubectl top pod <pod> -n <ns>
# If CPU is at limit constantly → throttled

# In Prometheus:
# container_cpu_cfs_throttled_seconds_total — cumulative throttle time
# rate(container_cpu_cfs_throttled_periods_total[5m]) / rate(container_cpu_cfs_periods_total[5m])
# > 0.25 means 25%+ of scheduling periods throttled → very bad for latency
```

**Fix:** Either raise the CPU limit, or remove it entirely (keep only `requests`). CPU limits cause p99 latency spikes even when average CPU is low.

```yaml
resources:
  requests:
    cpu: "500m"
  limits:
    cpu: "2000m"   # raise this, or remove limits entirely for latency-sensitive services
```

### Step 3 — GC pauses (Go / JVM)

```bash
# Go: check GC stats via pprof
kubectl port-forward pod/<pod> 6060:6060
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Check GC frequency
curl http://localhost:6060/debug/vars | jq '.memstats'
# NumGC high + PauseTotalNs high → GC pressure
```

**Fix for Go:** Set `GOMEMLIMIT` to give GC headroom before hitting k8s limit.

### Step 4 — connection pool exhaustion

```bash
# Symptoms: latency spikes at high concurrency, logs show "connection wait timeout"
# App is waiting for a DB connection from the pool

# Check pool metrics if exposed, or:
kubectl exec <pod> -- cat /proc/<pid>/net/tcp | wc -l   # open TCP connections
```

**Fix:** increase pool size, or add read replicas / RDS Proxy.

### Step 5 — DNS resolution slow

```bash
# DNS latency inside cluster
kubectl run dnstest --rm -it --image=busybox -- sh
time nslookup my-service.my-namespace.svc.cluster.local

# CoreDNS performance
kubectl top pods -n kube-system -l k8s-app=kube-dns
kubectl logs -n kube-system -l k8s-app=kube-dns | grep -i error
```

**Fix:** use fully qualified domain names (FQDN) to avoid search domain iteration, or tune CoreDNS replicas.

---

## PVC Stuck in `Pending`

```bash
kubectl get pvc -n <ns>
# NAME    STATUS    VOLUME   CAPACITY   STORAGECLASS
# data    Pending                       gp3

kubectl describe pvc data -n <ns>
# Events tell you why
```

| Event | Cause | Fix |
|-------|-------|-----|
| `no persistent volumes available` | No PV matches (static provisioning) | Create a PV or switch to dynamic |
| `storageclass not found` | Wrong `storageClassName` | `kubectl get storageclass` — check name |
| `waiting for first consumer` | StorageClass has `volumeBindingMode: WaitForFirstConsumer` | Normal — PVC binds when pod is scheduled |
| `failed to provision volume` | CSI driver error / IAM permissions | Check CSI driver pod logs |
| `exceeded quota` | ResourceQuota on namespace | `kubectl describe quota -n <ns>` |

```bash
# Check CSI driver (EBS on EKS)
kubectl logs -n kube-system -l app=ebs-csi-controller -c ebs-plugin | tail -50

# Check storage classes
kubectl get storageclass

# Check if IAM role for CSI has ebs:CreateVolume permission
```

---

## Resource Quota / LimitRange Blocking Pods

```bash
# Pod fails to create with "exceeded quota" or "must specify limits"
kubectl describe quota -n <ns>
kubectl describe limitrange -n <ns>

# Example output:
# Resource         Used    Hard
# requests.cpu     1800m   2000m    ← close to limit
# limits.memory    3Gi     4Gi
```

**LimitRange** sets defaults and min/max for containers in a namespace. If a pod has no `resources` set and LimitRange requires it, the pod is rejected.

```bash
# See what defaults are being injected
kubectl get limitrange -n <ns> -o yaml
```

---

## Quick Reference: All Pod States

| Status | Meaning | First command |
|--------|---------|--------------|
| `Pending` | Not scheduled yet | `kubectl describe pod` → Events: FailedScheduling |
| `Init:0/1` | Init container running/failed | `kubectl logs <pod> -c <init-container>` |
| `PodInitializing` | Init done, main container starting | Normal, wait |
| `Running` but not Ready | Readiness probe failing | `kubectl describe pod` → probe config + app logs |
| `CrashLoopBackOff` | Container crashing repeatedly | `kubectl logs --previous`, check exit code |
| `OOMKilled` | Memory limit exceeded | Increase `limits.memory`, check for leak |
| `Error` | Container exited non-zero once | `kubectl logs <pod>` |
| `Completed` | Container exited 0 (Job/init) | Normal for Jobs |
| `Terminating` | Delete issued, waiting grace period | Check finalizers, preStop hook |
| `ImagePullBackOff` | Can't pull image | `kubectl describe pod` → registry/auth/tag |
| `ErrImageNeverPull` | `imagePullPolicy: Never` + image missing | Push image to node or change policy |
| `NodeLost` / `Unknown` | Node unreachable | `kubectl get node`, check cloud console |
