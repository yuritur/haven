# Haven — Architecture

> Last updated: 2026-03
> Version: v0.1 MVP

---

## Deploy flow

```
haven deploy llama3.2:1b
  │
  ├─ 1. Validate AWS credentials                (STS GetCallerIdentity)
  ├─ 2. Bootstrap S3 state bucket if needed     (haven-state-{account_id}, once per account)
  ├─ 3. Generate CloudFormation template        (VPC, SG, EC2, EIP, IAM)
  ├─ 4. CloudFormation CreateStack
  │     └─ creates: VPC, Subnet, IGW, Route table, SG, EC2 (t3.large), EIP, IAM role
  ├─ 5. Poll stack events + show progress to user
  ├─ 6. EC2 runs user_data bootstrap            (~3–5 min: install Ollama, pull model)
  ├─ 7. Poll /api/tags (Ollama health) via HTTP until ready
  └─ 8. Save state JSON to S3, print endpoint URL + API key
```

---

## Module layout

```
cmd/haven/
  main.go                    # entry point

internal/
  aws/
    credentials.go           # STS GetCallerIdentity, region detection
    bootstrap.go             # Create S3 state bucket once per account
  cfn/
    template.go              # Generate CloudFormation JSON template
    deploy.go                # CreateStack, poll events, wait for complete
    destroy.go               # DeleteStack, wait for complete
  models/
    registry.go              # model name → instance type + ollama pull tag
  state/
    manager.go               # read/write deployment JSON to S3
  cli/
    deploy.go                # haven deploy <model> command
    destroy.go               # haven destroy <id> command
    status.go                # haven status command
```

---

## AWS resources per deployment

| Resource | Config |
|---|---|
| VPC | /16 CIDR, DNS hostnames enabled |
| Public subnet | /24, auto-assign public IP |
| Internet Gateway | 1 per VPC |
| Security Group | Inbound: 11434/tcp (Ollama) from user IP. Outbound: all |
| EC2 instance | t3.large, Amazon Linux 2023 AMI |
| EBS root volume | gp3, 30 GB, encrypted at rest |
| Elastic IP | Stable public IP, survives instance stop/start |
| IAM role | AmazonSSMManagedInstanceCore only |

**Shared per account (created once):**
- S3 bucket: `haven-state-{account_id}` — versioning on, public access blocked

---

## Serving backends

| Tier | Instance | Backend | Models |
|---|---|---|---|
| MVP | t3.large | Ollama (CPU) | Llama 3.2 1B, Phi 3 mini |

---

## State schema

Deployment state is stored as JSON in S3 at `s3://haven-state-{account_id}/deployments/{deployment_id}.json`:

```json
{
  "deployment_id": "haven-abc123",
  "created_at": "2026-03-01T12:00:00Z",
  "region": "us-east-1",
  "stack_name": "haven-abc123",
  "model": "llama3.2:1b",
  "instance_type": "t3.large",
  "resources": {
    "instance_id": "i-0abc...",
    "eip": "1.2.3.4"
  },
  "endpoint": "http://1.2.3.4:11434/v1",
  "api_key": "sk-haven-..."
}
```

---

## Security posture (v0.1)

| Concern | Mitigation |
|---|---|
| API authentication | Ollama `--api-key` (Bearer token) |
| Network access | Security Group restricts port 11434 to user's IP |
| Instance access | SSM only — no SSH, no port 22 |
| Data at rest | EBS encrypted (AES-256) |
| Instance metadata | IMDSv2 required (prevents SSRF) |
| IAM | Minimal: SSM only, no S3/EC2 write permissions |
| Transport | HTTP only in v0.1 — TLS in v0.2 |

---

## Key architectural decisions

| # | Decision | Rationale |
|---|---|---|
| ADR-001 | Go over Python | Faster cold start, single binary distribution, better AWS SDK support |
| ADR-002 | CloudFormation via aws-sdk-go-v2 | Direct API control, no external binary dependency, auditable templates |
| ADR-003 | S3 JSON state (no DynamoDB) | Simpler state management for MVP, versioning for auditability, no locking overhead |
| ADR-004 | SSM over SSH | No open ports, auto-logged, cleaner security group |
| ADR-005 | HTTP in MVP | Simpler, TLS deferred to v0.2 |
| ADR-006 | Ollama `--api-key` for POC | Lightweight CPU-based serving, rapid iteration over GPU optimization |
