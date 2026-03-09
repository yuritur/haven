# Iteration 010 — Login Command

**Date:** 2026-03-09
**Branch:** `feat/login-command`

## What was done

### Problem being solved

`Authenticate()` ran on every CLI command — detecting credentials, calling STS, and prompting "Continue with this account?" each time. This was repetitive and annoying for frequent use.

### Solution

Introduced `haven login` as a dedicated authentication command. Other commands now silently resume the saved session or fail with a clear error.

- `haven login` runs the interactive auth flow and saves session to `~/.haven/session.json`
- All other commands (`deploy`, `destroy`, `status`, `chat`, `cert`) call `ResumeSession()` which loads the saved session silently
- Session stores: provider, AWS profile name, account ID, region
- `ResumeSession` validates that the resolved credentials still match the saved account ID (prevents silent credential drift)

### Files created

| File | Description |
|---|---|
| `internal/provider/aws/session.go` | Session struct, SaveSession, LoadSession, ResumeSession |
| `internal/provider/aws/session_test.go` | Table-driven tests for session save/load round-trip |
| `internal/cli/login.go` | Cobra command for `haven login` |

### Files modified

| File | Change |
|---|---|
| `internal/provider/aws/authenticate.go` | Added `profile` to `authResult`, added `Login()` and `loginAndSave()`, removed dead `Authenticate()` |
| `internal/cli/root.go` | Registered login command, changed `buildProvider` to use `ResumeSession` (non-interactive) |
| `internal/cli/deploy.go` | Dropped prompter from `buildProvider` call |
| `internal/cli/destroy.go` | Dropped prompter from `buildProvider` call, removed unused prompter |
| `internal/cli/status.go` | Dropped prompter from `buildProvider` call, removed unused prompter |
| `internal/cli/chat.go` | Dropped prompter from `buildProvider` call |
| `internal/cli/cert.go` | Dropped prompter from `buildProvider` call, removed unused prompter |

## What works

- `go build ./cmd/haven/` — builds cleanly
- `go vet ./...` — no issues
- `go test -race ./...` — all tests pass with race detector
- `golangci-lint run` — passes (via pre-commit hooks)
- Session save/load round-trip tested (5 subtests)

## What's not covered

- `ResumeSession` integration test (requires real AWS credentials)
- Session TTL/expiry (sessions are valid until credentials expire or account changes)
- `haven logout` command (by design — re-run `haven login` to switch)

## What's left

- Manual testing: `haven login` → `haven status` → verify no prompts
- Manual testing: delete `~/.haven/session.json` → `haven deploy` → verify "not logged in" error
