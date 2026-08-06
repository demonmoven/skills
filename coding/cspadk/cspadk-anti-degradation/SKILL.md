---
name: cspadk-anti-degradation
description: Generic anti-degradation scanner for security, architecture, complexity, and test discipline across any codebase. Supports auto-fix mode.
version: 1.3.2
metadata:
  patterns:
    - generator
    - reviewer
  domain: generic
  i18n_level: 0
  prompt_version: "1.3.2"
  agent_support:
    - claude-code
    - trae
    - trae-cli
    - coco
  language:
    - en
---

# cspadk-anti-degradation

## Overview

A **generic** anti-degradation scanner that evaluates code quality across five dimensions: Security, Architecture, Complexity, Dead Code, and CI Checks. This skill is repository-agnostic and produces structured reports to help teams maintain engineering standards.

**New in v1.3.0:** Enhanced class refactoring rules with composition pattern examples.

**New in v1.2.0:** Supports `--fix` mode for automated remediation of certain issue types.

## Trigger Scenarios

- Before merge of substantial changes
- After AI-generated code modifications
- Periodic codebase health audits
- CI/CD quality gate integration
- Pre-release validation
- **Auto-fix mode**: `--fix` to automatically remediate detected issues

## Inputs

- **Target**: Files, directories, or entire repository to scan
- **Output Path**: Directory for the report (default: `docs/anti-degradation/`)
- **Focus Areas** (optional): Security, Architecture, Complexity, DeadCode, CI, Tests
- **Severity Threshold**: minimum severity to report (default: LOW)
- **--fix**: Enable auto-fix mode for supported issue types

## Outputs

- Structured markdown report saved to specified output directory
- Filename format: `{repository-name}-diagnostic-report-{date}.md`
- Summary of findings by category and severity
- Actionable remediation recommendations

## Execution Steps

### Step 1: Setup and Discovery

1. **Determine repository context**:
   - Identify project name from `package.json` or directory name
   - Detect tech stack (TypeScript, Go, Java, etc.)
   - Identify source directories (`src/`, `lib/`, `pkg/`, etc.)
2. **Establish output location**:
   - Create `docs/anti-degradation/` if not exists
   - Generate report filename: `{project-name}-diagnostic-report-{YYYY-MM-DD}.md`

### Step 2: Security Scan (SEC-\*)

Execute security checks in priority order:

#### SEC-001: Command Injection Prevention

**Severity:** CRITICAL
**Auto-Fix:** YES (P0)

Scan for unsafe shell command execution patterns:

| Language      | Pattern to Detect                                                           |
| ------------- | --------------------------------------------------------------------------- |
| TypeScript/JS | `execSync(\`...${...\`...\`)`, ` exec(\`...${...\`)\`                       |
| Python        | `os.system(f"...{...}")`, `subprocess.call(..., shell=True)` with f-strings |
| Go            | `exec.Command("sh", "-c", fmt.Sprintf(...))`                                |

**Safe alternatives:**

- Use array arguments: `execFileSync('cmd', [args])`
- Escape values: `'${value.replace(/'/g, "'\\''")}'`

**Auto-Fix Implementation:**

```typescript
// Before (vulnerable)
execSync(`git checkout ${branchName}`);

// After (safe)
execFileSync('git', ['checkout', branchName]);
```

**Steps:**

