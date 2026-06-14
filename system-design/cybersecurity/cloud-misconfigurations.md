# AWS Cloud Misconfigurations

> **Level:** SDE-2 | **Domain:** Security / Cloud

---

## 1. Misconfiguration Taxonomy

```mermaid
mindmap
  root((AWS<br/>Misconfigs))
    Identity
      IAM * actions
      No MFA on root
      Long-lived keys
      Unused roles
    Network
      SG 0.0.0.0/0
      S3 public access
      No VPC flow logs
      RDS public endpoint
    Data
      S3 no encryption
      EBS unencrypted
      Public RDS snapshot
      Secrets in env vars
    Logging
      CloudTrail disabled
      No S3 data events
      No Lambda logging
      GuardDuty off
```

| Category | Top Risk | Quick Fix |
|----------|----------|-----------|
| Identity | `"Action": "*"` IAM policies | Least-privilege; SCPs |
| Network | SG port 22 open to world | Restrict to known CIDRs / bastion |
| Data | Public S3 bucket | Block Public Access at account level |
| Logging | CloudTrail management-only | Enable data events for S3 + Lambda |

---

## 2. S3 Bucket Misconfiguration

### Attack Flow

```mermaid
flowchart LR
    A[Attacker] --> B[Discover<br/>bucket name]
    B --> C{Public<br/>access?}
    C -->|Yes| D[List objects<br/>s3:ListBucket]
    C -->|No| E[403 / blocked]
    D --> F[Download<br/>s3:GetObject]
    F --> G[Exfiltrate<br/>PII / secrets]

    B1[subdomain enum<br/>e.g. assets.corp.com] --> B
    B2[JS source<br/>s3.amazonaws.com/corp-] --> B
    B3[Error msg<br/>NoSuchBucket] --> B
```

### Block Public Access — Settings Hierarchy

```mermaid
flowchart TD
    ACC[Account-level BPA<br/>blocks all] -->|overrides| BKT[Bucket-level BPA]
    BKT -->|overrides| ACL[Bucket/Object ACLs]
    BKT -->|overrides| POL[Bucket Policy]

    style ACC fill:#d32f2f,color:#fff
    style BKT fill:#f57c00,color:#fff
    style ACL fill:#388e3c,color:#fff
    style POL fill:#388e3c,color:#fff
```

**Four BPA flags** (all should be `true`):

| Flag | Blocks |
|------|--------|
| `BlockPublicAcls` | New public ACLs |
| `IgnorePublicAcls` | Existing public ACLs |
| `BlockPublicPolicy` | New public bucket policies |
| `RestrictPublicBuckets` | Cross-account public access |

```bash
# Enforce at account level
aws s3control put-public-access-block \
  --account-id 123456789012 \
  --public-access-block-configuration \
    BlockPublicAcls=true,IgnorePublicAcls=true,\
    BlockPublicPolicy=true,RestrictPublicBuckets=true
```

---

## 3. IMDSv1 vs IMDSv2 — SSRF Attack Path

### IMDSv1 SSRF (no token required)

```mermaid
sequenceDiagram
    participant A as Attacker
    participant S as App Server
    participant M as IMDS<br/>169.254.169.254

    A->>S: GET /fetch?url=http://169.254.169.254/<br/>latest/meta-data/iam/security-credentials/
    S->>M: HTTP GET (no auth)
    M-->>S: role-name
    S->>M: GET /…/role-name
    M-->>S: AccessKeyId<br/>SecretAccessKey<br/>Token
    S-->>A: IAM keys returned
```

### IMDSv2 Blocks SSRF

```mermaid
sequenceDiagram
    participant A as Attacker
    participant S as App Server
    participant M as IMDS v2

    A->>S: GET /fetch?url=http://169.254.169.254/…
    S->>M: HTTP GET (no token)
    M-->>S: 401 Unauthorized
    Note over A,M: SSRF fails — PUT required for token
    Note over M: PUT /latest/api/token<br/>TTL-header mandatory<br/>SSRF cannot do PUT
```

