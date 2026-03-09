<p align="center">
  <img src="./data/haven.png" width="100%" alt="Haven">
</p>

# Haven

Deploy open-source LLMs to your own cloud with one command. No middlemen, no additional fees, no data leaks. Just your machine, your cloud, your models.

```bash
haven deploy llama3.2:1b
```

## Motivation

Open-source models are getting powerful enough to be useful for real work. But using them through third-party API providers means trusting someone else with your data — no guarantees it won't be logged, leaked, or used to train the next model.

Haven lets you deploy models to your own infrastructure — with no intermediaries and no fear of sending sensitive information to someone else's servers. It's also just a fast way to experiment without extra overhead or costs beyond the cloud resources themselves.

## How it works

Haven provisions a cloud instance, sets up the model behind an encrypted reverse proxy, and returns a ready-to-use API endpoint with an access key.

- Single binary, no external dependencies
- Infrastructure managed as code — automated provisioning and teardown
- TLS encryption with certificate pinning
- Network access restricted to your IP

## Supported models

| Model | GPU | ~$/hr |
|---|---|---|
| `llama3.2:1b` | — | $0.08 |
| `llama3.2:3b` | — | $0.17 |
| `phi3:mini` | — | $0.08 |
| `qwen3.5:4b` | NVIDIA A10G | $1.01 |
| `qwen3.5:9b` | NVIDIA A10G | $1.01 |
| `qwen3.5:27b` | NVIDIA A10G | $1.21 |

*Prices are approximate AWS on-demand rates for us-east-1.*

## Install

### macOS / Linux (Homebrew)

```bash
brew install yuritur/tap/haven
```

### macOS / Linux (script)

```bash
curl -sSL https://raw.githubusercontent.com/yuritur/haven/master/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/yuritur/haven/master/install.ps1 | iex
```

### From source

```bash
go install github.com/havenapp/haven/cmd/haven@latest
```

## Usage

> **Note:** Currently only AWS is supported as a cloud provider. You need an AWS account with credentials configured in your terminal:
>
> ```bash
> # Option 1: default profile
> aws configure
>
> # Option 2: named profile
> export AWS_PROFILE=my-profile
>
> # Option 3: explicit default profile
> export AWS_DEFAULT_PROFILE=my-profile
> ```

```bash
# Deploy a model
haven deploy llama3.2:1b

# List deployments
haven status

# Show TLS fingerprint
haven cert <deployment-id>

# Tear down
haven destroy <deployment-id>
```

## GPU models and vCPU quotas

AWS accounts have **0 vCPU quota** for GPU instance families (G, P) by default. When you deploy a GPU model for the first time, Haven will detect this and offer to request a quota increase automatically via the AWS Service Quotas API.

Small increases (e.g., 4 vCPUs for a single `g5.xlarge`) are typically auto-approved within a few minutes. Larger requests may take hours and require AWS support review.

If you prefer to request the increase manually:

```bash
aws service-quotas request-service-quota-increase \
  --service-code ec2 --quota-code L-DB2E81BA --desired-value 4
```

Or use the [AWS Console](https://console.aws.amazon.com/servicequotas/home#!/services/ec2/quotas/L-DB2E81BA).

## Use with OpenAI SDK

```python
from openai import OpenAI

client = OpenAI(
    base_url="https://<ip from haven deploy output>:11434/v1",
    api_key="sk-haven-...",  # from haven deploy output
)

response = client.chat.completions.create(
    model="llama3.2:1b",
    messages=[{"role": "user", "content": "Hello!"}],
)
```

## License

MIT — see [LICENSE](LICENSE)
