# Multi-Provider Design

**Date:** 2026-03-07

## Goal

Remove hard AWS coupling so future cloud providers (GCP, Azure, DigitalOcean, etc.) can be added
without touching the CLI or core logic. AWS remains the only implementation for now.

## Approach

Restructure into `internal/provider/` with explicit interfaces. AWS implementation lives at
`internal/provider/aws/`. Provider is selected via `--provider` flag (default: `aws`).

## Directory Structure

```
internal/
  provider/
    provider.go              — Provider + StateStore interfaces, shared types
    aws/
      provider.go            — AWSProvider implements Provider
      state.go               — S3StateStore implements StateStore
      cfn/
        template.go          — moved from internal/cfn/
        deploy.go            — moved from internal/cfn/
        destroy.go           — moved from internal/cfn/
      credentials.go         — moved from internal/aws/
      bootstrap.go           — moved from internal/aws/
  models/
    registry.go              — unchanged
  cli/
    root.go                  — --provider flag, constructor-based wiring (no init())
    deploy.go                — uses Provider + StateStore interfaces
    destroy.go               — uses Provider + StateStore interfaces
    status.go                — uses StateStore interface
```

Deleted: `internal/aws/`, `internal/cfn/`, `internal/state/`

## Interfaces

```go
// internal/provider/provider.go

type Identity struct {
    AccountID string
    Region    string
}

type DeployInput struct {
    DeploymentID string
    Model        string
    InstanceType string
    UserIP       string
    APIKey       string
}

type DeployResult struct {
    ProviderRef string
    InstanceID  string
    PublicIP    string
}

type Provider interface {
    Identity(ctx context.Context) (Identity, error)
    Deploy(ctx context.Context, input DeployInput) (DeployResult, error)
    Destroy(ctx context.Context, providerRef string) error
}

type StateStore interface {
    Save(ctx context.Context, d Deployment) error
    Load(ctx context.Context, id string) (*Deployment, error)
    List(ctx context.Context) ([]Deployment, error)
    Delete(ctx context.Context, id string) error
}
```

## Deployment Struct Changes

| Old field    | New field     | Reason                                      |
|---|---|---|
| `StackName`  | `ProviderRef` | CloudFormation-specific naming              |
| `EIP`        | `PublicIP`    | Elastic IP is AWS-specific terminology      |
| _(new)_      | `Provider`    | Track which provider created the deployment |

## CLI Changes

- `root.go`: remove `init()`, use `NewRootCmd()` constructor; add `--provider` flag
- All help text: remove "AWS" references
- Provider factory in each command:
  ```go
  func buildProviderAndStore(ctx context.Context, name string) (provider.Provider, provider.StateStore, error) {
      switch name {
      case "aws":
          return awsprovider.New(ctx)
      default:
          return nil, nil, fmt.Errorf("unknown provider %q — available: aws", name)
      }
  }
  ```

## Help Text Changes

| Location | Old | New |
|---|---|---|
| `root.go` Long | "Haven deploys LLM models to your AWS account..." | "Haven deploys LLM models to your cloud..." |
| `deploy.go` Short | "Deploy a model to AWS" | "Deploy a model to your cloud" |
| `destroy.go` Short | "Destroy a deployment and release all AWS resources" | "Destroy a deployment and release all cloud resources" |
| `destroy.go` output | "All AWS resources released." | "All resources released." |
| `deploy.go` Example | `haven deploy llama3.2:1b` | `haven deploy llama3.2:1b` + `haven deploy llama3.2:1b --provider aws` |

## What Does NOT Change

- CloudFormation template logic (`cfn/template.go`) — just moves location
- Polling logic (`cfn/deploy.go`, `cfn/destroy.go`) — just moves location
- S3 state storage implementation — just moves location
- Model registry — unchanged
- Ollama health check — unchanged
- `models.Config.InstanceType` — kept as-is; AWS terminology for now, providers use it directly
