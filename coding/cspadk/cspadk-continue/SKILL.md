---
name: cspadk-continue
description: Automatically proceed to the next workflow phase by invoking the appropriate skill for the current requirement. Supports auto-detection of missing artifacts and resume capability.
version: 1.2.1
metadata:
  patterns:
  - tool-wrapper
  - pipeline
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

# cspadk-continue

## Overview

This skill automatically proceeds to the next workflow phase by checking the current requirement status and invoking the appropriate skill. It enables seamless progression through the CSPADK workflow with auto-detection of missing artifacts and resume capability.

## Output Language

Respond in the same language as the user's request. Default to English for ambiguous cases.

## Trigger Scenarios

- When the user wants to continue working on a requirement
- When the user asks to "continue", "proceed", or "next step"
- When the user wants to advance the workflow
- When the user wants to resume from an interrupted workflow

## Phase → Skill Mapping

| Phase | Skill to Invoke |
|-------|-----------------|
| `init` | `/cspadk-requirement-init` |
| `brainstorm` | `/cspadk-brainstorming` |
| `tech-design` | `/cspadk-tech-design` |
| `breakdown` | `/cspadk-requirement-breakdown` |
| `openspec` | `/cspadk-openspec-flow` |
| `implementation` | `/cspadk-implementation` |
| `deploy` | `/csp-infra-hiworks-deploy` (optional) |
| `archived` | No action (complete) |

## Workflow Transition

`cspadk-continue` checks the current phase, reads the current requirement artifacts and module statuses from requirement status, and then either resumes unfinished work in the current phase or invokes the skill for the next phase. After implementation completes, the user is offered an optional deploy step via `/csp-infra-hiworks-deploy`.

| Current Phase | Next Phase | Skill to Invoke |
|---------------|------------|-----------------|
| (none) | `init` | `/cspadk-requirement-init` |
| `init` | `brainstorm` | `/cspadk-brainstorming` |
| `brainstorm` | `tech-design` | `/cspadk-tech-design` |
| `tech-design` | `breakdown` | `/cspadk-requirement-breakdown` |
| `breakdown` | `openspec` | `/cspadk-openspec-flow` |
| `openspec` | `implementation` | `/cspadk-implementation` |
| `implementation` | `deploy` (optional) OR `archived` | `/csp-infra-hiworks-deploy` (optional) OR archive workflow |
| `deploy` | `archived` (manual, NOT automatic) | User explicitly confirms after PPE testing is complete |
| `archived` | — | No action (complete) |

## Expected Artifacts by Phase

| Phase | Expected Artifacts |
|-------|-------------------|
| `init` | `prd`, `td`, `extraContext` (optional) |
| `brainstorm` | `design` |
| `tech-design` | `techDesign` (required), `techDesignRemote` (optional), `extraContext` (may be updated) |
| `breakdown` | `breakdown` |
| `openspec` | module-level `proposal`, `spec`, `tasks` artifacts in `modules[].documents.*` |
| `implementation` | (module status: completed) |
| `deploy` | BITS dev task created (optional phase) |
| `archived` | `archive` |

## Execution Steps

### Step 0: Ensure Correct Working Directory

Before running any CLI commands, ensure you are in the git repository root:

```bash
git rev-parse --show-toplevel
```

If the current working directory is not the git root, change to the git root first before proceeding. All subsequent CLI commands assume they are run from the repository root.

### Step 1: Check Current Requirement and Read Existing Context

Run the CLI to get the current requirement status:

```bash
cspadk req-current --json
```

Parse the JSON output to determine:
- Whether a requirement exists
- The current phase
- The requirement ID
- The documents and modules
- Which artifacts from previous phases are already available in requirement status
- Which modules are `pending`, `openspec`, `in_progress`, or `completed`

### Step 2: Handle No Requirement Case

If no requirement is found:

```
Error: No requirement for current branch. Run `/cspadk-requirement-init` to create one first.
```

### Step 3: Verify Phase Artifacts

Run the CLI to verify all artifact paths exist:

```bash
cspadk req-verify --json
```

