# ADR-004: SSM for instance access, no SSH in MVP

**Status:** Accepted
**Date:** 2026-03

## Context

Haven needs a way to access EC2 instances for debugging and log retrieval.
Options: SSH (port 22), AWS Systems Manager Session Manager (SSM).

## Decision

SSM only in MVP. SSH considered for v0.2 as an optional flag.

## Reasoning

- SSM requires no inbound port — Security Group is cleaner (only port 8000)
- No SSH key management — one less secret to generate and store
- Access is logged in AWS CloudTrail automatically
- Aligns with "security by default" principle

## Implementation

- Instance IAM role gets `AmazonSSMManagedInstanceCore` policy
- SSM Agent is pre-installed on AWS Deep Learning AMIs
- Access: `aws ssm start-session --target {instance_id}`

## Consequences

- Users need `aws ssm` CLI installed for debugging (it's part of standard awscli)
- Slightly slower session start vs SSH (~3-5 seconds)

## Revisit when

Users request SSH access for tooling compatibility (e.g. VSCode Remote, rsync).
Plan: `haven deploy ... --enable-ssh` flag that adds key pair + SG rule.
