# Iteration 004 — GPU model deployment (QWEN 3.5)

**Date:** 2026-03-09
**Branch:** `feat/gpu-deployment`

## What was done

### Problem being solved

Haven only supported CPU models on t3.large/t3.xlarge instances. Serious models like QWEN 3.5 require GPU instances with NVIDIA drivers and significantly more storage.

### Solution

Three changes: extend model registry with GPU models, use AWS Deep Learning AMI for GPU instances (pre-installed NVIDIA drivers), and parameterize EBS volume size.

### Key design decisions

1. **No `GPU bool` field** — GPU is derived from instance type via `IsGPUInstance()` helper. Checks prefix (`g4dn.`, `g5.`, `g6.`, `p3.`, `p4.`, `p5.`). Single source of truth.

2. **Deep Learning AMI instead of DKMS bootstrap** — Initially implemented runtime NVIDIA driver installation via DKMS in bootstrap script. Replaced with AWS Deep Learning Base GPU AMI (`/aws/service/deeplearning/ami/x86_64/base-oss-nvidia-driver-gpu-amazon-linux-2023/latest/ami-id`) resolved via SSM parameter at stack creation. Faster (~0 vs 5-10 min), more reliable (no kernel/driver mismatch), less code.

3. **Dynamic AMI selection** — CloudFormation `LatestAmiId` parameter Default is now chosen based on `IsGPUInstance(instanceType)`: AL2023 for CPU, Deep Learning AMI for GPU.

4. **EBSVolumeGB field** — Added to `Config` struct. Deep Learning AMI snapshot requires >= 75GB. CPU models use 30GB, GPU models 80-100GB.

5. **GPU-aware timeout** — `waitForOllama` accepts `time.Duration` parameter. CPU: 15 min, GPU: 30 min (larger models take longer to pull).

### Models added

| Model | Instance | EBS | VRAM (Q4) |
|---|---|---|---|
| `qwen3.5:4b` | g5.xlarge (A10G 24GB) | 80 GB | ~5 GB |
| `qwen3.5:9b` | g5.xlarge (A10G 24GB) | 100 GB | ~8 GB |
| `qwen3.5:27b` | g5.2xlarge (A10G 24GB) | 100 GB | ~17 GB |

### Files modified

| File | Change |
|---|---|
| `internal/models/registry.go` | Added `EBSVolumeGB` to Config, `IsGPUInstance()` helper, 3 QWEN models |
| `internal/provider/provider.go` | Added `EBSVolumeGB` to `DeployInput` |
| `internal/provider/aws/cfn/template.go` | Dynamic AMI SSM path, parameterized `VolumeSize` |
| `internal/provider/aws/cfn/deploy.go` | Added `EBSVolumeGB` to `cfn.DeployInput` |
| `internal/provider/aws/provider.go` | Threaded `EBSVolumeGB` through |
| `internal/cli/deploy.go` | `EBSVolumeGB` passthrough, `waitForOllama` timeout param, GPU-aware 30min timeout |
| `internal/bootstrap/bootstrap.go` | No changes (gpu param added then removed during AMI refactor) |
| `internal/bootstrap/ollama.sh` | No changes (GPU block added then removed during AMI refactor) |

### Tests added/updated

| Test file | What's tested |
|---|---|
| `internal/models/registry_test.go` | 3 QWEN model lookups (EBSVolumeGB, InstanceType), `TestIsGPUInstance` (7 cases) |
| `internal/provider/aws/cfn/template_test.go` | `TestGenerateTemplate_EBSVolumeSize` (80GB), `TestGenerateTemplate_GPUAmi` (deeplearning SSM path), `TestGenerateTemplate_CPUAmi` (al2023 SSM path) |

## What works

`go build ./...` passes. `go test -race ./...` passes — all tests green.

GPU deployment tested against real AWS:
- Deep Learning AMI resolves correctly via SSM
- EBS volume sizing works (fixed >= 75GB minimum for DLAMI snapshot)
- CloudFormation stack creation reaches EC2 instance provisioning
- Blocked by vCPU quota (default 0 for G instances) — expected, documented

## Known limitations

- **GPU vCPU quota**: AWS accounts default to 0 vCPU for G/P instances. Users must request increase via Service Quotas before first GPU deploy. CloudFormation fails with clear error message.
- **g5 availability**: Not available in all regions.
- **No auto-stop**: GPU instances cost ~$1/hr on-demand. No idle shutdown mechanism yet.
- **No quota pre-check**: Could detect 0 quota before attempting deploy to avoid failed stacks.

## What's next

- Pre-flight GPU quota check + automatic quota increase request
- Auto-stop for idle instances (cost control)
- More GPU models (Mistral, Llama 3.3)
