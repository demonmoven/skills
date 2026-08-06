---
name: csp-infra-hiworks-deploy
description: "Operate BITS DevOps platform via cspadk hiworks: create dev tasks"
version: 1.4.3
metadata:
  patterns:
    - pipeline
    - tool-wrapper
  domain: csp-infra
  i18n_level: 0
  prompt_version: "1.4.3"
  agent_support:
    - claude-code
    - trae
    - trae-cli
    - coco
  language:
    - en
---

# HiWorks: BITS Dev Task & Release Ticket Manager

Create BITS/HiWorks DevOps tasks (especially for multi-service or multi-branch changes) via `cspadk hiworks bits-task-create`, and bind Train release tickets.

## When to Use

- User wants to create a BITS dev task for deployment
- User mentions "create dev task", "hiworks", "bits task", "PPE deploy", or "开发任务"
- Working with multi-service or multi-branch changes that need a BITS workflow
- User wants to bind a Train release ticket to an existing dev task ("bind release ticket", "发布单关联")

## Input

| Input | Required | Example |
|-------|----------|---------|
| `changeName` | No | reads from user input or context from `openspec/changes/<changeName>` |
| User-provided parameters | No | Any `--flag value` pairs overriding defaults |

If `changeName` is provided, prefer reading task/deployment context from that change directory. Otherwise infer parameters from the project defaults.



## Workflow (Pattern: Pipeline)

```
Step 1: Read config ──► Step 2: Uncommitted changes ──► Step 3: Auth check ──► Step 4: Collect params ──► Step 5: Confirm ──► Step 6: Execute
                                                    │                        │                                         ▲
                                                    └── commit if needed ───┘                         (modify & re-confirm)
```


## Step 1: Read Configuration Defaults

- **Action**: Read `.ttadk/csp-config.json` to load default BITS configuration values.
- **Rule**: If the file does not exist, proceed with CLI defaults. Do not block the workflow.
- **Rule**: Store the resolved config for use in Step 4 parameter resolution.
- **Rule**: Read `bits.apps` map to resolve per-app `serviceType` and `services`. The top-level `bits.serviceType` and `bits.services` fields are deprecated — always use `bits.apps[<appName>].serviceType` and `bits.apps[<appName>].services`.
- **Rule**: Determine involved apps from the current requirement status `bits.apps` array. If `bits.apps` is not set in the requirement status, use all apps from `csp-config.json` `bits.apps`. If `bits.apps` is empty `{}`, report error: "No apps configured. Run init-infra first."
- **Rule**: For each involved app, create a separate BITS dev task.

Key config fields (see `references/parameters.md` for full mapping):

| Category | Config keys | Required |
|----------|-------------|----------|
| Site | `bits.cloudSite` | Yes |
| Space | `bits.spaceId` | Yes (frontend: `35997699842`, backend: `201141148930`) |
| Task | `bits.enableLanes` | Yes |
| App Service | `bits.apps[<name>].serviceType`, `bits.apps[<name>].services` | Yes (per app) |
| Template | `bits.devTaskTemplateId` | Yes — mutually exclusive with `teamFlowId` (frontend: `29396`, backend: `31265`) |
| Template | `bits.teamFlowId` | Optional — mutually exclusive with `devTaskTemplateId` |
| All others | — | Optional — omit if not set |

## Step 2: Check Uncommitted Changes

Before creating a BITS dev task, check if the current branch has uncommitted changes that should be pushed first.

- **Action**: Run `git status --porcelain` to detect uncommitted changes.
- **Rule**: If no uncommitted changes exist → proceed to Step 3.
- **Rule**: If uncommitted changes exist, use `AskUserQuestion` to ask the user:

> "There are uncommitted changes on the current branch. Would you like to commit and push them before creating the BITS dev task?"

Options:
- **Yes, commit and push** — Stage all non-sensitive files, commit with a descriptive message, and push to remote. Then proceed to Step 3.
- **No, continue without committing** — Proceed to Step 3 without committing. The BITS dev task will reference the current branch state.
- **Cancel** — Abort the task creation.

**When the user chooses to commit and push:**

