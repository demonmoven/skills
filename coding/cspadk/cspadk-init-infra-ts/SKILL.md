---
name: cspadk-init-infra-ts
description: AI infrastructure initialization Skill for TS/React frontend projects. When executed in the project root directory, it automatically scans TS project characteristics, generates or completes CI, dependency management, Lint, testing and other infrastructure configurations, and ensures each infrastructure item is correctly initialized through a validation mechanism.
version: 1.1.2
metadata:
  patterns:
    - generator
    - tool-wrapper
  domain: cspadk
  i18n_level: 0
  prompt_version: "1.1.2"
  agent_support:
    - claude-code
    - trae
    - trae-cli
    - coco
  language:
    - en
triggers:
  - cspadk init
  - Initialize TS project infrastructure
  - Frontend project CI configuration
  - TS dependency management
  - ESLint rules
---

# cspadk-init-infra-ts

AI infrastructure initialization Skill for CSP business scenario TS/React frontend projects.

## Trigger Scenarios

- Automatically triggered when the user executes `/cspadk:init` and the project is identified as a TS/React tech stack
- User explicitly requests initialization of frontend project infrastructure configuration
- First-time CSPADK workflow integration after creating a new frontend project repository

## Input

- Project root directory path (defaults to the current working directory)
- Optional: user-specified package manager preference (emo/pnpm), framework preference, etc.

## Output

- Complete TS/React project infrastructure configuration files (CI, Lint, testing, dependency management, etc.)

## Execution Steps

### Stage 0: Initialize Repository AI Infrastructure (AGENTS.md and docs Directory)

Refer to [references/agents-md.md](references/agents-md.md) and execute the following steps:

#### Language Detection

Before generating any docs content, determine the output language:

1. **Infer from AGENTS.md**: If the project's AGENTS.md is written primarily in Chinese, generate docs in Chinese. If in English, generate in English.
2. **Infer from repository content**: Check README.md, CLAUDE.md, and existing docs/ for language patterns.
3. **Fallback**: If language cannot be determined confidently, prompt the user to choose: "What language should the generated docs use? Chinese / English (default)"
4. **Default**: English.

#### Language Persistence

After determining the language:
1. Read the current AGENTS.md content.
2. Check if a `## Language` section already exists.
3. If not, append the following section:
   ```
   ## Language

   <lang>

   **Constraint**: All generated artifacts (docs, proposals, specs, designs, tasks, comments) MUST use this language. Do NOT mix languages within a single artifact.
   ```
   where `<lang>` is `zh` for Chinese or `en` for English.
4. If it exists, update the value only if the detected language differs.
5. All subsequent artifact generation in this skill and downstream skills MUST follow this language setting.

