---
name: git-workflow
description: "MUST trigger when starting implementation of a plan — before writing any code. Sets up a git branch, defines commit strategy, and ensures a PR is created at the end. Triggers on: beginning to execute a plan, starting development work from a design doc, transitioning from planning to coding. Even if only a few files will change, this workflow applies."
---

# Git Workflow for Plan Execution

## When This Activates

This skill activates at the boundary between planning and implementation — when a plan exists and you're about to start writing code. It defines how git is used throughout the implementation.

## Before Writing Any Code

1. **Create a branch** from `master` with a descriptive name:
   - Features: `feat/short-description`
   - Bug fixes: `fix/short-description`
   - Refactors: `refactor/short-description`
   - Keep names short (2-4 words, kebab-case)

2. **Announce the branch** to the user so they know where work is happening.

## During Implementation: Commit Strategy

The goal is to keep commits at a moderate size — each one should be a meaningful, reviewable unit of work. This matters because large commits bloat the LLM context when reviewing diffs, and tiny commits add noise.

**What makes a good commit:**
- One logical change that can be understood on its own
- Small related changes can be grouped (e.g., adding a function + its tests)
- A single step of the plan, or 2-3 steps if they're tightly coupled and small

**When to commit:**
- After completing a logical unit of work that compiles and passes tests
- Before moving to a different area of the codebase
- When the diff is getting large enough that it would be hard to review in one go

**When to group:**
- Trivial changes (imports, renames) can be folded into the next meaningful commit
- A new function and its tests belong together
- Config changes that only make sense alongside the code they configure

**Commit messages:** concise, descriptive, conventional format (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`).

## After All Tasks Complete

When the entire plan is implemented:

1. **Verify** all tests pass (`go test ./...`) and linter is clean (`golangci-lint run`)
2. **Create a PR** to `master` using `gh pr create` with:
   - Short title summarizing the change
   - Body with a summary of what was done (bullet points)
   - Test plan section
3. **Share the PR URL** with the user