1. **Review changes**: Run `git diff --stat` and `git status` to list pending files. Warn the user about sensitive files (`.env`, `credentials`, tokens) and exclude them.
2. **Stage and commit**: Stage ALL changed files (excluding sensitive files) with `git add` for each non-sensitive file, then commit with a message derived from the change context or requirement name. Do NOT leave any non-sensitive changes unstaged — the commit must include all pending changes so the BITS dev task references the complete branch state.
3. **Push**: Run `git push -u origin <current-branch>` to push the commit.
4. **Verify**: Run `git status` to confirm the working tree is clean, then proceed to Step 3.

## Step 3: Authentication Check

Before any write operation, verify the user is authenticated with the target ByteCloud site. Follow the priority chain below:

## 2a. Determine the target site

- Read `bits.cloudSite` from config (default: `i18n-tt`).
- If `cloudSite` is `i18n-tt`, the target SSO is TikTok (isolated from ByteDance SSO).
- If `cloudSite` is `prod`, `i18n-bd`, or `boei18n`, the target SSO is ByteDance (shared session).

## 2b. Check authentication status

```bash
# For default site (prod)
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest auth status

# For a specific site (e.g. i18n-tt)
BYTEDCLI_CLOUD_SITE=i18n-tt NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest auth status
```

- If the output shows the user is authenticated → proceed to Step 4.
- If the output contains `Not authenticated` → proceed to 3c.

### 3c. Guide the user to login

Use `AskUserQuestion` to ask the user to complete authentication:

> "You are not authenticated with the target site (`<site>`). Please log in before proceeding."

Options:
- **I've logged in** — Re-check auth status, then proceed to Step 4.
- **Help me login** — Show the login command and guide the user through the process.
- **Cancel** — Abort the task creation.

**Login commands by site:**

```bash
# Default site (prod)
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest auth login

# i18n-tt (TikTok SSO — requires separate login)
BYTEDCLI_CLOUD_SITE=i18n-tt NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest auth login

# i18n-bd (usually shares session with prod)
BYTEDCLI_CLOUD_SITE=i18n-bd NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest auth login
```

**Login tips to share with the user:**
- `auth login` defaults to terminal QR code scan.
- Add `--session --auto` to let the CLI automatically pick the best login path (browser cookie → QR → interactive browser).
- For headless/remote environments, use `--json auth login --begin` to get a QR image, then `--json auth login --complete <token>` to verify.
- After login, re-run `auth status` to confirm before continuing.

**Rule**: If Meego is used, also check Meego auth:

```bash
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest meego status
```

If not authenticated, guide login:

```bash
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest meego login
```

## Step 4: Collect and Resolve Parameters

Collect all parameters following the priority order defined in `references/parameters.md`:

1. **User input** — highest priority
2. **Change context** — from `openspec/changes/<changeName>/` and `requirement-status/<name>.json`
3. **Repository docs** — `AGENTS.md`, `deploy.md`
4. **csp-config.json defaults** — from Step 1
5. **Local runtime data** — `git branch --show-current`, etc.

**Rule**: For `--title`, infer from the requirement name or change context. Priority: user input → `changeName` → requirement `id` from status file. Convert kebab-case to human-readable title (e.g. `add-user-auth` → "Add user auth").
**Rule**: For `--scm-branch`, infer from `git branch --show-current` when not explicitly provided.
**Rule**: For `--lane`, infer from context in this priority: user input → `bits.lane` from config (if explicitly set) → swim lane from tech design document → branch name heuristic. **Tech design swim lane**: read the `# Swimlane` / `# 泳道` section from the requirement's TD file (path from `documents.techDesign` in the requirement status). Extract the `feature_slug` from any swim lane entry (e.g. `ppe_schedule_import_export` → `schedule_import_export`), stripping the `boe_` or `ppe_` prefix. Branch name heuristic: if the branch name starts with `feat/` or `fix/`, default lane is `test`; if it contains `hotfix`, default lane is `ppe`. If still unresolved, default to `test`.
**Rule**: For `--meego`, first check `documents.meegoLink` from the current requirement status file (`.ttadk/requirements-status/<id>.json`). If present, use its value as `--meego`. User input and config override this.
**Rule**: `--team-flow-id` and `--dev-task-template-id` are mutually exclusive. Use one or the other, never both. When `bits.teamFlowId` is set in config, use `--team-flow-id` and omit `--dev-task-template-id`. Otherwise use `--dev-task-template-id` from config.
**Rule**: All parameters not listed as required in Step 1 are optional. Only include optional flags (`--title`, `--change`, `--scm-mode`, `--scm-branch`, `--from-dev-id`, `--developer`, `--qa`, `--meego`, `--idcs`, `--var`, `--env-setting-map-json`, `--dev-task-mode`, `--workflow-snapshot-id`, `--api-base-url`, `--jwt-token`) in the CLI command when the user explicitly provides them (via input or config) or when auto-inferred. If not specified and not inferable, omit these flags entirely.
**Rule**: When required parameters are still missing after all inference, present the user with a CLI snippet using `<FILL_ME>` placeholders instead of asking plain-text questions.



