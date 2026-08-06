---
name: cspadk-init-infra-java
description: AI infrastructure initialization Skill for Java backend services. When executed in the project root directory, it automatically scans Java project characteristics, generates or completes CI, dependency management, Lint, testing and other infrastructure configurations, and ensures each infrastructure item is correctly initialized through a validation mechanism.
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
  - Initialize Java project infrastructure
  - Java project CI configuration
  - Java dependency management
  - Java Lint rules
---

# cspadk-init-infra-java

AI infrastructure initialization Skill for Java backend services in CSP business scenarios.

## Trigger Scenarios

- Automatically triggered when the user executes `/cspadk:init` and the project is identified as a Java tech stack
- The user explicitly requests initialization of Java project infrastructure configuration
- First-time onboarding to the CSPADK workflow after creating a new Java service repository

## Input

- Project root directory path (defaults to the current working directory)
- Optional: User-specified JDK version, build tool preference (Maven/Gradle), etc.

## Output

- Complete Java project infrastructure configuration files (CI, Lint, testing, dependency management, etc.)

## Execution Steps

### Stage 0: Initialize Repository AI Infrastructure (AGENTS.md and docs Directory)

**Language Detection**: Before generating docs content, determine the output language:

1. **Infer from AGENTS.md**: If primarily Chinese → generate Chinese docs; if English → English docs
2. **Infer from repository**: Check README.md, CLAUDE.md, existing docs/ for language patterns
3. **Fallback**: Prompt user to choose: "What language should the generated docs use? Chinese / English (default)"
4. **Default**: English

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

