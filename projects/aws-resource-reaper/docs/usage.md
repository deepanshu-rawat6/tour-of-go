# aws-resource-reaper — Usage Guide

A concurrent Go CLI tool for FinOps teams. Scans AWS resources across multiple accounts and regions, evaluates them against cost-optimization rules, and reports (or removes) idle/wasteful resources.

---

## Table of Contents

1. [How It Works](#how-it-works)
2. [Prerequisites](#prerequisites)
3. [IAM Setup](#iam-setup)
4. [Configuration File](#configuration-file)
5. [CLI Reference](#cli-reference)
6. [Deployment on EC2 / ECS](#deployment-on-ec2--ecs)
7. [Local Development](#local-development)
8. [Concurrency Tuning](#concurrency-tuning)
9. [Example Output](#example-output)
10. [Running in CI](#running-in-ci)
11. [FinOps Rules Reference](#finops-rules-reference)

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
        ▼ (only with --dry-run=false + confirmation)
  Execute deletions / terminations
```

The tool uses the **AWS SDK default credential chain** — it picks up credentials from:
1. EC2 Instance Metadata Service (IMDS) — when running on EC2/ECS with an instance profile
2. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`)
3. `~/.aws/credentials` — for local development

No credentials are stored in the config file.

---

## Prerequisites

- Go 1.22+ (for building from source)
- AWS credentials available via one of the methods above
- The management account role must have `sts:AssumeRole` permission on each target account role

---

## IAM Setup

### 1. Management Account — Instance Profile Policy

Attach this policy to the EC2 instance profile (or ECS task role) in the management account:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "sts:AssumeRole",
      "Resource": "arn:aws:iam::*:role/ResourceReaperReadOnly"
    }
  ]
}
```

### 2. Target Account — Cross-Account Role

Create a role named `ResourceReaperReadOnly` in each target account with this trust policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::MANAGEMENT_ACCOUNT_ID:root"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

#### Read-Only Permissions Policy (for `scan` only)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeVolumes",
        "ec2:DescribeAddresses",
        "ec2:DescribeInstances",
        "ec2:DescribeNetworkInterfaces",
        "ec2:DescribeSecurityGroups",
        "elasticloadbalancing:DescribeLoadBalancers",
        "cloudwatch:GetMetricStatistics",
        "rds:DescribeDBSnapshots"
      ],
      "Resource": "*"
    }
  ]
}
```

#### Additional Permissions for `execute` (destructive actions)

Add these to the target account role when using `execute --dry-run=false`:

```json
{
  "Effect": "Allow",
  "Action": [
    "ec2:DeleteVolume",
    "ec2:ReleaseAddress",
    "ec2:TerminateInstances",
    "ec2:DeleteSecurityGroup",
    "elasticloadbalancing:DeleteLoadBalancer",
    "rds:DeleteDBSnapshot"
  ],
  "Resource": "*"
}
```

> **Recommendation:** Keep a separate `ResourceReaperExecute` role with destructive permissions and only use it when you intend to apply changes.

---

## Configuration File

Copy `config.example.yaml` and edit it:

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

concurrency: 10
```

| Field | Description |
|-------|-------------|
| `accounts[].id` | AWS account ID (12-digit string) |
| `accounts[].role_arn` | Full ARN of the role to assume in that account |
| `regions` | List of AWS regions to scan |
| `concurrency` | Max concurrent account×region goroutines (default: 10) |

---

## CLI Reference

```
reaper [command] [flags]
```

### Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config`, `-c` | `config.yaml` | Path to YAML config file |
| `--output`, `-o` | `text` | Output format: `text` or `json` |
| `--concurrency` | (from config) | Override concurrency limit |
| `--dry-run` | `true` | Dry-run mode; set `false` to enable live execution |

### Commands

#### `reaper scan`

Scans all configured accounts and regions, evaluates FinOps rules, and prints a report. **No changes are made.**

```bash
reaper scan --config config.yaml --output text
reaper scan --config config.yaml --output json | jq '.findings[] | select(.action == "Delete")'
```

#### `reaper execute`

Scans, reports, then **executes** the remediation actions after an interactive confirmation prompt.

```bash
# This will prompt before making any changes:
reaper execute --config config.yaml --dry-run=false

# Dry-run execute (same as scan — no changes):
reaper execute --config config.yaml
```

---

## Deployment on EC2 / ECS

### EC2

1. Build the binary:
   ```bash
   make build
   ```
2. Copy to the EC2 instance:
   ```bash
   scp reaper ec2-user@<instance-ip>:/usr/local/bin/reaper
   scp config.yaml ec2-user@<instance-ip>:~/config.yaml
   ```
3. Ensure the instance has an IAM instance profile with `sts:AssumeRole` permission.
4. Run:
   ```bash
   reaper scan --config ~/config.yaml --output json > findings.json
   ```

### ECS (Fargate)

1. Build and push a Docker image:
   ```dockerfile
   FROM golang:1.22-alpine AS builder
   WORKDIR /app
   COPY . .
   RUN go build -o reaper ./cmd/reaper

   FROM alpine:3.19
   COPY --from=builder /app/reaper /usr/local/bin/reaper
   COPY config.yaml /etc/reaper/config.yaml
   ENTRYPOINT ["reaper"]
   ```
2. Assign a task role with `sts:AssumeRole` permission.
3. Run as a scheduled task (EventBridge Scheduler) for periodic FinOps reports.

### Cron / Scheduled Scan

```bash
# Run daily at 8am UTC, output JSON to S3
0 8 * * * reaper scan --config /etc/reaper/config.yaml --output json | aws s3 cp - s3://my-finops-bucket/reports/$(date +\%Y-\%m-\%d).json
```

---

## Local Development

```bash
# Use a named AWS profile
export AWS_PROFILE=my-dev-profile

# Or use environment variables
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export AWS_SESSION_TOKEN=...   # if using temporary credentials

# Build and run
make build
./reaper scan --config config.example.yaml --output text
```

---

## Concurrency Tuning

The `concurrency` setting controls how many `account × region` pairs are scanned simultaneously. Each slot makes multiple AWS API calls (EC2, RDS, ELBv2, CloudWatch).

| Fleet Size | Recommended Concurrency | Notes |
|------------|------------------------|-------|
| 1–5 accounts, 1–5 regions | 5–10 | Default is fine |
| 5–20 accounts, 5–10 regions | 10–20 | Monitor for throttling |
| 20+ accounts or 10+ regions | 20–50 | Enable AWS API rate limit monitoring; consider request retries |

AWS SDK v2 has built-in retry logic with exponential backoff. If you see `ThrottlingException` errors, reduce concurrency.

Override at runtime without editing the config:
```bash
reaper scan --config config.yaml --concurrency 5
```

---

## Example Output

### Text Format

```
ACCOUNT        REGION     TYPE            ID                    ACTION     ESTIMATED SAVINGS  REASON
-------        ------     ----            --                    ------     -----------------  ------
123456789012   us-east-1  ebs-volume      vol-0abc123def456     Delete     $10.00/mo          EBS volume is unattached (state: available)
123456789012   us-east-1  elastic-ip      eipalloc-0abc123      Delete     $3.60/mo           Elastic IP is not associated with any resource
234567890123   eu-west-1  ec2-instance    i-0abc123def456789    Recommend  $0.00/mo           Migrate from m5.large to m6g.large (Graviton) for ~20% cost reduction
234567890123   eu-west-1  alb             arn:aws:elasticlo...  Delete     $16.00/mo          ALB has received zero requests in the last 7 days

Total findings: 4 | Estimated monthly savings: $29.60
```

### JSON Format

```json
{
  "findings": [
    {
      "Resource": {
        "ID": "vol-0abc123def456",
        "Type": "ebs-volume",
        "Region": "us-east-1",
        "AccountID": "123456789012",
        "Tags": {"Name": "old-data-volume"},
        "Metadata": {"size_gb": "100", "volume_type": "gp2"}
      },
      "Action": "Delete",
      "Reason": "EBS volume is unattached (state: available)",
      "EstimatedMonthlySavings": 10.00
    }
  ],
  "total_findings": 1,
  "estimated_monthly_savings": 10.00
}
```

---

## Running in CI

For non-interactive execution in CI pipelines (e.g., GitHub Actions, Jenkins):

```bash
# Pipe "y" to auto-confirm (use with caution in production pipelines)
echo "y" | reaper execute --config config.yaml --dry-run=false

# Safer: scan only in CI, review findings, execute manually
reaper scan --config config.yaml --output json > findings.json
```

For CI, it's recommended to run `scan` only and store the JSON output as an artifact for human review before running `execute`.

---

## FinOps Rules Reference

| Rule | Resource Type | Action | Trigger Condition | Est. Savings |
|------|--------------|--------|-------------------|--------------|
| UnattachedEBSRule | `ebs-volume` | Delete | Volume state is `available` (not attached) | $0.10/GB/month |
| UnattachedEIPRule | `elastic-ip` | Delete | No `AssociationId` | ~$3.60/month |
| GravitonMigrationRule | `ec2-instance` | Recommend | Instance family in `m5`, `c5`, `r5`, `t3` (x86) | ~20% compute cost |
| OldRDSSnapshotRule | `rds-snapshot` | Delete | Manual snapshot older than 30 days | $0.095/GB/month |
| ZeroTrafficALBRule | `alb` | Delete | `RequestCount` sum = 0 over last 7 days | ~$16/month |
| UnusedSGRule | `security-group` | Delete | Not referenced by any ENI | $0 (hygiene) |

> **Note:** `Recommend` actions (Graviton migration) are **never executed** — they are advisory only. The tool will log them but make no API calls for them.