1. Import `execFileSync` if not already imported
2. Replace `execSync(\`cmd ${var}\`)`with`execFileSync('cmd', \[var])\`
3. Update `stdio` option if needed
4. Run typecheck and tests to verify

#### SEC-002: Path Traversal Prevention

**Severity:** HIGH

Check file path construction from user input:

| Pattern                      | Risk                  |
| ---------------------------- | --------------------- |
| `path.join(base, userInput)` | HIGH if no validation |
| `fs.readFile(userInput)`     | CRITICAL              |
| `os.Open(userInput)`         | CRITICAL              |

**Required fix:** Validate with whitelist regex before path operations.

#### SEC-003: Sensitive Data Exposure

**Severity:** MEDIUM

Check for:

- Hardcoded credentials (API keys, tokens, passwords)
- Logging of sensitive parameters
- Secrets in configuration files

#### SEC-004: Input Validation Gaps

**Severity:** MEDIUM

Identify:

- Missing validation on external inputs
- Unsanitized data in dangerous operations
- Missing bounds checking

### Step 3: Architecture Scan (ARCH-\*)

#### ARCH-001: God Object Detection

**Severity:** HIGH
**Auto-Fix:** YES (P2 - with planning)

**Thresholds:**

| Metric            | Warning | Failure |
| ----------------- | ------- | ------- |
| Lines per file    | >300    | >500    |
| Methods per class | >10     | >15     |
| Responsibilities  | >2      | >3      |

**Detection:**

- Count lines and methods per file/class
- Identify classes handling multiple concerns
- Flag violations for refactoring

**Class Refactoring Strategy:**

When a class or file exceeds thresholds, apply the following refactoring pattern:

##### Step 1: Responsibility Analysis

1. **Identify distinct responsibilities:**
   - Group methods by their concern area
   - Identify data/state that belongs to each concern
   - Map dependencies between concerns
2. **Classify method types:**
   - **Persistence**: save, load, delete, exists
   - **Business Logic**: calculate, validate, transform
   - **Orchestration**: coordinate multiple operations
   - **Query**: getters, finders, listers
   - **Mutation**: setters, updaters, modifiers

##### Step 2: Module Extraction Pattern

```
Original File: src/large-module.ts (750 lines, 25 methods)

Extracted Modules:
├── src/module/persistence.ts    - Data storage operations
├── src/module/validators.ts     - Validation logic
├── src/module/transformers.ts   - Data transformation
├── src/module/queries.ts        - Query/fetch operations
└── src/module/index.ts          - Orchestrator class (composition)
```

##### Step 3: Composition Pattern

**Before (God Object):**

```typescript
// req-tracker.ts - 400 lines, 20 methods
export class ReqTracker {
  private reqDir: string;
  private currentId: string | null = null;

  // Persistence methods
  exists(id: string): boolean { ... }
  delete(id: string): void { ... }
  save(id: string, data: object): void { ... }
  load(id: string): object { ... }

  // Module management methods
  addModule(id: string, title: string): void { ... }
  updateModuleStatus(id: string, status: string): void { ... }
  findModule(id: string): Module { ... }

  // Artifact verification methods
  verifyArtifacts(id: string): Result { ... }
  getMissingArtifacts(id: string): Artifact[] { ... }

  // Orchestration methods
  create(id: string): void { ... }
  getCurrent(): Requirement { ... }
}
```

**After (Composition):**

```typescript
// persistence.ts - Focused on storage
export class RequirementStore {
  constructor(private reqDir: string) {}
  exists(id: string): boolean { ... }
  delete(id: string): void { ... }
  save(id: string, data: Requirement): void { ... }
  load(id: string): Requirement | null { ... }
}

// module-manager.ts - Focused on module operations
export class ModuleManager {
  findModule(req: Requirement, id: string): Module | undefined { ... }
  updateStatus(req: Requirement, id: string, status: string): void { ... }
  add(req: Requirement, id: string, title: string): void { ... }
}

// artifact-verifier.ts - Focused on artifact validation
export class ArtifactVerifier {
  constructor(private pathChecker: (path: string) => boolean) {}
  verify(req: Requirement): VerificationResult { ... }
  getMissingArtifacts(req: Requirement): MissingArtifact[] { ... }
}

// req-tracker.ts - Orchestration only
export class ReqTracker {
  private store: RequirementStore;
  private moduleManager: ModuleManager;
  private artifactVerifier: ArtifactVerifier;
  private currentId: string | null = null;

  constructor(reqDir: string = '.ttadk/requirements-status') {
    this.store = new RequirementStore(reqDir);
    this.moduleManager = new ModuleManager();
    this.artifactVerifier = new ArtifactVerifier((path) => this.store.pathExists(path));
  }

  // Delegates to sub-modules
  exists(id: string): boolean { return this.store.exists(id); }
  addModule(id: string, title: string): void { ... }
  verifyArtifacts(): VerificationResult { ... }
}
```

##### Step 4: Extraction Checklist

For each extracted module:

1. **Create new file** with focused responsibility
2. **Move related methods** to the new class
3. **Identify shared dependencies** and inject them
4. **Update imports** in the main orchestrator
5. **Add re-exports** if needed for backward compatibility
6. **Run typecheck** after each extraction
7. **Run tests** to verify behavior preserved

##### Step 5: Test File Organization

Each extracted module should have its own test file:

```
Tests before:
└── src/core/req-tracker.test.ts (40 tests)

