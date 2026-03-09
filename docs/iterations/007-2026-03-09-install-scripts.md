# Iteration 007 — Install Scripts & Homebrew Tap

**Date:** 2026-03-09
**Branch:** `feat/install-scripts`

## What was done

### Problem being solved

Release binaries downloaded from GitHub Releases are blocked by macOS Gatekeeper, requiring users to manually allow them in System Preferences. No package manager or scripted install path existed.

### Solution

Added three install channels covering all platforms:
1. **Homebrew tap** (macOS/Linux) — GoReleaser auto-publishes formula on release
2. **Shell install script** (macOS/Linux) — `curl | sh` with SHA256 verification
3. **PowerShell install script** (Windows) — `irm | iex` with SHA256 verification

### Files created

| File | Description |
|---|---|
| `install.sh` | POSIX shell install script for Linux/macOS |
| `install.ps1` | PowerShell install script for Windows |

### Files modified

| File | Change |
|---|---|
| `.goreleaser.yaml` | Added `brews` section for Homebrew tap |
| `.github/workflows/release.yml` | Added `HOMEBREW_TAP_TOKEN` env to goreleaser step |
| `README.md` | Replaced Install section with Homebrew, script, and source options |

## What works

- GoReleaser config validates (brews section follows v2 schema)
- install.sh: OS/arch detection, GitHub API version fetch, SHA256 verification, sudo fallback, macOS quarantine removal
- install.ps1: arch detection, SHA256 verification, PATH management, temp cleanup

## What's not covered

- No automated tests for install scripts (would require actual GitHub Release to exist)
- Homebrew formula not yet testable (requires `yuritur/homebrew-tap` repo to exist)

## What's left

Before first release:
1. Create `yuritur/homebrew-tap` repo on GitHub
2. Create Personal Access Token with `repo` scope
3. Add `HOMEBREW_TAP_TOKEN` secret to `yuritur/haven` repo settings
4. Tag and push `v0.x.x` to trigger first release