**Note**: Skill content (SKILL.md, references/*) is always written in English. This policy only applies to generated docs content.

Follow [references/agents-md.md](references/agents-md.md) to execute the following steps:

1. Scan existing content: Read AGENTS.md, scan docs/ directory, scan project root directory structure, scan skills/ directory
2. Supplement missing files according to the docs directory specification (only create template skeletons, do not overwrite existing files)
3. Update AGENTS.md (preserve existing content, append constraint warnings, docs index, module index, workflow artifacts)
4. Validate: Confirm all links are valid, docs directory is compliant, and existing content has not been deleted

### Stage 1: Project Scanning and Feature Identification

1. Scan the project root directory to identify the following features:
   - `pom.xml` / `build.gradle` / `build.gradle.kts` → Confirm build tool
   - `src/main/java/` directory structure → Infer project layout
   - Existing CI configuration files (`.github/workflows/`, `.codebase/pipelines/`)
   - Existing Lint configuration (`checkstyle.xml`, `spotbugs-config.xml`, `pmd-ruleset.xml`)
   - Existing test directory (`src/test/java/`)
2. Output a project feature summary for use in subsequent stage decisions

### Stage 1.5: Infrastructure Gap Detection & Auto-Completion

Based on Stage 1 scan results, detect missing infrastructure items and offer selection-based auto-completion:

1. **Scan**: Check for missing infrastructure against the 5 categories below
2. **Present**: Show detected gaps as a checklist for user selection
3. **Execute**: Install dependencies, create configurations, write examples for selected items
4. **Verify**: Confirm each completed item meets acceptance criteria

| Infrastructure | Detection Method | Reference |
|---------------|-----------------|-----------|
| Test Framework | No test config (`src/test/java/` missing or no JUnit 5 deps) | [tests.md](references/tests.md) |
| CI Pipeline | No YAML under `.codebase/pipelines/` | [ci.md](references/ci.md) |
| Lint Config | No `checkstyle.xml` found | [lint.md](references/lint.md) |
| Coverage | No JaCoCo plugin in build config | [tests.md](references/tests.md) |
| Dependency Mgmt | Missing `pom.xml` or `build.gradle` | [deps.md](references/deps.md) |

**Acceptance Criteria** (all items):

1. Dependencies installed and versions correct
2. Configuration files created with valid syntax
3. A simple runnable example written (test case / checkstyle config / CI YAML)
4. Execution verification passes (`mvn test` / `mvn checkstyle:check` / CI YAML valid)

### Stage 2: Dependency Management Initialization

Follow [references/deps.md](references/deps.md) to execute:

1. Check if the build tool configuration file exists
2. Verify that the JDK version meets the minimum requirement (≥ JDK 17)
3. Check if dependency version management is unified (BOM or dependencyManagement)
4. Confirm that the internal repository source is configured correctly

### Stage 3: Lint Rule Configuration

Follow [references/lint.md](references/lint.md) to execute:

1. Check if Checkstyle / SpotBugs / PMD configuration exists
2. If not, generate configuration files based on CSP standard templates
3. If it exists, compare against the CSP standard rule set, highlight missing items, and suggest additions
4. Validate that Lint tools are executable and configurations are valid

### Stage 4: CI Pipeline Configuration

Follow [references/ci.md](references/ci.md) to execute:

1. Check existing CI configuration
2. Generate or complete Codebase CI Pipeline YAML (`.codebase/pipelines/`)
3. Configure pipeline stages: lint → test → build
4. Ensure the pipeline includes JDK version checking and dependency caching

### Stage 5: Test Framework Initialization

Follow [references/tests.md](references/tests.md) to execute:

1. Check existing test directory and framework dependencies
2. Confirm JUnit 5 basic configuration
3. Check if auxiliary libraries such as Mockito need to be introduced
4. Generate test templates
5. Configure coverage output target (JaCoCo)

### Stage 6: Infrastructure Status Report & User Confirmation

After completing Stages 0–5, present a unified status table of all infrastructure items and let the user decide whether to fix remaining issues.

1. **Run validation checklist**: Check each infrastructure item's current status:

| Check Item | Validation Method | Expected Result |
|------------|-------------------|-----------------|
| AGENTS.md | `AGENTS.md` exists and contains core constraints section | ✅ AI infrastructure navigation ready |
| Build Tool | `pom.xml` or `build.gradle` exists and is parseable | ✅ Build configuration ready |
| JDK Version | JDK version ≥ 17 in configuration | ✅ JDK version compliant |
| Lint Configuration | Checkstyle configuration exists and is executable | ✅ Lint rules executable |
| CI Pipeline | At least one YAML exists under `.codebase/pipelines/` with valid syntax | ✅ Pipeline configuration ready |
| Test Executable | `mvn test` or `gradle test` can run | ✅ Test framework available |
| Coverage Configuration | JaCoCo plugin is configured | ✅ Coverage trackable |
| docs/modules/ | `docs/modules/` exists and contains at least one `<module-name>.md` file | ✅ Module documentation available |

2. **Output a unified status table**: Present the results in the following format to the user:

```
| # | Infrastructure Item | Status | Detail |
|---|---------------------|--------|--------|
| 1 | AGENTS.md           | ✅ PASS / ❌ FAIL | ... |
| 2 | Build Tool          | ✅ PASS / ❌ FAIL | ... |
| ... | ... | ... | ... |
```

3. **Ask user for confirmation**: If any items show FAIL, present the list of failing items and ask:
   - "The following infrastructure items do not meet requirements: [list]. Would you like to fix them? (Fix all / Select items / Skip)"
4. **Fix selected items**: For each item the user confirms, re-execute the corresponding Stage (0–5) to fix the issue.
5. **Re-validate**: After fixes, re-run the validation checklist and present the updated table. Repeat until the user is satisfied or all items pass.

### Step 7: Detect Sub-applications and PSM Mappings

- **Action**: Detect sub-applications in the target repository and identify PSM/service mappings.
- **Action**: Check for multiple sub-module `pom.xml` or `build.gradle` files to identify sub-applications.
- **Action**: If multiple modules found, each becomes a sub-application. If only one, treat as single application with `"main"` and `path: "."`.
- **Action**: Scan the repository for PSM/service names in:
  1. Documentation files: `AGENTS.md`, `deploy.md` and other markdown files
  2. Configuration files: `yaml`/`toml`/`json` config files, `application.properties`, `Dockerfile`
  3. Match PSM patterns: fields containing `psm`, `service_name`, `service-name`, `ies.kefu.*`
- **Action**: Associate identified PSMs to the corresponding sub-application based on file path ownership.
- **Action**: Default `serviceType` to `PROJECT_TYPE_TCE` for Java backend projects.
- **Action**: If PSM is not found for a sub-application, leave `services` as empty string `""`.
- **Action**: Write the detection results to `.ttadk/csp-config.json` under `bits.apps` using `ConfigManager.setLocal('bits.apps', appsConfig)` or by directly editing the file.
- **Rule**: Do NOT scan `target/`, `.git/`, or `.gradle/` directories.
- **Rule**: The `bits.apps` value is a map where keys are kebab-case app names and values are `{ path, serviceType, services }`.
- **Rule**: Single-app repositories use `"main": { "path": ".", "serviceType": "PROJECT_TYPE_TCE", "services": "" }`.

## Relationship with Other Skills

- Invoked by `/cspadk:init` orchestration
- Output is scanned and validated by `cspadk-anti-degradation`
- Stage 0 includes built-in AGENTS.md and docs directory initialization capabilities (refer to `references/agents-md.md`)

## Output Language

Detect the user's input language and respond in the same language.

- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).
