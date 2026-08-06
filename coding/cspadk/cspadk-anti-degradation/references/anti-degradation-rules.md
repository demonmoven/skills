# Anti-Degradation Rules

This document defines the engineering guardrails and quality standards used by the anti-degradation scanner. These rules are repository-agnostic and apply across different tech stacks.

---

## Rule Categories

| Category | Prefix | Focus |
|----------|--------|-------|
| Security | SEC | Vulnerabilities, injection, data exposure |
| Architecture | ARCH | Structure, layering, dependencies |
| Complexity | COMP | Size, nesting, duplication |
| Testing | TEST | Coverage, organization, quality |

---

## 1. Security Rules

### SEC-001: Command Injection Prevention

**Severity:** CRITICAL
**Category:** Security

**Description:** Shell command execution must not interpolate unsanitized user input.

**Rationale:** Unsanitized input in shell commands allows arbitrary code execution.

**Detection Patterns:**

| Language | Pattern | Risk |
|----------|---------|------|
| TypeScript | `execSync(\`...\${var}\`)` | HIGH |
| TypeScript | `execFileSync(cmd, [args])` | SAFE |
| Python | `os.system(f"...{var}")` | HIGH |
| Python | `subprocess.run([cmd, *args])` | SAFE |
| Go | `exec.Command("sh", "-c", str)` | HIGH |
| Go | `exec.Command(cmd, args...)` | SAFE |

**Remediation:**
1. Use array-based argument passing
2. Escape all interpolated values
3. Validate input against whitelist patterns

**Example Fix:**
```typescript
// BAD
execSync(`git checkout ${branchName}`);

// GOOD
execFileSync('git', ['checkout', branchName]);

// GOOD (with validation)
if (!/^[a-zA-Z0-9_/-]+$/.test(branchName)) {
  throw new Error('Invalid branch name');
}
execFileSync('git', ['checkout', branchName]);
```

---

### SEC-002: Path Traversal Prevention

**Severity:** HIGH
**Category:** Security

**Description:** File paths from user input must be validated before use.

**Rationale:** Unvalidated paths allow reading/writing files outside intended directories.

**Detection Patterns:**

| Pattern | Risk |
|---------|------|
| `fs.readFile(userInput)` | CRITICAL |
| `path.join(base, userInput)` without validation | HIGH |
| `os.Open(userInput)` | CRITICAL |

**Remediation:**
```typescript
// Validate path doesn't escape base directory
function safePath(base: string, input: string): string {
  // Validate input format
  if (!/^[a-zA-Z0-9_.-]+$/.test(input)) {
    throw new Error('Invalid path component');
  }
  
  const resolved = path.resolve(base, input);
  if (!resolved.startsWith(path.resolve(base))) {
    throw new Error('Path traversal detected');
  }
  return resolved;
}
```

---

### SEC-003: Sensitive Data Exposure

**Severity:** MEDIUM
**Category:** Security

**Description:** No sensitive data should be logged, hardcoded, or exposed.

**Detection:**
- Hardcoded patterns: `api_key`, `password`, `secret`, `token`, `credential`
- Logging of request bodies, headers, or parameters
- Secrets in configuration files (unless encrypted)

**Remediation:**
- Use environment variables for secrets
- Redact sensitive fields before logging
- Use secret management systems

---

### SEC-004: Input Validation Gaps

**Severity:** MEDIUM
**Category:** Security

**Description:** All external inputs must be validated before use.

**Detection:**
- Function parameters from external sources without validation
- API request bodies without schema validation
- Query parameters used directly in operations

**Remediation:**
```typescript
// Use schema validation
const schema = z.object({
  id: z.string().regex(/^[a-zA-Z0-9_-]+$/),
  count: z.number().int().min(1).max(100),
});
const validated = schema.parse(input);
```

---

## 2. Architecture Rules

### ARCH-001: God Object Prevention

**Severity:** HIGH
**Category:** Architecture

**Description:** Classes/modules should follow Single Responsibility Principle.

**Thresholds:**

| Metric | Warning | Failure |
|--------|---------|---------|
| Lines of code | >300 | >500 |
| Public methods | >10 | >15 |
| Responsibilities | >2 | >3 |

**Remediation:**
- Extract focused classes/modules
- Apply composition over inheritance
- Use facade pattern for complex operations

---

### ARCH-002: Layering Integrity

**Severity:** MEDIUM
**Category:** Architecture

**Description:** Dependencies must follow layer hierarchy.

**Layer Hierarchy:**
```
Presentation (handlers, controllers, CLI)
    ↓
Business Logic (services, core)
    ↓
Data Access (repositories, stores)
    ↓
Utilities (helpers, logging)
```

**Violations:**
- Core importing from handlers
- Utilities importing from services
- Data access importing from presentation

**Remediation:**
- Move shared types to dedicated layer
- Use dependency injection
- Apply interface segregation

