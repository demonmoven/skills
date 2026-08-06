---
name: cspadk-implementation
description: Implements ONE OpenSpec change at a time using test-driven development. After completion, prompts to continue with next module or run openspec-flow for pending modules.
version: 1.3.1
metadata:
  patterns:
  - pipeline
  - reviewer
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

# cspadk-implementation

## Overview

This skill implements ONE OpenSpec change at a time using a test-driven development (TDD) approach. It delegates the actual implementation to sub-agents, ensuring that tests are written first, production code follows, and all tests pass before marking a task complete. After completion, it prompts to continue with the next module.

## Trigger Scenarios

- When the user wants to implement an OpenSpec change that already has generated proposals and tasks.
- When the user asks to "implement", "apply changes", or "start coding" for a requirement.
- After completing OpenSpec generation for a module.
- When continuing from a previous module's implementation.

## Inputs

- The requirement status obtained via `cspadk req-current --json`.
- Modules with status `openspec` (have generated artifacts, ready for implementation).
- Previous-step OpenSpec artifacts from the selected module's document references: `modules[].documents.proposal`, `modules[].documents.spec`, and `modules[].documents.tasks`.
- Additional workflow context from requirement status, especially `documents.design`, `documents.techDesign`, and `documents.breakdown`.
- Any extra context documents from `extraContext` array in requirement status (e.g., backend tech designs, data warehouse designs).

## Outputs

- Implemented and tested code changes in the working directory.
- Updated module status in requirement status file.

## Requirement Status Tracking

This skill updates the requirement status via CLI commands:

| Field | Value | CLI Command |
|-------|-------|-------------|
| `currentPhase` | `implementation` | `cspadk req-update-phase implementation` |
| `modules[].status` | `in_progress` | `cspadk req-module <id> --status in_progress` |
| `modules[].status` | `completed` | `cspadk req-module <id> --status completed` |

**Status file location**: `.ttadk/requirements-status/<requirement-name>.json`

**Module status values**:
- `pending` - Module identified, no OpenSpec generated
- `openspec` - OpenSpec artifacts generated, ready for implementation
- `in_progress` - Implementation started
- `completed` - Implementation finished

**Status schema during implementation**:
```json
{
  "updatedAt": "<iso-timestamp>",
  "currentPhase": "implementation",
  "documents": {
    "design": "docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md",
    "breakdown": "docs/requirements-breakdown/<requirement-name>.md"
  },
  "modules": [
    {
      "id": "<module-1-id>",
      "title": "<module-1-title>",
      "status": "completed",
      "documents": {
        "proposal": "openspec/changes/<change-key-1>/proposal.md",
        "spec": "openspec/changes/<change-key-1>/spec.md",
        "tasks": "openspec/changes/<change-key-1>/tasks.md"
      }
    },
    {
      "id": "<module-2-id>",
      "title": "<module-2-title>",
      "status": "openspec",
      "documents": {
        "proposal": "openspec/changes/<change-key-2>/proposal.md",
        "spec": "openspec/changes/<change-key-2>/spec.md",
        "tasks": "openspec/changes/<change-key-2>/tasks.md"
      }
    }
  ]
}
```

## Core Principles

This skill adheres to the shared workflow principles defined in [references/cspadk-workflow-principles.md](references/cspadk-workflow-principles.md).

## Execution Steps (Pattern: Pipeline & Reviewer)

### Step 0: Get Current Requirement Status and Read Required References

- **Action**: Read `references/openspec-apply-change.md` BEFORE selecting any module or starting implementation.
- **Action**: Treat `references/openspec-apply-change.md` as the source of truth for how the OpenSpec artifacts must be applied during implementation.
- **Action**: Read `references/test-driven-development.md` BEFORE selecting any module or starting implementation.
- **Action**: Treat `references/test-driven-development.md` as the source of truth for the test-first workflow and verification rules.
- **Action**: Run `cspadk req-current --json` to get the current requirement status.
- **Action**: Parse the output to understand:
  - Current phase
  - Available modules and their statuses
  - Per-module document references
  - Additional upstream requirement documents such as `documents.design`, `documents.techDesign`, and `documents.breakdown`
