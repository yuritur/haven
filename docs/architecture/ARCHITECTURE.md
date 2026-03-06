# Haven — Architecture

> Last updated: 2026-03
> Version: v0.1 MVP

---

## Deploy flow

```
haven deploy llama3.1-8b
  │
  ├─ 1. Validate AWS credentials                (boto3)
  ├─ 2. Bootstrap tfstate backend if needed     (boto3 → S3 + DynamoDB, once per account)
  ├─ 3. Ensure terraform binary in PATH         (~/.haven/bin/terraform, auto-downloaded)
  ├─ 4. Generate deployment config              (Jinja2 → terraform.tfvars)
  ├─ 5. terraform init + apply                  (python-terraform)
  │     └─ creates: VPC, SG, EC2, EIP, IAM
  ├─ 6. EC2 runs user_data bootstrap            (~10–20 min: Docker + nvidia-toolkit + vLLM)
  ├─ 7. Haven polls /health via SSM port-forward
  └─ 8. Print endpoint URL + API key
```

---

## Module layout

```
terraform/
├── modules/
│   ├── networking/          # VPC, subnet, internet gateway, security group
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   └── outputs.tf
│   └── gpu-instance/        # EC2, EBS (encrypted), EIP, IAM role (SSM), user_data
│       ├── main.tf
│       ├── variables.tf
│       ├── outputs.tf
│       └── templates/
│           └── user_data.sh.tpl   # Docker + nvidia-container-toolkit + vLLM/Ollama
└── deployments/             # Generated per-deploy, gitignored
    └── {deployment_id}/
        ├── main.tf
        ├── backend.tf
        └── terraform.tfvars

haven/                       # Python package
├── cli.py                   # Click: deploy, destroy, status
├── aws/
│   ├── credentials.py       # Validate creds, detect region, check GPU quota
│   ├── bootstrap.py         # Create S3 + DynamoDB for tfstate
│   └── terraform_dl.py      # Download terraform binary if absent
├── terraform/
│   ├── runner.py            # Wraps python-terraform: init, apply, destroy, output
│   └── templates/           # Jinja2 templates for main.tf / backend.tf / tfvars
├── models/
│   └── registry.py          # Model → recommended instance + backend mapping
└── state/
    └── manager.py           # ~/.haven/state.json: active deployments, endpoints, keys
```

---

## AWS resources per deployment

| Resource | Config |
|---|---|
| VPC | /16 CIDR, DNS hostnames enabled |
| Public subnet | /24, auto-assign public IP |
| Internet Gateway | 1 per VPC |
| Security Group | Inbound: 8000/tcp from user IP. Outbound: all |
| EC2 instance | GPU instance (g4dn/g5 family), Deep Learning Base AMI |
| EBS root volume | gp3, 200 GB, encrypted at rest |
| Elastic IP | Stable public IP, survives instance stop/start |
| IAM role | AmazonSSMManagedInstanceCore only |

**Shared per account (created once):**
- S3 bucket: `haven-tfstate-{account_id}` — versioning on, public access blocked
- DynamoDB table: `haven-tfstate-lock` — state locking

---

## Serving backends

| Tier | Instance | GPU VRAM | Backend | Models |
|---|---|---|---|---|
| Budget | g4dn.xlarge | T4 16GB | Ollama (GGUF Q4) | Mistral 7B, Qwen 2.5 7B |
| Standard | g5.xlarge | A10G 24GB | vLLM (FP16) | Llama 3.1 8B, DeepSeek R1 8B |
| Performance | g5.12xlarge | 4× A10G 96GB | vLLM (tensor parallel) | Llama 3.3 70B, Qwen 2.5 72B |

---

## Security posture (v0.1)

| Concern | Mitigation |
|---|---|
| API authentication | vLLM `--api-key` (Bearer token) |
| Network access | Security Group restricts port 8000 to user's IP |
| Instance access | SSM only — no SSH, no port 22 |
| Data at rest | EBS encrypted (AES-256) |
| Instance metadata | IMDSv2 required (prevents SSRF) |
| IAM | Minimal: SSM only, no S3/EC2 write permissions |
| Transport | HTTP only in v0.1 ⚠️ — TLS in v0.2 |

---

## Key architectural decisions

| # | Decision | Rationale |
|---|---|---|
| ADR-001 | Python over Go | Author expertise, distribution solved via pipx/brew |
| ADR-002 | Terraform for infra | Auditable, declarative, state management built-in |
| ADR-003 | S3 state via boto3 bootstrap | No manual setup, multi-machine support |
| ADR-004 | SSM over SSH | No open ports, auto-logged, cleaner security group |
| ADR-005 | HTTP in MVP | Simpler, TLS deferred to v0.2 |
| ADR-006 | vLLM `--api-key` | Native auth, no nginx needed |
