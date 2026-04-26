# Use Cases & Scenarios

## 1. Catching Manual Console Changes

**Scenario:** An engineer SSHes into an EC2 instance and changes the instance type via the AWS console to handle a traffic spike. They forget to update the Terraform code.

**How tf-drift-detector helps:**
```
🚨 Terraform Drift Detected — 1 resource(s)
• aws_instance / i-0abc123def456
  - instance_type: t3.micro → t3.large
```
The team is alerted within one polling cycle. They can either update the TF code to match, or revert the instance type.

---

## 2. Security Compliance Monitoring

**Scenario:** Your security policy requires all S3 buckets to have server-side encryption enabled. Someone disables it on a bucket directly via the CLI.

**How tf-drift-detector helps:**
```
🚨 Terraform Drift Detected — 1 resource(s)
• aws_s3_bucket / my-data-bucket
  - sse_algorithm: AES256 → (none)
```
The drift is caught immediately. Combined with a Slack alert to the security channel, this becomes an automated compliance check.

---

## 3. Lambda Configuration Drift in Production

**Scenario:** A developer increases a Lambda function's memory from 128MB to 512MB directly in the console during an incident. The incident is resolved but the change is never committed to Terraform.

**How tf-drift-detector helps:**
```
🚨 Terraform Drift Detected — 1 resource(s)
• aws_lambda_function / payment-processor
  - memory_size: 128 → 512
  - timeout: 30 → 60
```
The daemon catches this on the next cycle. The team can decide whether to codify the change or revert it.

---

## 4. Drift Resolution Tracking

**Scenario:** You've been tracking drift on an IAM role for 3 days. The team finally applies `terraform apply` to fix it.

**How tf-drift-detector helps:**
```
✅ Drift Resolved — 1 resource(s)
• aws_iam_role / my-service-role
```
The daemon detects that the live state now matches TF state and sends a resolution alert. The drift record is removed from the persisted state file.

---

## 5. Pre-Deployment Drift Check in CI

**Scenario:** Before running `terraform plan` in CI, you want to know if there's existing drift that would cause the plan to be noisy.

```bash
# In your CI pipeline
./detect run --config config.yaml --output json > drift-report.json
if [ $(jq '.new_drift | length' drift-report.json) -gt 0 ]; then
  echo "WARNING: Drift detected before deployment"
  cat drift-report.json
fi
```

---

## 6. Multi-Environment Drift Monitoring

**Scenario:** You manage dev, staging, and prod environments each with their own Terraform state. You want drift monitoring for all three.

Run three daemon instances with separate configs:

```bash
./detect daemon --config config.dev.yaml &
./detect daemon --config config.staging.yaml &
./detect daemon --config config.prod.yaml &
```

Each has its own `drift_state_file` and alert channels (e.g., prod alerts go to `#alerts-prod`, dev to `#alerts-dev`).

---

## Summary

| Scenario | Mode | Alert Channel | Frequency |
|----------|------|---------------|-----------|
| Console change detection | daemon | Slack | 5m |
| Security compliance | daemon | Slack #security | 5m |
| Lambda config drift | daemon | Discord | 5m |
| Drift resolution tracking | daemon | Slack | 5m |
| Pre-deployment CI check | run | stdout/JSON | Per deploy |
| Multi-environment | daemon × N | Per-env channels | 5m each |
