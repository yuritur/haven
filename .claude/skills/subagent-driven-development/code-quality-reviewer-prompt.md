# Code Quality Reviewer Prompt Template

Use this template when dispatching a code quality reviewer subagent.

**Purpose:** Verify implementation is well-built (clean, tested, maintainable)

**Only dispatch after spec compliance review passes.**

```
Agent tool (code-reviewer):
  description: "Review code quality for Task N"
  prompt: |
    Review the code changes between BASE_SHA and HEAD_SHA for quality.

    ## What Was Implemented

    [from implementer's report]

    ## Requirements

    Task N from [plan-file]:
    [FULL TEXT of task requirements]

    ## Review Scope

    git diff BASE_SHA..HEAD_SHA

    ## Your Job

    Review for:
    - Code quality, readability, maintainability
    - Error handling correctness
    - Test coverage and test quality
    - Naming conventions and consistency with codebase
    - YAGNI violations (over-engineering)
    - Security issues

    Report:
    - Strengths: what's done well
    - Issues (Critical/Important/Minor): specific problems with file:line
    - Assessment: Approved / Changes Requested
```

**Code reviewer returns:** Strengths, Issues (Critical/Important/Minor), Assessment
