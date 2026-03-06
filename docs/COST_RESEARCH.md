# Cost Research: LLM Deployment on AWS

> Last updated: 2026-03
> Prices are for us-east-1 (N. Virginia), Linux. Always verify at [instances.vantage.sh](https://instances.vantage.sh) before release.

---

## AWS GPU Instances


| Instance     | GPU     | VRAM  | On-Demand/hr | Spot/hr | Reserved 1yr/hr |
| ------------ | ------- | ----- | ------------ | ------- | --------------- |
| g4dn.xlarge  | 1× T4   | 16 GB | $0.526       | ~$0.158 | ~$0.315         |
| g4dn.2xlarge | 1× T4   | 16 GB | $0.752       | ~$0.226 | ~$0.451         |
| g5.xlarge    | 1× A10G | 24 GB | $1.006       | ~$0.352 | ~$0.604         |
| g5.2xlarge   | 1× A10G | 24 GB | $1.212       | ~$0.424 | ~$0.727         |
| g5.12xlarge  | 4× A10G | 96 GB | $5.672       | ~$1.985 | ~$3.403         |
| p3.2xlarge   | 1× V100 | 16 GB | $3.060       | ~$0.918 | —               |


> **p3 is not recommended** for inference: same 16 GB as g4dn.xlarge but 6× more expensive.

---

## Model VRAM Requirements

VRAM depends on precision. Rule of thumb:

- **FP16**: ~~2 GB per billion parameters + KV cache overhead (~~1–4 GB for typical contexts)
- **Q4**: ~0.5 GB per billion parameters + overhead


| Model                   | Params  | FP16 VRAM | Q4 VRAM |
| ----------------------- | ------- | --------- | ------- |
| Phi-3.5 Mini            | 3.8B    | ~8 GB     | ~3 GB   |
| Mistral 7B              | 7B      | ~14 GB    | ~5 GB   |
| Llama 3.1 8B            | 8B      | ~16 GB*   | ~5.5 GB |
| Gemma 2 9B              | 9B      | ~18 GB    | ~6 GB   |
| Qwen 2.5 7B             | 7B      | ~14 GB    | ~5 GB   |
| DeepSeek R1 Distill 7B  | 7B      | ~14 GB    | ~5 GB   |
| DeepSeek R1 Distill 8B  | 8B      | ~16 GB*   | ~5.5 GB |
| Qwen 2.5 14B            | 14B     | ~28 GB    | ~9 GB   |
| DeepSeek R1 Distill 14B | 14B     | ~28 GB    | ~9 GB   |
| DeepSeek R1 Distill 32B | 32B     | ~64 GB    | ~20 GB  |
| Llama 3.1 70B           | 70B     | ~140 GB   | ~40 GB  |
| Llama 3.3 70B           | 70B     | ~140 GB   | ~40 GB  |
| Qwen 2.5 72B            | 72B     | ~144 GB   | ~42 GB  |
| Mixtral 8x7B            | 47B MoE | ~90 GB    | ~28 GB  |


*8B models in FP16 are borderline on T4 (16 GB) — KV cache for long contexts will exceed VRAM. Q4 is strongly recommended on g4dn.

---

## Model-to-Instance Compatibility


| Model                    | g4dn.xlarge (T4 16GB) | g5.xlarge (A10G 24GB) | g5.12xlarge (4×A10G 96GB) |
| ------------------------ | --------------------- | --------------------- | ------------------------- |
| Phi-3.5 Mini             | FP16 + Q4             | FP16 + Q4             | FP16 + Q4                 |
| Mistral 7B / Qwen 2.5 7B | Q4 / INT8             | FP16 + Q4             | FP16 + Q4                 |
| Llama 3.1 8B / DS R1 8B  | Q4 only               | FP16 + Q4             | FP16 + Q4                 |
| Gemma 2 9B               | Q4 only               | INT8 + Q4             | FP16 + Q4                 |
| Qwen 2.5 14B / DS R1 14B | NO                    | Q4 only               | FP16 + Q4                 |
| DS R1 Distill 32B        | NO                    | NO                    | Q4 (~20 GB)               |
| Llama 3.1/3.3 70B        | NO                    | NO                    | Q4 (~40 GB)               |
| Mixtral 8x7B             | NO                    | NO                    | Q4 (~28 GB)               |


---

## Monthly Cost Estimates

Usage patterns:

- **Light**: 4 hrs/day × 20 days = 80 hrs/month (dev/testing)
- **Medium**: 8 hrs/day × 20 days = 160 hrs/month (team use, business hours)
- **Heavy**: 24/7 = 730 hrs/month (production API)

### On-Demand


| Instance    | Light | Medium | Heavy  |
| ----------- | ----- | ------ | ------ |
| g4dn.xlarge | $42   | $84    | $384   |
| g5.xlarge   | $80   | $161   | $734   |
| g5.2xlarge  | $97   | $194   | $885   |
| g5.12xlarge | $454  | $907   | $4,141 |


### Spot (saves 65–70%)


| Instance    | Light | Medium | Heavy   |
| ----------- | ----- | ------ | ------- |
| g4dn.xlarge | ~$13  | ~$25   | ~$115   |
| g5.xlarge   | ~$28  | ~$56   | ~$257   |
| g5.12xlarge | ~$159 | ~$318  | ~$1,449 |


> Spot instances can be interrupted with 2-minute notice. Use for dev/batch; use on-demand or reserved for production APIs.

---

## Recommended Tiers for MVP

### Tier 1 — Budget (`g4dn.xlarge`, T4 16GB)

- **Cost**: ~$13–84/month depending on usage
- **Best for**: Mistral 7B, Qwen 2.5 7B (Q4), Phi-3.5 Mini
- **Serving**: Ollama or llama.cpp (GGUF Q4/Q8)
- **Use case**: Individual developers, light team use

### Tier 2 — Standard (`g5.xlarge`, A10G 24GB)

- **Cost**: ~$28–734/month depending on usage
- **Best for**: Llama 3.1 8B (FP16), Gemma 2 9B, Qwen 2.5 14B (Q4)
- **Serving**: vLLM
- **Use case**: Small team production use, good throughput

### Tier 3 — Performance (`g5.12xlarge`, 4× A10G 96GB)

- **Cost**: ~$159–4,141/month depending on usage
- **Best for**: Llama 3.1/3.3 70B (Q4), Mixtral 8x7B, Qwen 2.5 72B
- **Serving**: vLLM with tensor parallelism (`--tensor-parallel-size 4`)
- **Use case**: Production API serving large models

---

## Key Considerations

1. **AWS GPU quota**: By default AWS accounts have 0 GPU quota. Users must request an increase for `g4dn` or `g5` families. This is the #1 friction point to address in UX (clear error message + link to quota request).
2. **Context length matters**: All VRAM numbers assume 2K–4K token context. Running 32K+ context adds 4–16 GB VRAM for KV cache.
3. **vLLM vs Ollama**:
  - **Ollama**: easier setup, supports GGUF quantization, good for single-user or dev use
  - **vLLM**: better throughput for concurrent requests, FP16/INT8/FP8, recommended for production
4. **Auto-stop is critical**: A forgotten running g5.12xlarge costs ~$4,100/month. Auto-stop after inactivity should be a default-on feature, not optional.
5. **Additional costs**: EBS storage ($0.08/GB/month), data transfer out ($0.09/GB), Elastic IP if needed. Typically <$5/month for typical model weights stored locally.

---

## Recommended MVP Model List

Focus on models that cover most use cases with good quality-to-cost ratio:


| Model                  | Recommended Instance | Monthly (Medium use) | Notes                                    |
| ---------------------- | -------------------- | -------------------- | ---------------------------------------- |
| Llama 3.1 8B           | g5.xlarge            | $161                 | Best general purpose, OpenAI alternative |
| Mistral 7B             | g4dn.xlarge          | $84                  | Fast, efficient, code-capable            |
| Qwen 2.5 14B           | g5.xlarge            | $161                 | Strong multilingual and coding           |
| DeepSeek R1 Distill 8B | g5.xlarge            | $161                 | Best reasoning at this size              |
| Llama 3.3 70B          | g5.12xlarge          | $907                 | GPT-4-level quality, max privacy         |


