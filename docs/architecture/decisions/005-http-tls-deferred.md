# ADR-005: Self-signed TLS with certificate pinning

**Status:** Implemented
**Date:** 2026-03

## Context

The API endpoint needs to be accessible to clients. Options for transport security:
1. Plain HTTP (port 11434)
2. Self-signed HTTPS with certificate pinning
3. ALB + ACM certificate (proper TLS, requires domain or uses AWS default)
4. Let's Encrypt (requires domain)

## Decision

Self-signed ECDSA P-256 certificate generated at deploy time. Client trust established via SHA-256 fingerprint pinning (TOFU model) instead of CA chain verification.

## Reasoning

- No external dependencies (no domain, no CA, no ALB)
- Single binary — certificate generation in Go stdlib (`crypto/x509`)
- Fingerprint pinning provides MITM protection without CA infrastructure
- nginx as TLS terminator because Ollama doesn't support TLS natively
- Cost: $0 (no ALB, no ACM, no Route53)

## Implementation

- `internal/certutil/` — `GenerateSelfSigned()`, `NewPinnedTransport(fingerprint)`
- Certificate and key delivered to EC2 via CloudFormation user_data (base64-encoded in bootstrap script)
- nginx terminates TLS on port 11434, proxies to Ollama on 127.0.0.1:11435
- `haven cert <id>` exports PEM for use with curl / OpenAI SDK

## Tech debt

**Private key in CloudFormation user_data.** The TLS private key is embedded in the EC2 user_data script, which is visible in the AWS Console (EC2 > Instance > Actions > Instance Settings > View User Data). Acceptable for personal dev use, but for production hardening the key should be delivered via AWS Secrets Manager or SSM Parameter Store SecureString instead.

## Consequences

- All traffic encrypted with TLS 1.2+
- Clients must either pin the fingerprint or trust the self-signed cert (`--cacert` / `SSL_CERT_FILE`)
- No domain or DNS configuration required
