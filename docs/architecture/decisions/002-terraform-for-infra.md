# ADR-002: Direct AWS API calls via aws-sdk-go-v2

**Status:** Accepted (revised from Terraform, 2026-03)
**Date:** 2026-03

## Context

Haven needs to create and destroy AWS resources (VPC, EC2, IAM, EIP).
Options: aws-sdk-go-v2 directly, Terraform, Pulumi, AWS CDK.

Haven's infrastructure scope for MVP is bounded and well-known:
VPC → Subnet → IGW → Route table → Security Group → EC2 → EIP (~8 resources, fixed dependency order).

## Decision

aws-sdk-go-v2 directly. No external infrastructure tooling.

## Reasoning

- **No external binary dependency**: Terraform requires a binary on the host (Go binary, but separate install). Auto-downloading it adds complexity and fails in corporate environments (proxy, antivirus, UAC on Windows).
- **Wrapper overhead**: Terraform has no stable programmatic API — only CLI. Using it from Go means subprocess management, stdout/stderr parsing, working directory with generated .tf files, exit code handling. This is a significant maintenance surface for fragile code.
- **Scope fit**: The infrastructure Haven creates is simple and fixed. The problems Terraform solves (complex dependency graphs, iterative human-driven changes, multi-team state) do not apply here. Haven *generates* infrastructure programmatically under a known topology.
- **State management is trivial at this scope**: storing 8 resource IDs in a JSON file is sufficient (see ADR-003).
- **Destroy is straightforward**: delete resources in reverse creation order using stored IDs.

## Implementation

```
haven deploy
  → aws-sdk-go-v2: create VPC
  → aws-sdk-go-v2: create Subnet, IGW, Route table
  → aws-sdk-go-v2: create Security Group
  → aws-sdk-go-v2: launch EC2
  → aws-sdk-go-v2: allocate + associate EIP
  → save resource IDs to state (see ADR-003)

haven destroy
  → load state
  → delete resources in reverse order
```

## Consequences

- Zero external tool dependencies for infrastructure management
- Full control over error messages and UX — AWS errors surface directly
- Auditable: users can read Go code to see exactly what API calls are made in their account (aligns with core "auditable" principle)
- If infrastructure scope grows significantly (multi-region, complex topologies), revisit with Pulumi Go SDK (no external binary, Python-free)

## Alternatives considered

- **Terraform**: declarative, auditable .tf files, good state management — but requires external binary, subprocess wrapper, .tf file generation, and is overengineered for Haven's fixed-topology MVP
- **Pulumi**: Python-native (rejected) or Go SDK available, but adds heavy dependency and Pulumi Cloud account friction for state
- **AWS CDK**: tied to AWS, CloudFormation under the hood, slow convergence, requires Node.js runtime