---

### ARCH-003: Type Duplication Prevention

**Severity:** MEDIUM
**Category:** Architecture

**Description:** Types should have a single source of truth.

**Detection:**
- Identical type definitions in multiple files
- Similar types that could be unified
- Re-defined types instead of imports

**Remediation:**
- Create centralized `types/` directory
- Export from single location
- Use shared type packages

---

### ARCH-004: Dead Code Elimination

**Severity:** LOW
**Category:** Architecture

**Description:** Remove unused code and exports.

**Detection:**
- Exports never imported elsewhere
- Commented-out code blocks
- Deprecated functions still present
- Unused dependencies

**Remediation:**
- Remove unused exports
- Delete commented code
- Archive deprecated functions

---

### ARCH-005: Circular Dependency Prevention

**Severity:** MEDIUM
**Category:** Architecture

**Description:** Modules should not have circular dependencies.

**Detection:**
- Module A imports B, B imports A
- Longer cycles (A→B→C→A)

**Remediation:**
- Extract shared code to separate module
- Use dependency injection
- Apply inversion of control

---

## 3. Complexity Rules

### COMP-001: File Size Limits

**Severity:** MEDIUM
**Category:** Complexity

**Thresholds:**

| Lines | Status |
|-------|--------|
| <300 | OK |
| 300-500 | WARNING |
| >500 | FAILURE |

**Exemptions:**
- Test files
- Template/data files (mostly string literals)
- Generated code

---

### COMP-002: Function Length Limits

**Severity:** MEDIUM
**Category:** Complexity

**Thresholds:**

| Lines | Status |
|-------|--------|
| <40 | OK |
| 40-80 | WARNING |
| >80 | FAILURE |

**Remediation:**
- Extract helper functions
- Apply single responsibility
- Use early returns

---

### COMP-003: Nesting Depth Limits

**Severity:** LOW
**Category:** Complexity

**Thresholds:**

| Indentation Levels | Status |
|-------------------|--------|
| <3 | OK |
| 3-5 | WARNING |
| >5 | FAILURE |

**Remediation:**
```typescript
// BAD: 4 levels of nesting
function process(data) {
  if (data) {
    if (data.items) {
      for (const item of data.items) {
        if (item.active) {
          // do something
        }
      }
    }
  }
}

// GOOD: Early returns
function process(data) {
  if (!data?.items) return;
  
  for (const item of data.items) {
    if (!item.active) continue;
    // do something
  }
}
```

---

### COMP-004: Code Duplication Prevention

**Severity:** MEDIUM
**Category:** Complexity

**Description:** Avoid duplicating code across files.

**Thresholds:**
- 6+ consecutive duplicate lines
- 70%+ similarity in code blocks

**Remediation:**
- Extract to shared utilities
- Apply DRY principle
- Use composition

---

### COMP-005: Cyclomatic Complexity

**Severity:** MEDIUM
**Category:** Complexity

**Thresholds:**

| Branch Count | Status |
|--------------|--------|
| <10 | OK |
| 10-20 | WARNING |
| >20 | FAILURE |

**Remediation:**
- Simplify conditional logic
- Extract decision tables
- Use polymorphism

---

## 4. Testing Rules

### TEST-001: Test Coverage

**Severity:** MEDIUM
**Category:** Testing

**Metrics:**

| Metric | Target |
|--------|--------|
| Test/Source Ratio | >0.8 |
| Branch Coverage | >70% |
| Public Function Coverage | >80% |

**Remediation:**
- Add tests for uncovered functions
- Cover edge cases and error paths
- Test critical business logic first

---

### TEST-002: Test Organization

**Severity:** LOW
**Category:** Testing

**Standards:**
- Tests in dedicated `test/` or `__tests__/` directory
- Test files mirror source structure
- Consistent naming: `*.test.ts`, `*_test.go`, `Test*.java`

---

### TEST-003: Test Quality

**Severity:** LOW
**Category:** Testing

**Detection:**
- Empty test bodies
- Skipped tests (`it.skip`, `t.Skip`)
- Tests without assertions
- Tests that always pass

**Remediation:**
- Remove or implement empty tests
- Fix skipped tests
- Add meaningful assertions

---

## Severity Escalation

When multiple related issues exist:

| Scenario | Action |
|----------|--------|
| 3+ LOW issues in same area | Escalate to MEDIUM |
| 3+ MEDIUM issues in same area | Escalate to HIGH |
| Any CRITICAL issue | Block immediately |

## Scan Configuration

Default configuration for anti-degradation scans:

```yaml
anti-degradation:
  output_path: docs/anti-degradation/
  severity_threshold: LOW
  categories:
    - security
    - architecture
    - complexity
    - testing
  exemptions:
    - "**/*.test.ts"
    - "**/__tests__/**"
    - "**/templates/**"
    - "**/generated/**"
```
