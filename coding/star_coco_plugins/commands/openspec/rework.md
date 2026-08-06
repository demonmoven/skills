---
description: Rework an OpenSpec change when modifications are needed after the initial apply.
argument-hint: change-id and rework reason
---
<!-- OPENSPEC:START -->
**Guardrails**
- Favor straightforward, minimal implementations first and add complexity only when it is requested or clearly required.
- Keep changes tightly scoped to the requested outcome.
- Refer to `openspec/AGENTS.md` (located inside the `openspec/` directory) if you need additional OpenSpec conventions or clarifications.
- Ensure the updated proposal and tasks are validated before restarting implementation.

**Steps**
Track these steps as TODOs and complete them one by one.
1. Identify the `change-id` to rework and the specific reasons (e.g., failed tests, missed requirements, or feedback).
2. Review the current implementation state and the original `openspec/changes/<id>/tasks.md`.
3. If requirements have changed, update `proposal.md` and the delta specs in `openspec/changes/<id>/specs/`.
4. Update `openspec/changes/<id>/tasks.md` to reflect the rework:
   - Add new tasks for the required modifications.
   - Reset any previously "completed" tasks that now require changes back to pending (`- [ ]`).
   - Update `design.md` if the technical approach needs adjustment.
5. Run `openspec validate <id> --strict` to ensure the updated proposal is consistent.
6. Re-implement the pending tasks sequentially, focusing only on the modifications.
7. Confirm all tasks in the updated `tasks.md` are completed and marked as `- [x]`.

**Reference**
- Use `openspec show <id> --json --deltas-only` to review the updated requirements.
- Use `openspec list` to verify the change status.

$ARGUMENTS
<!-- OPENSPEC:END -->
