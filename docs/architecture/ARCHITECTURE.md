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
  ├─ 6. EC2 runs user_data bootstrap            (per-runtime: Ollama pull or llama.cpp + GGUF download)
  ├─ 7. Poll runtime health (Ollama: /api/tags, llama.cpp: /v1/models) via TLS until ready
  └─ 8. Save state JSON to S3, print endpoint URL + API key
```

---

## Module layout

```
cmd/haven/
  main.go                    # entry point

internal/
  cli/                       # cobra: deploy, destroy, status, cert, login, chat
  provider/
    provider.go             # Provider + StateStore interfaces, Deployment struct
    aws/                    # AWS: credentials, S3 bootstrap, S3 state, instance/cost
      cfn/                  # CloudFormation template, create/poll, delete/poll
  models/
    registry.go             # model name → Config (Ollama tag and/or LlamaCpp GGUF); SupportsRuntime(rt)
  runtime/
    runtime.go              # Runtime interface; Resolve(model, override) → Runtime + kind
    ollama.go, llamacpp.go  # health path, chat path, wire format, WaitForReady
  bootstrap/
    bootstrap.go            # user_data generation; dispatches by runtime (ollama.sh / llamacpp.sh)
  certutil/                 # TLS cert, fingerprint-pinned transport
  tui/                      # spinner for deploy feedback
```

---

## AWS resources per deployment

| Resource | Config |
|---|---|
| VPC | /16 CIDR, DNS hostnames enabled |
| Public subnet | /24, auto-assign public IP |
| Internet Gateway | 1 per VPC |
| Security Group | Inbound: 11434/tcp (runtime) from user IP. Outbound: all |
| EC2 instance | t3.large, Amazon Linux 2023 AMI |
| EBS root volume | gp3, 30 GB, encrypted at rest |
| Elastic IP | Stable public IP, survives instance stop/start |
| IAM role | AmazonSSMManagedInstanceCore only |

**Shared per account (created once):**
- S3 bucket: `haven-state-{account_id}` — versioning on, public access blocked

---

## Serving runtimes

Haven supports multiple runtimes per model. The registry (`internal/models`) defines for each model which runtimes are supported (Ollama and/or llama.cpp). Default is llama.cpp when available; user can override with `--runtime ollama`.

| Runtime | Health path | Chat path | Use case |
|---------|-------------|-----------|----------|
| **ollama** | `/api/tags` | `/api/chat` | Ease of use, Ollama ecosystem |
| **llamacpp** | `/v1/models` | `/v1/chat/completions` | OpenAI-compatible API, GGUF from Hugging Face |

Instance type and resources are the same for both runtimes today; bootstrap script and readiness check differ. See ADR-007.

## Serving backends (tiers)

| Tier | Instance | Runtimes | Models |
|---|---|---|---|
| CPU | t3.large | ollama, llamacpp | Llama 3.2 1B/3B, Phi 3 mini, Qwen 3.5 4B/9B/27B (quantized) |
| GPU | g5.xlarge | ollama, llamacpp | Qwen 3.5 (larger quants / GPU) |

---

## State schema

Deployment state is stored as JSON in S3 at `s3://haven-state-{account_id}/deployments/{deployment_id}.json`:

```json
{
  "id": "haven-abc123",
  "provider": "aws",
  "provider_ref": "stack name / resource id",
  "model": "llama3.2:1b",
  "runtime": "llamacpp",
  "instance_type": "t3.large",
  "instance_id": "i-0abc...",
  "public_ip": "1.2.3.4",
  "endpoint": "https://1.2.3.4:11434",
  "api_key": "sk-haven-...",
  "tls_cert": "...",
  "tls_fingerprint": "...",
  "region": "us-east-1",
  "created_at": "2026-03-01T12:00:00Z"
}
```

---

## Security posture (v0.1)

| Concern | Mitigation |
|---|---|
| API authentication | Runtime API key (Bearer token); Ollama via proxy, llama.cpp native |
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
| ADR-006 | vLLM / Ollama API key | Native or proxy auth for API endpoints |
| ADR-007 | Multiple runtimes (Ollama, llama.cpp) | Per-model backend choice, default llamacpp; health/chat paths differ by runtime |
