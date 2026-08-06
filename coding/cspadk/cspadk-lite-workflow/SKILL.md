---
name: cspadk-lite-workflow
description: Orchestrates a streamlined, interactive workflow for small requirements - runs brainstorming, OpenSpec generation, implementation, and archiving with user confirmation at each step using AskUserQuestion tool. Accepts single sentence descriptions or Feishu document URLs as input. Trigger with "start lite workflow", "quick feature", or "simple requirement".
version: 1.3.2
metadata:
  patterns:
    - pipeline
    - tool-wrapper
  domain: cspadk
  i18n_level: 0
  prompt_version: "1.3.2"
  agent_support:
    - claude-code
    - trae
    - trae-cli
    - coco
  language:
    - en
    - zh
---

# cspadk-lite-workflow

## Overview

This skill orchestrates the requirement workflow from requirement status instead of forcing the user to manually remember phase boundaries. It reads the active requirement `type` and stable `currentPhase`, then chooses the next valid stage automatically.

For `type: 'lite'`, the workflow is intentionally simplified:

1. **Requirement Initialization** - Creates branch and status tracking when no requirement exists
2. **Brainstorming** - Clarifies and expands the requirement
3. **OpenSpec Generation** - Creates proposal, spec, and tasks
4. **Implementation** - Executes tasks with TDD
5. **Optional PPE** - Runs only when project config enables PPE and the user agrees
6. **Archive** - Finalizes the completed work

Lite workflow skips `tech-design` and `breakdown`, and downstream OpenSpec / implementation stages use only the brainstorm design artifact as context.

For `type: 'full'`, the workflow preserves the full progression through `tech-design` and `breakdown` before OpenSpec and implementation.

The workflow is re-enterable: rerunning this skill resumes from the next valid stage derived from the last stable completed phase.

## Trigger Scenarios

- When the user says "start lite workflow", "quick feature", or "simple requirement"
- When the user has a small, well-defined feature to implement
- When the user wants to quickly prototype or implement a single-module change
- When the user provides a single sentence description or Feishu doc URL

## Inputs

- **Requirement description** - A single sentence describing the feature
- **OR Feishu document URL** - Link to a PRD or design doc
- **OR simple text description** - Brief explanation of what needs to be built

## Outputs

- Initialized requirement with branch and status tracking
- Brainstorming output with clarified requirements
- OpenSpec artifacts (proposal.md, spec.md, tasks.md)
- Implemented code changes
- Archived requirement status

## Workflow Phases

### Lite workflow (`type: 'lite'`)

```
┌─────────────────┐    ┌───────────────┐    ┌────────────────┐    ┌───────────────┐    ┌────────────┐    ┌──────────┐
│ 1. Initialize   │───▶│ 2. Brainstorm │───▶│ 3. OpenSpec    │───▶│ 4. Implement  │───▶│ 5. PPE?    │───▶│ 6. Archive│
│    (req init)   │    │   (ideas)     │    │   (artifacts)  │    │   (tasks)     │    │  optional  │    │          │
└─────────────────┘    └───────────────┘    └────────────────┘    └───────────────┘    └────────────┘    └──────────┘
       │                      │                      │                      │                    │                │
       ▼                      ▼                      ▼                      ▼                    ▼                ▼
   [Confirm]              [Confirm]              [Confirm]              [Confirm]            [Confirm]        [Done]
```

**After each step, you MUST**:
1. Output a clear summary of what was completed
2. Use `AskUserQuestion` tool to ask the user whether to proceed

Lite mode skips `tech-design` and `breakdown`. It resumes from the next valid stage based on the last stable completed phase in requirement status.

### Full workflow (`type: 'full'`)

```
init → brainstorm → tech-design → breakdown → openspec → implementation → optional PPE → archive
```

Full mode preserves the existing upstream artifact chain and should continue passing all available upstream documents to downstream stages.

## Core Principles

This skill adheres to the shared workflow principles defined in [references/cspadk-workflow-principles.md](references/cspadk-workflow-principles.md).

Additional principles for lite workflow:
- **Progressive disclosure** - Only show what's needed at each step
- **User control** - Always ask for confirmation before proceeding
- **Fail fast** - Stop and report issues early rather than continuing blindly
- **Sensible defaults** - Make reasonable assumptions for small requirements

