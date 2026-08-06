---
name: cspadk-requirement-init
description: Initializes new requirements in the CSPADK repository using CLI commands only. NEVER implements real code - only sets up configuration tracking.
version: 1.4.2
metadata:
  patterns:
    - pipeline
  domain: cspadk
  i18n_level: 0
  prompt_version: "1.4.2"
  agent_support:
    - claude-code
    - trae
    - trae-cli
    - coco
  language:
    - en
---

# cspadk-requirement-init

## Overview

You are responsible for initializing a new requirement in the CSPADK repository. This skill handles branch management and requirements status tracking initialization using CLI commands.

**IMPORTANT**: This skill ONLY initializes requirement configuration. You MUST NOT write any implementation code, create files in `src/`, or modify business logic. Your role is strictly limited to:

- Setting up the requirement status tracking
- Managing branches
- Downloading PRD/TD documents (if URLs provided)

## Trigger Scenarios

- When the user indicates the need to "start a new requirement", "initialize a requirement", or "develop a new feature".

## Inputs

- A brief description of the requirement, or the URL of the Feishu PRD/Technical Design document.

## Outputs

- The locally initialized `.ttadk/requirements-status/<name>.json` status file.
- The requirement documents downloaded to the `.cspadk/docs` directory first (if URLs provided), then recorded into requirement status as local `documents.prd` / `documents.td` paths.
- The newly created and switched requirement branch.

<br />

## Requirement Status Tracking

This skill initializes the requirement status via CLI commands only:

| Field           | Value        | CLI Command                                                   |
| --------------- | ------------ | ------------------------------------------------------------- |
| Initialize      | `<name>`     | `cspadk req-init <name> --json --no-branch`                   |
| `documents.meegoLink` | `<url>` | `cspadk req-init <name> --json --no-branch --meego-link <url>` |
| `currentPhase`  | `init`        | `cspadk req-update-phase init --json --req <name>`                  |
| `documents.prd` | `<prd-path>`  | `cspadk req-update-document prd <path> --json --req <name>`         |
| `documents.td`  | `<td-path>`   | `cspadk req-update-document td <path> --json --req <name>`          |
| `documents.prdRemote` | `<prd-url>` | `cspadk req-update-document prdRemote <url> --json --req <name>` |
| `documents.tdRemote`  | `<td-url>`  | `cspadk req-update-document tdRemote <url> --json --req <name>`  |
| `extraContext[]` | `{ path, label, remote? }` | `cspadk req-extra-context-add <path> --label <label> [--remote <url>] --json --req <name>` |

**Important**: When using `--no-branch`, you MUST use `--req <name>` with update commands to specify which requirement to update, since the branch name won't match the requirement ID.

**Status file location**: `.ttadk/requirements-status/<requirement-name>.json`

**Initial status schema**

```json
{
  "updatedAt": "<iso-timestamp>",
  "currentPhase": "init",
  "branch": "feat/<requirement-name>",
  "documents": {
    "meegoLink": "<meego-work-item-url>",
    "prd": ".cspadk/docs/prd-<name>.md",
    "td": ".cspadk/docs/td-<name>.md",
    "prdRemote": "<original-feishu-prd-url>",
    "tdRemote": "<original-feishu-td-url>"
  },
  "extraContext": [
    { "path": ".cspadk/docs/backend-td-<name>.md", "label": "Backend Tech Design", "remote": "<original-feishu-url>" }
  ],
  "modules": []
}
```

**Note**: The `extraContext` array stores additional context documents (beyond PRD/TD) such as backend tech designs, data warehouse designs, infrastructure plans, etc. Each entry has a `path` (local file), `label` (human-readable description), and optional `remote` (original URL).

**Note**: The `branch` field stores the original branch name associated with this requirement. When using `--no-branch`, this field will be omitted.

## Execution Steps (Pattern: Pipeline)

Follow these steps precisely:

**Interaction rule**: Use the `AskUserQuestion` tool whenever the user must make a bounded choice, confirm an inferred value, or choose between workflow branches. Use normal conversational prompts only for open-ended inputs such as requirement descriptions or document URLs.

### Step 1: Check CLI Availability

