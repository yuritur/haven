<p align="center">
  <img src="./data/haven.png" width="100%" alt="Haven">
</p>

# Haven

Deploy open-source LLMs to your AWS account with one command. Get an OpenAI-compatible HTTPS endpoint in minutes.

```bash
haven deploy llama3.2:1b
```

## Motivation

Open-source models are getting powerful enough to be useful for real work. But using them through third-party API providers means trusting someone else with your data -- no guarantees it won't be logged, leaked, or used to train the next model.

Haven gives you a way to deploy uncensored, open-source models to your own cloud -- with no intermediaries, no data sharing, and no fear of sending sensitive information to someone else's servers. Your prompts stay on your infrastructure, period.

It's also just a fast way to experiment with open-source models without extra overhead or costs beyond the cloud resources themselves.

## How it works

Haven provisions an EC2 instance via CloudFormation, runs [Ollama](https://ollama.com) behind an nginx TLS reverse proxy, and returns a ready-to-use endpoint with an API key.

- No external dependencies -- single Go binary
- Infrastructure as code via CloudFormation (no Terraform required)
- State stored in S3 (no local files, no DynamoDB)
- Self-signed TLS with certificate pinning (TOFU)
- Security group restricted to your IP

## Supported models

| Model | Instance | GPU | ~$/hr |
|---|---|---|---|
| `llama3.2:1b` | t3.large | -- | $0.08 |
| `llama3.2:3b` | t3.xlarge | -- | $0.17 |
| `phi3:mini` | t3.large | -- | $0.08 |
| `qwen3.5:4b` | g5.xlarge | NVIDIA A10G | $1.01 |
| `qwen3.5:9b` | g5.xlarge | NVIDIA A10G | $1.01 |
| `qwen3.5:27b` | g5.2xlarge | NVIDIA A10G | $1.21 |

*Prices are approximate AWS on-demand rates for us-east-1. Actual costs may vary by region.*

## Install

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

### Use with OpenAI SDK

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

MIT -- see [LICENSE](LICENSE)
