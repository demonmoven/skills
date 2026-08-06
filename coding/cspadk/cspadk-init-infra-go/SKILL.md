---
name: cspadk-init-infra-go
description: AI infrastructure initialization Skill for Go backend services. When executed in the project root directory, it automatically scans Go project features, generates or completes CI, dependency management, Lint, testing and other infrastructure configurations, and ensures each infrastructure item is correctly initialized through a validation mechanism.
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
  - Initialize Go project infrastructure
  - Go project CI configuration
  - Go dependency management
  - Go Lint rules
---

# cspadk-init-infra-go

AI infrastructure initialization Skill for Go backend services in CSP business scenarios.

## Trigger Scenarios

- Automatically triggered when the user executes `/cspadk:init` and the project is identified as a Go tech stack
- User explicitly requests initialization of Go project infrastructure configuration
- First-time onboarding to the CSPADK workflow after creating a new Go service repository

## Input

- Project root directory path (defaults to the current working directory)
- Optional: user-specified Go version, framework preferences, etc.

## Output

- Complete Go project infrastructure configuration files (CI, Lint, testing, dependency management, etc.)

## Execution Steps

### Stage 0: Initialize Repository AI Infrastructure (AGENTS.md and docs Directory)

Refer to [references/agents-md.md](references/agents-md.md) and execute the following steps:

#### Language Detection

Before generating docs content, determine the output language:

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

1. Scan existing content: read AGENTS.md, scan docs/ directory, scan project root directory structure, scan skills/ directory
2. Complete missing files according to the docs directory specification (only create template skeletons, do not overwrite existing files)
3. Update AGENTS.md (preserve existing content, append constraint warnings, docs index, module index, workflow artifacts)
4. Validate: confirm all links are valid, docs directory is compliant, existing content has not been deleted

### Stage 1: Project Scanning and Feature Identification

1. Scan the project root directory and identify the following features:
   - Whether `go.mod` / `go.sum` exist → confirm Go Module management approach
   - Whether `Makefile` / `build.sh` exist → confirm build approach
   - `cmd/` / `internal/` / `pkg/` directory structure → infer project layout
   - Existing CI configuration files (`.github/workflows/`, `.codebase/pipelines/`)
   - Existing Lint configuration (`.golangci.yml`, `.revive.toml`)
   - Existing test files (`*_test.go`)
2. Output a project feature summary for subsequent stage decision-making

### Stage 1.5: Infrastructure Gap Detection and Auto-Completion

1. Based on Stage 1 scan results, identify missing infrastructure items
2. Present a checklist of detected gaps to the user for selection
3. For each selected item, execute the auto-completion procedure from the corresponding reference
4. Verify each completed item meets the acceptance criteria

**Detection Rules:**

| Infrastructure | Detection Method | Reference |
|----------------|-----------------|-----------|
| Test Framework | No `*_test.go` files or test config found | references/tests.md → Auto-Completion Procedure |
| CI Pipeline | No YAML under `.codebase/pipelines/` | references/ci.md → Auto-Completion Procedure |
| Lint Config | No `.golangci.yml` found | references/lint.md → Auto-Completion Procedure |
| Coverage | CI test stage lacks coverage flags | references/tests.md → Auto-Completion Procedure |
| Dependency Mgmt | Missing `go.mod` or `GOPRIVATE` not configured | references/deps.md → Auto-Completion Procedure |

**Acceptance Criteria (all items):**

1. Dependencies installed and versions correct
2. Configuration files created with valid syntax
3. A simple runnable example written (test case / lint example / CI YAML)
4. Execution verification passes (test green / lint no config errors / CI YAML valid)

### Stage 2: Dependency Management Initialization

Refer to [references/deps.md](references/deps.md) and execute:

1. Check if `go.mod` exists; if not, guide the user to execute `go mod init`
2. Verify that the Go version meets the minimum requirement (≥ 1.21)
3. Check if indirect dependencies are clean (`go mod tidy`)
4. Confirm `go.sum` integrity

### Stage 3: Lint Rule Configuration

Refer to [references/lint.md](references/lint.md) and execute:

1. Check if `.golangci.yml` exists
2. If it does not exist, generate `.golangci.yml` based on the CSP standard template
3. If it already exists, compare against the CSP standard rule set, highlight missing items and suggest additions
4. Verify that `golangci-lint` is executable and the configuration is valid

### Stage 4: CI Pipeline Configuration

