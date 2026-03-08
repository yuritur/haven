# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build ./cmd/haven/

# Run
go run ./cmd/haven/ deploy llama3.2:1b

# Test
go test ./...

# Test single package
go test ./internal/provider/aws/cfn/...

# Lint
golangci-lint run

# Format
gofmt -w .

# Vet
go vet ./...

# Tidy deps
go mod tidy
```

## Architecture

Haven is a CLI tool that deploys open-source LLM models to the user's AWS account with one command. It provisions EC2 via CloudFormation and returns an OpenAI-compatible HTTPS endpoint.

**Deploy flow:** `haven deploy llama3.2:1b`
1. Validate AWS credentials (STS)
2. Bootstrap S3 state bucket `haven-state-{account_id}` (idempotent, once per account)
3. Detect user's public IP (for Security Group restriction)
4. Generate self-signed TLS cert (ECDSA P-256) + API key + deployment ID
5. Generate CloudFormation template and create stack (VPC, Subnet, IGW, SG, EC2, EIP, IAM)
6. Poll stack events until complete
7. Poll `GET /api/tags` on the Ollama endpoint via pinned TLS until the model is ready (~3–15 min)
8. Save deployment state JSON to S3, print endpoint + API key + TLS fingerprint

**Key design constraints:**
- No external binaries — CloudFormation managed entirely via `aws-sdk-go-v2` direct API calls
- State in S3 JSON files, no DynamoDB or local disk
- Single binary distribution (no runtime deps for users)
- Multi-provider abstraction — CLI code is provider-agnostic via `Provider` and `StateStore` interfaces

**Module layout:**
- `cmd/haven/main.go` — entry point, calls `cli.Execute()`
- `internal/cli/` — cobra commands: `deploy`, `destroy`, `status`, `cert`
- `internal/provider/provider.go` — `Provider` and `StateStore` interfaces, `Deployment` struct
- `internal/provider/aws/` — AWS provider: credentials, S3 bootstrap, S3 state store
- `internal/provider/aws/cfn/` — CloudFormation template generation, stack create/poll, stack delete/poll
- `internal/provider/mock/` — mock Provider and StateStore for unit tests
- `internal/models/` — model name → `{Runtime, Tag, InstanceType, MinRAMGB}` registry
- `internal/bootstrap/` — EC2 user data script generation (embeds `ollama.sh` template)
- `internal/certutil/` — self-signed TLS cert generation (ECDSA P-256), fingerprint-pinned `http.Transport`
- `internal/tui/` — terminal spinner for provisioning feedback

**AWS resources per deployment:** VPC, Subnet, IGW, RouteTable, SecurityGroup (port 11434 restricted to user IP), EC2 t3.large (Amazon Linux 2023, 30GB gp3 EBS, IMDSv2), EIP, IAM role (SSM only). AMI resolved via SSM parameter. Nginx as TLS reverse proxy (port 11434 SSL → 127.0.0.1:11435 Ollama).

**State schema:** stored at `s3://haven-state-{account_id}/deployments/{deployment_id}.json` with fields: `id`, `provider`, `provider_ref`, `model`, `instance_type`, `instance_id`, `public_ip`, `endpoint`, `api_key`, `tls_cert`, `tls_fingerprint`, `region`, `created_at`.

## Testing

Tests use Go standard `*_test.go` naming, same-package access:
- Mock provider/store via `internal/provider/mock/` with pluggable function fields
- Table-driven tests, helper functions (`testInput()`, `parseTemplate()`)
- Run with `go test -race ./...` (CI uses `-race`)

## Code conventions

- No docstrings by default — only add comments when behavior is non-obvious or surprising. Focus comments on "why", not "what".
- Code and comments in English only.
- Minimize code changes — less is more, as long as functionality isn't impacted.
- Prefer clear naming and small functions over documentation.
