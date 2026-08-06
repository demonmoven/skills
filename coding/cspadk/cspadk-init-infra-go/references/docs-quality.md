# Quality Documentation Templates

## 1. quality/testing-strategy.md — Testing Strategy & Coverage Requirements

| Field | Content |
|-------|---------|
| Purpose | Define testing approach, tooling, coverage requirements, and best practices |
| Sections | 1) Testing Philosophy 2) Test Types (unit/integration/E2E) 3) Tooling (table: Tool / Version / Purpose) 4) Coverage Requirements (table: Module Type / Line / Branch) 5) Test Conventions 6) Running Tests |
| Content Guidelines | Testing philosophy must state the team's approach clearly. Test types must describe when to use each. Tooling table must list all test-related packages. Coverage requirements must specify thresholds by module type. Test conventions must cover naming, file location, and patterns. Running tests must include all common commands. |
| Quality Checks | Testing philosophy is stated; All test types are described; Tooling table is complete; Coverage thresholds are specified by module type; Test conventions are documented; Common test commands are provided |

### Go Tooling Table

| Tool | Version | Purpose |
|------|---------|---------|
| go testing | stdlib | Test runner (built-in) |
| testify | latest | Assertions and suite helpers |
| go.uber.org/mock | latest | Interface mock generation |
| go test -cover | stdlib | Coverage collection |
| net/http/httptest | stdlib | HTTP handler testing |
| golangci-lint | latest | Static analysis including test file checks |

### Coverage Requirements Table

| Module Type | Line Coverage | Branch Coverage |
|-------------|--------------|----------------|
| Core logic (internal/) | >= 60% | >= 50% |
| Utility (pkg/) | >= 80% | >= 70% |
| Handlers (internal/handler/) | >= 50% | >= 40% |

### Test Conventions Example

#### Test Conventions

- **File naming**: `*_test.go` — co-located with source files in the same directory
- **Package naming**: Use `xxx_test` for black-box testing; same package for white-box testing
- **Test function naming**: `Test<FunctionName>`, `Test<FunctionName>_<Scenario>`
- **Table-driven tests**: Preferred for testing multiple scenarios
- **Benchmark naming**: `Benchmark<FunctionName>`
- **Mocking**: Use `go.uber.org/mock` for interface mocks; `testify/mock` for simple mocks

### Running Tests Example

#### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with race detector
go test ./... -race

# Run tests for a specific package
go test ./internal/ticket/...

# Run a specific test
go test ./internal/ticket/... -run TestRouteTicket -v

# Run tests with coverage
go test ./... -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# View coverage by function
go tool cover -func=coverage.out

# Run benchmarks
go test ./... -bench=. -benchmem

# Run only unit tests (exclude integration)
go test ./... -short
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

#### Review Process

1. **Who**: At least one peer reviewer + one senior reviewer for architecture changes
2. **When**: Review requested within 1 business day of MR creation
3. **How**: Use merge request comments; approve only when all comments are resolved
4. **Resolution**: Author addresses comments; reviewer re-reviews after changes
5. **Merge**: Squash merge after all approvals and CI passes

### Review Checklist Example

#### Review Checklist (≤ 15 items)

**Correctness**
- [ ] Code does what the MR description says
- [ ] Error handling follows the `fmt.Errorf("context: %w", err)` pattern
- [ ] Edge cases are handled (nil checks, empty slices, context cancellation)

**Security**
- [ ] No hardcoded secrets or credentials
- [ ] Input validation is in place for all handler inputs
- [ ] SQL queries use parameterized statements (no string concatenation)

**Performance**
- [ ] No goroutine leaks (context cancellation handled)
- [ ] Database queries are optimized (no N+1 queries)
- [ ] Appropriate use of pointers vs values (avoid unnecessary copies)

**Maintainability**
- [ ] Code follows Go naming conventions (PascalCase for exported, camelCase for unexported)
- [ ] Functions are single-responsibility
- [ ] Tests cover the new/changed code
- [ ] No dead code or commented-out code

**Compatibility**
- [ ] Changes are backward-compatible or properly versioned
- [ ] Breaking changes are documented

### Common Issues Example

#### Common Issues with Examples

##### 1. Missing Error Check

```go
// Anti-pattern
result, _ := riskyOperation()

// Fix
result, err := riskyOperation()
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

##### 2. Goroutine Leak

```go
// Anti-pattern — goroutine never exits
go func() {
    for {
        doWork()
    }
}()

// Fix — respect context cancellation
go func(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            doWork()
        }
    }
}(ctx)
```

##### 3. Non-Pointer Receiver on Mutating Method

```go
// Anti-pattern — changes lost because receiver is a copy
func (s Service) Update(name string) {
    s.name = name
}

// Fix — use pointer receiver for mutations
func (s *Service) Update(name string) {
    s.name = name
}
```

### Review Etiquette Example

#### Review Etiquette

- **Be constructive**: Frame feedback as suggestions, not commands
- **Explain why**: Provide rationale, not just "change this"
- **Be timely**: Review within 1 business day of request
- **Keep it focused**: Comment on code quality, not personal style preferences
- **Resolve disagreements**: Discuss in comments or synchronous call; escalate to tech lead if needed
- **Acknowledge good work**: Call out well-written code and clever solutions
