# Iteration 006 — Cross-platform release binaries CI

**Date:** 2026-03-09
**Branch:** `feat/release-binaries`

## What was done

### Problem being solved

No automated way to build and distribute Haven binaries. Users would need to clone the repo and build from source, which requires Go toolchain.

### Solution

GoReleaser-based GitHub Actions workflow that triggers on tag push (`v*`), builds static binaries for 6 platforms (linux/darwin/windows x amd64/arm64), generates SHA256 checksums, and attaches everything to a GitHub Release.

### Files created

| File | Description |
|---|---|
| `.goreleaser.yaml` | GoReleaser v2 config: 6 targets, CGO_ENABLED=0, ldflags version injection, tar.gz/zip archives, checksums |
| `.github/workflows/release.yml` | Release workflow: checkout, setup-go, test with -race, goreleaser release |
| `internal/cli/version.go` | `var version = "dev"` — linker-injected at build time |

### Files modified

| File | Change |
|---|---|
| `internal/cli/root.go` | Added `Version: version` to cobra command for `--version` flag |
| `.gitignore` | Added `dist/` (GoReleaser local build output) |

## What works

`go build ./...` passes. `go test -race ./...` — all tests green. `go vet ./...` — no issues.

Key behaviors:
- `haven --version` prints `haven version dev` (local) or `haven version X.Y.Z` (release)
- Tag push `v*` triggers release workflow
- Release includes 6 platform archives + SHA256 checksums file
- Tests run before release build (prevents broken binaries)

## What's not covered

- No Homebrew tap or Scoop manifest (future)
- No snapshot/nightly builds
- Action versions pinned to major tags, not SHA (industry norm)

## What's left

- First release tag (`v0.1.0`)
- Homebrew formula for macOS users
- Install script (`curl | sh` pattern)