Tests after:
├── src/core/requirement-store.test.ts (10 tests)
├── src/core/module-manager.test.ts (12 tests)
├── src/core/artifact-verifier.test.ts (10 tests)
└── src/core/req-tracker.test.ts (8 tests - orchestration only)
```

**Auto-Fix Implementation:**

1. **Analyze the file structure:**
   - Identify distinct responsibilities
   - Map functions to their concern areas
   - Identify shared dependencies
2. **Generate refactoring plan:**
   ```
   File: src/large-module.ts (750 lines)

   Suggested split:
   - src/module/api.ts (150 lines) - API interactions
   - src/module/transformers.ts (200 lines) - Data transformations
   - src/module/validators.ts (100 lines) - Input validation
   - src/module/index.ts (100 lines) - Orchestration

   Shared imports: utils/logger, types/common
   Breaking changes: None (public API preserved)
   ```
3. **Get user confirmation before proceeding**
4. **Execute refactoring:**
   - Create new files
   - Move functions to appropriate modules
   - Update imports in dependent files
   - Run typecheck and tests

**Refactoring Principles:**

- Single Responsibility Principle: Each class should have one reason to change
- Dependency Injection: Pass dependencies through constructor
- Interface Segregation: Split large interfaces into smaller ones
- Preserve Public API: Keep backward compatibility when possible

#### ARCH-002: Layering Integrity

**Severity:** MEDIUM

**Layer hierarchy (strict):**

```
Presentation → Business Logic → Data Access → Utilities
```

**Check:**

- Lower layers should NOT import from higher layers
- Cross-layer imports indicate violations

#### ARCH-003: Type Duplication

**Severity:** MEDIUM

**Detection:**

- Scan for identical type definitions
- Flag types defined in multiple locations
- Suggest centralized type location

#### ARCH-004: Dead Code Detection

**Severity:** MEDIUM

**Detection:**

- Exports never imported elsewhere (using grep/import analysis)
- Commented-out code blocks (scan for `//` or `/* */` blocks >5 lines)
- Deprecated functions still present
- Unused imports
- Unreachable code paths

**Automated checks:**

```bash
# Find unused exports (TypeScript)
npx ts-prune --unused

# Find commented-out code blocks
grep -r "//.*" --include="*.ts" | wc -l
```

#### ARCH-005: Circular Dependencies

**Severity:** MEDIUM

**Detection:**

- Build dependency graph
- Identify circular import chains
- Report affected modules

### Step 4: Complexity Scan (COMP-\*)

#### COMP-001: File Size

**Severity:** MEDIUM

| Lines   | Status  |
| ------- | ------- |
| <300    | OK      |
| 300-500 | WARNING |
| >500    | FAILURE |

**Exemptions:** Test files, template/data files

#### COMP-002: Function Length

**Severity:** MEDIUM

| Lines | Status  |
| ----- | ------- |
| <40   | OK      |
| 40-80 | WARNING |
| >80   | FAILURE |

#### COMP-003: Nesting Depth

**Severity:** LOW

| Levels | Status  |
| ------ | ------- |
| <3     | OK      |
| 3-5    | WARNING |
| >5     | FAILURE |

#### COMP-004: Code Duplication

**Severity:** MEDIUM

**Detection:**

- Identify duplicate code blocks (>6 lines, >70% similarity)
- Find repeated constants
- Detect copy-paste patterns

#### COMP-005: Cyclomatic Complexity

**Severity:** MEDIUM

| Branches | Status  |
| -------- | ------- |
| <10      | OK      |
| 10-20    | WARNING |
| >20      | FAILURE |

### Step 5: Dead Code Scan (DEAD-\*)

**ALWAYS EXECUTE THIS STEP** - Dead code increases maintenance burden and confusion.

#### DEAD-001: Unused Exports

**Severity:** MEDIUM
**Auto-Fix:** YES (P1)

**Detection:**

- Scan all exported functions/types/classes
- Cross-reference with imports across the codebase
- Flag exports with no consumers

**Automated check:**

```bash
# TypeScript unused exports
npx ts-prune --unused --project tsconfig.json
```

**Auto-Fix Implementation:**

1. For each unused export, determine if it's:
   - **Dead code**: Remove the export entirely
   - **Future use**: Add `// @deprecated - kept for future use` comment
   - **External API**: Add documentation explaining external usage
