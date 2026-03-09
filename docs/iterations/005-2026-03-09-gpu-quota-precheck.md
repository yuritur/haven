# Iteration 005 — GPU vCPU quota pre-check

**Date:** 2026-03-09
**Branch:** `master`

## What was done

### Problem being solved

AWS accounts have 0 vCPU quota by default for G/P instance families. When deploying GPU models, CloudFormation fails with an opaque `CREATE_FAILED` error. Users don't know what happened or how to fix it.

### Solution

Pre-flight quota check before GPU deploy. If quota is insufficient, the user chooses between requesting the increase manually (Haven prints AWS Console URL + CLI command) or letting Haven submit the request via AWS Service Quotas API. Quota requests are persisted to S3 so they survive script restarts.

### Files created

| File | Description |
|---|---|
| `internal/provider/aws/quota/quota.go` | Quota check, increase request, status polling via Service Quotas API |
| `internal/provider/aws/quota/store.go` | S3 persistence for quota requests (`quota-requests/{code}.json`) |
| `internal/provider/aws/quota/quota_test.go` | Table-driven tests for quota code and vCPU mappings |
| `internal/cli/gpu_quota.go` | Interactive quota flow: check, prompt, request, poll with spinner |
| `internal/cli/gpu_quota_test.go` | Handler tests with mock QuotaChecker (6 test cases) |

### Files modified

| File | Change |
|---|---|
| `internal/provider/aws/provider.go` | Added `bucketName`, `quotaStore` fields; implemented 5 quota methods |
| `internal/cli/deploy.go` | Inserted quota pre-check before `prov.Deploy()` for GPU instances |
| `internal/provider/mock/mock.go` | Added `QuotaChecker` mock with pluggable function fields |
| `go.mod` | Added `github.com/aws/aws-sdk-go-v2/service/servicequotas` |

## What works

`go build ./...` passes. `go test -race ./...` — all tests green. `go vet ./...` — no issues.

Key behaviors:
- GPU deploy detects 0 vCPU quota and prompts before CloudFormation
- Option 1 prints AWS Console URL and CLI command, then exits
- Option 2 submits Service Quotas request, saves to S3, polls with spinner
- Re-running `haven deploy` finds pending request in S3 and resumes polling
- Graceful degradation: if Service Quotas API is inaccessible, skips check

## What's not covered

- No integration test against real AWS Service Quotas (mock-only)
- No test for `quota.Store` S3 persistence (would need S3 mock)
- Quota approval can take hours/days for large increases — user may need to wait

## What's left

- Auto-stop for idle GPU instances (cost control)
- More GPU models (Mistral, Llama 3.3)
- g5 regional availability checks