- **Action**: If no requirement is found, inform the user to run `cspadk-requirement-init` first.

### Step 1: Select ONE Module

- **Action**: Filter modules to show those with status `openspec` or `in_progress`.
- **Action**: If no such modules exist, check for `pending` modules and suggest running `/cspadk-openspec-flow` first.
- **Action**: Use the `AskUserQuestion` tool to let the user select ONE module to implement.
- **Action**: Verify that the selected module has previous-step OpenSpec artifacts recorded in requirement status:
  - `modules[].documents.proposal`
  - `modules[].documents.spec`
  - `modules[].documents.tasks`
- **Action**: If any of these document references are missing, inform the user to run `/cspadk-openspec-flow` for that module first.

### Step 2: Update Module Status to In Progress

- **Action**: Before starting implementation, update the module status:
  ```bash
  cspadk req-update-phase implementation --json
  cspadk req-module <module-id> --status in_progress --json
  ```

### Step 3: Delegate Implementation to Sub-Agent

- **Action**: Invoke a sub-agent with the `openspec-apply-change` skill.
- **Action**: Pass the module's OpenSpec document paths from requirement status (`modules[].documents.proposal`, `modules[].documents.spec`, `modules[].documents.tasks`) to the sub-agent.
- **Action**: Also pass upstream context artifacts such as `documents.design`, `documents.techDesign`, and `documents.breakdown` when present so the implementation step has more product context, design rationale, and technical detail.
- **Action**: Require the implementation flow to obey `references/openspec-apply-change.md` and `references/test-driven-development.md`, not just the summary in this skill.
- **Action**: The sub-agent MUST adopt TDD by following the `test-driven-development` skill:
  1. **Write tests first**: Generate test cases before writing any production code.
  2. **Implement production code**: Write the minimum code to make tests pass.
  3. **Verify tests pass**: Run all tests and confirm they pass.
- **Reference**: See [references/openspec-apply-change.md](references/openspec-apply-change.md) for the openspec-apply-change skill details.
- **Reference**: See [references/test-driven-development.md](references/test-driven-development.md) for the test-driven-development skill details.

### Step 4: Auto-Fix Test Failures

- **Action**: If a sub-agent reports test failures after implementation, instruct the sub-agent to automatically fix the failing code.
- **Action**: Re-run tests after the fix to confirm they pass.
- **Action**: If tests still fail after auto-fix, report the failure to the user and pause for guidance.

### Step 5: Mark Module as Completed

- **Action**: After successful implementation, update the module status:
  ```bash
  cspadk req-module <module-id> --status completed --json
  ```

### Step 6: Prompt for Next Action

- **Action**: After successfully implementing a module, check for remaining modules:
  - Modules with status `openspec`: Ready for implementation
  - Modules with status `pending`: Need OpenSpec generation

- **Action**: Use `AskUserQuestion` to ask the user what to do next only when more work remains:
  - If `openspec` modules exist: "Implement another module?"
  - If `pending` modules exist: "Generate OpenSpec for next module?"
  - If all modules completed: do not ask a follow-up question here; let `/cspadk-continue` advance the workflow automatically.

- **If user chooses to implement another module**:
  - Loop back to Step 1 with modules having status `openspec`.

- **If user chooses to generate OpenSpec for next module**:
  - Instruct the user to run `/cspadk-openspec-flow`.

## Next Step

After completing this skill:

- **Recommended**: Use `/cspadk-continue` to automatically proceed to the next workflow phase (deploy or archive).
- **If more modules to implement**: Continue with `/cspadk-implementation`.
- **If pending modules need OpenSpec**: Run `/cspadk-openspec-flow`.
- **If all modules completed**: Return control to `/cspadk-continue`, which should invoke `/cspadk-archive` automatically.

Update the requirement status via CLI before proceeding.

## Output Language

Detect the user's input language and respond in the same language.
- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).
