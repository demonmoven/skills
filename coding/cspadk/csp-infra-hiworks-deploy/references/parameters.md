# Task parameter collection and resolution (Parameter specification)

This document defines how the agent should collect a complete and accurate set of **core parameters** before creating or updating a BITS/HiWorks dev task, including but not limited to: target lane/environment, change description, branch information, service list, and any custom variables.

## 1. Priority order

When inferring or collecting any parameter, the agent **must** follow the priority order below (from high to low). Once a parameter is provided or confirmed by a higher-priority source, do not overwrite it with lower-priority values.

1. **User input**
   - Highest priority. Any parameter explicitly specified, modified, or overridden by the user in the current conversation (for example: "set title", "use target branch master").
2. **Change context**
   - If `changeName` is provided, it maps to `*/openspec/changes/<changeName>` (for example: `*/openspec/changes/add-approval-config`).
   - The agent should proactively read relevant files under that directory (configs, change logs, context docs, etc.) and extract task/deployment-related parameters.
   - Also read the current requirement status file (`.ttadk/requirements-status/<id>.json`). If `documents.meegoLink` is present, use it as the `--meego` value (can be overridden by user input or config).
3. **Repository docs (fallback)**
   - Read and parse `AGENTS.md` and `deploy.md` (if present) under the project directory.
   - Example: extract the parameters defined therein, such as team-flow-id, services, service-type, etc.
4. **csp-config.json defaults**
   - Read `.ttadk/csp-config.json` for default BITS configuration values.
   - Example: `bits.lane`, `bits.serviceType`, `bits.fromDevId`, etc.
5. **Local runtime data**
   - Use local shell commands to derive the current repository/system state.
   - Example: to get the current `branch`, run `git branch --show-current` (or equivalent) instead of asking the user to type it.

## 2. Configuration File

The `.ttadk/csp-config.json` file contains BITS configuration defaults (all fields align with `hiworks bits-task-create` CLI options). Fields marked **(optional)** should be omitted from the CLI command when not explicitly provided by the user or config — the BITS API will apply its own defaults.

## 3. Parameter Mapping

**Required parameters** — always include in the CLI command:

| CLI Parameter            | csp-config.json Key       | Description                                                                                                |
| ------------------------ | ------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `--cloud-site`           | `bits.cloudSite`          | Cloud site (e.g. `i18n-tt`, `prod`)                                                                        |
| `--space-id`             | `bits.spaceId`            | BITS space ID (frontend: `35997699842`, backend: `201141148930`)                                           |
| `--service-type`         | `bits.apps[<appName>].serviceType` | Service type per app (e.g. `PROJECT_TYPE_TCE`, `PROJECT_TYPE_WEB`)                             |
| `--enable-lanes`         | `bits.enableLanes`        | Enable lanes (ppe, boe, both)                                                                              |
| `--lane`                 | (runtime inference)       | Target lane name — inferred from TD swim lane or branch name heuristic                                                     |
| `--services`             | `bits.apps[<appName>].services` | Comma-separated service PSMs per app                                                             |
| `--dev-task-template-id` | `bits.devTaskTemplateId`  | Dev task template ID (**mutually exclusive with `--team-flow-id`**; frontend: `29396`, backend: `31265`)   |

**Mutually exclusive**: `--team-flow-id` and `--dev-task-template-id` cannot be used together. When `--team-flow-id` is provided, omit `--dev-task-template-id` — the template ID will be resolved from the team flow workflow.

| CLI Parameter            | csp-config.json Key       | Description                                                                                                |
| ------------------------ | ------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `--team-flow-id`         | `bits.teamFlowId`         | Team flow ID (replaces `--dev-task-template-id` when set)                                                  |

**Optional parameters** — only include when explicitly provided by the user or config. Omit entirely if not set; do not pass empty values.

| CLI Parameter            | csp-config.json Key       | Description                                                                                                |
| ------------------------ | ------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `--title`                | `bits.title`              | Task title (optional)                                                                                      |
| `--change`               | `bits.change`             | Service change specs                                                                                       |
| `--scm-mode`             | `bits.scmMode`            | SCM mode (branch, version)                                                                                 |
| `--scm-branch`           | `bits.scmBranch`          | Source branch                                                                                              |
| `--from-dev-id`          | `bits.fromDevId`          | Template dev task ID                                                                                       |
| `--idcs`                 | `bits.idcs`               | IDC list                                                                                                   |
| `--qa`                   | `bits.qa`                 | QA email                                                                                                   |
| `--developer`            | `bits.developer`          | Developer email                                                                                            |
| `--meego`                | `bits.meego`              | Meego work item URL or ID (also read from `documents.meegoLink` in requirement status)                     |
| `--var`                  | `bits.vars`               | Custom variables                                                                                           |
| `--env-setting-map-json` | `bits.envSettingMapJson`  | Env setting map JSON                                                                                       |
| `--dev-task-mode`        | `bits.devTaskMode`        | Dev task mode                                                                                              |
| `--workflow-snapshot-id` | `bits.workflowSnapshotId` | Workflow snapshot ID                                                                                       |
| `--api-base-url`         | `bits.apiBaseUrl`         | API base URL                                                                                               |
| `--jwt-token`            | `bits.jwtToken`           | JWT token                                                                                                  |

