# ADR-003: Deployment state as JSON in S3

**Status:** Accepted (revised from Terraform state, 2026-03)
**Date:** 2026-03

## Context

Haven creates AWS resources and needs to track their IDs for `haven status` and `haven destroy`.
Since Terraform is no longer used (see ADR-002), there is no tfstate. A lightweight alternative is needed.

Requirements:
- `haven destroy` must reliably find and delete every resource created by `haven deploy`
- `haven status` should work from any machine with AWS credentials (not just the machine that deployed)
- No manual setup

## Decision

Store deployment state as a JSON file in S3. Haven creates the bucket automatically on first deploy.

## State schema

```json
{
  "deployment_id": "haven-abc123",
  "created_at": "2026-03-01T12:00:00Z",
  "region": "us-east-1",
  "resources": {
    "vpc_id": "vpc-0abc...",
    "subnet_id": "subnet-0abc...",
    "igw_id": "igw-0abc...",
    "route_table_id": "rtb-0abc...",
    "security_group_id": "sg-0abc...",
    "instance_id": "i-0abc...",
    "eip_allocation_id": "eipalloc-0abc..."
  },
  "endpoint": "http://1.2.3.4:8000/v1",
  "api_key": "sk-haven-..."
}
```

## Implementation

```
haven deploy (first run in account)
  → aws-sdk-go-v2: create s3://haven-state-{aws_account_id}
      versioning: enabled
      encryption: AES256
      public access: blocked
  → deploy resources, collect IDs
  → upload state JSON to s3://haven-state-{account_id}/{deployment_id}.json

haven destroy --deployment haven-abc123
  → download state JSON
  → delete resources in reverse order
  → delete state JSON

haven status
  → list and download state JSONs from bucket
  → display all active deployments
```

Bucket name includes account ID to avoid global S3 name collisions.

## Consequences

- Zero manual setup — bucket is created transparently on first `haven deploy`
- `haven status` and `haven destroy` work from any machine with AWS credentials
- State is human-readable JSON — no proprietary format, easy to inspect or recover manually
- `haven destroy --all` can optionally delete the state bucket for full account cleanup
- Additional AWS costs: S3 (~$0.023/GB/month, negligible for small JSON files)

## Alternatives considered

- **Local file (`~/.haven/state/`)**: simple, but breaks multi-machine usage and is lost on disk wipe. Acceptable for solo dev, but not for team use or compliance scenarios.
- **Terraform tfstate in S3**: removed along with Terraform dependency (ADR-002)
- **DynamoDB**: overkill for key-value JSON blobs at this scale; adds another AWS service to manage
