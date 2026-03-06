# ADR-001: Go as implementation language

**Status:** Accepted (revised from Python, 2026-03)
**Date:** 2026-03

## Context

Haven needs a CLI language. Two main candidates: Python and Go.

Core project requirements that drive this decision:
- Security-first, minimal dependency surface
- Cross-platform distribution (including Windows)
- Target audience: companies with compliance requirements (HIPAA, GDPR, SOC2)

## Decision

Go.

## Reasoning

- **Single binary, zero runtime deps**: users download one executable, no `pip install`, no Python environment, no package manager attack surface
- **Minimal dependency surface**: Go stdlib covers HTTP, JSON, crypto, process management. aws-sdk-go-v2 (AWS-maintained) covers the rest. Total transitive deps: ~8 modules with checksums in `go.sum`
- **Reproducible builds**: `go.sum` contains cryptographic hashes of all dependencies — mandatory for compliance-conscious users
- **Supply chain security**: PyPI has a documented history of supply chain attacks. Go module proxy + checksumdb provides stronger integrity guarantees
- **Cross-platform by default**: `GOOS=windows go build` produces a native exe with no additional setup
- **Startup latency**: instant, no interpreter warmup — important for CLI UX

## Consequences

- Distribution: single binary per platform, published via GitHub Releases + Homebrew formula
- No Python environment required on user machine
- AWS access via `aws-sdk-go-v2` (maintained by AWS, same trust level as boto3)

## Alternatives considered

- **Python**: faster initial prototyping if author knows Python well, but boto3 brings ~15-20 PyPI transitive deps, requires Python runtime on user machine, weaker supply chain guarantees. Rejected due to security and distribution concerns.
- **Rust**: similar security/binary benefits, but steeper learning curve and smaller AWS SDK ecosystem.
