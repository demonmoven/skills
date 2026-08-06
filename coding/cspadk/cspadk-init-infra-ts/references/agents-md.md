# AGENTS.md & docs Directory Initialization Reference

This reference document defines the complete execution logic for initializing the repository's `AGENTS.md` knowledge base map and `docs/` directory structure.

## Core Principles

1. **Append-Only, Never Delete**: Content in `AGENTS.md` and files under `docs/` may only be appended or updated — **never delete** existing content.
2. **AGENTS.md as the Map**: `AGENTS.md` serves as the repository's navigation hub and must index all core documents under `docs/`.
3. **docs Directory Standardization**: The `docs/` directory must follow the unified directory specification (see below).

## Input

- Project root directory path (defaults to the current working directory)
- Optional: user-specified business context description

## Output

- Updated `AGENTS.md` (appended new content, preserving existing content)
- Supplemented `docs/` directory structure per specification (missing files are created from templates)

## Execution Steps

### Stage 1: Scan Existing Content

1. Read the full contents of the current `AGENTS.md` (if it exists)
2. Scan all existing files under the `docs/` directory
3. Scan the project root directory structure (identify tech stack, modules, configs, etc.)
4. Scan all Skill modules under the `skills/` directory
5. Output a current-state summary

### Stage 2: Supplement Missing Files per docs Directory Specification

Check and create missing files according to the following docs directory specification (create template skeletons only; do not overwrite existing files):

```
docs/
├── index.md                          # Documentation center homepage
├── architecture/
│   ├── overview.md                   # Repository architecture overview
│   └── module-index.md               # Module introductions & file directory index
├── constraints/
│   ├── repository-constraints.md     # Repository constraint specs (deps, dirs, CI, etc.)
│   ├── development-guidelines.md     # Development & version control guidelines
│   └── constraint-validation.md      # Constraint validation guide
├── development/
│   ├── getting-started.md            # Quick start guide
│   └── workflow.md                   # Development workflow description
├── quality/
│   ├── testing-strategy.md           # Testing strategy & coverage requirements
│   └── code-review-guide.md          # Code review guide
├── business/
│   ├── background.md                 # Business background introduction
│   └── processes.md                  # Core business process descriptions
├── modules/                           # Per-module detailed documentation
│   └── <module-name>.md              # One file per module: architecture, breakdown, code index
├── experiences/                       # Experience repository directory
└── writing-guide.md                   # Documentation writing guide
```

**Key Rules**:
- Existing files are **not overwritten**; only missing template files are created
- Template files contain standard heading structures and placeholder descriptions to be filled in
- For files that exist at a different path (e.g., `repository-constraints.md` already at the root), create a reference link under `constraints/`

#### Module Documentation Generation (within Stage 2)

The `docs/modules/` directory MUST be populated with per-module files — creating an empty directory is not acceptable. Execute the following steps:

1. **Discover modules**: Scan the project root directory to identify top-level modules. For TS/React projects, look for:
   - Directories under `packages/` (monorepo packages)
   - Directories under `apps/` or `services/` (application entries)
   - Directories with their own `package.json` (workspace members)
   - Top-level `src/` directory (if single-package project, treat as one module)
2. **Create one file per module**: For each discovered module, create `docs/modules/<module-name>.md` using the template from [references/docs-modules.md](docs-modules.md). Fill in all 8 sections with actual content derived from scanning the module's source code.
3. **Do not skip modules**: Every module directory that contains `.ts`/`.tsx` source files must have a corresponding documentation file. If a module has no source files, skip it.
4. **Do not overwrite**: If a `docs/modules/<module-name>.md` file already exists, skip it.

### Stage 3: Update AGENTS.md

1. **Preserve existing content**: Read the full contents of the current `AGENTS.md`
2. **Append constraint warning**: If the "Repository Core Constraints" section does not exist, append it at the top
3. **Append docs index**: Based on the Stage 2 directory specification, append or update the docs directory index
4. **Append module index**: Based on the `skills/` directory scan results, append or update the skill module index
5. **Append workflow artifacts**: Append or update the workflow artifacts directory description