## Step 5: Present Parameters and Confirm

### Capability 1: Create Dev Task
After all parameters are resolved:

1. **For each involved app**, show parameter summary — Present a human-readable table grouped by category, with the source of each value. For multi-app scenarios, show one table per app:

```
### App: <app-name> (path: <app-path>)

| Category | Parameter | Value | Source |
|----------|-----------|-------|--------|
| Task     | --title   | "Fix login issue" | User input or generated from change name(requirement name) |
| Task     | --lane    | schedule_import_export | TD swim lane (ppe_schedule_import_export → slug) |
| Service  | --service-type | PROJECT_TYPE_WEB | bits.apps[<app-name>].serviceType |
| Service  | --services | psm.web.svc | bits.apps[<app-name>].services |
| SCM      | --scm-branch | fix/login | git branch |
| ...      | ...       | ... | ... |
```

2. **For each involved app**, show the complete CLI command — Build the full command with the invocation prefix and `--json`:

```bash
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @cspadk/cli@latest hiworks bits-task-create \
  --cloud-site i18n-tt \
  --space-id 201141148930 \
  --service-type PROJECT_TYPE_TCE \
  --enable-lanes both \
  --lane test \
  --services ies.kefu.schedule_service_i18n \
  --dev-task-template-id 31265 \
  --json
```

> **Note**: Only required parameters are shown above. Optional parameters (`--title`, `--scm-branch`, `--meego`, `--developer`, `--qa`, `--idcs`, `--scm-mode`, etc.) are only included when explicitly provided by the user or config. When `--team-flow-id` is provided, omit `--dev-task-template-id` (they are mutually exclusive).

3. **Ask for confirmation** — Use `AskUserQuestion` with the question "Create BITS dev task with the above parameters?" and options:
   - **Yes, create now** — Execute the command immediately (proceed to Step 6).
   - **I want to modify some parameters** — Let the user specify which parameters to change, update them, and re-present this step.
   - **No, cancel** — Abort the task creation.

## Step 6: Execute and Handle Result

### Capability 1: Create Dev Task

- **Action**: Run the confirmed CLI command(s) via bash. For multi-app scenarios, execute one command per app sequentially.
- **Rule**: Parse the JSON output to determine success or failure for each app's task.
- **On success**: Construct the BITS dev task link from the response and present it with key details (lane, services, IDCs, SCM branch). For multi-app, present a summary of all created tasks.

**Constructing the dev task link**: The BITS dev task URL follows the pattern `https://bits.bytedance.net/devops/<spaceId>/develop/detail/<devTaskId>`, where `<spaceId>` is the value passed to `--space-id` and `<devTaskId>` is the task ID returned in the JSON response (`data.id` or `data.devTaskId`). Example: `https://bits.bytedance.net/devops/35997699842/develop/detail/2319568`.
- **Action**: After successfully creating a BITS dev task, persist the dev task link to the current requirement status by running `cspadk req-update-dev-task-link <devTaskLink> --json`. This writes the link to `bits.devTaskLink` in the requirement status file.
- **Rule**: Only update if the requirement status file exists. If no requirement is active, skip the update and still show the link to the user.
- **On failure**: Show the error message and hint from the JSON output. Reference `references/troubleshooting.md` for common error recovery.