- **Action**: Check if `cspadk` CLI is available
  ```bash
  cspadk --version
  ```
- **Action**: If not available, install it globally
  ```bash
  npm install -g @cspadk/cli --registry https://bnpm.byted.org
  ```

### Step 2: Download Documents (Optional)

If the user provides Feishu/Lark document URLs (PRD, Technical Design, or other context documents), download them using lark-cli:

1. **Ensure the docs directory exists**:
   ```bash
   mkdir -p .cspadk/docs
   ```

2. **Fetch each Feishu document**:
   ```bash
   lark-cli docs +fetch --doc "<FEISHU_URL>" > /tmp/doc-raw.json
   ```

3. **Parse and save the markdown**:
   ```javascript
   const fs = require('fs');
   const raw = fs.readFileSync('/tmp/doc-raw.json', 'utf8');
   const parsed = JSON.parse(raw);
   const markdown = parsed.data.markdown;
   const title = parsed.data.title;
   fs.writeFileSync('.cspadk/docs/prd-<name>.md', markdown, 'utf8');
   ```

4. **Repeat for all provided documents** (PRD, TD, etc.), saving each with an appropriate name:
   - PRD: `.cspadk/docs/prd-<name>.md`
   - Technical Design: `.cspadk/docs/td-<name>.md`
   - Other context documents (backend TD, data warehouse TD, infrastructure plans, etc.): `.cspadk/docs/<descriptive-name>.md`

**Important**:
- Download to `.cspadk/docs` first, then update requirement status with those local file paths.
- Preserve the original Feishu/Lark source links in requirement status as `documents.prdRemote` / `documents.tdRemote` when the user provided them.
- All following workflow skills must read PRD/TD from `.ttadk/requirements-status/<name>.json` (`documents.prd` / `documents.td`), not from the original remote Feishu URLs.
- For documents that are NOT the primary PRD/TD (e.g., backend tech design, data warehouse design, API specs from other teams), store them in the `extraContext` array rather than `documents`. Use `cspadk req-extra-context-add` to register them.
- Only download documents. Do NOT create or modify any source code files.

### Step 3: Infer Requirement Details

- **Action**: Ask the user for the requirement description if the user did not provide a Feishu PRD/Technical Design document URL either.
- **Action**: Automatically infer the requirement `<name>` from the user's input:
  - Extract 2-4 keywords from the description or document's content.
  - Convert to kebab-case (lowercase, hyphen-separated)
  - Example: "Add user authentication feature" → `add-user-auth`
  - Example: "Fix memory leak in data processing" → `fix-memory-leak`
- **Action**: Use the `AskUserQuestion` tool to let the user choose whether to:
  - Accept the inferred requirement name
  - Provide a custom requirement name
- **Action**: If the user chooses a custom requirement name, ask for the custom value and continue with that name.
- **Action**: Detect document URLs (Feishu/Lark links) in the user's input and extract them for document download.
- **Action**: Detect Meego work item URLs in the user's input (e.g. `https://meego.feishu.cn/xxx/issue/detail/123` or standalone numeric IDs). If found, store as `meegoLink` for use in Step 4. If not found, ask the user if they have a Meego ticket for this requirement.

### Step 4: Initialize Tracking via CLI

**For AI/Non-Interactive Mode (Recommended)**:

Use the CLI variant that matches the branch choice already collected from the user.

If the user chose no branch (append `--meego-link <url>` if a Meego link was collected in Step 3):
```bash
cspadk req-init <name> --json --no-branch [--meego-link <meego-url>]
```

If the user chose the default branch `feat/<name>`:
```bash
cspadk req-init <name> --json --branch feat/<name> [--meego-link <meego-url>]
```

If the user chose a custom branch name:
```bash
cspadk req-init <name> --json --branch <branch-name> [--meego-link <meego-url>]
```

For overwriting an existing requirement:
```bash
cspadk req-init <name> --json --no-branch --force [--meego-link <meego-url>]
```

**JSON Response Format**:
```json
{
  "success": true,
  "data": {
    "id": "<name>",
    "branch": null,
    "currentPhase": "init"
  }
}
```

### Step 5: Branch Management (Optional)

