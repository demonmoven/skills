# Docs Sync

Sync implementation changes to the project's `docs/` directory to ensure documentation stays accurate and up-to-date.

This is an **agent-driven** operation — you will analyze what changed during implementation, discover the project's current docs structure, and directly edit the corresponding docs files. This allows intelligent updates (e.g., adding a new command to a reference table without rewriting the entire doc).

**Input**: The requirement status (from `cspadk req-current --json`) and the OpenSpec change directory.

**Steps**

1. **Discover current docs structure**

   Scan the project's `docs/` directory to understand what documentation exists:

   - List all files and subdirectories under `docs/`.
   - Skim each doc file's headings to build a map of: which doc covers which topic.
   - This map is used to match implementation changes to the right docs.

2. **Identify changed areas**

   Analyze the implementation to determine what areas were affected. Use these sources:

   - **OpenSpec change artifacts**: Read `openspec/changes/<name>/proposal.md` and `tasks.md` to understand the scope of changes.
   - **Git diff**: Run `git diff <base-branch>...HEAD --name-only` to see which files were actually modified.
   - **Requirement status**: Check `documents` field for design/breakdown references that indicate the change scope.

   Categorize the changes into high-level areas (e.g., architecture, modules, commands, configuration, workflow, constraints, etc.) based on what actually changed in the codebase.

3. **Map changes to docs**

   Using the docs map from Step 1, determine which docs are affected:

   - For each changed area, find the doc(s) whose topic overlaps with that area.
   - A single change may affect multiple docs; a single doc may cover multiple areas.
   - If no existing doc covers a changed area, note it — you may need to add content to the closest relevant doc.

4. **If no doc updates needed, skip**

   If the changes don't affect any documented topics (e.g., pure bug fix with no doc impact), inform the user and proceed without doc updates.

5. **For each affected doc, apply updates**

   For each identified doc:

   a. **Read the current doc** to understand existing content.

   b. **Read the source of truth** — the actual code/config that changed — to determine what the doc should say.

   c. **Apply updates intelligently**:
      - **New items**: Add them in the appropriate section, following existing format and style.
      - **Modified items**: Update only the changed parts. Preserve surrounding content.
      - **Removed items**: Remove outdated entries. If a section becomes empty, remove the section heading too.
      - **Style consistency**: Match the existing doc's language, heading levels, and formatting.

   d. **Keep docs concise**: Docs should describe the *current state*, not the history of changes. Don't add changelog-style entries.

6. **Show summary**

   After applying all updates, summarize:
   - Which docs were updated and what changed
   - Which docs were already up-to-date (no changes needed)

**Output On Success**

```
## Docs Synced

Updated docs:
- **docs/<path>/file.md**: <brief description of what was updated>
- **docs/<path>/other.md**: <brief description of what was updated>

Already up-to-date:
- docs/<path>/ (no related changes)
```

**Guardrails**
- Always read the current doc before editing — don't assume its content.
- Verify against the actual codebase, not just the OpenSpec artifacts — implementation may have diverged.
- Preserve existing content that is not affected by the changes.
- Follow the existing doc's language (Chinese vs English) and formatting style.
- Don't create new doc files unless the change clearly warrants a new category — prefer updating existing files.
- The operation should be idempotent — running twice should give the same result.
- Do NOT add changelog entries or "updated on" timestamps to docs. Docs describe current state, not history.
