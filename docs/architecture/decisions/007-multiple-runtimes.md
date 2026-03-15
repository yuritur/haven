# ADR-007: Multiple runtimes (Ollama, llama.cpp)

**Status:** Accepted
**Date:** 2026-03

## Context

Models can be served by different backends. We need a single deploy path that supports more than one runtime, with a clear default and an override so users can pick Ollama or llama.cpp when a model supports both.

## Decision

- **Runtimes:** `ollama` and `llamacpp`. Each is a first-class backend with its own bootstrap script, health endpoint, and chat API shape.
- **Model registry:** Each model has a `Config` with optional `Ollama` and/or `LlamaCpp` config (tag for Ollama, HF repo/file for GGUF). `SupportsRuntime(rt)` decides if a model can use a given runtime.
- **Default:** If a model supports both, default to **llamacpp**. Rationale: OpenAI-compatible `/v1/chat/completions`, same port and TLS story, GGUF from Hugging Face.
- **Override:** CLI flag `--runtime ollama` (or `llamacpp`) to force a runtime. No flag = use default above.
- **Abstraction:** `internal/runtime` exposes a `Runtime` interface (Port, ChatPath, MarshalChatRequest, ParseChatToken, WaitForReady). `Resolve(modelName, override)` returns the concrete runtime and its kind; bootstrap and CLI use the kind to select scripts and readiness checks.

## Reasoning

- One code path for deploy/chat/status; runtime is a dimension of deployment state.
- Ollama: simple `ollama run <tag>`, good for quick CPU runs.
- Llama.cpp: OpenAI-like API, GGUF from HF, same 11434 port behind nginx; default keeps API shape consistent for SDK users.
- Instance type and sizing are runtime-agnostic for now; future ADRs can add per-runtime instance rules if needed.

## Implementation

- `internal/models/registry.go`: `RuntimeName` (ollama, llamacpp), `Config` with `Ollama` / `LlamaCpp`, `SupportsRuntime`.
- `internal/runtime/`: `Resolve(model, override)`, `OllamaRuntime`, `LlamaCppRuntime`; health paths `/api/tags` vs `/v1/models`, chat paths `/api/chat` vs `/v1/chat/completions`.
- `internal/bootstrap/`: switch on `input.Runtime`, embed `ollama.sh` or `llamacpp.sh` in user_data.
- State: persist `runtime` in deployment JSON so `haven chat` and cost/status use the correct paths and labels.

## Consequences

- New runtimes (e.g. vLLM) require a new `RuntimeName`, a `Runtime` impl, and bootstrap script; registry and CLI stay unchanged.
- Default “llamacpp when available” is a product choice; can be revisited if Ollama becomes the preferred default.
