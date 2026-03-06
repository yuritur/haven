# Iteration 001 — Go + CloudFormation POC

**Date:** 2026-03-07

## What was done

### Architecture decisions

**Rejected Terraform:**
- Requires external binary — especially problematic on Windows (antivirus, UAC, PATH)
- Lots of wrapper code needed: subprocess management, parsing stdout/stderr, generating .tf files
- For Haven's fixed topology (VPC + EC2 + EIP), the complexity is not justified

**Rejected Pulumi:**
- Despite being written in Go, Pulumi Automation API shells out to the `pulumi` CLI
- Pulumi's architecture is gRPC-based (engine + provider plugins as separate processes) to support multi-language — cannot be embedded into a single binary
- Installing Pulumi means: pulumi CLI (~70MB) + pulumi-resource-aws provider (~400-600MB), downloaded at runtime from GitHub Releases — incompatible with "one-click install" requirement
- No mature alternative to Pulumi exists as a pure Go library with state management

**Chosen CloudFormation via aws-sdk-go-v2:**
- Pure API calls, no external binary
- AWS manages state, rollback, dependency ordering
- Single `CreateStack` / `DeleteStack` call replaces all Terraform boilerplate
- AMI resolved via SSM public parameter (always latest Amazon Linux 2023)

**Chosen Go over Python:**
- Single static binary, zero runtime dependencies for end user
- `go.sum` checksums for all dependencies — supply chain security
- Python's PyPI has a larger attack surface for supply chain attacks
- Target audience (HIPAA/SOC2 companies) benefits from auditable single binary

**POC serving: Ollama on t3.large instead of vLLM on GPU:**
- Validates end-to-end flow (deploy → endpoint → API call) without GPU cost
- Ollama: single binary install, OpenAI-compatible API, supports OLLAMA_API_KEY
- Model: llama3.2:1b (~700MB GGUF Q4), fits in 8GB RAM, pulls in ~1 min
- Bootstrap time: ~3-5 min vs 10-20 min for GPU + nvidia-toolkit + Docker + vLLM

**State management (no Terraform state):**
- No DynamoDB — was only needed for Terraform state locking
- Simple JSON file per deployment stored in S3
- Bucket `haven-state-{account_id}` created automatically on first deploy

### Code written

Initial Go project from scratch. `go mod init github.com/havenapp/haven`.

**Files created:**

| File | Description |
|---|---|
| `cmd/haven/main.go` | Entry point |
| `internal/cli/root.go` | cobra root command |
| `internal/cli/deploy.go` | `haven deploy <model>`: auto-detect IP, generate API key, call cfn.Deploy, health poll, save state |
| `internal/cli/destroy.go` | `haven destroy <id>`: load state, call cfn.Destroy, delete state |
| `internal/cli/status.go` | `haven status`: list deployments from S3 |
| `internal/aws/credentials.go` | LoadConfig (default credential chain), GetIdentity (STS) |
| `internal/aws/bootstrap.go` | EnsureStateBucket: idempotent S3 create with versioning + public access block |
| `internal/cfn/template.go` | GenerateTemplate: CloudFormation JSON with VPC/SG/EC2/EIP/IAM, Ollama user_data |
| `internal/cfn/deploy.go` | CreateStack + pollStackEvents (chronological progress output) |
| `internal/cfn/destroy.go` | DeleteStack + pollStackEvents |
| `internal/models/registry.go` | Lookup(name) → {OllamaTag, InstanceType} |
| `internal/state/manager.go` | Deployment struct, Save/Load/List/Delete to S3 |

**Dependencies (go.sum checksums on everything):**
- `github.com/spf13/cobra` — CLI framework
- `github.com/aws/aws-sdk-go-v2/{config,service/sts,service/s3,service/cloudformation}` — AWS SDK

**CloudFormation resources per deployment:**
VPC → Subnet → IGW → VPCGWAttachment → RouteTable → Route → SubnetRTAssoc → SecurityGroup → IAMRole → InstanceProfile → EC2Instance → EIP → EIPAssociation

Security: IMDSv2 required, EBS encrypted, port 11434 restricted to deployer's IP, SSM access only (no SSH).

### Docs updated

- `docs/architecture/ARCHITECTURE.md` — rewritten for Go + CloudFormation + Ollama
- `docs/architecture/decisions/001` — Go over Python
- `docs/architecture/decisions/002` — CloudFormation over Terraform
- `docs/architecture/decisions/003` — S3 JSON state (no DynamoDB)

## What works

`go build -o haven ./cmd/haven && ./haven --help` — binary builds, all commands present.

Not yet tested against real AWS. Next step: run `haven deploy llama3.2:1b` against a real account.

## What's next

- Test real AWS deploy end-to-end
- Verify CloudFormation stack creation and event polling output
- Verify Ollama health check and model pull detection
- Verify `haven destroy` cleans up all resources
- Add `--region` flag to deploy command
