# ADR-005: HTTP endpoint in MVP, TLS deferred to v0.2

**Status:** Accepted
**Date:** 2026-03

## Context

The API endpoint needs to be accessible to clients. Options for transport security:
1. Plain HTTP (port 8000)
2. Self-signed HTTPS (vLLM supports --ssl-certfile)
3. ALB + ACM certificate (proper TLS, requires domain or uses AWS default)
4. Let's Encrypt (requires domain)

## Decision

HTTP for MVP. TLS in v0.2.

## Reasoning

- Target users are engineers connecting from their own machines or CI
- API key provides authentication; HTTP is acceptable when client is on a trusted network or VPN
- Adding ALB adds ~$20/month and significant Terraform complexity
- Self-signed cert requires `verify=False` on client, which is arguably worse UX than documented HTTP
- Scope: getting to working deploy fast is higher priority than transport encryption in v0.1

## Security documentation

This tradeoff MUST be documented clearly in the README:
> The API endpoint uses HTTP. Your API key and inference data are transmitted unencrypted.
> If you're sending sensitive data, restrict `allowed_cidr_blocks` to your IP only
> and consider upgrading to v0.2 which adds TLS.

## Consequences

- Simple Terraform — no ALB, no certificate resources
- vLLM runs directly on port 8000
- Security Group should default to user's own IP, not 0.0.0.0/0

## v0.2 plan

Options (to be decided):
- ALB + ACM (requires Route53 or user-provided domain)
- Self-signed cert auto-generated at boot, distributed via Haven state
- Caddy reverse proxy (auto TLS with Let's Encrypt, requires domain)