### Risk Model

```
IMDSv1 risk = P(SSRF exists) × P(metadata reachable)
                                 ↑ ~1.0 (always reachable)

IMDSv2 risk = P(SSRF exists) × P(SSRF can do PUT with headers)
                                 ↑ ~0.0 (standard SSRF is GET-only)
```

**Enforcement:**

```bash
# Require IMDSv2 on existing instance
aws ec2 modify-instance-metadata-options \
  --instance-id i-1234567890abcdef0 \
  --http-tokens required \
  --http-put-response-hop-limit 1

# Enforce via Launch Template
aws ec2 create-launch-template-version \
  --launch-template-id lt-xxx \
  --launch-template-data '{"MetadataOptions":{"HttpTokens":"required"}}'
```

---

## 4. Overpermissioned IAM

### Dangerous vs Least-Privilege Policy

```json
// ❌ Wildcard — grants everything
{
  "Effect": "Allow",
  "Action": "*",
  "Resource": "*"
}

// ✅ Least-privilege
{
  "Effect": "Allow",
  "Action": ["s3:GetObject", "s3:PutObject"],
  "Resource": "arn:aws:s3:::my-bucket/*"
}
```

### Privilege Escalation: PassRole + CreateFunction

```mermaid
flowchart TD
    ATK[Attacker<br/>limited IAM user] --> PR[iam:PassRole<br/>permission]
    ATK --> CF[lambda:CreateFunction<br/>permission]
    PR --> ESC{Privilege<br/>Escalation}
    CF --> ESC
    ESC --> LAM[Create Lambda<br/>with admin role attached]
    LAM --> EXE[Invoke Lambda<br/>executes as admin role]
    EXE --> FULL[Full AWS access<br/>arbitrary code]

    style ESC fill:#d32f2f,color:#fff
    style FULL fill:#b71c1c,color:#fff
```

**Other dangerous permission combos:**

| Permission A | Permission B | Result |
|---|---|---|
| `iam:PassRole` | `ec2:RunInstances` | Launch EC2 as any role |
| `iam:PassRole` | `glue:CreateJob` | Run Glue job as any role |
| `iam:CreatePolicyVersion` | — | Overwrite policy with `*` |
| `iam:AttachUserPolicy` | — | Attach managed admin policy to self |

**Detection:** AWS IAM Access Analyzer + `cloudsplaining` tool.

---

## 5. Public RDS Snapshots

### Exposure Mechanism

```mermaid
flowchart LR
    DEV[Developer] -->|modify-db-snapshot-attribute<br/>--attribute-name restore<br/>--values-to-add all| SNAP[RDS Snapshot<br/>visibility: public]
    SNAP --> AWS[AWS public<br/>snapshot registry]
    AWS -->|automated scan<br/>every ~15 min| ATK[Attacker tool<br/>e.g. RDS Snapshot Hunter]
    ATK --> RST[Restore snapshot<br/>to attacker account]
    RST --> DB[Full DB access<br/>no password needed]

    style SNAP fill:#d32f2f,color:#fff
    style DB fill:#b71c1c,color:#fff
```

### Attacker Enumeration

```bash
# Attackers run this to find public snapshots
aws rds describe-db-snapshots \
  --snapshot-type public \
  --region us-east-1 \
  --query 'DBSnapshots[?contains(DBSnapshotIdentifier,`corp`)]'
```

### AWS Config Rule Detection

```json
{
  "ConfigRuleName": "rds-snapshots-public-prohibited",
  "Source": {
    "Owner": "AWS",
    "SourceIdentifier": "RDS_SNAPSHOTS_PUBLIC_PROHIBITED"
  }
}
```

**Auto-remediation SSM document:** `AWS-DisablePublicAccessForRDSSnapshot`

---

## 6. Security Group Misconfigurations