If artifacts are missing, parse the output to identify which artifacts are missing for the current phase. Keep local file-path artifacts as the required workflow contract for `req-verify`; treat remote URLs as optional metadata.

### Step 4: Detect Missing Artifacts

Based on the current phase, check if expected artifacts exist in requirement status:

| Current Phase | Check For |
|---------------|-----------|
| `init` | `documents.prd` OR `documents.td` |
| `brainstorm` | `documents.design` |
| `tech-design` | `documents.techDesign` (required local path), `documents.techDesignRemote` (optional remote URL) |
| `breakdown` | `documents.breakdown` |
| `openspec` | each generated target module has the required `modules[].documents.*` artifacts recorded in requirement status |
| `implementation` | All modules with `status: completed` |

### Step 5: Handle Missing Artifacts

If artifacts are missing, use `AskUserQuestion` with options:

```
Some artifacts are missing for the current phase:
- <artifact-key>: <expected-path>

What would you like to do?
1. Re-run <skill-name> to generate missing artifacts
2. Provide path manually
3. Proceed anyway (may cause errors)
```

**Option 1: Re-run previous skill**
- Invoke the skill that should have produced the missing artifacts
- After completion, return to Step 3

**Option 2: Provide path manually**
- Prompt for each missing artifact path
- Run `cspadk req-update-document <key> <path> --json` for each
- Return to Step 3

**Option 3: Proceed anyway**
- Continue to next phase with a warning
- May cause errors in downstream skills

### Step 6: Resume Unfinished Work or Invoke the Next Skill

Based on the current phase, existing artifacts, and module statuses, choose the correct continuation point:

- **`init`** → Invoke `/cspadk-brainstorming`
- **`brainstorm`** → Invoke `/cspadk-tech-design`
- **`tech-design`** → Invoke `/cspadk-requirement-breakdown`
- **`breakdown`** → Invoke `/cspadk-openspec-flow`
- **`openspec`**:
  - If unfinished modules are detected, use `AskUserQuestion` to let the user select one or more modules to continue with `/cspadk-implementation`
  - Then invoke `/cspadk-implementation` for the selected module(s), following that skill's module-selection rules
- **`implementation`**:
  - If any modules are still `openspec` or `in_progress`, use `AskUserQuestion` to let the user select one or more unfinished modules to continue with `/cspadk-implementation`
  - If any modules are still `pending`, use `AskUserQuestion` to let the user select one or more pending modules to return to `/cspadk-openspec-flow`, or choose unfinished implementation modules instead
  - If all modules are `completed`, use `AskUserQuestion` to ask whether to deploy:
    - **Deploy**: Invoke `/csp-infra-hiworks-deploy`, then after it completes, print the BITS dev task link (constructed as `https://bits.bytedance.net/devops/<spaceId>/develop/detail/<devTaskId>` from the BITS API response), update phase to `deploy`. **Do NOT automatically proceed to archive.** The `deploy` phase is for PPE testing — the user may find bugs or requirement changes that need vibe-coding fixes. Simply inform the user that they can run `/cspadk-continue` again when PPE testing is complete and they are ready to archive.
    - **Skip deployment**: Invoke `/cspadk-archive` immediately
- **`deploy`** → The `deploy` phase is a PPE testing phase. **Do NOT automatically transition to archive.** Print the BITS dev task link (constructed as `https://bits.bytedance.net/devops/<spaceId>/develop/detail/<devTaskId>`), then use `AskUserQuestion` to ask how the user wants to proceed:
  - **Continue to archive**: PPE testing is complete, invoke `/cspadk-archive`
  - **Fix bugs / make changes**: Stay in `deploy` phase. The user can vibe-code fixes, push changes, and re-run `/cspadk-continue` when ready to archive
- **`archived`** → Inform user: "Requirement is already archived. No further actions."

### Step 7: Update Phase After Completion

After the invoked skill completes successfully, update the phase only when the workflow actually advanced to a new phase. If the skill resumed unfinished work within the same phase, do not force a phase transition.

```bash
cspadk req-update-phase <next-phase> --json
```

