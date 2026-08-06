---
name: cspadk-requirement-breakdown
description: Splits complex requirements into logical modules using a top-down approach, ensuring zero code and directory overlap between modules for parallel execution.
version: 1.3.1
metadata:
  patterns:
  - generator
  domain: cspadk
  i18n_level: 0
  prompt_version: "1.3.1"
  agent_support:
  - claude-code
  - trae
  - trae-cli
  - coco
  language:
  - en
---

# cspadk-requirement-breakdown

## Overview

This skill splits a complex requirement into logical modules using a top-down approach. The highest-priority constraint is that split modules MUST have zero code and directory overlap, guaranteeing that multiple tasks can be implemented in parallel by different developers or agent sessions.

## Trigger Scenarios

- After tech design, when the requirement is complex enough to need modular decomposition.
- When the user explicitly asks to break down a requirement into modules.
- When the user asks to "split", "decompose", or "break down" a requirement.

## Inputs

- The requirement status obtained via `cspadk req-current`.
- The brainstorming output referenced from requirement status (`documents.design`).
- The tech design document referenced from requirement status (`documents.techDesign`).
- Any local PRD or technical design documents referenced from requirement status (`documents.prd`, `documents.td`).
- Any extra context documents from `extraContext` array in requirement status (e.g., backend tech designs, data warehouse designs).

## Outputs

- A structured breakdown document in `docs/requirements-breakdown/<requirement-name>.md`.
- Updated requirement status in `.ttadk/requirements-status/<name>.json`.

## Requirement Status Tracking

This skill updates the requirement status via CLI commands:

| Field | Value | CLI Command |
|-------|-------|-------------|
| `currentPhase` | `breakdown` | `cspadk req-update-phase breakdown` |
| `documents.breakdown` | `<breakdown-path>` | `cspadk req-update-document breakdown <path>` |
| `modules` | `<modules-array>` | `cspadk req-module-set-all '<json>'` |

**Status file location**: `.ttadk/requirements-status/<requirement-name>.json`

**Status schema after breakdown**:
```json
{
  "updatedAt": "<iso-timestamp>",
  "currentPhase": "breakdown",
  "documents": {
    "design": "docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md",
    "breakdown": "docs/requirements-breakdown/<requirement-name>.md"
  },
  "modules": [
    { "id": "<module-1-id>", "title": "<module-1-title>", "status": "pending" },
    { "id": "<module-2-id>", "title": "<module-2-title>", "status": "pending" }
  ]
}
```

## Core Principles

This skill adheres to the shared workflow principles defined in [references/cspadk-workflow-principles.md](references/cspadk-workflow-principles.md).

## Execution Steps (Pattern: Generator)

### Step 0: Get Current Requirement Status

- **Action**: Run `cspadk req-current --json` to get the current requirement status.
- **Action**: Parse the output to understand the current phase and any existing documents.
- **Action**: Extract the brainstorming design path from `documents.design`.
- **Action**: Extract the local PRD/TD paths from `documents.prd` and `documents.td` when present.
- **Action**: Extract any extra context documents from the `extraContext` array and read them for additional context.
- **Action**: If no requirement is found, inform the user to run `cspadk-requirement-init` first.

### Step 1: Review Input

- **Action**: Read the brainstorming output from the path recorded in requirement status (`documents.design`) and use it as the primary input for requirement breakdown.
- **Action**: Supplement the breakdown with any local PRD/technical design documents recorded in requirement status (`documents.prd`, `documents.td`, and any optional tech design artifact path).
- **Action**: Also reference any `extraContext` documents from requirement status for additional technical context.
- **Action**: If `documents.design` is missing, stop and ask the user to run `cspadk-brainstorming` or provide the missing design artifact path before continuing.
- **Action**: Identify the overall scope and complexity of the requirement.

### Step 2: Identify Logical Modules