### Dangerous Open Ports

| Port | Service | Risk |
|------|---------|------|
| 22 | SSH | Brute-force, credential stuffing |
| 3389 | RDP | BlueKeep, ransomware entry |
| 5432 | PostgreSQL | Direct DB access, SQL injection |
| 27017 | MongoDB | Unauthenticated access common |
| 0 | All traffic | Full exposure |

### Correct vs Incorrect SG Design

```mermaid
flowchart TD
    subgraph WRONG["❌ Incorrect"]
        I1[Internet<br/>0.0.0.0/0] -->|port 22| W1[EC2 Instance]
        I1 -->|port 5432| W2[RDS Database]
        I1 -->|port 3389| W3[Windows EC2]
    end

    subgraph RIGHT["✅ Correct"]
        I2[Internet] -->|443 only| ALB[ALB / NLB]
        ALB -->|8080 from ALB SG| R1[EC2 App]
        R1 -->|5432 from App SG| R2[RDS Private]
        OPS[Ops CIDR<br/>10.0.0.0/8] -->|22 bastion only| BAS[Bastion Host]
        BAS -->|22 from Bastion SG| R1
    end
```

```bash
# Detect with AWS Config managed rule
aws configservice put-config-rule --config-rule '{
  "ConfigRuleName": "restricted-ssh",
  "Source": {"Owner":"AWS","SourceIdentifier":"INCOMING_SSH_DISABLED"}
}'

# Audit open SGs with CLI
aws ec2 describe-security-groups \
  --filters Name=ip-permission.cidr,Values='0.0.0.0/0' \
  --query 'SecurityGroups[*].[GroupId,GroupName,IpPermissions]'
```

---

## 7. CloudTrail Gaps

### What Is / Isn't Logged by Default

```mermaid
flowchart TD
    CT[CloudTrail<br/>Trail] --> ME[Management Events<br/>✅ on by default]
    CT --> DE[Data Events<br/>❌ off by default]
    CT --> IN[Insights Events<br/>❌ off by default]

    ME --> ME1[CreateBucket]
    ME --> ME2[RunInstances]
    ME --> ME3[CreateUser]

    DE --> DE1[S3 GetObject<br/>PutObject]
    DE --> DE2[Lambda Invoke]
    DE --> DE3[DynamoDB GetItem]

    IN --> IN1[Unusual API<br/>call volume]

    style DE fill:#d32f2f,color:#fff
    style IN fill:#f57c00,color:#fff
```

### Attacker Blind Spots

| Action | Logged? | Attacker Advantage |
|--------|---------|-------------------|
| `s3:GetObject` (data exfil) | ❌ default | Silently download all objects |
| `lambda:InvokeFunction` | ❌ default | Run malicious Lambda undetected |
| `dynamodb:GetItem` | ❌ default | Query DB without trace |
| `ec2:DescribeInstances` | ✅ | Recon is visible |
| `iam:CreateUser` | ✅ | Persistence attempt visible |

### Enable Data Events

```bash
aws cloudtrail put-event-selectors \
  --trail-name my-trail \
  --event-selectors '[
    {
      "ReadWriteType": "All",
      "IncludeManagementEvents": true,
      "DataResources": [
        {"Type": "AWS::S3::Object", "Values": ["arn:aws:s3:::"]},
        {"Type": "AWS::Lambda::Function", "Values": ["arn:aws:lambda"]}
      ]
    }
  ]'
```

**Additional gaps:**
- Route53 DNS query logs (off by default)
- VPC Flow Logs (off by default — set to `ACCEPT/REJECT ALL`)
- RDS slow query / error logs
- CloudFront access logs

---

## 8. Detection: AWS Config + GuardDuty Auto-Remediation

### Detection & Response Pipeline

