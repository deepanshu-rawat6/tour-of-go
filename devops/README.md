# DevOps Reference

Kubernetes, Linux, Docker, AWS, GCP, CI/CD, IaC, Monitoring, Git, and SRE — ordered from foundation to advanced.

---

## Recommended Order

```
1. Linux fundamentals        → understand what everything runs on
2. Docker                    → containers before orchestration
3. Kubernetes                → orchestration, networking, security
4. CI/CD                     → build and deploy pipelines
5. IaC                       → Terraform, Helm, CloudFormation
6. AWS                       → cloud infrastructure
7. GCP                       → GCP equivalent concepts
8. Monitoring                → Prometheus, Grafana, OTel, Loki
9. Performance debugging     → profiling, USE/RED, pprof, eBPF
10. Git                      → internals, workflows, fixing mistakes
11. Advanced                 → service mesh, eBPF, chaos, DR
12. SRE & Debugging          → production incident runbooks
```

---

## 1. Linux

| File | Topics | Level |
|------|--------|-------|
| [linux/README.md](./linux/README.md) | Process model, memory, filesystem, signals, load average | SDE-1 |
| [linux/commands.md](./linux/commands.md) | grep, awk, sed, find, ps, ss, curl, disk management | SDE-1 |
| [linux/boot.md](./linux/boot.md) | BIOS → GRUB → kernel → initramfs → systemd | SDE-1 |
| [linux/systemd.md](./linux/systemd.md) | Unit files, service lifecycle, timers, journalctl, systemd-analyze | SDE-1 |
| [linux/networking.md](./linux/networking.md) | TCP/IP stack, sockets, iptables, netfilter, TIME_WAIT | SDE-1 |
| [linux/network-tools.md](./linux/network-tools.md) | ss, tcpdump, curl -v, nc, iperf3, mtr, dig, ip, tcp tuning | SDE-1 |
| [linux/io-models.md](./linux/io-models.md) | Blocking, non-blocking, select/poll/epoll, io_uring | SDE-2 |
| [linux/scheduler.md](./linux/scheduler.md) | CFS, nice values, cgroups, context switches, GMP model | SDE-2 |
| [linux/security.md](./linux/security.md) | Capabilities, seccomp, AppArmor, SELinux, namespaces | SDE-2 |
| [linux/memory-tuning.md](./linux/memory-tuning.md) | /proc/meminfo, swap, OOM killer, dirty pages, huge pages, cgroups v2 | SDE-2 |
| [linux/strace-perf.md](./linux/strace-perf.md) | strace, perf stat/top/record, flame graphs, Brendan Gregg methodology | SDE-2 |
| [linux/containers-evolution.md](./linux/containers-evolution.md) | chroot → LXC → Docker → OCI. Image size optimization. | SDE-1 |

**Read order:** README → commands → boot → systemd → networking → network-tools → io-models → scheduler → security → memory-tuning → strace-perf → containers-evolution

---

## 2. Docker

| File | Topics | Level |
|------|--------|-------|
| [docker/README.md](./docker/README.md) | Architecture (dockerd→containerd→runc), image layers COW, commands, Dockerfile, Compose, runtime flags, storage drivers, build cache, health checks | SDE-1 |
| [docker/networking.md](./docker/networking.md) | bridge/host/overlay/macvlan/none, iptables NAT, container DNS, port publishing internals, deep mode comparison | SDE-1/2 |
| [docker/buildkit.md](./docker/buildkit.md) | Parallel stages, cache mounts, secret mounts, SSH agent, multi-platform buildx, inline cache | SDE-1/2 |
| [docker/docker-security.md](./docker/docker-security.md) | Rootless Docker, docker.sock danger, image scanning, cosign/Sigstore, cap-drop, BuildKit secrets | SDE-2 |
| [docker/debugging.md](./docker/debugging.md) | 10 scenarios: exit codes, OOM, port issues, networking, build cache, image size, volumes, compose, disk | SDE-1 |

**Read order:** README → networking → buildkit → docker-security → debugging

---

## 3. Kubernetes

