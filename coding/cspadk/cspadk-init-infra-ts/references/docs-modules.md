# Per-Module Documentation Template

## modules/\<module-name\>.md — Module Documentation

| Field | Content |
|-------|---------|
| Purpose | In-depth view of a single module: internal architecture, sub-module breakdown, core flows, complete code index |
| When to Create | For every module/sub-project/package/service with its own directory |

### Section Structure

Every module documentation file must include all 8 sections below:

#### Section 1: Module Overview

- Must be ≤ 3 sentences
- States what this module does and its role in the system

```markdown
## Module Overview

@cspadk/cli is the CLI for the CSPADK toolkit, providing project initialization, skill management, and requirement tracking. It serves as the primary developer interface for interacting with the CSPADK ecosystem.
```

#### Section 2: Internal Architecture

- Directory tree with annotations for every directory
- Architectural pattern description
- DI/wiring mechanism if applicable

```markdown
## Internal Architecture

packages/cli/
├── bin/              # CLI entry point (thin shim)
├── src/
│   ├── commands/     # Command handlers (one file per group)
│   ├── core/         # Business logic (ReqTracker, UpdateChecker)
│   ├── templates/    # Template definitions and management
│   ├── utils/        # Shared utilities (logger, color, error)
│   └── index.ts      # CLI registration and startup
└── package.json

**Pattern**: CAC-based command registration with delegation to `core/` classes.
**Wiring**: Commands are registered in `index.ts` using CAC; each command delegates to a core class that encapsulates business logic.
```

#### Section 3: Sub-Module Breakdown

- Table: Sub-Module / Path / Responsibility / Key Exports

```markdown
## Sub-Module Breakdown

| Sub-Module | Path | Responsibility | Key Exports |
|-----------|------|---------------|-------------|
| Commands | src/commands/ | CLI command handlers | initCommand, skillCommand, reqCommand |
| Core | src/core/ | Business logic classes | ReqTracker, UpdateChecker, TemplateManager |
| Templates | src/templates/ | Template definitions and rendering | getTemplate, renderTemplate |
| Utils | src/utils/ | Shared utilities | logger, colorize, handleError |
```

#### Section 4: Core Execution Flows

- For each flow: name, entry point (file:function), step-by-step call chain, error handling path

```markdown
## Core Execution Flows

### Flow: Project Initialization

- **Entry Point**: `src/commands/init.ts:initCommand()`
- **Call Chain**:
  1. `initCommand()` parses user options
  2. Delegates to `src/core/TemplateManager.ts:selectTemplate()`
  3. `TemplateManager` renders selected template via `src/templates/index.ts:getTemplate()`
  4. Writes rendered files to target directory
  5. Runs validation via `src/core/Validator.ts:validate()`
- **Error Handling**: If template rendering fails, logs error via `src/utils/logger.ts` and exits with code 1. If validation fails, reports specific failures and suggests remediation.

### Flow: Requirement Tracking

- **Entry Point**: `src/commands/req.ts:reqCommand()`
- **Call Chain**:
  1. `reqCommand()` parses subcommand (list/status)
  2. Delegates to `src/core/ReqTracker.ts:listRequirements()` or `getStatus()`
  3. `ReqTracker` reads `.ttadk/requirements-status/` JSON files
  4. Formats and displays results
- **Error Handling**: Missing `.ttadk/` directory triggers initialization prompt. Corrupt JSON files are skipped with a warning.
```

#### Section 5: Code Location Index

- Table: File Path / Responsibility / Key Exports
- Must cover ALL source files in the module

```markdown
## Code Location Index

| File Path | Responsibility | Key Exports |
|-----------|---------------|-------------|
| src/index.ts | CLI registration and startup | main |
| src/commands/init.ts | Init command handler | initCommand |
| src/commands/skill.ts | Skill command handler | skillCommand |
| src/commands/req.ts | Requirement command handler | reqCommand |
| src/core/ReqTracker.ts | Requirement state management | ReqTracker |
| src/core/UpdateChecker.ts | Update availability checking | UpdateChecker |
| src/core/TemplateManager.ts | Template selection and rendering | TemplateManager |
| src/templates/index.ts | Template registry | getTemplate, renderTemplate |
| src/utils/logger.ts | Logging utility | logger |
| src/utils/color.ts | Terminal color helpers | colorize |
| src/utils/error.ts | Error handling utilities | handleError |
```

#### Section 6: Configuration Reference

- Config files, env vars, feature flags

```markdown
## Configuration Reference

| Config | Location | Description | Default |
|--------|----------|-------------|---------|
| .cspadk.json | Project root | CSPADK project configuration | Auto-generated |
| CSPADK_REGISTRY_URL | Environment | Custom skill registry URL | https://registry.cspadk.io |
| CSPADK_LOG_LEVEL | Environment | Logging verbosity | info |
```

#### Section 7: Testing Guide

- Test directory structure
- Exact command to run this module's tests in isolation
- Test conventions

```markdown
## Testing Guide

**Test Directory**: `packages/cli/__tests__/`

**Run in Isolation**:
```bash
npx vitest run packages/cli/
```

**Conventions**:
- Test files named `*.test.ts`
- One describe block per command or core class
- Use `vi.mock()` for external dependencies
- Snapshot tests for CLI output formatting
```

#### Section 8: Dependencies

- Table: Dependency / Version / Purpose / Internal or External

```markdown
## Dependencies

| Dependency | Version | Purpose | Type |
|-----------|---------|---------|------|
| @i18n-cs/ai-toolkit-core | workspace:* | Core skill validation utilities | Internal |
| @i18n-cs/infra-config | workspace:* | Shared toolchain configuration | Internal |
| cac | ^0.18.x | CLI argument parsing | External |
| chalk | ^5.x | Terminal color output | External |
| dotenv | ^16.x | Environment variable loading | External |
```

---

### Content Rules

1. Core execution flows must reference actual file paths and function names (e.g., `src/commands/init.ts:initCommand()`)
2. Code location index must cover ALL source files in the module — no file may be omitted
3. Testing guide must include the exact command to run this module's tests in isolation (e.g., `npx vitest run packages/cli/`)

### Quality Checks

- [ ] Module overview ≤ 3 sentences
- [ ] Internal architecture includes directory tree with annotations
- [ ] Every sub-module has a row in the breakdown table
- [ ] At least one core execution flow is described with file:line references
- [ ] Code location index covers all source files
- [ ] Testing guide includes isolation run command
- [ ] Dependencies table distinguishes internal vs. external