- **Action**: Use the `AskUserQuestion` tool to ask whether to:
  - Skip branch creation
  - Create the default branch `feat/<name>`
  - Provide a custom branch name
- **Action**: If the user chooses the default branch, use:
  ```bash
  cspadk req-init <name> --json --branch feat/<name> [--meego-link <meego-url>]
  ```
- **Action**: If the user provides a custom branch name, use:
  ```bash
  cspadk req-init <name> --json --branch <branch-name> [--meego-link <meego-url>]
  ```

### Step 6: Update Documents Reference (if documents downloaded)

- **Action**: After the local files are written into `.cspadk/docs`, update requirement status via CLI for each downloaded document (use `--req <name>` when using `--no-branch`):
  ```bash
  cspadk req-update-document prd .cspadk/docs/prd-<name>.md --json --req <name>
  cspadk req-update-document td .cspadk/docs/td-<name>.md --json --req <name>
  ```
- **Action**: If the original PRD/TD Feishu links were provided by the user, also record the remote source links for provenance:
  ```bash
  cspadk req-update-document prdRemote <original-prd-url> --json --req <name>
  cspadk req-update-document tdRemote <original-td-url> --json --req <name>
  ```
- **Action**: For any extra context documents (backend TD, data warehouse TD, etc.) that were downloaded, register them in the `extraContext` array:
  ```bash
  cspadk req-extra-context-add .cspadk/docs/<descriptive-name>.md --label "Backend Tech Design" --remote <original-url> --json --req <name>
  ```

### Step 7: Infer Involved Sub-applications

- **Action**: Read `.ttadk/csp-config.json` `bits.apps` to get the list of available sub-applications. You can also use `cspadk config-list --json` and read the `local.bits.apps` field from the JSON response.
- **Rule**: If `bits.apps` has only one entry, auto-select it without prompting.
- **Rule**: If `bits.apps` has multiple entries, analyze the requirement description to infer which sub-applications are involved. Use `AskUserQuestion` to let the user confirm which apps are involved.
- **Rule**: If `bits.apps` is empty or not present, skip this step.
- **Action**: Write the selected apps to requirement status:
  ```bash
  cspadk req-update-bits --apps "web-app,server-app" --json --req <name>
  ```

### Step 8: Update Phase

- **Action**: Update the phase to `init` (use `--req <name>` when using `--no-branch`):
  ```bash
  cspadk req-update-phase init --json --req <name>
  ```

## AI Compatibility

All CLI commands support `--json` flag for non-interactive, machine-readable output:

| Command | --json Support | Notes |
|---------|---------------|-------|
| `req-init` | ✅ | Use `--no-branch` or `--branch` to avoid prompts |
| `req-update-phase` | ✅ | |
| `req-update-document` | ✅ | |
| `req-extra-context-add` | ✅ | |
| `req-update-bits` | ✅ | `--apps <comma-separated-names>` |
| `req-current` | ✅ | |
| `req-list` | ✅ | |
| `req-status` | ✅ | Takes positional `<name>` arg (NOT `--req`): `cspadk req-status <name> --json` |
| `req-verify` | ✅ | |

## Error Handling

All commands return JSON with `success` field:

**Success Response**:
```json
{
  "success": true,
  "data": { ... }
}
```

**Error Response**:
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message",
    "hint": "Optional hint for resolution"
  }
}
```

## What You MUST NOT Do

- **NEVER** write implementation code (no `src/` files)
- **NEVER** create business logic, functions, or classes
- **NEVER** modify existing code files
- **NEVER** generate test files
- **NEVER** create OpenSpec proposals or designs yourself (these come from separate workflow steps)

Your responsibility is **strictly limited to**:

- Requirement configuration via CLI
- Branch management
- Document downloading (if URLs provided)

## Core Principles

This skill adheres to the shared workflow principles defined in [references/cspadk-workflow-principles.md](references/cspadk-workflow-principles.md).

## Next Step

After completing this skill:

- **Recommended**: Use `/cspadk-continue` to automatically proceed to the next workflow phase (brainstorming).
- **Manual**: Use `/cspadk-brainstorming` to clarify requirements and design.

Update the requirement phase via CLI before proceeding.

## Output Language

Detect the user's input language and respond in the same language.

- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).