| File | Topics | Level |
|------|--------|-------|
| [kubernetes/README.md](./kubernetes/README.md) | Architecture, kubectl apply flow, scheduler internals, taints, nodeAffinity, podAffinity/anti-affinity, GPU scenarios | SDE-1/2 |
| [kubernetes/kubectl-cheatsheet.md](./kubernetes/kubectl-cheatsheet.md) | Context switching, pod/deployment/debug operations, one-liners, jsonpath | SDE-1 |
| [kubernetes/workloads.md](./kubernetes/workloads.md) | Pod lifecycle, probes, QoS, Deployments, StatefulSets, DaemonSets, Jobs | SDE-1 |
| [kubernetes/networking.md](./kubernetes/networking.md) | Services, ClusterIP/iptables, DNS, Ingress, NetworkPolicy, CNI, per-node virtual networks | SDE-1/2 |
| [kubernetes/resource-limits.md](./kubernetes/resource-limits.md) | Requests vs limits, CPU throttling math, OOMKill, QoS classes, LimitRange, ResourceQuota, VPA | SDE-1/2 |
| [kubernetes/storage.md](./kubernetes/storage.md) | PV/PVC/StorageClass, dynamic provisioning, CSI, volume snapshots | SDE-1 |
| [kubernetes/autoscaling.md](./kubernetes/autoscaling.md) | HPA, VPA, KEDA, Cluster Autoscaler, Karpenter | SDE-2 |
| [kubernetes/rbac.md](./kubernetes/rbac.md) | ServiceAccount, Role/ClusterRole, RoleBinding, auth chain | SDE-1 |
| [kubernetes/helm.md](./kubernetes/helm.md) | Chart structure, templating, hooks, library charts, Helmfile, debugging | SDE-1/2 |
| [kubernetes/eks-architecture.md](./kubernetes/eks-architecture.md) | EKS managed control plane, VPC CNI, IRSA, node groups, Fargate | SDE-1/2 |

**Read order:** kubectl-cheatsheet → README → workloads → resource-limits → networking → storage → rbac → autoscaling → helm → eks-architecture

---

## 4. CI/CD

| File | Topics | Level |
|------|--------|-------|
| [cicd/README.md](./cicd/README.md) | CI vs CD vs GitOps, pipeline patterns, deployment strategies, secrets | SDE-1 |
| [cicd/github-actions/README.md](./cicd/github-actions/README.md) | Workflows, OIDC to AWS, matrix builds, caching, reusable workflows | SDE-1 |
| [cicd/jenkins/README.md](./cicd/jenkins/README.md) | Declarative pipeline, shared libraries, master-agent architecture | SDE-1 |
| [cicd/jenkins/ecs-agents.md](./cicd/jenkins/ecs-agents.md) | Dynamic Jenkins agents on ECS Fargate, cost optimization | SDE-2 |
| [cicd/argocd/README.md](./cicd/argocd/README.md) | GitOps, Application CRD, sync waves, multi-cluster with ApplicationSet | SDE-1/2 |

---

## 5. Infrastructure as Code

| File | Topics | Level |
|------|--------|-------|
| [iac/README.md](./iac/README.md) | IaC overview, Terraform vs CloudFormation, state, drift | SDE-1 |
| [iac/terraform/README.md](./iac/terraform/README.md) | HCL, state, modules, workspaces, remote backend, import | SDE-1 |
| [iac/cloudformation/README.md](./iac/cloudformation/README.md) | Templates, stacks, change sets, nested stacks, CDK | SDE-1 |

---

## 6. AWS

| File | Topics | Level |
|------|--------|-------|
| [aws/README.md](./aws/README.md) | VPC, subnets, SGs vs NACLs, NAT GW, IGW, VPC Peering, Transit Gateway | SDE-1 |
| [aws/services-overview.md](./aws/services-overview.md) | IAM, ALB/NLB, Route 53, ECS vs EKS, TLS/mTLS | SDE-1 |
| [aws/storage-databases.md](./aws/storage-databases.md) | S3, RDS, Aurora, ElastiCache, DynamoDB, EBS vs EFS | SDE-1 |
| [aws/databases-deep-dive.md](./aws/databases-deep-dive.md) | Aurora internals, DocumentDB, DynamoDB single-table design, DAX, 8 scenarios | SDE-2 |
| [aws/messaging-serverless-observability.md](./aws/messaging-serverless-observability.md) | SQS, SNS, EventBridge, Lambda, CloudWatch, Secrets Manager, multi-account, cost | SDE-1/2 |

