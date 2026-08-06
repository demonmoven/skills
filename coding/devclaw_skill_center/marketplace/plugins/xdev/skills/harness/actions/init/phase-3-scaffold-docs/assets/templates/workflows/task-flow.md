# Default Task Flow

本文件描述 Agent 执行任务的默认工作流。

## Default Working Loop

1. **Orient** — Read AGENTS.md, understand the task scope, locate relevant files
2. **Plan** — For complex tasks (>1 hour), create an ExecPlan in `docs/plans/proposal/`
3. **Implement** — Write code, following invariants and project conventions
4. **Validate** — Run tests, linters, type checks. All must pass
5. **Document** — Update docs if the change affects architecture, APIs, or workflows
6. **Commit** — Atomic commits with clear messages

## Bugfix Flow

1. Reproduce the bug (write a failing test if possible)
2. Identify root cause (read surrounding code, check git blame)
3. Fix the root cause (not just the symptom)
4. Verify the fix (run the failing test, run full test suite)
5. Check for similar bugs elsewhere

## Feature Flow

1. Understand the requirements (read existing docs, ask clarifying questions)
2. Design the approach (for complex features, use ExecPlan)
3. Implement incrementally (commit after each working milestone)
4. Write tests alongside implementation
5. Update documentation
6. Run full validation suite

## Definition of Done

A task is done when:

- All tests pass (no SKIP, no failures)
- Linters pass (no warnings, no errors)
- Documentation is updated (if applicable)
- ExecPlan progress is updated (if applicable)
- Code is committed with a clear message
