# aws-resource-reaper

A concurrent Go CLI tool for FinOps teams. Scans AWS resources across multiple accounts and regions, evaluates them against cost-optimization rules, and reports — or removes — idle and wasteful resources.

Designed to run on EC2/ECS in a central management account, assuming IAM roles into target accounts via STS. No static credentials required.

---

## How It Works

```
Management Account (EC2/ECS with Instance Profile)
  └─ STS AssumeRole ──► Target Account A
  └─ STS AssumeRole ──► Target Account B
  └─ STS AssumeRole ──► Target Account C
        │
        ▼
  Scan all regions concurrently (errgroup + semaphore)
        │
        ▼
  Evaluate FinOps rules → Findings
        │
        ▼
  Report (text table or JSON)
        │
        ▼  (only with --dry-run=false + confirmation prompt)
  Execute deletions
```

**Dry-run is on by default.** The tool never makes destructive API calls unless you explicitly pass `--dry-run=false` and confirm the prompt.

---

## What It Scans

| Resource | Rule | Action | Est. Savings |
|----------|------|--------|--------------|
| EBS Volumes | State is `available` (unattached) | Delete | $0.10/GB/month |
| Elastic IPs | No `AssociationId` | Delete | ~$3.60/month |
| EC2 Instances | x86 family with a Graviton equivalent | Recommend | ~20% compute cost |
| RDS Snapshots | Manual snapshots older than 30 days | Delete | $0.095/GB/month |
| ALBs | Zero `RequestCount` over last 7 days | Delete | ~$16/month |
| Security Groups | Not attached to any ENI | Delete | $0 (hygiene) |

> `Recommend` actions (Graviton migration) are **advisory only** — no API calls are ever made for them.

---

## Quick Start

### Build

```bash
make build
# or
go build -o reaper ./cmd/reaper
```

### Configure

```bash
cp config.example.yaml config.yaml
# Edit config.yaml with your account IDs, role ARNs, and regions
```

### Scan (dry-run)

```bash
./reaper scan --config config.yaml --output text
./reaper scan --config config.yaml --output json | jq .
```

### Execute (live)

```bash
./reaper execute --config config.yaml --dry-run=false
# Prints the full plan, then prompts: "Proceed? [y/N]"
```

---

## Configuration

```yaml
accounts:
  - id: "123456789012"
    role_arn: "arn:aws:iam::123456789012:role/ResourceReaperReadOnly"
  - id: "234567890123"
    role_arn: "arn:aws:iam::234567890123:role/ResourceReaperReadOnly"

regions:
  - us-east-1
  - us-west-2
  - eu-west-1

concurrency: 10  # max concurrent account×region goroutines
```

See [`config.example.yaml`](./config.example.yaml) for an annotated example.

---

## CLI Reference

```
reaper [command] [flags]

Commands:
  scan      Scan resources and report findings (no changes made)
  execute   Scan and execute remediation (requires --dry-run=false)

Flags:
  -c, --config string      Path to YAML config file (default "config.yaml")
  -o, --output string      Output format: text|json (default "text")
      --concurrency int    Override concurrency limit from config
      --dry-run            Dry-run mode, default true (set false to execute)
```

---

## Example Output

### Text

```
ACCOUNT        REGION     TYPE            ID                 ACTION     ESTIMATED SAVINGS  REASON
-------        ------     ----            --                 ------     -----------------  ------
123456789012   us-east-1  ebs-volume      vol-0abc123        Delete     $10.00/mo          EBS volume is unattached (state: available)
123456789012   us-east-1  elastic-ip      eipalloc-0abc      Delete     $3.60/mo           Elastic IP is not associated with any resource
234567890123   eu-west-1  ec2-instance    i-0abc123def       Recommend  $0.00/mo           Migrate from m5.large to m6g.large (Graviton) for ~20% cost reduction
234567890123   eu-west-1  alb             arn:aws:...        Delete     $16.00/mo          ALB has received zero requests in the last 7 days

Total findings: 4 | Estimated monthly savings: $29.60
```

### JSON

```bash
reaper scan --output json | jq '.estimated_monthly_savings'
# 29.60
```

---

## IAM Setup

### Management Account — Instance Profile

```json
{
  "Effect": "Allow",
  "Action": "sts:AssumeRole",
  "Resource": "arn:aws:iam::*:role/ResourceReaperReadOnly"
}
```

### Target Account — Cross-Account Role (Read-Only)

Trust policy principal: `arn:aws:iam::MANAGEMENT_ACCOUNT_ID:root`

Permissions needed for `scan`:
```
ec2:DescribeVolumes, ec2:DescribeAddresses, ec2:DescribeInstances,
ec2:DescribeNetworkInterfaces, ec2:DescribeSecurityGroups,
elasticloadbalancing:DescribeLoadBalancers,
cloudwatch:GetMetricStatistics, rds:DescribeDBSnapshots
```

Additional permissions for `execute`:
```
ec2:DeleteVolume, ec2:ReleaseAddress, ec2:TerminateInstances,
ec2:DeleteSecurityGroup, elasticloadbalancing:DeleteLoadBalancer,
rds:DeleteDBSnapshot
```

See [`docs/usage.md`](./docs/usage.md) for full IAM policy JSON.

---

## Project Structure

```
cmd/reaper/          — Cobra CLI entrypoint
internal/
  config/            — YAML config loader
  auth/              — STS AssumeRole provider with caching
  scanner/           — Scanner interface, engine, 6 resource scanners
  rules/             — Rule interface + 6 FinOps evaluators
  report/            — text/tabwriter + JSON formatter
  executor/          — AWS delete/terminate APIs + slog audit logging
docs/
  usage.md           — Full runbook (IAM, deployment, CI, concurrency tuning)
  scenarios.md       — Real-world use case scenarios
config.example.yaml
Makefile
```

---

## Development

```bash
make test    # run all tests
make lint    # go vet
make tidy    # go mod tidy
make build   # compile binary
```

All packages have unit tests with mock AWS clients — no real AWS credentials needed to run tests.

---

## Docs

- [Full Usage Guide & Runbook](./docs/usage.md)
- [Design Decisions](./docs/design-decisions.md)
- [Use Case Scenarios](./docs/scenarios.md)