---

## 7. GCP

| File | Topics | Level |
|------|--------|-------|
| [gcp/README.md](./gcp/README.md) | Global VPC, subnets, firewall rules, Cloud NAT, VPC Peering, Shared VPC | SDE-1/2 |
| [gcp/services-overview.md](./gcp/services-overview.md) | IAM, Workload Identity, Cloud LB, Cloud DNS, Cloud Run vs GKE | SDE-1/2 |

---

## 8. Monitoring & Observability

| File | Topics | Level |
|------|--------|-------|
| [monitoring/prometheus.md](./monitoring/prometheus.md) | Architecture, data model (4 types + math), TSDB internals, 12+ PromQL queries, scrape config, recording rules, 5 production alerts, scenarios | SDE-1/2 |
| [monitoring/alertmanager.md](./monitoring/alertmanager.md) | Routing tree, grouping, inhibition, silences, complete config, debugging | SDE-1/2 |
| [monitoring/grafana.md](./monitoring/grafana.md) | Panel types, variables, USE/RED/SLO dashboards, provisioning as code | SDE-1/2 |
| [monitoring/opentelemetry.md](./monitoring/opentelemetry.md) | Three pillars (traces/metrics/logs), OTEL Collector, Go SDK, auto-instrumentation | SDE-2 |
| [monitoring/loki.md](./monitoring/loki.md) | Architecture, labels vs content, LogQL, Promtail config, trace correlation | SDE-1/2 |
| [monitoring/performance-debugging.md](./monitoring/performance-debugging.md) | USE method, RED method, 60-second checklist, Go pprof, bpftrace one-liners | SDE-2 |
| [monitoring/monitoring-scenarios.md](./monitoring/monitoring-scenarios.md) | 12 scenarios with Prevention: target DOWN, missing metrics, Prometheus OOM, slow PromQL, alert not notifying, alert storm, Grafana no data, Loki missing logs, no OTEL traces, K8s scrape issues | SDE-1/2 |
| [monitoring/slo-sli.md](./monitoring/slo-sli.md) | SLI/SLO/Error Budget math, multi-window multi-burn-rate alerts, recording rules, Grafana SLO dashboard, decision framework | SDE-2 |
| [monitoring/alerting-philosophy.md](./monitoring/alerting-philosophy.md) | Four Golden Signals + PromQL, symptoms vs causes, alert fatigue, urgency tiers, runbook structure, USE vs RED vs Golden Signals | SDE-1/2 |
| [monitoring/thanos-mimir.md](./monitoring/thanos-mimir.md) | Thanos components (sidecar/querier/store/compactor), Mimir distributed TSDB, long-term S3 storage, deduplication, downsampling, when to use each | SDE-2 |
| [sre/db-monitoring.md](./sre/db-monitoring.md) | Prometheus + Grafana for PostgreSQL, MySQL, Redis, MongoDB | SDE-2 |

---

## 9. Git

| File | Topics | Level |
|------|--------|-------|
| [git/git-internals.md](./git/git-internals.md) | Object model (blob/tree/commit/tag), refs, pack files, merge vs rebase internals, reflog | SDE-1/2 |
| [git/git-workflows.md](./git/git-workflows.md) | Trunk-based vs GitFlow vs GitHub Flow, monorepo, branch protection, conventional commits | SDE-1 |
| [git/git-fixes.md](./git/git-fixes.md) | reset modes, amend, reflog recovery, interactive rebase, bisect, detached HEAD, secrets removal | SDE-1 |

---

## 10. Advanced

