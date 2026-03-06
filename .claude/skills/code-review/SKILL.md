---
name: code-review
description: Run a comprehensive code review
---

# Senior Code Review

Code review from a senior developer perspective: design, architecture, maintainability, anti-patterns.

## When to Use

- "review code", "code review", "check this code"
- Verify code is sensible before merge
- Find poorly designed areas
- Assess solution maintainability

## Review Focus

### 1. Architecture & Design

- **Separation of concerns** — one module = one responsibility
- **Right abstractions** — not too generic, not too specific
- **Dependencies** — direction, cycles, unnecessary coupling
- **Layering** — business logic separated from infrastructure

### 2. Readability & Simplicity

- Is code understandable without comments?
- Does naming reflect intent?
- No over-engineering?
- No magic numbers/strings without explanation?

### 3. Anti-patterns

- **God Object** — class/module does everything
- **Shotgun Surgery** — one change requires edits in 10 places
- **Feature Envy** — method works more with other's data
- **Primitive Obsession** — strings/numbers instead of types
- **Copy-Paste** — duplication instead of abstraction
- **Leaky Abstraction** — implementation details exposed
- **Callback Hell / Promise Chain** — deep async nesting
- **Boolean Blindness** — `process(true, false, true)` without context

### 4. Maintainability

- Easy to add a new feature?
- Easy to modify existing one?
- What breaks if we change X?
- Any fragile spots (fragile base class)?

### 5. Testability

- Can it be tested in isolation?
- No hidden dependencies?
- Pure functions where possible?

## Review Process

1. **Scope** — determine what to review (git diff / specific files)
2. **Context** — understand what code should do
3. **Top-down** — architecture first, then details
4. **Be specific** — file:line + what's wrong + how to fix

## Issue Levels

| Level | Description | Action |
|-------|-------------|--------|
| **BLOCKER** | Broken design, won't work | Redo before merge |
| **MAJOR** | Bad pattern, will cause pain | Fix before merge |
| **MINOR** | Could be better, but works | Fix when convenient |
| **NIT** | Style, preference | Author's discretion |

## Output Format

```
## Code Review

**Scope:** [what was reviewed]
**Verdict:** APPROVE / NEEDS WORK / RETHINK

### Issues

**MAJOR** `src/service.py:42`
Problem: Service directly works with HTTP request object
Why bad: Business logic coupled to web framework
Solution: Accept specific parameters, not entire request

**MINOR** `src/models.py:15`
Problem: User model contains email sending methods
Why bad: Model knows about infrastructure
Solution: Extract to separate EmailService

### Good

- Clear router separation by domain
- Understandable function naming
- Proper use of dependency injection

### Recommendations

1. [specific improvement]
2. [specific improvement]
```

## What NOT to Do

- Don't nitpick style (linters exist for that)
- Don't rewrite someone's code "your way"
- Don't ignore context and constraints
- Don't demand perfection when good enough suffices

## Self-check Questions

When reviewing, ask yourself:

1. Would I understand this code in 6 months?
2. Could a new developer figure it out in an hour?
3. What breaks if requirements change slightly?
4. Where's the most fragile spot?
5. Does this solution scale?
