---
name: cspadk-openspec-flow
description: Translates designs into structured OpenSpec changes with TDD-enforced task generation. Supports one or more requirement modules in a single run. Use this skill when the user asks to generate specs, proposals, or tasks, or says "create openspec", "generate artifacts", or "start spec".
version: 1.4.1
metadata:
  patterns:
  - generator
  - tool-wrapper
  domain: cspadk
  i18n_level: 0
  prompt_version: "1.4.1"
  agent_support:
  - claude-code
  - trae
  - trae-cli
  - coco
  language:
  - en
---

# cspadk-openspec-flow

## Overview

This skill translates designs into structured OpenSpec changes for one or more requirement modules in a single run. It follows the artifact contract defined by `references/openspec-propose.md` with TDD enforcement. After completion, it prompts the user to proceed with implementation or generate OpenSpec for any remaining modules.

## Trigger Scenarios

- After requirement breakdown (or after tech-design for simpler requirements).
- When the user asks to generate specs, proposals, or tasks.
- When the user asks to "create openspec", "generate artifacts", or "start spec".
- When continuing from a previous module's implementation.

## Inputs

- The requirement status obtained via `cspadk req-current --json`.
- Modules with status `pending` or modules that need OpenSpec regeneration.
- Previous-step artifacts referenced from requirement status, especially `documents.breakdown`, `documents.design`, and `documents.techDesign`.
- Any optional supporting PRD/TD or source-document links already recorded in requirement status.
- Any extra context documents from `extraContext` array in requirement status (e.g., backend tech designs, data warehouse designs).

## Outputs

- OpenSpec change artifacts produced by `references/openspec-propose.md` under `openspec/changes/<change-key>/`, including nested generated files under `specs/` when the schema produces them.
- Updated module status in requirement status file.
- Per-module document references synced only after the generated change files have been read back from disk.

## Requirement Status Tracking

This skill updates the requirement status via CLI commands:

| Field | Value | CLI Command |
|-------|-------|-------------|
| `currentPhase` | `openspec` | `cspadk req-update-phase openspec` |
| `modules[].status` | `openspec` | `cspadk req-module <id> --status openspec` |
| `modules[].documents.proposal` | `<path>` | `cspadk req-module-document <id> proposal <path>` |
| `modules[].documents.specs` | `<array of spec paths>` | `cspadk req-module-document-append <id> specs <path>` |
| `modules[].documents.tasks` | `<path>` | `cspadk req-module-document <id> tasks <path>` |

**Adding modules when none exist**:
```bash
cspadk req-module-add <id> <title> [--days <days>] [--json]
```

**IMPORTANT**: Use `specs` (plural) as an array to track multiple spec files. Use `cspadk req-module-document-append` to add each spec file to the array. This allows a single module to have multiple spec files when the schema produces nested specs.

**Status file location**: `.ttadk/requirements-status/<requirement-name>.json`

**Module status values**:
- `pending` - Module identified, no OpenSpec generated
- `openspec` - OpenSpec artifacts generated, ready for implementation
- `in_progress` - Implementation started
- `completed` - Implementation finished

**Status schema after openspec generation**:
```json
{
  "updatedAt": "<iso-timestamp>",
  "currentPhase": "openspec",
  "documents": {
    "design": "docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md",
    "breakdown": "docs/requirements-breakdown/<requirement-name>.md"
  },
  "modules": [
    {
      "id": "<module-1-id>",
      "title": "<module-1-title>",
      "status": "openspec",
      "documents": {
        "proposal": "openspec/changes/<change-key-1>/proposal.md",
        "specs": [
          "openspec/changes/<change-key-1>/spec.md",
          "openspec/changes/<change-key-1>/specs/<capability>/spec.md"
        ],
        "tasks": "openspec/changes/<change-key-1>/tasks.md"
      }
    },
    {
      "id": "<module-2-id>",
      "title": "<module-2-title>",
      "status": "pending"
    }
  ]
}
```

