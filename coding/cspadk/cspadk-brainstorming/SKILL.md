---
name: cspadk-brainstorming
description: You MUST use this before any creative work - creating features, building components, adding functionality, or modifying behavior. Explores user intent, requirements and design before implementation.
version: 1.2.1
metadata:
  patterns:
    - generator
    - reviewer
  domain: cspadk
  i18n_level: 0
  prompt_version: "1.2.1"
  agent_support:
    - claude-code
    - trae
    - trae-cli
    - coco
  language:
    - en
---

# cspadk-brainstorming

## Overview

Help turn ideas into fully formed designs and specs through natural collaborative dialogue. This skill wraps the core brainstorming process with CSPADK-specific workflow integration and packages the original brainstorming support materials under `references/`.

This skill is **clarification-first**. Its default behavior is to surface unknowns, ask follow-up questions, and refine scope before proposing designs or any downstream planning. Do not drift into implementation planning, task breakdown, or execution thinking while material requirements, constraints, or success criteria are still unclear.

## Bundled References

This skill keeps the source brainstorming materials as local references:

- Core workflow: [references/brainstorming.md](references/brainstorming.md)
- Shared workflow principles: [references/cspadk-workflow-principles.md](references/cspadk-workflow-principles.md)
- Visual companion guide: [references/visual-companion.md](references/visual-companion.md)
- Spec reviewer prompt template: [references/spec-document-reviewer-prompt.md](references/spec-document-reviewer-prompt.md)
- Visual companion assets: `references/scripts/`
- Original skill metadata snapshot: `references/.metadata.json`

## Trigger Scenarios

- When the user indicates they want to start a new requirement or feature.
- After `cspadk-requirement-init` has completed initialization.
- When the user asks to "brainstorm", "design", or "explore ideas".

## Inputs

- The requirement description from `cspadk-requirement-init`.
- The requirement status obtained via `cspadk req-current`.
- Any local PRD or technical design documents referenced from requirement status (`documents.prd`, `documents.td`).
- Any extra context documents from `extraContext` array in requirement status (e.g., backend tech designs, data warehouse designs).

## Outputs

- A design document at `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md`.
- Updated requirement status in `.ttadk/requirements-status/<name>.json`.

## Requirement Status Tracking

This skill updates the requirement status via CLI commands:

| Field              | Value               | CLI Command                                |
| ------------------ | ------------------- | ------------------------------------------ |
| `currentPhase`     | `brainstorm`        | `cspadk req-update-phase brainstorm`       |
| `documents.design` | `<design-doc-path>` | `cspadk req-update-document design <path>` |

**Status file location**: `.ttadk/requirements-status/<requirement-name>.json`

**Status schema**:

```json
{
  "updatedAt": "<iso-timestamp>",
  "currentPhase": "brainstorm",
  "documents": {
    "design": "docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md"
  },
  "modules": []
}
```

## Core Principles

This skill adheres to the shared workflow principles defined in [references/cspadk-workflow-principles.md](references/cspadk-workflow-principles.md).

**Hard gate**: Do NOT invoke downstream planning or implementation-oriented skills, generate code, scaffold artifacts, draft implementation tasks, or take any implementation action until a design has been presented and approved. This applies even when the request seems simple.

**Clarification rule**: Stay in brainstorming and clarification mode while material unknowns remain. If unresolved details would change the scope, workflows, interfaces, constraints, edge cases, or success criteria, ask more questions instead of guessing.

**Question depth rule**: For simple, already-specific asks, the clarification phase may be brief. For ambiguous, user-facing, multi-step, cross-system, or constraint-heavy work, you should usually ask many detailed follow-up questions before proposing approaches or drafting a design.

## Execution Steps (Pattern: Generator & Reviewer)

### Step 0: Get Current Requirement Status

- **Action**: Run `cspadk req-current --json` to get the current requirement status.
- **Action**: Parse the output to understand the current phase and any existing documents.
- **Action**: Extract the local PRD/TD paths from `documents.prd` and `documents.td` when present.
- **Action**: Extract any extra context documents from the `extraContext` array and read them for supplementary context.
- **Action**: If no requirement is found, inform the user to run `cspadk-requirement-init` first.

### Step 1: Explore Project Context

- **Action**: Check files, docs, and recent commits to understand the current project state.
- **Action**: Read any PRD or technical design documents from the local paths recorded in requirement status (`documents.prd`, `documents.td`).
- **Action**: Read any extra context documents from `extraContext` array in requirement status.
- **Action**: If the status file does not contain local PRD/TD links, report the missing artifact and ask the user whether to continue with limited context.

### Step 2: Execute Brainstorming

- **Action**: \[Force] Read and follow the brainstorming process defined in [references/brainstorming.md](references/brainstorming.md), but enforce the sequence explicitly in this skill.
- **Action**: Assess scope first. If the request contains multiple independent subsystems or is too large for one coherent design, stop and help the user decompose it before continuing.
- **Action**: Use [references/visual-companion.md](references/visual-companion.md) when the discussion benefits from visual mockups, diagrams, or side-by-side comparisons.
- **Action**: Reuse the companion assets in `references/scripts/` if you need the original browser-based brainstorming support files.
- **Action**: Enter a clarification loop before proposing any approaches.
  - Ask one question per message using the `AskUserQuestion` tool.
  - Prefer multiple-choice questions when possible, but use open-ended questions when needed.
  - Usually ask many detailed follow-up questions for non-trivial or ambiguous requirements.
  - Keep clarifying until the requirement is sufficiently determined.
- **Action**: During clarification, make sure you understand at least the following before moving on:
  - Purpose / problem being solved
  - Target users or stakeholders
  - Scope and non-goals
  - Constraints and dependencies
  - Success criteria
  - Edge cases / failure expectations
  - Key assumptions that still need confirmation
- **Action**: If any unresolved detail would materially affect the design, continue clarifying instead of proposing solutions.
- **Action**: Only after the clarification loop has produced a stable enough understanding, propose 2-3 approaches with trade-offs and a recommendation.
- **Action**: Only after the user has reacted to the approaches and the direction is stable, present the design and get user approval.
- **Action**: If new ambiguity or disagreement appears during approach discussion or design review, return to clarification instead of pushing forward.

### Step 3: Write Design Doc

- **Action**: Write the validated design (spec) to `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md`.
- **Action**: Use [references/spec-document-reviewer-prompt.md](references/spec-document-reviewer-prompt.md) as the review template when dispatching a spec review pass.
- **Action**: Do NOT commit automatically - leave changes for user to review.

### Step 4: Update Status via CLI

- **Action**: Update the phase:
  ```bash
  cspadk req-update-phase brainstorm --json
  ```
- **Action**: Update the document reference:
  ```bash
  cspadk req-update-document design docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md --json
  ```

## Next Step

After completing this skill:

- **Recommended**: Use `/cspadk-continue` to automatically proceed to the next workflow phase.
- **Tech Design**: Use `/cspadk-tech-design` to create a detailed Technical Design document (Frontend or Backend).
- **Full workflow**: After tech design, use `/cspadk-requirement-breakdown` to split complex requirements into logical modules.
- **Lite workflow**: After tech design, use `/cspadk-openspec-flow` to generate OpenSpec artifacts directly.

Do not draft implementation steps, task breakdowns, or other plan-shaped output inside this skill before the design is validated and approved.

Update the requirement status via CLI before proceeding.

## Output Language

Detect the user's input language and respond in the same language.

- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).

