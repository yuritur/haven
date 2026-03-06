# ADR-006: vLLM native --api-key for authentication

**Status:** Accepted
**Date:** 2026-03

## Context

The API endpoint needs authentication to prevent unauthorized access.
Options: nginx/Caddy reverse proxy with auth, vLLM native --api-key, custom auth middleware.

## Decision

vLLM native `--api-key` flag. No reverse proxy in MVP.

## Reasoning

- vLLM supports `--api-key` natively — enforces Bearer token on all endpoints
- OpenAI SDK and clients already know how to send `Authorization: Bearer {key}`
- Eliminates nginx/Caddy as a dependency — fewer moving parts, simpler user_data script
- One less process running on the instance

## Implementation

```bash
docker run vllm/vllm-openai:latest \
  --model meta-llama/Llama-3.1-8B-Instruct \
  --api-key {generated_32char_key} \
  ...
```

API key is generated at deploy time by Haven (cryptographically random, 32 chars), passed to user_data via Terraform template, stored in `~/.haven/state.json`.

## Consequences

- Ollama does not have native API key support → for Ollama backend, a lightweight proxy is needed (future work)
- For MVP: Ollama deployments rely on Security Group IP restriction as sole auth
- API key is stored in Terraform state (S3, encrypted) — acceptable security posture