2. Run typecheck to verify no breakage
3. Run tests to verify functionality

**Example:**

```typescript
// Before
export function unusedHelper() { ... }

// After (removed)
// (deleted)
```

#### DEAD-002: Commented-Out Code

**Severity:** LOW
**Auto-Fix:** YES (P1)

**Detection:**

- Find large commented blocks (>5 consecutive comment lines)
- Look for code-like patterns in comments
- Identify commented imports or function definitions

**Auto-Fix Implementation:**

1. Identify commented blocks >5 lines
2. Check if block contains code patterns (function signatures, imports, etc.)
3. Delete the commented block
4. Run tests to verify no regression

#### DEAD-003: Unused Imports

**Severity:** LOW

**Detection:**

- Import statements with no usage in file
- TypeScript unused variable warnings

**Automated check:**

```bash
# TypeScript compiler catches these
npx tsc --noEmit
```

#### DEAD-004: Unreachable Code

**Severity:** MEDIUM

**Detection:**

- Code after return/throw/break statements
- Functions never called
- Dead branches (if(false) blocks)

### Step 6: CI Checks (CI-\*)

**ALWAYS EXECUTE THIS STEP** - CI failures indicate broken builds.

#### CI-001: TypeScript Compilation

**Severity:** HIGH

**Check:**

```bash
npm run typecheck  # or npx tsc --noEmit
```

Report all compilation errors.

#### CI-002: Lint Check

**Severity:** MEDIUM

**Check:**

```bash
npm run lint  # or npx eslint .
```

Report lint errors (not warnings unless --strict).

#### CI-003: Test Suite

**Severity:** HIGH

**Check:**

```bash
npm run test  # or npx vitest run / npx jest
```

Report test failures.

#### CI-004: Build Check

**Severity:** HIGH

**Check:**

```bash
npm run build
```

Report build failures.

### Step 7: Test Scan (TEST-\*)

#### TEST-001: Test Coverage

**Severity:** MEDIUM

**Metrics:**

- Test files vs source files ratio (target: >0.8)
- Uncovered public functions
- Missing edge case tests

#### TEST-002: Test Organization

**Severity:** LOW

**Check:**

- Tests in dedicated `test/` or `__tests__/` directory
- Test files mirror source structure
- Consistent naming convention

#### TEST-003: Test Quality

**Severity:** LOW

**Detection:**

- Empty test bodies
- Skipped tests
- Tests without assertions

### Step 8: Generate Report

Create the diagnostic report with this structure:

```markdown
# {Project Name} Diagnostic Report

**Generated:** {ISO timestamp}
**Tech Stack:** {detected stack}
**Scope:** {scanned directories}

## Executive Summary

| Category | Issues Found | Critical | High | Medium | Low |
|----------|-------------|----------|------|--------|-----|
| Security | X | X | X | X | X |
| Architecture | X | X | X | X | X |
| Complexity | X | - | X | X | X |
| Dead Code | X | - | - | X | X |
| CI Checks | X | - | X | X | - |
| Testing | X | - | - | X | X |

**Overall Verdict:** [PASS | PASS WITH FOLLOW-UPS | FAIL]

---

## 1. Security Analysis

### 1.1 {Issue Title} ({SEVERITY})

**File:** `{path}:{line}`

```{language}
// Problematic code
```

**Issue:** {description}
**Risk:** {risk explanation}
**Recommendation:** {specific fix}

---

## 2. Architecture Analysis

{Similar structure for each category}

---

## 3. Complexity Analysis

{Similar structure}

---

## 4. Dead Code Analysis

{Similar structure}

---

## 5. CI Checks Analysis

{Similar structure}

---

## 6. Test Analysis

{Similar structure}

---

## 7. Recommended Actions

### Immediate (P0)

1. {Critical fixes}

### Short-term (P1)

1. {High priority fixes}

### Medium-term (P2)

1. {Medium priority fixes}

### Long-term (P3)

1. {Low priority items}

---

## Appendix: Metrics

| Metric                   | Value | Target | Status   |
| ------------------------ | ----- | ------ | -------- |
| Max File Lines           | X     | <500   | {status} |
| Max Function Lines       | X     | <80    | {status} |
| Test/Source Ratio        | X     | >0.8   | {status} |
| Security Vulnerabilities | X     | 0      | {status} |
| Dead Code Exports        | X     | 0      | {status} |
| CI Failures              | X     | 0      | {status} |

```