- **Action**: Use a top-down approach to split the requirement into logical modules based on effort and responsibility boundaries.
- **Action**: Treat the first module split as a draft until the user confirms it; be ready to revise the module plan based on user feedback.
- **Action**: For each module, define:
  - **Name**: A clear, concise module name (kebab-case).
  - **Description**: What this module covers and its boundaries.
  - **Affected directories/files**: The specific code paths this module will touch.
  - **Estimated effort**: A rough size estimate (S/M/L).

#### Backend-Specific Splitting Strategy

For backend requirements, apply these additional rules:

- **Split by API dimension**: Each sub-module should correspond to one API endpoint or a tightly-coupled group of related APIs. Name modules after the API they implement (e.g., `create-order-api`, `query-order-api`).
- **Keep modules at S or M size**: Avoid L-sized backend modules. If an API-centric module evaluates to L, decompose it further — for example, split a single API module into separate modules for the handler layer, data layer, or cross-cutting concerns (validation, transformation, persistence).
- **Grouping rule**: Only group multiple APIs into one module when they share the same handler and data layer with minimal branching logic; otherwise, keep them separate to preserve parallelism.
- **Extract shared infrastructure**: If multiple API modules depend on common utilities, shared types, client wrappers, or base infrastructure, extract these into a standalone module (e.g., `shared-types`, `infra-client`). This shared module is implemented first; API modules depend on it but remain independent of each other, preserving parallel execution.

### Step 3: Validate Zero Overlap (CRITICAL)

- **Action**: Verify that no two modules share any code files or directories.
- **Action**: If overlap exists, re-split the modules until zero overlap is achieved.
- **Constraint**: This is the highest-priority constraint. Zero overlap MUST be guaranteed before proceeding. This ensures that multiple tasks can be implemented in parallel by different developers or agent sessions without conflicts.

### Step 4: Confirm Breakdown with User

- **Action**: Present the proposed module breakdown to the user and explicitly ask for confirmation before recording it.
- **Action**: If the user requests changes, update the module split, boundaries, titles, or affected paths based on that feedback.
- **Action**: Repeat the confirmation loop until the user accepts the breakdown modules.

### Step 5: Record Breakdown

- **Action**: After the user confirms the modules, record the breakdown in `docs/requirements-breakdown/<requirement-name>.md` using the following structure:

```markdown
# Requirement Breakdown: <requirement-name>

## Overview
<Brief summary of the requirement>

## Modules

### Module 1: <module-name>
- **Description**: <what this module covers>
- **Affected paths**: <list of directories/files>
- **Effort**: <S/M/L>

### Module 2: <module-name>
- **Description**: <what this module covers>
- **Affected paths**: <list of directories/files>
- **Effort**: <S/M/L>

## Overlap Validation
- [ ] Module 1 and Module 2: No shared paths ✓
- [ ] Module 1 and Module 3: No shared paths ✓
- [ ] Module 2 and Module 3: No shared paths ✓
```

### Step 6: Sync Status via CLI

- **Action**: Update the phase:
  ```bash
  cspadk req-update-phase breakdown --json
  ```
- **Action**: Update the document reference:
  ```bash
  cspadk req-update-document breakdown docs/requirements-breakdown/<requirement-name>.md --json
  ```
- **Action**: Update the modules array with all identified modules:
  ```bash
  cspadk req-module-set-all '[{"id":"<module-1-id>","title":"<module-1-title>"},{"id":"<module-2-id>","title":"<module-2-title>"}]' --json
  ```
  Alternatively, write to a temporary file and use `--file`:
  ```bash
  echo '[{"id":"module-1","title":"Core Feature"},{"id":"module-2","title":"UI Components"}]' > /tmp/modules.json
  cspadk req-module-set-all --file /tmp/modules.json --json
  ```

## Next Step

After completing this skill:

- **Recommended**: Use `/cspadk-continue` to automatically proceed to the next workflow phase.
- **Full workflow**: Use `/cspadk-openspec-flow` to generate OpenSpec artifacts for the selected modules.
- **Lite workflow**: Use `/cspadk-openspec-flow` to generate OpenSpec artifacts.

Update the requirement status via CLI before proceeding.

## Output Language

Detect the user's input language and respond in the same language.
- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).