### Step 8: Print Progress Summary (Always)

**IMPORTANT**: Before exiting, always print a progress summary of the entire requirement. This gives the user visibility into the current state.

Run the CLI to get the current status:

```bash
cspadk req-current --json```

The CLI already outputs a formatted summary with phase, documents, and modules when using `--json`, parse the JSON output and display a human-readable summary. Do not add an extra `AskUserQuestion` when `cspadk-continue` finishes. If the skill has just auto-invoked the next workflow stage, print only the overall progress summary after that stage completes.

## Edge Cases

| Scenario | Behavior |
|----------|----------|
| No requirement found | Error with hint to run `/cspadk-requirement-init` |
| Already `archived` | Info: "Requirement is already archived. No further actions." |
| Skill invocation fails | Report error and halt, do not update phase |
| Missing artifacts detected | Prompt user with options to re-run, provide path, or skip |
| Module not completed in implementation | Prompt the user to select one or more unfinished modules and resume `/cspadk-implementation` instead of suggesting archive |

## Example Usage

```bash
# User invokes the skill
/cspadk-continue

# Skill checks current status
cspadk req-current --json
# Output: { "success": true, "data": { "currentPhase": "implementation", "documents": { ... }, "modules": [ ... ] } }

# Skill verifies artifacts
cspadk req-verify
# Output: All artifact paths are valid

# Skill sees unfinished implementation modules
# Prompt the user to select one or more unfinished modules to continue
AskUserQuestion: select one or more modules with status `openspec` or `in_progress`

# Then resume the current phase for the selected module set
/cspadk-implementation

# Print progress summary before exit
cspadk req-current --json# Output:
# ℹ Requirement: example-requirement
# ℹ Phase: implementation
# ℹ Documents:
#   prd: .cspadk/docs/prd.md
#   ...
# ℹ Modules:
#   ✔ Module A (completed)
#   ► Module B (in_progress)
#   ☐ Module C (pending)
```

### Example: All Modules Completed

```bash
# User invokes the skill
/cspadk-continue

# Skill checks current status
cspadk req-current --json
# Output: { "currentPhase": "implementation", "modules": [{ "id": "module-a", "status": "completed" }, { "id": "module-b", "status": "completed" }] }

# Skill verifies artifacts
cspadk req-verify
# Output: All artifact paths are valid

# Skill sees implementation is fully complete and asks about deploy
AskUserQuestion: "All modules are completed. Would you like to create a BITS dev task for deployment?"
  - Deploy → /csp-infra-hiworks-deploy
  - Skip deployment → /cspadk-archive

# If user chose Deploy:
/csp-infra-hiworks-deploy

# After deploy completes, print the BITS dev task link:
# "BITS dev task created: https://bits.bytedance.net/devops/<spaceId>/develop/detail/<devTaskId>"
# Update phase to 'deploy'
cspadk req-update-phase deploy --json

# Ask how the user wants to proceed (deploy is a PPE testing phase, NOT auto-archive)
AskUserQuestion: "PPE dev task created. What would you like to do next?"
  - Continue to archive → /cspadk-archive (PPE testing is complete)
  - Fix bugs / make changes → Stay in deploy phase (vibe-code fixes, push, re-run /cspadk-continue when ready)

# After archive completes, print the overall progress summary only
cspadk req-current --json```

## Example: Missing Artifacts

```bash
# User invokes the skill
/cspadk-continue

# Skill checks current status
cspadk req-current --json
# Output: { "currentPhase": "implementation", "modules": [{ "id": "module-a", "status": "completed" }, { "id": "module-b", "status": "openspec" }] }

# Skill detects unfinished work in the current phase
# Completed: module-a
# Remaining: module-b

# Skill should resume implementation, not jump to archive
/cspadk-implementation

# Print progress summary before exit
cspadk req-current --json# Next: Continue with remaining modules or run /cspadk-archive when all are completed.
```

## Dependencies

- `@cspadk/cli` must be installed and available in PATH
- Current directory must be a git repository
- Must be on a branch with an associated requirement