Refer to [references/ci.md](references/ci.md) and execute:

1. Check existing CI configuration
2. Generate or complete Codebase CI Pipeline YAML (`.codebase/pipelines/`)
3. Configure pipeline stages: lint → test → build
4. Ensure the pipeline includes Go version checking and dependency caching

### Stage 5: Test Framework Initialization

Refer to [references/tests.md](references/tests.md) and execute:

1. Check existing test files and directory structure
2. Confirm usage of the `testing` standard library
3. Check whether auxiliary libraries such as testify need to be introduced
4. Generate test templates (e.g., `cmd/.../main_test.go`, `internal/.../handler_test.go`)
5. Configure coverage output targets

### Stage 6: Infrastructure Status Report & User Confirmation

After completing Stages 0–5, present a unified status table of all infrastructure items and let the user decide whether to fix remaining issues.

1. **Run validation checklist**: Check each infrastructure item's current status:

| Check Item | Validation Method | Expected Result |
|------------|-------------------|-----------------|
| AGENTS.md | `AGENTS.md` exists and contains core constraints section | ✅ AI infrastructure navigation ready |
| Go Module | `go mod verify` executes successfully | ✅ Dependencies complete and verified |
| Lint Configuration | `.golangci.yml` exists and `golangci-lint run --max-issues-per-linter=0` has no configuration errors | ✅ Lint rules executable |
| CI Pipeline | At least one YAML exists under `.codebase/pipelines/` with valid syntax | ✅ Pipeline configuration ready |
| Tests Executable | `go test ./... -count=1` can run (0 test cases allowed) | ✅ Test framework available |
| Coverage Configuration | CI includes a coverage collection step | ✅ Coverage trackable |
| docs/modules/ | `docs/modules/` exists and contains at least one `<module-name>.md` file | ✅ Module documentation available |

2. **Output a unified status table**: Present the results in the following format to the user:

```
| # | Infrastructure Item | Status | Detail |
|---|---------------------|--------|--------|
| 1 | AGENTS.md           | ✅ PASS / ❌ FAIL | ... |
| 2 | Go Module           | ✅ PASS / ❌ FAIL | ... |
| ... | ... | ... | ... |
```

3. **Ask user for confirmation**: If any items show FAIL, present the list of failing items and ask:
   - "The following infrastructure items do not meet requirements: [list]. Would you like to fix them? (Fix all / Select items / Skip)"
4. **Fix selected items**: For each item the user confirms, re-execute the corresponding Stage (0–5) to fix the issue.
5. **Re-validate**: After fixes, re-run the validation checklist and present the updated table. Repeat until the user is satisfied or all items pass.

### Step 7: Detect Sub-applications and PSM Mappings

- **Action**: Detect sub-applications in the target repository and identify PSM/service mappings.
- **Action**: Check directory structure for multiple `cmd/` sub-directories or multiple `main.go` entry points.
- **Action**: If multiple entry points found, each becomes a sub-application. If only one, treat as single application with `"main"` and `path: "."`.
- **Action**: Scan the repository for PSM/service names in:
  1. Documentation files: `AGENTS.md`, `deploy.md` and other markdown files
  2. Configuration files: `yaml`/`toml`/`json` config files, `application.properties`, `Dockerfile`
  3. Match PSM patterns: fields containing `psm`, `service_name`, `service-name`, `ies.kefu.*`
- **Action**: Associate identified PSMs to the corresponding sub-application based on file path ownership.
- **Action**: Default `serviceType` to `PROJECT_TYPE_TCE` for Go backend projects.
- **Action**: If PSM is not found for a sub-application, leave `services` as empty string `""`.
- **Action**: Write the detection results to `.ttadk/csp-config.json` under `bits.apps` using `ConfigManager.setLocal('bits.apps', appsConfig)` or by directly editing the file.
- **Rule**: Do NOT scan `vendor/`, `.git/`, or binary output directories.
- **Rule**: The `bits.apps` value is a map where keys are kebab-case app names and values are `{ path, serviceType, services }`.
- **Rule**: Single-app repositories use `"main": { "path": ".", "serviceType": "PROJECT_TYPE_TCE", "services": "" }`.

## Relationship with Other Skills

- Outputs are scanned and validated by `cspadk-anti-degradation`
- Stage 0 includes built-in AGENTS.md and docs directory initialization capabilities (refer to `references/agents-md.md`)

## Output Language

Detect the user's input language and respond in the same language.

- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).