**CRITICAL - Mandatory Sub-Skill Delegation**:
This skill is an **orchestrator only**. It MUST NOT implement any phase functionality itself. Instead, it MUST delegate to the appropriate sub-skill for each phase:

| Phase | Required Sub-Skill | Purpose |
|-------|-------------------|---------|
| Init | `/cspadk-requirement-init` | Initialize requirement tracking and branch |
| Brainstorm | `/cspadk-brainstorming` | Clarify requirements and produce design |
| OpenSpec | `/cspadk-openspec-flow` | Generate proposal, spec, and tasks artifacts |
| Implementation | `/cspadk-implementation` | Execute tasks with TDD |
| Archive | `/cspadk-archive` | Finalize and archive completed work |

**Violation of this rule** (e.g., generating designs, writing code, or creating OpenSpec artifacts directly in this skill) is a critical error that breaks workflow traceability and artifact consistency.

## Execution Steps (Pattern: Pipeline)

### Step 1: Read current requirement state

- **Action**: Run `cspadk req-current --json` to determine whether an active requirement already exists
- **Action**: Parse:
  - requirement ID
  - `type: 'lite' | 'full'`
  - `currentPhase`
  - document references
  - module statuses
- **Rule**: Treat `currentPhase` as the last stable completed phase, not the currently running transient step

### Step 2: Handle new requirement vs re-entry

- **If no requirement exists**:
  - Ask the user for a requirement description or Feishu document URL
  - Generate a kebab-case requirement ID
  - Invoke `/cspadk-requirement-init` via `Skill` tool
  - **IMPORTANT**: Set the workflow type to lite:
    ```bash
    cspadk req-update-type lite --json
    ```
  - **IMPORTANT**: For lite workflow, create a default module to track progress:
    ```bash
    cspadk req-module-add main "Main implementation" --json
    ```
  - This requirement will now follow the lite workflow
- **If a requirement exists**:
  - Re-enter from the next valid stage derived from `type` and `currentPhase`
  - Do not restart from `init`

## Module Management CLI Commands

| Action | CLI Command |
|--------|-------------|
| Add a new module | `cspadk req-module-add <id> <title> [--days <days>] --json` |
| Update module status | `cspadk req-module <id> --status <status> --json` |
| Add module document | `cspadk req-module-document <id> <key> <path> --json` |
| Append to document array | `cspadk req-module-document-append <id> <key> <path> --json` |
| Replace all modules | `cspadk req-module-set-all '<json>' --json` |

**Common module status values**: `pending`, `openspec`, `in_progress`, `completed`

### Step 3: Resolve the next stage from workflow type and stable phase

- **Lite workflow routing**:
  - no requirement → init
  - `init` → brainstorming
  - `brainstorm` → openspec
  - `openspec` → implementation
  - `implementation` → deploy decision or archive
  - `deploy` → archive (manual, only after PPE testing is complete; NOT automatic)
  - `archive` / `archived` → report completion
- **Full workflow routing**:
  - `init` → brainstorming
  - `brainstorm` → tech-design
  - `tech-design` → breakdown
  - `breakdown` → openspec
  - `openspec` → implementation
  - `implementation` → deploy decision or archive
  - `deploy` → archive (manual, only after PPE testing is complete; NOT automatic)

### Step 4: Pass the correct upstream context

- **Lite workflow**:
  - downstream OpenSpec and implementation stages MUST use only `documents.design`
  - they MUST NOT require `documents.techDesign` or `documents.breakdown`
- **Full workflow**:
  - continue passing the relevant available upstream artifacts, including `documents.design`, `documents.techDesign`, and `documents.breakdown`

### Step 5: Invoke the next workflow stage (MANDATORY SUB-SKILL DELEGATION)

**CRITICAL**: You MUST invoke the corresponding sub-skill using the `Skill` tool. Do NOT implement any phase functionality directly in this workflow skill.

- **Brainstorm**: invoke `/cspadk-brainstorming` via `Skill` tool
  - This skill handles requirement clarification, design exploration, and spec document creation
  - Do NOT create design documents or clarify requirements yourself
- **OpenSpec**: invoke `/cspadk-openspec-flow` via `Skill` tool
  - This skill generates proposal.md, spec.md, and tasks.md artifacts
  - Do NOT generate OpenSpec artifacts yourself
- **Implementation**: invoke `/cspadk-implementation` via `Skill` tool
  - This skill executes tasks with TDD discipline
  - Do NOT write implementation code yourself
