# Quality Documentation Templates

## 1. quality/testing-strategy.md — Testing Strategy & Coverage Requirements

| Field | Content |
|-------|---------|
| Purpose | Define testing approach, tooling, coverage requirements, and best practices |
| Sections | 1) Testing Philosophy 2) Test Types (unit/integration/E2E) 3) Tooling (table: Tool / Version / Purpose) 4) Coverage Requirements (table: Module Type / Line / Branch) 5) Test Conventions 6) Running Tests |
| Content Guidelines | Testing philosophy must state the team's approach clearly. Test types must describe when to use each. Tooling table must list all test-related packages. Coverage requirements must specify thresholds by module type. Test conventions must cover naming, file location, and patterns. Running tests must include all common commands. |
| Quality Checks | Testing philosophy is stated; All test types are described; Tooling table is complete; Coverage thresholds are specified by module type; Test conventions are documented; Common test commands are provided |

### TS/React Tooling Table

```markdown
| Tool | Version | Purpose |
|------|---------|---------|
| Vitest | 1.x | Test runner |
| @testing-library/react | 14.x | Component testing |
| @vitest/coverage-v8 | latest | Coverage collection |
| msw | latest | API mocking |
| jsdom | latest | DOM environment for component tests |
```

### Coverage Requirements Table

```markdown
| Module Type | Line Coverage | Branch Coverage |
|-------------|--------------|----------------|
| Core / Utility | >= 80% | >= 75% |
| Feature / Component | >= 60% | >= 50% |
| Integration | >= 50% | >= 40% |
```

### Test Conventions Example

```markdown
### Test Conventions

- **File naming**: `*.test.ts` or `*.test.tsx` — co-located with source files
- **Directory alternative**: `__tests__/` for centralized test organization
- **Describe blocks**: Group by feature or component name
- **Test naming**: Use `it('should <expected behavior> when <condition>')` pattern
- **Mocking**: Use `vi.mock()` for module mocks, `vi.fn()` for function mocks
- **Async testing**: Use `async/await` with `vi.useFakeTimers()` for time-dependent tests
```

### Running Tests Example

```markdown
### Running Tests

```bash
# Run all tests
npm run test

# Run tests in watch mode
npm run test -- --watch

# Run a single test file
npx vitest run src/utils/format.test.ts

# Run tests with coverage
npm run test -- --coverage

# Run only unit tests
npx vitest run --exclude "**/*.integration.test.*"

# Run only integration tests
npx vitest run "**/*.integration.test.*"

# Run tests for a specific package
cd packages/cli && npm run test

# Update snapshots
npm run test -- --update
```
```

---

## 2. quality/code-review-guide.md — Code Review Guide

| Field | Content |
|-------|---------|
| Purpose | Code review standards, checklist, and process |
| Sections | 1) Review Process (who/when/how) 2) Review Checklist (correctness, security, performance, maintainability) 3) Common Issues (with examples) 4) Review Etiquette |
| Content Guidelines | Review process must define who reviews, when reviews happen, and how they are conducted. Checklist must be concise — no more than 15 items. Common issues must include code examples showing the anti-pattern and the fix. Review etiquette must cover tone, timeliness, and resolution of disagreements. |
| Quality Checks | Checklist has ≤ 15 items; Common issues include code examples; Review process is clearly defined; Review etiquette is documented |

### Review Process Example

```markdown
### Review Process

1. **Who**: At least one peer reviewer + one senior reviewer for architecture changes
2. **When**: Review requested within 1 business day of MR creation
3. **How**: Use merge request comments; approve only when all comments are resolved
4. **Resolution**: Author addresses comments; reviewer re-reviews after changes
5. **Merge**: Squash merge after all approvals and CI passes
```

### Review Checklist Example

```markdown
### Review Checklist (≤ 15 items)

**Correctness**
- [ ] Code does what the MR description says
- [ ] Edge cases are handled
- [ ] Error handling is appropriate

**Security**
- [ ] No hardcoded secrets or credentials
- [ ] Input validation is in place
- [ ] Dependencies do not introduce known vulnerabilities

**Performance**
- [ ] No unnecessary re-renders (React)
- [ ] No N+1 queries or unnecessary loops
- [ ] Large lists use virtualization

**Maintainability**
- [ ] Code is readable and self-documenting
- [ ] Functions are single-responsibility
- [ ] Types are explicit and accurate
- [ ] Tests cover the new/changed code
- [ ] No dead code or commented-out code

**Compatibility**
- [ ] Changes are backward-compatible or properly versioned
- [ ] Breaking changes are documented
```

### Common Issues Example

```markdown
### Common Issues with Examples

#### 1. Missing Error Handling

```typescript
// ❌ Anti-pattern
async function fetchData() {
  const response = await fetch('/api/data');
  return response.json();
}

// ✅ Fix
async function fetchData() {
  try {
    const response = await fetch('/api/data');
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
    return response.json();
  } catch (error) {
    logger.error('Failed to fetch data', { error });
    throw error;
  }
}
```

#### 2. Implicit Any Type

```typescript
// ❌ Anti-pattern
function processConfig(config) {
  return config.name;
}

// ✅ Fix
interface Config {
  name: string;
  version: number;
}

function processConfig(config: Config): string {
  return config.name;
}
```

#### 3. Unnecessary Re-render

```typescript
// ❌ Anti-pattern — new object reference on every render
function Parent() {
  return <Child style={{ color: 'red' }} />;
}

// ✅ Fix — stable reference
const childStyle = { color: 'red' };

function Parent() {
  return <Child style={childStyle} />;
}
```
```

### Review Etiquette Example

```markdown
### Review Etiquette

- **Be constructive**: Frame feedback as suggestions, not commands
- **Explain why**: Provide rationale, not just "change this"
- **Be timely**: Review within 1 business day of request
- **Keep it focused**: Comment on code quality, not personal style preferences
- **Resolve disagreements**: Discuss in comments or synchronous call; escalate to tech lead if needed
- **Acknowledge good work**: Call out well-written code and clever solutions
```