All skill content (SKILL.md, references/*) remains in English regardless.

#### Docs Generation

1. Scan existing content: read AGENTS.md, scan docs/ directory, scan project root directory structure, scan skills/ directory
2. Fill in missing files according to docs directory conventions — use content templates from [references/docs-architecture.md](references/docs-architecture.md), [references/docs-constraints.md](references/docs-constraints.md), [references/docs-dev-workflow.md](references/docs-dev-workflow.md), [references/docs-quality.md](references/docs-quality.md), [references/docs-business-context.md](references/docs-business-context.md), [references/docs-modules.md](references/docs-modules.md), and [references/docs-meta.md](references/docs-meta.md) (only create files if not already existing, never overwrite)
3. Update AGENTS.md (preserve existing content, append constraint warnings, docs index, module index, workflow artifacts)
4. Validate: confirm all links are valid, docs directory is compliant, existing content has not been deleted

### Stage 1: Project Scanning and Feature Identification

1. Scan the project root directory to identify the following features:
   - `package.json` → Confirm project information and dependencies
   - `tsconfig.json` → Confirm TypeScript configuration
   - `eden.monorepo.json` → Confirm whether it is an emo monorepo project
   - Existing CI configuration files (`.github/workflows/`, `.codebase/pipelines/`)
   - Existing Lint configuration (`.eslintrc.*`, `.prettierrc.*`)
   - Existing test configuration (`jest.config.*`, `vitest.config.*`)
   - Framework features (`next.config.*`, `vite.config.*`, `craco.config.*`)
2. Output a project feature summary for subsequent stage decision-making

### Stage 1.5: Infrastructure Gap Detection and Auto-Completion

1. Based on Stage 1 scan results, identify missing infrastructure items
2. Present a checklist of detected gaps to the user for selection
3. For each selected item, execute the auto-completion procedure from the corresponding reference
4. Verify each completed item meets the acceptance criteria

**Detection Rules:**

| Infrastructure | Detection Method | Reference |
|----------------|-----------------|-----------|
| Test Framework | No vitest.config.* or jest.config.* found | [references/tests.md](references/tests.md) → Auto-Completion Procedure |
| CI Pipeline | No YAML under .codebase/pipelines/ | [references/ci.md](references/ci.md) → Auto-Completion Procedure |
| Lint Config | No .eslintrc.* or eslint.config.* found | [references/lint.md](references/lint.md) → Auto-Completion Procedure |
| Coverage | Test config lacks coverage settings | [references/tests.md](references/tests.md) → Auto-Completion Procedure |
| Dependency Mgmt | Missing emo config or root package.json issues | [references/deps.md](references/deps.md) → Auto-Completion Procedure |

**Acceptance Criteria (all items):**

1. Dependencies installed and versions correct
2. Configuration files created with valid syntax
3. A simple runnable example written (test case / lint example / CI YAML)
4. Execution verification passes (test green / lint no config errors / CI YAML valid)

### Stage 2: Dependency Management Initialization

Refer to [references/deps.md](references/deps.md) and execute:

1. Check if `package.json` exists
2. Verify whether `emo` is used to manage dependencies (check `eden.monorepo.json`)
3. Check if root `package.json` contains `devDependencies` (should be moved to `infra/`)
4. Confirm `.npmrc` has `shamefully-hoist=false`
5. Verify `pnpm-lock.yaml` is located under the `infra/` directory

### Stage 3: Lint Rule Configuration

Refer to [references/lint.md](references/lint.md) and execute:

1. Check if ESLint configuration exists
2. If not, generate `.eslintrc.cjs` based on the CSP standard template
3. If it exists, compare against the CSP standard rule set, highlight missing items and suggest additions
4. Check Prettier configuration
5. Verify that `eslint` and `prettier` are executable and configurations are valid

### Stage 4: CI Pipeline Configuration

Refer to [references/ci.md](references/ci.md) and execute:

1. Check existing CI configuration
2. Generate or complete Codebase CI Pipeline YAML (`.codebase/pipelines/`)
3. Configure pipeline stages: lint → test → build
4. Ensure the pipeline includes Node.js version checking and dependency caching

### Stage 5: Test Framework Initialization

Refer to [references/tests.md](references/tests.md) and execute:

1. Check existing test framework and configuration
2. Confirm Jest or Vitest base configuration
3. Check whether React Testing Library needs to be introduced
4. Generate test templates
5. Configure coverage output target

### Stage 6: Infrastructure Status Report & User Confirmation

After completing Stages 0–5, present a unified status table of all infrastructure items and let the user decide whether to fix remaining issues.

1. **Run validation checklist**: Check each infrastructure item's current status:

| Check Item | Validation Method | Expected Result |
|------------|-------------------|-----------------|
| AGENTS.md | `AGENTS.md` exists and contains core constraints section | ✅ AI infrastructure navigation ready |
| Package Manager | `emo` available and `eden.monorepo.json` exists | ✅ emo management ready |
| Root Directory Clean | Root `package.json` has no `devDependencies`, no `node_modules` | ✅ Directory structure compliant |
| Lint Configuration | `.eslintrc.*` exists and `npx eslint .` has no configuration errors | ✅ Lint rules executable |
| CI Pipeline | At least one YAML exists under `.codebase/pipelines/` with valid syntax | ✅ Pipeline configuration ready |
| Tests Executable | `npm test` or `emo test` can run | ✅ Test framework available |
| Coverage Configuration | Test configuration includes coverage collection | ✅ Coverage trackable |
| docs/modules/ | `docs/modules/` exists and contains at least one `<module-name>.md` file | ✅ Module documentation available |

2. **Output a unified status table**: Present the results in the following format to the user:

```
| # | Infrastructure Item | Status | Detail |
|---|---------------------|--------|--------|
| 1 | AGENTS.md           | ✅ PASS / ❌ FAIL | ... |
| 2 | Package Manager     | ✅ PASS / ❌ FAIL | ... |
| ... | ... | ... | ... |
```

3. **Ask user for confirmation**: If any items show FAIL, present the list of failing items and ask:
   - "The following infrastructure items do not meet requirements: [list]. Would you like to fix them? (Fix all / Select items / Skip)"
4. **Fix selected items**: For each item the user confirms, re-execute the corresponding Stage (0–5) to fix the issue.
5. **Re-validate**: After fixes, re-run the validation checklist and present the updated table. Repeat until the user is satisfied or all items pass.

### Step 7: Detect Sub-applications and PSM Mappings

- **Action**: Detect sub-applications in the target repository and identify PSM/service mappings.
- **Action**: Read `eden.monorepo.json` from the repository root to get the sub-application list and paths.
- **Action**: If `eden.monorepo.json` does not exist, fall back to reading `package.json` `workspaces` field for sub-package paths.
- **Action**: If neither exists, treat as single application and use `"main"` with `path: "."`.
- **Action**: Scan the repository for PSM/service names in:
  1. Documentation files: `AGENTS.md`, `deploy.md`, `README.md` and other markdown files
  2. Configuration files: `package.json`, `tsconfig.json`, `.env`, `yaml`/`toml`/`json` config files
  3. Match PSM patterns: fields containing `psm`, `service_name`, `service-name`, `ies.kefu.*`
- **Action**: Associate identified PSMs to the corresponding sub-application based on file path ownership.
- **Action**: Default `serviceType` to `PROJECT_TYPE_WEB` for frontend projects.
- **Action**: If PSM is not found for a sub-application, leave `services` as empty string `""`.
- **Action**: Write the detection results to `.ttadk/csp-config.json` under `bits.apps` using `ConfigManager.setLocal('bits.apps', appsConfig)` or by directly editing the file.
- **Rule**: Do NOT scan `node_modules/`, `.git/`, `dist/`, or `build/` directories.
- **Rule**: The `bits.apps` value is a map where keys are kebab-case app names and values are `{ path, serviceType, services }`.
- **Rule**: Single-app repositories use `"main": { "path": ".", "serviceType": "PROJECT_TYPE_WEB", "services": "" }`.

## Relationship with Other Skills

- Invoked by `/cspadk:init` orchestration
- Output is validated by `cspadk-anti-degradation` scanning
- Stage 0 includes built-in AGENTS.md and docs directory initialization capability (refer to `references/agents-md.md`)

## Output Language

Detect the user's input language and respond in the same language.

- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).
