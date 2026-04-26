# Usage Guide — tf-drift-detector

## Prerequisites

- Go 1.22+
- AWS credentials (env vars, `~/.aws`, or EC2 instance profile)
- Terraform state accessible via S3 or local file

## IAM Permissions Required

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "ec2:DescribeInstances",
        "ec2:DescribeSecurityGroups",
        "rds:DescribeDBInstances",
        "iam:GetRole",
        "lambda:GetFunction",
        "s3:GetBucketVersioning",
        "s3:GetBucketEncryption",
        "s3:GetBucketTagging"
      ],
      "Resource": "*"
    }
  ]
}
```

## Config File Format

| Field | Type | Description |
|-------|------|-------------|
| `backend.type` | string | `s3` or `local` |
| `backend.bucket` | string | S3 bucket name (s3 only) |
| `backend.key` | string | S3 object key (s3 only) |
| `backend.region` | string | S3 bucket region (s3 only) |
| `backend.file_path` | string | Local tfstate path (local only) |
| `aws_region` | string | Region for live AWS API calls |
| `interval` | string | Daemon check interval (e.g. `5m`, `1h`) |
| `concurrency` | int | Max concurrent API calls (default: 10) |
| `drift_state_file` | string | Path to persist drift state JSON |
| `alerts.stdout` | bool | Print JSON to stdout |
| `alerts.slack.webhook_url` | string | Slack incoming webhook URL |
| `alerts.discord.webhook_url` | string | Discord webhook URL |
| `ignore_fields` | map | Extra fields to ignore per resource type |

## CLI Reference

```bash
# One-shot check
detect run --config config.yaml

# Daemon with default interval from config
detect daemon --config config.yaml

# Daemon with interval override
detect daemon --config config.yaml --interval 10m

# Help
detect --help
detect run --help
detect daemon --help
```

## Ignore Rules Guide

The diff engine has two layers of false-positive suppression:

### Built-in (hardcoded)

These fields are always ignored regardless of config:

| Resource | Ignored Fields |
|----------|---------------|
| `aws_instance` | `metadata_options`, `credit_specification`, `enclave_options`, `root_block_device.0.volume_id` |
| `aws_s3_bucket` | `arn`, `bucket_domain_name`, `hosted_zone_id`, `region`, `website_endpoint` |
| `aws_security_group` | `owner_id` |
| `aws_db_instance` | `address`, `endpoint`, `hosted_zone_id`, `latest_restorable_time` |
| `aws_iam_role` | `create_date`, `unique_id` |
| `aws_lambda_function` | `arn`, `invoke_arn`, `last_modified`, `source_code_hash`, `version` |

### Config-driven

Add your own in `config.yaml`:

```yaml
ignore_fields:
  aws_instance:
    - "user_data"          # ignore user_data changes
    - "ebs_optimized"      # managed outside TF in your org
  aws_lambda_function:
    - "environment.MY_ENV" # environment variable managed separately
```

## Deployment

### Local / CI

```bash
make build
./detect run --config config.yaml
```

### EC2 with instance profile

```bash
# No credentials needed — uses IMDS automatically
./detect daemon --config /etc/tf-drift/config.yaml --interval 5m
```

### Systemd service

```ini
[Unit]
Description=Terraform Drift Detector

[Service]
ExecStart=/usr/local/bin/detect daemon --config /etc/tf-drift/config.yaml
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

### Docker

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o detect ./cmd/detect

FROM alpine:3.19
COPY --from=builder /app/detect /usr/local/bin/detect
COPY config.yaml /etc/tf-drift/config.yaml
ENTRYPOINT ["detect", "daemon", "--config", "/etc/tf-drift/config.yaml"]
```
