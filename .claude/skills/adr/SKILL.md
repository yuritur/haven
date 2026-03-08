---
name: adr
description: "Record an architectural decision as an ADR and update ARCHITECTURE.md. Use this skill whenever an important architectural decision has been made or agreed upon during conversation — such as choosing a technology, changing a design pattern, adding/removing a component, or revising infrastructure. Trigger even if the user doesn't say 'ADR' explicitly — any resolved architectural discussion should be captured."
---

# Architecture Decision Record

When an architectural decision has been made in conversation, capture it before it gets lost. This skill creates an ADR file and updates the architecture document so the team always has a written trail of what was decided and why.

## Step 1: Confirm the decision

Before writing anything, briefly restate the decision back to the user in 1-2 sentences. Make sure you've captured:
- **What** was decided
- **Why** (the key reasoning)
- **What alternatives** were considered (if discussed)

Get a confirmation before proceeding. If the decision is ambiguous or incomplete, ask one clarifying question — no more.

## Step 2: Determine the next ADR number

Look at existing files in `docs/architecture/decisions/` and pick the next sequential number (e.g., if 006 exists, create 007).

## Step 3: Create the ADR file

Create `docs/architecture/decisions/{NNN}-{slug}.md` where `{slug}` is a short kebab-case summary (2-4 words).

Follow this structure — it matches the existing ADRs in the project:

```markdown
# ADR-{NNN}: {Title}

**Status:** Accepted
**Date:** {YYYY-MM}

## Context

What problem or question prompted this decision? Keep it to 2-4 sentences.

## Decision

What was decided. Be specific and concrete — one paragraph or a few bullet points.

## Reasoning

Why this option was chosen over alternatives. Focus on the trade-offs that mattered most.

## Consequences

What follows from this decision — both positive implications and any new constraints or trade-offs the team accepts.

## Alternatives considered

Brief description of each alternative and why it was rejected. Skip this section if no alternatives were discussed.
```

Guidelines:
- Write in third person, present tense ("Haven uses X" not "we decided to use X")
- Keep the total ADR under 60 lines — concise is better than exhaustive
- The "Reasoning" section is the most important part — it answers "why" for future readers

## Step 4: Update ARCHITECTURE.md

Open `docs/architecture/ARCHITECTURE.md` and:

1. **Always**: add a row to the "Key architectural decisions" table at the bottom with the ADR number, short decision summary, and one-line rationale.

2. **If relevant**: update other sections of the document that are affected by this decision (e.g., Module layout, AWS resources, Security posture, Serving backends). Only change sections where the decision materially changes what's described — don't force updates where nothing changed.

## Step 5: Summary

After creating both files, tell the user:
- The ADR file path
- What was updated in ARCHITECTURE.md
- A one-line summary of the recorded decision
