# Haven — Idea & Vision

## What Is This?

**Haven** is a CLI tool for one-command deployment of open-source LLM models to your own cloud infrastructure.

```bash
haven deploy llama3.1-8b --cloud aws
# → asks for AWS credentials
# → provisions GPU instance
# → downloads and serves the model
# → returns an OpenAI-compatible endpoint
```

## Problem Being Solved

There is a clear gap between:
- **"Run locally"** (Ollama, LM Studio) — simple but limited by local hardware
- **"Managed cloud inference"** (HuggingFace Inference Endpoints, Replicate, Modal) — simple but data leaves your control

Users who care about **data privacy** (GDPR, HIPAA, financial, legal sectors) cannot use managed services. Setting up self-hosted cloud inference today requires deep expertise: Kubernetes, GPU node configuration, model serving frameworks, VPC/networking, TLS, auth. There is no simple tool for this.

## Target Users

- Companies with compliance requirements (GDPR, HIPAA, SOC2) that need LLM inference without sending data to third parties
- Engineering teams that want self-hosted inference without dedicated DevOps knowledge
- Security-conscious developers and researchers

## Core Principles

1. **Simplicity first** — one command to deploy, one command to destroy. No YAML manifests, no Helm charts.
2. **Security by default** — private VPC, minimal IAM permissions, encrypted storage, API key auth out of the box.
3. **Open source and auditable** — fully open code, security model documented in plain English/Markdown so it can be reviewed by humans and LLMs alike.
4. **Your infrastructure, your data** — model runs in your cloud account. We never touch your data.

## Competitive Landscape

| Tool | Type | Problem |
|---|---|---|
| Ollama | Local only | No cloud, limited by local GPU |
| HuggingFace Inference Endpoints | Managed cloud | Data on HF infrastructure |
| Replicate / Modal | Managed cloud | Data on their infrastructure |
| SkyPilot | Multi-cloud orchestrator | General-purpose, complex for LLM use case, no security focus |
| BentoML / Ray Serve | Serving frameworks | Require infrastructure on top |

**Our angle:** open source + your cloud account + truly simple CLI + security-first documentation.

## Roadmap

### v0.1 — MVP (AWS only)
- Deploy top 5-10 popular models to EC2 GPU instances
- Docker + vLLM serving backend
- OpenAI-compatible REST API endpoint
- API key authentication
- `deploy` and `destroy` commands

### v0.2 — Security hardening
- Private VPC isolation
- Minimal IAM permissions (documented)
- Security model documentation in Markdown
- Audit trail for deploy/destroy operations

### v0.3 — Cost management
- Spot instance support
- Auto-stop on inactivity / time limit
- Cost estimate before deploy

### Future
- GCP, Azure support
- Bare metal deployment
- Auto-scaling
- Multiple models / model routing

## Technical Approach

**MVP stack (no Kubernetes — intentionally simple):**
```
CLI (Python or Go)
  → AWS API
    → EC2 GPU instance (g4dn / g5 family)
      → Docker + vLLM
        → HTTPS endpoint + API key auth
```

Kubernetes is deliberately avoided in v1. It adds operational complexity that contradicts the simplicity principle. A single GPU VM with Docker is sufficient and easier to reason about for security.

## Security Model

What the tool creates in your AWS account:
- A VPC with private/public subnets
- A Security Group allowing HTTPS (443) only from specified IPs
- An EC2 instance with an IAM role that has read-only S3 access (for model cache, optional)
- An EBS volume with encryption enabled
- No credentials are stored by the tool — AWS credentials are used only during provisioning

Full security documentation will be maintained in `docs/security/`.