- **Archive**: invoke `/cspadk-archive` via `Skill` tool
  - This skill finalizes and archives completed work
  - Do NOT create archive artifacts yourself

**Sub-skill invocation example**:
```javascript
// Correct: Use Skill tool to delegate
Skill({ skill: "cspadk-brainstorming" })

// WRONG: Implementing functionality directly
// Do NOT do this - write design documents, clarify requirements, etc.
```

### Step 6: Step Completion and User Confirmation (MANDATORY)

**After each major phase completes successfully**, you MUST:

1. **Output a step summary** with the following format:

```
## Step Complete: [Phase Name]

### Summary
[Brief summary of what was accomplished in this step - 2-3 sentences max]

### Artifacts Produced
- [List of files/documents created or modified]

### Next Step
[Brief description of what the next step will do]
```

2. **Use `AskUserQuestion` tool** to ask the user how to proceed:

```javascript
AskUserQuestion({
  questions: [{
    question: "What would you like to do next?",
    header: "Next Step",
    multiSelect: false,
    options: [
      {
        label: "Continue to next step",
        description: "Proceed with /cspadk-continue or [next phase name] - [brief description of what it will do]"
      },
      {
        label: "Adjust current artifacts",
        description: "Make changes to the current output before proceeding - describe what needs adjustment"
      },
      {
        label: "Abort workflow",
        description: "Stop the workflow here and exit"
      }
    ]
  }]
})
```

3. **Handle user response**:
   - If "Continue to next step": Proceed to the next phase
   - If "Adjust current artifacts": Ask user to describe what needs to be changed, then make adjustments and re-confirm
   - If "Abort workflow": Exit the skill gracefully

**Phases requiring confirmation**:
- After `init` → before brainstorming
- After `brainstorm` → before openspec
- After `openspec` → before implementation
- After `implementation` → before PPE (if enabled) or archive

### Step 7: Optional PPE stage after implementation

- **Action**: After implementation, ask the user whether to proceed with PPE deployment using `AskUserQuestion` tool
- **If user declines PPE**:
  - skip PPE and proceed toward archive
- **If user confirms PPE**:
  - load reusable BITS config
  - infer runtime-only values such as branch, lane, and change set
  - show inferred values for confirmation before any mutating command
  - run the BITS command sequence:
    1. `bytedcli bits develop create`
    2. optional `update-lane`
    3. optional `bind-branch`
    4. optional `quick-run`
  - persist successful PPE runtime results and update the stable phase to `deploy`

### Step 8: Stable-state re-entry and failure behavior

- **Action**: Update requirement status only after a stable stage completes successfully
- **Rule**: If a major stage fails, do not advance `currentPhase`
- **Rule**: On the next run, inspect requirement state again and resume from the next valid stage instead of restarting
- **Rule**: Unsupported or inconsistent phases should stop with a clear explanation rather than guessing

## Input Handling

### Single Sentence Input

When user provides a simple description like "add a user login feature":
- Use it directly for brainstorming
- Generate a kebab-case ID: `add-user-login-feature`
- Treat as single-module requirement

### Feishu Document URL

When user provides a Feishu URL:
- Download the document using `lark-cli docs +fetch` command
- Extract key information for brainstorming
- Use document title to generate requirement ID

**Document fetching command**:
```bash
lark-cli docs +fetch --doc "<FEISHU_URL>" > /tmp/doc-raw.json
```

Then parse the JSON and extract the markdown content from `parsed.data.markdown` and the title from `parsed.data.title`.

### Multi-Step Input

When the requirement spans multiple logical modules:
- **Option A** (Recommended): Split into separate lite workflow runs
- **Option B**: Convert to full workflow with `/cspadk-requirement-breakdown`

## Error Handling

| Scenario | Action |
|----------|--------|
| Requirement init fails | Report error, suggest checking branch status |
| Brainstorming unclear | Ask user to clarify, do not proceed |
| OpenSpec generation fails | Show partial results, ask to retry or abort |
| Implementation fails | Stop, report failing task, ask to fix or skip |
| Archive fails | Report error, manual cleanup may be needed |

## Next Step

After completing this skill, the requirement is fully implemented and archived. The user may:
- Start a new lite workflow with `/cspadk-lite-workflow`
- Start a new requirement with `/cspadk-requirement-init`
- Review archived work in `.ttadk/csp-archives/`

## Output Language

Detect the user's input language and respond in the same language.
- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).
