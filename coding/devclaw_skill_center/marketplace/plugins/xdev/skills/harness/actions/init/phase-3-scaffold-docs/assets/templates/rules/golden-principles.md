# Golden Principles

本文件记录项目的长期指导原则。不变量（invariants）是硬性约束，原则（principles）是软性指导。

## Working Principles

1. **Read before write**: Before modifying any code, read the surrounding context, existing tests, and relevant documentation. Never assume the structure.

2. **Verify, don't trust**: After any code generation or modification, run the relevant tests and linters. Don't assume correctness.

3. **One thing at a time**: Each commit should have a single, clear purpose. Don't mix refactoring with feature work.

4. **Leave it better**: If you encounter tech debt while working on a task, log it in `docs/quality/debt-log.md` rather than fixing it in the same PR (unless trivial).

## Cleanup Heuristics

Low-risk cleanup that can be done opportunistically:

- Fix typos in comments and documentation
- Remove unused imports
- Add missing type annotations
- Update outdated comments

NOT low-risk (requires separate PR):

- Renaming public APIs
- Changing data structures
- Modifying database schemas
- Altering configuration formats