**Common failure patterns:**

| Error | Cause | Fix |
|-------|-------|-----|
| `Not authenticated` / `AUTH_REQUIRED` | Not logged in or token expired | Return to Step 3 |
| `401` on specific site | Site SSO not authenticated | Login to target site (Step 3c) |
| `BITS_INPUT_ERROR` | Missing required parameters | Return to Step 4 to collect missing params |
| Network error | No intranet access | Check VPN / network connectivity |

### Capability 2: Bind Train Release Ticket

If the user wants to bind a Train release ticket to an existing dev task (or directly after creating one):

1. **List bindable tickets**: Run `NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @cspadk/cli@latest cspadk hiworks bits-release-ticket-find <taskId> --json` (using the provided dev task ID or the one obtained in Step 6). Alternatively, use `--meego-id <meegoId>` if the task ID is unknown. This will return a JSON list of bindable Train release tickets in `bindableTickets` and related task(s) in `tasks`.
2. **Handle dev task selection (meego-id only)**: If `--meego-id` was used, use `AskUserQuestion` to ask the user which dev task they want to use, **even if only one task was found**. This allows the user to verify the correct task is selected before proceeding.
3. **Handle empty list**: If no bindable tickets are found, inform the user and conclude the task.
4. **Ask for selection**: If bindable tickets exist, use the `AskUserQuestion` tool to present the options to the user. Ask them "Which Train release ticket would you like to bind to this development task?". The options should include the ticket name, integration ID, and creation date.
5. **Execute binding**: Once the user selects a ticket, run the specific bind CLI command:
   ```bash
   NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @cspadk/cli@latest cspadk hiworks bits-release-ticket-bind --task-id <taskId> --integration-id <selectedIntegrationId> --json
   ```
6. **Report bind result**: Show the bind result to the user.
7. **Read config for reviewers**: Read `.ttadk/csp-config.json` to get recent reviewer values from `bits.reviewerEmails` (an array of strings).
8. **Ask for MR Reviewer**: Use `AskUserQuestion` to ask the user for the MR reviewer's email. Provide the historical values from `bits.reviewerEmails` as priority options, and allow the user to input a new email manually.
9. **Save manual input**: If the user inputs a new email manually, prepend it to the `bits.reviewerEmails` array in `.ttadk/csp-config.json`. Deduplicate the array and keep a maximum length of 5.
10. **Invite MR Reviewers**: Run the CLI command to invite the reviewer:
   ```bash
   NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @cspadk/cli@latest cspadk hiworks bits-mr-reviewer-invite --task-id <taskId> --reviewer-email <selectedEmail> --json
   ```
11. **Report invitation result**: Show the reviewer invitation result to the user.

## Quick Start

### Create a dev task

```bash
# Minimal: only required parameters
cspadk hiworks bits-task-create \
  --cloud-site i18n-tt \
  --space-id 201141148930 \
  --service-type PROJECT_TYPE_TCE \
  --enable-lanes both \
  --lane test \
  --services ies.kefu.schedule_service_i18n \
  --dev-task-template-id 31265

# With optional parameters
cspadk hiworks bits-task-create \
  --cloud-site i18n-tt \
  --space-id 201141148930 \
  --service-type PROJECT_TYPE_TCE \
  --enable-lanes both \
  --lane test \
  --services ies.kefu.schedule_service_i18n \
  --dev-task-template-id 31265 \
  --title "Fix login issue" \
  --scm-branch fix/login \
  --meego "https://meego.feishu.cn/xxx/issue/detail/123"

# With --meego from requirement status (documents.meegoLink)
# The agent reads .ttadk/requirements-status/<id>.json → documents.meegoLink automatically in Step 3
cspadk hiworks bits-task-create \
  --cloud-site i18n-tt \
  --space-id 201141148930 \
  --service-type PROJECT_TYPE_TCE \
  --enable-lanes both \
  --lane test \
  --services ies.kefu.schedule_service_i18n \
  --dev-task-template-id 31265 \
  --meego "https://meego.feishu.cn/xxx/issue/detail/123"

# Using --team-flow-id instead of --dev-task-template-id (mutually exclusive)
cspadk hiworks bits-task-create \
  --cloud-site i18n-tt \
  --space-id 201141148930 \
  --service-type PROJECT_TYPE_TCE \
  --enable-lanes both \
  --lane test \
  --services ies.kefu.schedule_service_i18n \
  --team-flow-id 1234567890

# Frontend project (PROJECT_TYPE_WEB) with meego
cspadk hiworks bits-task-create \
  --cloud-site i18n-tt \
  --space-id 35997699842 \
  --service-type PROJECT_TYPE_WEB \
  --enable-lanes both \
  --lane test \
  --services my-frontend-app \
  --dev-task-template-id 29396 \
  --meego "https://meego.feishu.cn/xxx/issue/detail/456"
```