**AGENTS.md Standard Structure**:

```markdown
# Repository Knowledge Base Map

## 1. ⚠️ Repository Core Constraints & Experience Reference
[Constraint document index, with mandatory reading warning]

## 2. Core Documents (Docs)
[Index links to all core documents under docs/]

### 2.1 Architecture & Design
- Repository architecture overview
- CSPADK architecture & design description
- Module introductions & file directory index

### 2.2 Constraints & Specifications
- Repository constraint specs
- Development & version control guidelines
- Constraint validation guide

### 2.3 Development
- Quick start guide
- Development workflow description

### 2.4 Quality Assurance
- Testing strategy
- Code review guide

### 2.5 Business Background & Processes
- Business background introduction
- Core business process descriptions

### 2.6 Module Documentation
- Per-module architecture and code index

## 3. Workflow Artifacts & Status Tracking
[.ttadk/, openspec/, and other artifact directory descriptions]

## 4. Experience & Design Decisions
[docs/experiences/ and other experience directory descriptions]

## 5. Repository Directory Structure Overview
[Complete directory tree of the project root with brief descriptions]
```

### Stage 4: Validation

1. Confirm all links in `AGENTS.md` point to existing files
2. Confirm the `docs/` directory structure conforms to the specification
3. Confirm no existing content has been deleted
4. Output an update summary report

## docs Directory Specification Details

### architecture/ — Architecture & Design

| File | Responsibility | Required | Template Reference |
|------|---------------|----------|-------------------|
| `overview.md` | Repository architecture overview, including tech stack, module breakdown, and dependencies | Yes | `references/docs-architecture.md` (Section 1) |
| `module-index.md` | Module (sub-project/Skill) introductions, responsibilities, and file directory index | Yes | `references/docs-architecture.md` (Section 2) |

### constraints/ — Constraints & Specifications

| File | Responsibility | Required | Template Reference |
|------|---------------|----------|-------------------|
| `repository-constraints.md` | Dependency management, directory configuration, CI/CD, and other low-level mandatory constraints | Yes | `references/docs-constraints.md` (Section 1) |
| `development-guidelines.md` | Development & version control guidelines | Yes | `references/docs-constraints.md` (Section 2) |
| `constraint-validation.md` | Constraint validation guide (how to check compliance) | Yes | `references/docs-constraints.md` (Section 3) |

### development/ — Development

| File | Responsibility | Required | Template Reference |
|------|---------------|----------|-------------------|
| `getting-started.md` | Quick start guide for new contributors (environment setup, first run) | Yes | `references/docs-dev-workflow.md` (Section 1) |
| `workflow.md` | SDD workflow description (brainstorm → TD → openspec → TDD) | No | `references/docs-dev-workflow.md` (Section 2) |

### quality/ — Quality Assurance

| File | Responsibility | Required | Template Reference |
|------|---------------|----------|-------------------|
| `testing-strategy.md` | Testing strategy & coverage requirements | No | `references/docs-quality.md` (Section 1) |
| `code-review-guide.md` | Code review guide | No | `references/docs-quality.md` (Section 2) |

### business/ — Business Background & Processes

| File | Responsibility | Required | Template Reference |
|------|---------------|----------|-------------------|
| `background.md` | Business background introduction | No | `references/docs-business-context.md` (Section 1) |
| `processes.md` | Core business process descriptions | No | `references/docs-business-context.md` (Section 2) |

### modules/ — Per-Module Documentation

| File | Responsibility | Required | Template Reference |
|------|---------------|----------|-------------------|
| `<module-name>.md` | Per-module architecture, breakdown, and code index | No | `references/docs-modules.md` |

### Root Files

| File | Responsibility | Required | Template Reference |
|------|---------------|----------|-------------------|
| `index.md` | Documentation center homepage with organized links | Yes | `references/docs-meta.md` (Section 1) |
| `writing-guide.md` | Documentation writing guide | No | `references/docs-meta.md` (Section 2) |