### Step 9: Save Report

1. Write report to `{output_path}/{project-name}-diagnostic-report-{date}.md`
2. Print summary to console
3. Return path to generated report

### Step 10: Auto-Fix Mode (when --fix is enabled)

When `--fix` flag is provided, automatically remediate detected issues in priority order:

#### 10.1 Auto-Fixable Issues

| Issue Code | Issue Type | Auto-Fix Strategy |
|------------|------------|-------------------|
| SEC-001 | Command Injection | Replace `execSync(\`...\`)` with `execFileSync('cmd', [args])` |
| DEAD-001 | Unused Exports | Remove unused exports or add `// @deprecated` comment |
| DEAD-002 | Commented-Out Code | Delete large commented blocks (>5 lines) |
| DEAD-003 | Unused Imports | Remove unused import statements |
| ARCH-001 | God Object | Split file into smaller modules (requires confirmation) |
| COMP-001 | File Size | Refactor into smaller files (requires confirmation) |

#### 10.2 Auto-Fix Execution Order

1. **P0 Critical (Immediate)**:
   - SEC-001: Command injection fixes
   - Apply fixes, run typecheck, run tests

2. **P1 High Priority (With Confirmation)**:
   - DEAD-001: Remove unused exports
   - DEAD-002: Remove commented-out code
   - DEAD-003: Remove unused imports
   - Apply fixes, run typecheck, run tests

3. **P2 Medium Priority (Requires Planning)**:
   - ARCH-001: God Object refactoring
   - COMP-001: File size reduction
   - Generate refactoring plan, require explicit user approval

#### 10.3 Auto-Fix Workflow

```

For each auto-fixable issue:

1. Backup original file (optional: create git stash)
2. Apply the fix
3. Run typecheck
4. Run lint
5. Run tests
6. If any check fails:
   - Revert the change
   - Report failure with details
7. If all checks pass:
   - Report success
   - Add to summary

After all fixes:

1. Run full test suite
2. Generate before/after summary
3. List remaining manual fixes needed
```

#### 10.4 Auto-Fix Output Format

```markdown
## Auto-Fix Summary

### Applied Fixes (X of Y successful)

| Issue | File | Fix Applied | Status |
|-------|------|-------------|--------|
| SEC-001 | src/commands/init.ts | execSync → execFileSync | ✅ Pass |
| DEAD-001 | src/utils.ts | Removed unused `foo` export | ✅ Pass |
| DEAD-003 | src/index.ts | Removed 2 unused imports | ✅ Pass |

### Requiring Manual Fix

| Issue | File | Reason |
|-------|------|--------|
| ARCH-001 | src/large-file.ts | Requires architectural planning |

### Verification Results

- TypeScript: ✅ Pass
- Lint: ✅ Pass
- Tests: ✅ 203 passed

### Remaining Issues

1. **ARCH-001**: `src/large-file.ts` (500+ lines) - Requires refactoring
   - Suggested modules: `module-a.ts`, `module-b.ts`
   - Estimated effort: 2 hours
```

## Severity Classification

| Severity | Criteria                                       | Action                        | Auto-Fix     |
| -------- | ---------------------------------------------- | ----------------------------- | ------------ |
| CRITICAL | Security vulnerability, data loss risk         | Blocks merge, fix immediately | Yes (P0)     |
| HIGH     | Architectural violation, significant tech debt | Fix within sprint             | Planned (P2) |
| MEDIUM   | Complexity threshold exceeded, minor issues    | Fix next sprint               | Yes (P1)     |
| LOW      | Documentation gaps, style inconsistencies      | Technical debt backlog        | Yes (P1)     |

## Tech Stack Adaptations

The scanner adapts rules based on detected tech stack:

| Stack         | Security Patterns                    | Architecture Patterns |
| ------------- | ------------------------------------ | --------------------- |
| TypeScript/JS | execSync, eval, prototype pollution  | Classes, modules      |
| Go            | exec.Command, os.Open, filepath.Join | Packages, interfaces  |
| Java          | Runtime.exec, FileInputStream        | Classes, packages     |
| Python        | os.system, subprocess, eval          | Modules, classes      |

## Reference Files

- [Anti-Degradation Rules](./references/anti-degradation-rules.md) - Complete rule definitions

## Output Language

Detect the user's input language and respond in the same language.

- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).