```mermaid
flowchart TD
    MC[Misconfiguration<br/>occurs] --> CR[Config Rule<br/>evaluates]
    CR -->|NON_COMPLIANT| SNS[SNS Topic<br/>alert]
    SNS --> SL[Slack / PagerDuty<br/>notify team]
    SNS --> LM[Lambda<br/>auto-remediate]
    LM -->|e.g. revoke SG rule| FIX[Resource<br/>fixed]
    LM --> CT2[CloudTrail<br/>log remediation]

    GD[GuardDuty<br/>finding] --> EB[EventBridge<br/>rule match]
    EB --> LM2[Lambda<br/>isolate instance]
    LM2 -->|quarantine SG| ISO[Instance<br/>isolated]
    LM2 --> JIRA[Jira ticket<br/>created]

    style MC fill:#d32f2f,color:#fff
    style FIX fill:#388e3c,color:#fff
    style ISO fill:#388e3c,color:#fff
```

### Key GuardDuty Finding Types

| Finding | Indicates |
|---------|-----------|
| `UnauthorizedAccess:IAMUser/ConsoleLoginSuccess.B` | Login from unusual location |
| `Recon:IAMUser/MaliciousIPCaller` | Recon from known bad IP |
| `CredentialAccess:EC2/MetadataDNSRebind` | IMDS DNS rebind attack |
| `Exfiltration:S3/MaliciousIPCaller` | Data exfil to bad IP |
| `Persistence:IAMUser/UserPermissions` | Attacker adding permissions |

### Lambda Auto-Remediation Skeleton

```python
import boto3, json

def handler(event, context):
    detail = event['detail']
    config_item = detail['configurationItem']
    resource_type = config_item['resourceType']

    if resource_type == 'AWS::EC2::SecurityGroup':
        remediate_sg(config_item['resourceId'])
    elif resource_type == 'AWS::S3::Bucket':
        remediate_s3(config_item['resourceId'])

def remediate_sg(sg_id):
    ec2 = boto3.client('ec2')
    sg = ec2.describe_security_groups(GroupIds=[sg_id])['SecurityGroups'][0]
    for perm in sg['IpPermissions']:
        for r in perm.get('IpRanges', []):
            if r['CidrIp'] in ('0.0.0.0/0', '::/0') and perm.get('FromPort') == 22:
                ec2.revoke_security_group_ingress(GroupId=sg_id, IpPermissions=[perm])

def remediate_s3(bucket):
    s3 = boto3.client('s3control',
        region_name='us-east-1')
    # Enable BPA on bucket
    boto3.client('s3').put_public_access_block(
        Bucket=bucket,
        PublicAccessBlockConfiguration={
            'BlockPublicAcls': True, 'IgnorePublicAcls': True,
            'BlockPublicPolicy': True, 'RestrictPublicBuckets': True
        }
    )
```

---

## Quick-Reference Checklist

```
Identity
  [ ] No IAM wildcard Action/Resource in policies
  [ ] MFA on root + all human users
  [ ] Rotate/delete unused access keys (>90 days)
  [ ] Use IAM roles for EC2/Lambda (no embedded keys)
  [ ] Enable IAM Access Analyzer

Network
  [ ] No SG inbound 0.0.0.0/0 on 22/3389/5432
  [ ] S3 Block Public Access ON at account level
  [ ] VPC Flow Logs enabled (ACCEPT + REJECT)
  [ ] RDS not publicly accessible

Data
  [ ] S3 default encryption (SSE-S3 minimum, SSE-KMS preferred)
  [ ] EBS volumes encrypted at launch
  [ ] RDS snapshots private
  [ ] Secrets in Secrets Manager, not env vars

Logging
  [ ] CloudTrail in all regions + S3 data events
  [ ] GuardDuty enabled in all regions
  [ ] AWS Config rules deployed
  [ ] CloudWatch alarms on root login, config changes

Compute
  [ ] IMDSv2 required (HttpTokens=required)
  [ ] IMDS hop limit = 1 (blocks container escape)
  [ ] No public ECR repositories with sensitive images
```