| File | Topics | Level |
|------|--------|-------|
| [advanced/service-mesh.md](./advanced/service-mesh.md) | Istio control/data plane, VirtualService, mTLS, circuit breaking, Linkerd vs Istio | SDE-2 |
| [advanced/ebpf-observability.md](./advanced/ebpf-observability.md) | eBPF verifier, bpftrace, BCC tools, Cilium, Tetragon, Hubble | SDE-2 |
| [advanced/chaos-engineering.md](./advanced/chaos-engineering.md) | Litmus Chaos, Chaos Mesh, game days, failure injection patterns | SDE-2 |
| [advanced/backup-dr.md](./advanced/backup-dr.md) | RTO/RPO math, Velero, etcd backup, PITR, AWS DR patterns, 3-2-1 rule | SDE-2 |

---

## 11. SRE & Debugging

| File | Topics | Level |
|------|--------|-------|
| [sre/README.md](./sre/README.md) | Index + quick triage cheatsheets | — |
| [sre/k8s-debugging.md](./sre/k8s-debugging.md) | 5XX runbooks (K8s + EKS), OOMKilled recovery | SDE-1/2 |
| [sre/k8s-scenarios.md](./sre/k8s-scenarios.md) | 20 K8s scenarios: CrashLoop, DNS, NetworkPolicy, HPA, rollout, webhooks, etcd, RBAC | SDE-1/2 |
| [sre/linux-debugging.md](./sre/linux-debugging.md) | 10 Linux scenarios: high CPU, I/O wait, zombies, FD exhaustion, inodes, NFS, OOM, kernel panic | SDE-1/2 |
| [sre/aws-scenarios.md](./sre/aws-scenarios.md) | 10 AWS scenarios: EC2 SSH, Lambda timeout, ALB 502, S3 denied, RDS refused, EKS nodes, bill | SDE-1/2 |
| [sre/cicd-scenarios.md](./sre/cicd-scenarios.md) | 8 CI/CD scenarios: GHA OIDC, ArgoCD sync, Jenkins Docker, flaky tests | SDE-1/2 |
| [sre/iac-scenarios.md](./sre/iac-scenarios.md) | 8 IaC scenarios: tf resource exists, state lock, drift, Helm timeout, CF rollback | SDE-1/2 |
| [sre/sre-concepts.md](./sre/sre-concepts.md) | SLOs, error budgets, leader election (client-go Lease API) | SDE-2 |

---

## Full Learning Path

```
── SDE-1 ──────────────────────────────────────────────────
linux/README.md → linux/commands.md → linux/boot.md
linux/systemd.md → linux/networking.md → linux/network-tools.md
linux/containers-evolution.md
docker/README.md → docker/networking.md → docker/debugging.md
kubernetes/kubectl-cheatsheet.md → kubernetes/README.md
kubernetes/workloads.md → kubernetes/resource-limits.md
kubernetes/networking.md → kubernetes/storage.md → kubernetes/rbac.md
kubernetes/helm.md
cicd/README.md → cicd/github-actions → cicd/argocd
iac/terraform → iac/cloudformation
aws/README.md → aws/services-overview.md → aws/storage-databases.md
aws/messaging-serverless-observability.md
monitoring/prometheus.md → monitoring/alertmanager.md → monitoring/grafana.md
monitoring/loki.md
git/git-workflows.md → git/git-fixes.md
sre/k8s-debugging.md → sre/k8s-scenarios.md (first half)
sre/aws-scenarios.md

── SDE-2 ──────────────────────────────────────────────────
linux/io-models.md → linux/scheduler.md → linux/security.md
linux/memory-tuning.md → linux/strace-perf.md
docker/buildkit.md → docker/docker-security.md
kubernetes/autoscaling.md → kubernetes/eks-architecture.md
aws/databases-deep-dive.md
monitoring/opentelemetry.md → monitoring/performance-debugging.md
sre/db-monitoring.md
gcp/README.md → gcp/services-overview.md
git/git-internals.md
advanced/service-mesh.md → advanced/ebpf-observability.md
advanced/chaos-engineering.md → advanced/backup-dr.md
sre/k8s-scenarios.md (advanced) → sre/linux-debugging.md
sre/cicd-scenarios.md → sre/iac-scenarios.md
sre/sre-concepts.md
```