## Core Principles

This skill adheres to the shared workflow principles defined in [references/cspadk-workflow-principles.md](references/cspadk-workflow-principles.md).

## Execution Steps (Pattern: Generator & Tool-Wrapper)

### Step 0: Get Current Requirement Status and Read Required References

- **Action**: Read `references/openspec-propose.md` BEFORE selecting any module or generating any OpenSpec artifact.
- **Action**: Treat `references/openspec-propose.md` as the source of truth for what artifacts must be produced and what rules they must obey.
- **Action**: Read `openspec/schemas/<schema-name>/schema.yaml` to understand the full artifact dependency chain. The schema defines which artifacts exist and their `requires` dependencies — this is essential to ensure NO artifact is skipped (e.g., `specs` requires `proposal`, `tasks` requires `specs` + `design`).
- **Action**: Run `cspadk req-current --json` to get the current requirement status.
- **Action**: Parse the output to understand:
  - Current phase
  - Available modules and their statuses
  - Existing documents
- **Action**: Read the previous-step artifact paths from requirement status, prioritizing `documents.breakdown`, then `documents.techDesign`, and then `documents.design`.
- **Action**: Also collect any supporting PRD/TD context from `documents.prd` and `documents.td` when available.
- **Action**: If no requirement is found, inform the user to run `cspadk-requirement-init` first.

### Step 1: Validate Upstream Artifacts and Select Modules

- **Action**: Read the breakdown artifact from `documents.breakdown` when present and use it as the primary module-splitting artifact.
- **Action**: Read `documents.techDesign` and `documents.design` as additional upstream context so the generated OpenSpec artifacts inherit more implementation detail and rationale.
- **Action**: If `documents.breakdown` is missing, fall back to `documents.design` for lite workflow; if both are missing, stop and ask the user to run the previous workflow step or provide the missing artifact path.
- **Action**: Filter modules to show only those with status `pending` (or `openspec` if regenerating).
- **Action**: Use the `AskUserQuestion` tool to let the user select one or more modules from the available modules.
- **Action**: If no eligible modules exist, inform the user that all modules already have OpenSpec generated or are in progress.

### Step 2: Process Selected Modules Sequentially

- **Action**: For each selected module, derive a kebab-case change name from the module ID: `<requirement-id>-<module-id>`.
- **Action**: For each selected module, invoke `openspec-propose` and follow the artifact workflow defined by `references/openspec-propose.md`.
- **Action**: Generate artifacts in the schema-defined dependency order: proposal → specs + design → tasks. **Do NOT skip the specs artifact.** The `specs` artifact creates `specs/<capability>/spec.md` files for each capability declared in the proposal's "New Capabilities" or "Modified Capabilities" sections.
- **Action**: After generation for that module, read back all generated files under `openspec/changes/<change-key>/` before syncing requirement status.
- **Action**: The readback MUST include nested generated files under `openspec/changes/<change-key>/specs/` when present, not just files at the change root.
- **Action**: Determine the canonical file paths to record for the module from the files that were actually generated on disk.
- **Action**: Process selected modules one by one, not in parallel, so each module's artifacts and status updates remain isolated and traceable.

### Step 3: Enforce TDD in Task Generation

- **Action**: When generating `tasks.md`, enforce that every task MUST include three steps:
  1. **Pre-step (Write Test)**: Write the test case(s) first before any production code.
  2. **Implementation Step**: Write the minimum production code to make the test pass.
  3. **Post-step (Verify Test)**: Run all tests and verify they pass. If tests fail, fix automatically.