## 4. Auto-inference rules

To minimize manual input, the agent **must** attempt to auto-infer the following parameters from available context before asking the user. Only fall back to `<FILL_ME>` placeholders when no inference source exists.

| Parameter | Inference priority (high → low) | Fallback |
|-----------|-------------------------------|----------|
| `--title` | User input → `changeName` (convert to human-readable) → requirement `id` from status file | Omit |
| `--lane` | User input → `bits.lane` from config → TD swim lane → branch name heuristic | `test` |
| `--scm-branch` | User input → `bits.scmBranch` from config → `git branch --show-current` | `<FILL_ME>` |
| `--meego` | User input → `bits.meego` from config → `documents.meegoLink` from requirement status | Omit |
| `--services` | User input → `bits.apps[<appName>].services` from config → search psm/service from the repository | `<FILL_ME>` |
| `--service-type` | User input → `bits.apps[<appName>].serviceType` from config | `<FILL_ME>` |

**Branch name heuristic for `--lane`**:
- Branch contains `hotfix` → `ppe`
- Branch starts with `feat/` or `fix/` → `test`
- Otherwise → `test`

**Tech design swim lane inference for `--lane`**:
- Read the TD file path from `documents.techDesign` in the current requirement status (`.ttadk/requirements-status/<id>.json`)
- Find the `# Swimlane` / `# 泳道` section in the TD markdown
- Extract the `feature_slug` from any swim lane entry (e.g. `boe_schedule_import_export` or `ppe_schedule_import_export` → `schedule_import_export`), stripping the `boe_` or `ppe_` prefix
- The slug is used as the `--lane` value — it is identical across boe and ppe by convention

**Title inference examples**:
- `changeName` = `add-approval-config` → title = "Add approval config"
- Requirement `id` = `fix-memory-leak` → title = "Fix memory leak"

## 5. Interaction guidelines

To improve Agent DX, the following rules must be followed during parameter collection:

- When required parameters are missing, validation fails, or a final confirmation is needed, **do not** ask follow-up questions in plain text only.
- You **must** provide a complete CLI/bash snippet:
  - **Keep known parameters**: inject all parameters already collected into the snippet.
  - **Highlight unknown parameters**: for missing values, use a prominent placeholder like `<FILL_ME>`.
  - **Guide user action**: ask the user to copy the snippet, fill the placeholders, and send it back (or run it locally).

### Example 1 — Only truly unknown params use `<FILL_ME>`

Scenario: the current branch is `feat-login`, change context is known, lane inferred as `test`, but `services` is missing.

> "I've collected most parameters from context. The target lane is inferred as `test` from the branch name, but the service PSM is missing. Please fill in the `<FILL_ME>` part and confirm execution:
>
> ```bash
> cspadk hiworks bits-task-create \
>   --cloud-site i18n-tt \
>   --space-id 35997699842 \
>   --service-type PROJECT_TYPE_WEB \
>   --enable-lanes both \
>   --lane test \
>   --services <FILL_ME> \
>   --dev-task-template-id 29396 \
>   --scm-branch feat-login \
>   --json
> ```
>
> Replace `<FILL_ME>` with the service PSM (e.g. `ies.kefu.schedule_service_i18n`) and send it back to continue."

### Example 2 — All params inferred, only confirmation needed

Scenario: the current branch is `feat/add-approval-config`, config has all defaults, meegoLink from requirement status.

> "All parameters resolved from context and config:
>
> | Parameter | Value | Source |
> |-----------|-------|--------|
> | --lane | schedule_import_export | TD swim lane (ppe_schedule_import_export → slug) |
> | --title | Add approval config | Inferred from changeName |
> | --scm-branch | feat/add-approval-config | git branch |
> | --meego | https://meego.feishu.cn/xxx/issue/detail/123 | Requirement status |
> | --services | ies.kefu.schedule_service_i18n | Config default |
>
> ```bash
> cspadk hiworks bits-task-create \
>   --cloud-site i18n-tt \
>   --space-id 35997699842 \
>   --service-type PROJECT_TYPE_WEB \
>   --enable-lanes both \
>   --lane test \
>   --services ies.kefu.schedule_service_i18n \
>   --dev-task-template-id 29396 \
>   --scm-branch feat/add-approval-config \
>   --meego "https://meego.feishu.cn/xxx/issue/detail/123" \
>   --json
> ```
>
> Proceed with these parameters?"

