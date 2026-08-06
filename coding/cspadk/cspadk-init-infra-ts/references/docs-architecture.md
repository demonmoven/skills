# Architecture Documentation Templates

## 1. architecture/overview.md — Repository Architecture Overview

| Field | Content |
|-------|---------|
| Purpose | Describe the repository's overall architecture, tech stack, and module dependencies |
| Sections | 1) Project Overview (≤3 sentences) 2) Tech Stack (table: Technology / Version / Purpose) 3) Repository Structure (directory tree with annotations) 4) Module Dependency Graph (Mermaid preferred) 5) Deployment Architecture (staging + production) 6) External Dependencies (service / purpose) |
| Content Guidelines | Overview must be concise — no more than 3 sentences. Tech stack table must include all major tools with accurate versions. Directory tree must have one-line annotations for every top-level directory. Module dependency graph should use Mermaid syntax for visual clarity. External dependencies must list both the service name and its purpose. |
| Quality Checks | Overview ≤ 3 sentences; Tech stack table is complete with all major tools; Every top-level directory is annotated in the repository structure; External dependencies are listed with service and purpose |

### TS/React Example — Tech Stack Table

```markdown
| Technology | Version | Purpose |
|-----------|---------|---------|
| TypeScript | 5.x | Primary language |
| React | 18.x | UI framework |
| Vite | 5.x | Build tool |
| Vitest | 1.x | Test runner |
| emo | latest | Monorepo management |
```

### TS/React Example — Repository Structure

```markdown
├── packages/              # TypeScript toolkits (pnpm workspace)
│   ├── ai-toolkit-core/   # Core utilities for skill linting and validation
│   └── cli/               # CLI entry point (cspadk command)
├── skills/                # Self-contained AI skills (NOT in pnpm workspace)
├── infra/                 # Shared dev tools (ESLint, Prettier, TypeScript, Vitest)
├── docs/                  # Architecture docs, conventions, experience records
├── .codebase/             # CI pipeline definitions
└── openspec/              # SDD workflow artifacts
```

### TS/React Example — Module Dependency Graph

```mermaid
graph TD
    CLI[@cspadk/cli] --> Core[@i18n-cs/ai-toolkit-core]
    CLI --> Infra[@i18n-cs/infra-config]
    Core --> Infra
```

---

## 2. architecture/module-index.md — Module Index

| Field | Content |
|-------|---------|
| Purpose | Concise index of each module with responsibility and key file locations |
| Sections | 1) Module Overview (table: Module / Path / Responsibility / Key Files) 2) Module Details (for each module: responsibility statement, key files table, core execution paths, inter-module dependencies) |
| Content Guidelines | Module Overview table must list every module/sub-project in the repository. Each Module Details subsection must include a responsibility statement (1-2 sentences), a key files table (File / Responsibility), a description of core execution paths, and explicit inter-module dependencies. |
| Quality Checks | Every module is listed in the overview table; Key files table is consistent across overview and details; Core execution paths are described for each module; Inter-module dependencies are explicit and accurate |

### Module Overview Table Format

```markdown
| Module | Path | Responsibility | Key Files |
|--------|------|---------------|-----------|
| @cspadk/cli | packages/cli/ | CLI entry point for toolkit commands | src/index.ts, src/commands/ |
| @i18n-cs/ai-toolkit-core | packages/ai-toolkit-core/ | Core utilities for skill linting | src/skill-lint.ts, src/index.ts |
```

### Module Details Subsection Format

```markdown
### @cspadk/cli

**Responsibility**: CLI for the CSPADK toolkit, providing project init, skill management, and requirement tracking.

**Key Files**:

| File | Responsibility |
|------|---------------|
| src/index.ts | CLI registration and startup |
| src/commands/init.ts | Project initialization command |
| src/core/ReqTracker.ts | Requirement state management |

**Core Execution Paths**: `src/index.ts` → registers commands → delegates to `src/commands/<cmd>.ts` → calls `src/core/` business logic.

**Dependencies**: @i18n-cs/ai-toolkit-core (internal), cac (external)
```
