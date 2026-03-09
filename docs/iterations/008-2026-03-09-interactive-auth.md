# Iteration 008 — Interactive AWS Authentication

**Date:** 2026-03-09
**Branch:** `feat/interactive-auth`

## What was done

### Problem being solved

Haven silently used the AWS SDK default credential chain. If credentials weren't configured, the user got a cryptic STS error. There was no confirmation of which account would be used, no onboarding for new users, and no way to switch profiles.

### Solution

Added an interactive authentication flow that probes credentials, confirms identity, supports profile switching, and onboards new users — all inside the AWS provider.

The CLI provides a `Prompter` interface for terminal I/O, keeping the provider testable and the auth flow portable to future providers (GCP, Azure).

### Files created

| File | Description |
|---|---|
| `internal/provider/aws/authenticate.go` | Full interactive auth flow: probe, confirm, switch profile, onboard |
| `internal/cli/prompt.go` | `terminalPrompter` implementing `provider.Prompter` via stdin/stdout |

### Files modified

| File | Change |
|---|---|
| `internal/provider/provider.go` | Added `Prompter` interface, `ARN` field to `Identity` |
| `internal/provider/aws/credentials.go` | Added `loadConfigWithProfile`, `loadConfigWithStaticCredentials`, `ARN` capture |
| `internal/provider/aws/provider.go` | Populated `ARN` in Identity |
| `internal/cli/root.go` | Added `buildProvider()` dispatch function |
| `internal/cli/deploy.go` | Uses `buildProvider`, removed duplicate identity print, routed quota prompts through shared scanner |
| `internal/cli/destroy.go` | Uses `buildProvider` for interactive auth |
| `internal/provider/mock/mock.go` | Added mock `Prompter` |
| `go.mod` | Added `golang.org/x/term`, promoted `aws-sdk-go-v2/credentials` |

## What works

- `go build ./cmd/haven/` — compiles
- `go test -race ./...` — 8/8 packages pass (including new authenticate_test.go)
- `go vet ./...` — clean
- `golangci-lint run` — no new issues from this iteration

Auth flows:
- Existing credentials: shows identity (account, region, ARN), asks Y/n
- Profile switching: lists ~/.aws/config + ~/.aws/credentials profiles
- New user onboarding: guides to IAM console, collects keys, validates via STS, saves to [haven] profile
- Fallback: tries [haven] profile when default credentials fail
- Read-only commands (status, cert) bypass interactive auth
- Retry loop: up to 3 attempts for invalid credentials during onboarding/manual entry

Unit tests (16 cases in authenticate_test.go):
- `TestUpsertINISection` — 5 cases: new file, append, replace, preserve others, profile-style sections
- `TestParseINISections` — 4 cases: missing file, single, multiple, non-section lines
- `TestCollectCredentials` — 5 cases: valid input, default region, empty key/secret errors
- `TestConfirmIdentity` — 2 cases: confirm/decline, verifies identity values are printed

## What's not covered

- Unit tests for `Authenticate` orchestration (requires AWS SDK dependency injection)
- Unit tests for retry loop in `collectAndProbe` (same reason — calls real `loadConfigWithStaticCredentials`/`getIdentity`)
- Confirmation before saving credentials to disk (decided against — unnecessary friction)

## What's left

- Consider extracting AWS SDK calls behind interfaces for deeper unit testing of `Authenticate` and `collectAndProbe`
