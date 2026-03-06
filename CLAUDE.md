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
go test ./internal/cfn/...

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

Haven is a CLI tool that deploys open-source LLM models to the user's AWS account with one command. It provisions EC2 via CloudFormation and returns an OpenAI-compatible endpoint.

**Deploy flow:** `haven deploy llama3.2:1b`
1. Validate AWS credentials (STS)
2. Bootstrap S3 state bucket `haven-state-{account_id}` (idempotent, once per account)
3. Detect user's public IP (for Security Group restriction)
4. Generate CloudFormation template and create stack (VPC, Subnet, IGW, SG, EC2, EIP, IAM)
5. Poll stack events until complete
6. Poll `GET /api/tags` on the Ollama endpoint until the model is ready (~3–15 min)
7. Save deployment state JSON to S3, print endpoint + API key

**Key design constraints:**
- No external binaries — CloudFormation managed entirely via `aws-sdk-go-v2` direct API calls
- State in S3 JSON files, no DynamoDB or local disk
- Single binary distribution (no runtime deps for users)

**Module layout:**
- `cmd/haven/main.go` — entry point, calls `cli.Execute()`
- `internal/aws/` — AWS config loading (`credentials.go`), S3 bucket bootstrap (`bootstrap.go`)
- `internal/cfn/` — CloudFormation template generation (`template.go`), stack create/poll (`deploy.go`), stack delete/poll (`destroy.go`)
- `internal/models/` — model name → `{OllamaTag, InstanceType}` registry
- `internal/state/` — `Deployment` struct, Save/Load/List/Delete to S3
- `internal/cli/` — cobra commands: `deploy`, `destroy`, `status`

**AWS resources per deployment:** VPC, Subnet, IGW, RouteTable, SecurityGroup (port 11434 restricted to user IP), EC2 t3.large (Amazon Linux 2023, 30GB gp3 EBS), EIP, IAM role (SSM only). AMI resolved via SSM parameter.

**State schema:** stored at `s3://haven-state-{account_id}/deployments/{deployment_id}.json` with fields: `deployment_id`, `stack_name`, `model`, `instance_type`, `instance_id`, `eip`, `endpoint`, `api_key`, `region`, `created_at`.

## Code conventions

- No docstrings by default — only add comments when behavior is non-obvious or surprising. Focus comments on "why", not "what".
- Code and comments in English only.
- Minimize code changes — less is more, as long as functionality isn't impacted.
- Prefer clear naming and small functions over documentation.