- **Constraint**: No task in `tasks.md` is complete without all three steps. This ensures TDD discipline is built into the task structure.
- **Action**: Tasks MUST be grouped by file path, derived from design.md "Implementation Details" section:
  ```
  ## N. File: `path/to/file.ts`

  ### N.1 Tests
  - [ ] N.1.1 Write failing test for <specific change>
  - [ ] N.1.2 Verify test fails for expected reason

  ### N.2 Implementation
  - [ ] N.2.1 <specific change 1 from design>
  - [ ] N.2.2 <specific change 2 from design>

  ### N.3 Verification
  - [ ] N.3.1 Run `<test command>` - must pass
  - [ ] N.3.2 Run `<lint command>` - must pass
  ```

### Step 3.5: Validate Artifact Quality

- **Action**: After each artifact is created, validate against quality requirements:
  - **proposal.md**: Has Problem Analysis table, Module Decomposition table with effort estimates, Success Criteria with checkboxes, Capabilities section listing "New Capabilities" and/or "Modified Capabilities"
  - **specs**: One spec file per capability from proposal's Capabilities section (under `specs/<capability>/spec.md`), each with at least one requirement section (ADDED/MODIFIED/REMOVED/RENAMED), each requirement has at least one scenario with `#### Scenario:` header in WHEN/THEN format
  - **design.md**: Has Implementation Details section with per-file subsections covering change type, specific changes, key responsibilities, flow, change diffs (+/- line prefixes), and quality checks for each affected file; Detection Rules table, Exact Commands
  - **tasks.md**: Has task groups matching each file from design.md, Tests/Implementation/Verification subsections, Final Verification section, test tasks reference spec scenarios when applicable
- **Action**: Use `AskUserQuestion` if critical sections are missing:
  > "The [artifact] is missing [X, Y]. Would you like me to enhance these sections?"
- **Action**: Do not proceed to status sync until quality gate passes.

### Step 4: Sync Status via CLI

- **Action**: Update the phase (if not already `openspec`):
  ```bash
  cspadk req-update-phase openspec --json
  ```
- **Action**: After each selected module is processed, update that module's status:
  ```bash
  cspadk req-module <module-id> --status openspec --json
  ```
- **Action**: Only after reading back the generated change directory, update the per-module document references that were actually produced for that module according to `references/openspec-propose.md`.
- **Action**: For `proposal` and `tasks`, use single path:
  ```bash
  cspadk req-module-document <module-id> proposal <path> --json
  cspadk req-module-document <module-id> tasks <path> --json
  ```
- **Action**: For `specs`, use append to add each spec file to the array:
  ```bash
  cspadk req-module-document-append <module-id> specs <spec-path-1> --json
  cspadk req-module-document-append <module-id> specs <spec-path-2> --json
  ```
- **Action**: Treat the readback result as the source of truth for sync, including canonical nested `specs/<capability>/spec.md` paths when the generated schema uses them.
- **Action**: Ensure the requirement status records the final discovered paths from disk, not guessed or assumed locations.
- **Action**: If one selected module fails, report which modules were completed successfully and which modules remain pending or need regeneration before stopping.

### Step 5: Prompt for Next Action

- **Action**: After all selected modules have been processed, use `AskUserQuestion` to ask the user:
  - "Proceed to implementation?" (for one generated module at a time)
  - "Generate OpenSpec for remaining modules?" (if pending modules remain)

- **If user chooses to proceed to implementation**:
  - Instruct the user to run `/cspadk-implementation` to implement one generated module with TDD.

- **If user chooses to generate OpenSpec for remaining modules**:
  - Loop back to Step 1 with the remaining pending modules.

- **If user declines both options**:
  - Display a summary of completed modules, remaining modules, and available next steps.

## Next Step

After completing this skill:

- **Recommended**: Use `/cspadk-continue` to automatically proceed to the next workflow phase.
- **Alternative**: Use `/cspadk-implementation` to implement the generated OpenSpec changes with TDD, one module at a time.
- **Alternative**: Generate OpenSpec for other pending modules.

Update the requirement status via CLI before proceeding.

## Output Language

Detect the user's input language and respond in the same language.
- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).
