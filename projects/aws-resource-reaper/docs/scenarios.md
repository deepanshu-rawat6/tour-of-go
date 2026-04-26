# Use Case Scenarios

Real-world scenarios where `aws-resource-reaper` provides immediate value.

---

## 1. Weekly FinOps Audit (Scheduled ECS Task)

Run this on a Friday night via an EventBridge-scheduled ECS Fargate task. It scans all accounts, outputs JSON, and uploads to S3. Your FinOps team reviews the report Monday morning before approving any deletions.

```bash
reaper scan --config /etc/reaper/config.yaml --output json \
  | aws s3 cp - s3://finops-reports/$(date +%Y-%m-%d).json
```

**What it catches:** Accumulated unattached EBS volumes from dev/test environments that engineers forgot to clean up after terminating instances.

---

## 2. Post-Sprint Cleanup in Dev/Staging Accounts

After every sprint, dev accounts accumulate orphaned resources — EIPs left over from quick experiments, security groups from deleted stacks, manual RDS snapshots from debugging sessions. Run the tool against dev/staging accounts only (separate config) and auto-confirm since the risk is low.

```bash
echo "y" | reaper execute --config config.dev.yaml --dry-run=false
```

**What it catches:** EIPs ($3.60/mo each), stale RDS snapshots, unused security groups — small individually but adds up across 10+ dev accounts.

---

## 3. Pre-Migration Cost Baseline (Graviton Recommendation)

Before a cloud cost review with leadership, run a scan to identify all x86 EC2 instances that have a direct Graviton equivalent. The output gives you a concrete migration list with instance-type suggestions already filled in.

```bash
reaper scan --config config.yaml --output json \
  | jq '[.findings[] | select(.Action == "Recommend")]'
```

**What it catches:** `m5.large` → `m6g.large`, `c5.xlarge` → `c6g.xlarge`, etc. Graviton is ~20% cheaper and often faster. A fleet of 50 `m5.large` instances saves ~$500/month.

> Graviton findings are **advisory only** — the tool never terminates instances, only reports them.

---

## 4. Zombie ALB Detection After Service Decommission

When a service is decommissioned, the ALB is often left running because it's not obviously visible in cost reports. At ~$16/month idle, 10 forgotten ALBs = $160/month. The tool queries CloudWatch `RequestCount` over 7 days — if it's zero, the ALB is flagged.

```bash
reaper scan --config config.yaml --output text | grep alb
```

**What it catches:** ALBs left behind after ECS services, EKS ingresses, or Elastic Beanstalk environments were deleted without cleaning up the load balancer.

---

## 5. Compliance & Hygiene Audit (Security Groups)

Unused security groups are a security hygiene issue — they represent attack surface that isn't actively monitored. Before a SOC 2 or ISO 27001 audit, run the tool to produce a list of all SGs not attached to any ENI, then review and delete them.

```bash
reaper scan --config config.yaml --output json \
  | jq '[.findings[] | select(.Resource.Type == "security-group")]' \
  > sg-cleanup-$(date +%Y-%m-%d).json
```

**What it catches:** SGs from deleted EC2 instances, old CloudFormation stacks that didn't clean up, or manually created groups that were never used.

---

## 6. Multi-Account Onboarding Check (New Account Hygiene)

When a new AWS account is provisioned and handed to a team, run the reaper against it after 30 days to catch any resources created during exploration and never cleaned up — before they become a permanent cost line item.

```bash
reaper scan --config config.new-account.yaml --output text
```

**What it catches:** EBS volumes from AMI experiments, Elastic IPs from VPN testing, manual RDS snapshots from initial data migrations — all the "I'll clean this up later" resources that never get cleaned up.

---

## The Common Thread

This tool is most valuable when run **regularly and automatically** (scenario 1), not just once. The dry-run default means you can safely schedule it everywhere and only trigger `execute` after human review.

| Scenario | Accounts | Mode | Frequency |
|----------|----------|------|-----------|
| Weekly FinOps Audit | All prod | `scan` → S3 | Weekly (cron) |
| Post-Sprint Cleanup | Dev/staging only | `execute` | Per sprint |
| Graviton Cost Baseline | All | `scan` (Recommend) | Quarterly |
| Zombie ALB Detection | All prod | `scan` | Weekly |
| Compliance Audit | All | `scan` → JSON | Pre-audit |
| New Account Hygiene | Single new account | `scan` | 30 days post-provision |
