---
name: cspadk-archive
description: Archive a completed change in the CSPADK workflow. Finalizes and archives both OpenSpec changes and requirement status after implementation is complete.
version: 1.1.0
metadata:
  patterns:
    - pipeline
  domain: cspadk
  i18n_level: 0
  prompt_version: "1.1.0"
  agent_support:
    - claude-code
    - trae
    - trae-cli
    - coco
  language:
    - en
---

# cspadk-archive

## Overview

Archive a completed change in the CSPADK workflow. This skill finalizes and archives:
1. **OpenSpec changes** - Moves to `openspec/changes/archive/YYYY-MM-DD-<change-name>/`
2. **Requirement status** - Moves to `.ttadk/csp-archives/<requirement-name>.json`
3. **Docs sync** - Updates `docs/` directory to reflect implementation changes

This ensures all artifacts, tracking data, and documentation are properly preserved and up-to-date.

## Output Language

Detect the user's input language and respond in the same language.
- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).

## Trigger Scenarios

- After `cspadk-implementation` has completed implementing a change.
- When the user asks to "archive", "finalize", or "complete" a requirement.
- When all tasks in a change are complete and the user wants to close it out.

## Input Validation

| Input Key | Required | Source Skill | Fallback |
|-----------|----------|--------------|----------|
| requirement | Yes | cspadk-implementation | None - error if missing |
| openspec-change | Yes | cspadk-implementation | None - error if missing |

## Inputs

- The requirement status obtained via `cspadk req-current`.
- The OpenSpec change directory at `openspec/changes/<change-key>/`.

## Outputs

- Archived OpenSpec change at `openspec/changes/archive/YYYY-MM-DD-<change-name>/`.
- Archived requirement status at `.ttadk/csp-archives/<requirement-name>.json`.
- Updated docs under `docs/` reflecting implementation changes.

## Requirement Status Tracking

This skill archives the requirement status by moving it to the archive directory:

| Action | Source | Destination |
|--------|--------|-------------|
| Archive Status | `.ttadk/requirements-status/<name>.json` | `.ttadk/csp-archives/<name>.json` |
| Archive OpenSpec | `openspec/changes/<change-key>/` | `openspec/changes/archive/YYYY-MM-DD-<change-key>/` |

**Final status schema (archived)**:
```json
{
  "updatedAt": "<iso-timestamp>",
  "currentPhase": "archived",
  "archivePath": ".ttadk/csp-archives/<requirement-name>.json",
  "documents": {
    "design": "docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md",
    "breakdown": "docs/requirements-breakdown/<requirement-name>.md",
    "openspec-proposal": "openspec/changes/archive/YYYY-MM-DD-<change-key>/proposal.md",
    "openspec-spec": "openspec/changes/archive/YYYY-MM-DD-<change-key>/spec.md",
    "openspec-tasks": "openspec/changes/archive/YYYY-MM-DD-<change-key>/tasks.md"
  },
  "modules": [
    { "id": "<module-1-id>", "title": "<module-1-title>", "status": "completed" }
  ]
}
```

## Core Principles

This skill adheres to the shared workflow principles defined in [references/cspadk-workflow-principles.md](references/cspadk-workflow-principles.md).

## Execution Steps (Pattern: Pipeline)

### Step 0: Validate Inputs

- **Action**: Run `cspadk req-current --json` to get the current requirement status.
- **Action**: Parse the output to understand:
  - The requirement ID/name
  - The current phase
  - Available modules and their status
  - Document references
- **Error Handling**: If no requirement is found, inform the user to run `/cspadk-requirement-init` first.

### Step 1: Verify Completion Status

- **Action**: Check that all modules have status `completed`.
- **Action**: Read the tasks file (`tasks.md`) to verify all tasks are marked complete `- [x]`.
- **Error Handling**: If incomplete items found:
  - Display warning showing incomplete modules/tasks
  - Use `AskUserQuestion` to confirm user wants to proceed
  - Proceed if user confirms

### Step 2: Sync Delta Specs (if applicable)

Follow the procedure in [references/openspec-sync-specs.md](references/openspec-sync-specs.md):

- **Action**: Check for delta specs at `openspec/changes/<name>/specs/`.
- **Action**: If delta specs exist, compare with main specs at `openspec/specs/<capability>/spec.md`.
- **Action**: Show a summary of changes and prompt user:
  - "Sync now (recommended)" - sync delta specs to main specs before archiving
  - "Archive without syncing" - proceed without sync

### Step 3: Sync Docs

Follow the procedure in [references/docs-sync.md](references/docs-sync.md):

- **Action**: Analyze the implementation changes to identify which `docs/` areas are affected (architecture, modules, CLI commands, skills, constraints, workflow, experiences).
- **Action**: Use git diff and OpenSpec artifacts to determine the change scope.
- **Action**: For each affected doc, read the current content and the source of truth, then apply updates to reflect the current state.
- **Action**: Verify `docs/architecture/module-index.md` is current.
- **Action**: Show summary of docs updated or confirm no updates needed.
- **Error Handling**: If a doc references something that no longer exists, remove the stale reference. If a new area has no corresponding doc, prefer updating an existing doc over creating a new one.

### Step 4: Archive OpenSpec Changes

Follow the procedure in [references/openspec-archive.md](references/openspec-archive.md):

- **Action**: Create the archive directory if it doesn't exist:
  ```bash
  mkdir -p openspec/changes/archive
  ```
- **Action**: Generate target name using current date: `YYYY-MM-DD-<change-name>`
- **Action**: Check if target already exists. If yes, fail with error.
- **Action**: Move the OpenSpec change directory to archive:
  ```bash
  mv openspec/changes/<name> openspec/changes/archive/YYYY-MM-DD-<name>
  ```

### Step 5: Archive Requirement Status

- **Action**: Create the archive directory if it doesn't exist:
  ```bash
  mkdir -p .ttadk/csp-archives
  ```
- **Action**: Get the current requirement ID from the status.
- **Action**: Move the requirement status file to archive:
  ```bash
  mv .ttadk/requirements-status/<requirement-name>.json .ttadk/csp-archives/<requirement-name>.json
  ```

### Step 6: Display Summary

- **Action**: Show archive completion summary including:
  - Requirement name/ID
  - OpenSpec archive location: `openspec/changes/archive/YYYY-MM-DD-<name>/`
  - Requirement status archive location: `.ttadk/csp-archives/<requirement-name>.json`
  - Whether specs were synced
  - Which docs were updated (or "No doc updates needed")
  - Note about any warnings

## References

- [CSPADK Workflow Principles](references/cspadk-workflow-principles.md) - Shared workflow principles
- [OpenSpec Archive](references/openspec-archive.md) - OpenSpec change archiving procedure
- [OpenSpec Sync Specs](references/openspec-sync-specs.md) - Delta specs sync procedure
- [Docs Sync](references/docs-sync.md) - Docs directory sync procedure

## Next Step

After completing this skill, the requirement workflow is complete. All artifacts have been archived and preserved for future reference.

No further workflow steps are needed. The user may:
- Start a new requirement with `/cspadk-requirement-init`
- Review archived OpenSpec changes in `openspec/changes/archive/`
- Review archived requirement statuses in `.ttadk/csp-archives/`