### Bind Train Release Ticket

```bash
# 1. First query bindable tickets for a task
cspadk hiworks bits-release-ticket-find 1234567 --json

# 1a. Or query bindable tickets using meego-id
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @cspadk/cli@latest hiworks bits-release-ticket-find --meego-id 1234567 --json

# 2. Then bind the selected ticket
cspadk hiworks bits-release-ticket-bind --task-id 1234567 --integration-id 9876543 --json

# 3. Invite MR reviewers
cspadk hiworks bits-mr-reviewer-invite --task-id 1234567 --reviewer-email user@example.com --json
```

### Meego link resolution

The `--meego` value is resolved in this priority order:

1. **User input** — explicitly provided by the user in the conversation
2. **Requirement status** — read from `.ttadk/requirements-status/<id>.json` → `documents.meegoLink` (set during `cspadk req-init --meego-link <url>`)
3. **Config default** — `bits.meego` from `.ttadk/csp-config.json`

When `--meego` is provided, BITS associates the task with the Meego work item.

## Notes

- Add `--json` for structured output.
- `--change` format: `service=<PSM>,branch=<sourceBranch>[,target=<targetBranch>]`.
- `--var` can be repeated and uses `name=value`.
- `--env-setting-map-json` overrides the `envSettingMap` parameter in the create API.
- Supported `--service-type`: `PROJECT_TYPE_WEB`, `PROJECT_TYPE_TCE`, `PROJECT_TYPE_FAAS`, `PROJECT_TYPE_HYBRID`, `PROJECT_TYPE_CRONJOB`, `PROJECT_TYPE_CUSTOM`. When using `PROJECT_TYPE_WEB`, service names are parsed into `projectUniqueId` automatically.
- `--title` and `--meego` are independent optional parameters.
- `--team-flow-id` and `--dev-task-template-id` are mutually exclusive. Use one or the other, never both.
- All parameters besides `--cloud-site`, `--space-id`, `--dev-task-template-id` (or `--team-flow-id`), `--service-type`, `--enable-lanes`, `--lane`, and `--services` are optional. Only include optional flags when explicitly provided by the user or config.

## Common Pitfalls

1. **Skipping auth check before creating a task**. BITS API calls require a valid JWT token. If the token is expired or the wrong site is authenticated, the create call will fail with 401. Always check auth status first (Step 2).
2. **Wrong `cloudSite` SSO isolation**. `i18n-tt` (TikTok SSO) is isolated from `prod/i18n-bd/boei18n` (ByteDance SSO). Being logged in to `prod` does NOT mean you are authenticated for `i18n-tt`. Check the target site specifically.
3. **Using `--team-flow-id` and `--dev-task-template-id` together**. These two parameters are mutually exclusive. `--dev-task-template-id` specifies a workflow template directly; `--team-flow-id` resolves the template via team flow. Using both causes undefined behavior — pick one.
4. **Including optional parameters with empty values**. Never pass `--qa ""` or `--idcs ""`. If the user hasn't provided a value, simply omit the flag entirely.

## References

- `references/invocation.md`
- `references/parameters.md`
- `references/troubleshooting.md`

## Next Step

After completing this skill:

- **Recommended**: Use `/cspadk-continue` to automatically proceed to the next workflow phase (archive).
- **Manual**: Use `/cspadk-archive` to archive the completed requirement.

Update the requirement phase via CLI before proceeding.

## Output Language

Detect the user's input language and respond in the same language.
- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).
