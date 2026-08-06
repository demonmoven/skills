# Constraints Documentation Templates

## 1. constraints/repository-constraints.md — Repository Constraint Specifications

| Field | Content |
|-------|---------|
| Purpose | Document mandatory low-level constraints: dependency management, directory structure, CI/CD, and security |
| Sections | 1) Dependency Management 2) Directory Structure Rules 3) CI/CD Requirements 4) Security Constraints 5) Repository-Specific Rules |
| Content Guidelines | Every constraint must be prefixed with MUST / MUST NOT / SHOULD. Each constraint must include a rationale explaining why it exists. Each constraint must provide a validation command that can be run to check compliance. |
| Quality Checks | Every constraint has a MUST/MUST NOT/SHOULD prefix; Rationale is provided for each constraint; Validation command is provided for each constraint |

### Go-Specific Constraint Examples

#### Dependency Management

- **MUST** use Go Modules (`go.mod` must exist). Rationale: Go Modules is the standard dependency management system; ensures reproducible builds. Validation: `test -f go.mod && echo PASS || echo FAIL`
- **MUST** configure `GOPRIVATE=code.byted.org`. Rationale: Internal packages must be fetched through the company proxy, not the public proxy. Validation: `go env GOPRIVATE | grep code.byted.org`
- **MUST NOT** commit `vendor/` directory unless explicitly required. Rationale: Go Modules provides reproducible builds without vendoring; vendor directory bloats the repository. Validation: `test -d vendor && echo 'FAIL' || echo 'PASS'`
- **MUST** run `go mod tidy` before every commit. Rationale: Ensures `go.mod` and `go.sum` are clean and consistent. Validation: `go mod tidy && git diff --exit-code go.mod go.sum`

#### Directory Structure Rules

- **MUST** place internal packages under `internal/`. Rationale: Go's `internal` convention prevents external imports. Validation: `find . -name '*.go' -not -path './internal/*' -not -path './pkg/*' -not -path './cmd/*' | head`
- **MUST** place public libraries under `pkg/`. Rationale: `pkg/` convention signals importability by external projects. Validation: `ls pkg/`
- **MUST** place entry points under `cmd/`. Rationale: Standard Go project layout for multiple binaries. Validation: `ls cmd/`

#### CI/CD Requirements

- **MUST** include lint, test, and build stages in CI pipeline. Rationale: Prevents merging code that fails quality checks. Validation: `grep -E 'lint|test|build' .codebase/pipelines/*.yml`
- **MUST** run `golangci-lint` with zero errors before merging. Rationale: Enforces code quality standards consistently. Validation: `golangci-lint run ./...`
- **MUST** cache Go modules in CI. Rationale: Reduces build times significantly. Validation: `grep -E 'cache|go\\.pkg' .codebase/pipelines/*.yml`

#### Security Constraints

- **MUST NOT** commit secrets, tokens, or credentials to the repository. Rationale: Prevents credential leakage. Validation: `git log --all --diff-filter=A -- '*.env' '*.key' '*.pem'`
- **MUST** use environment variables for configuration. Rationale: Separates config from code. Validation: `grep -r 'hardcoded' internal/ || echo 'PASS'`
- **MUST NOT** use `unsafe` package without explicit approval. Rationale: `unsafe` bypasses Go's type safety guarantees. Validation: `grep -r 'import "unsafe"' internal/ pkg/ || echo 'PASS'`

#### Repository-Specific Rules

- **SHOULD** follow the `handler → service → repository` layered architecture. Rationale: Separation of concerns enables independent testing and evolution. Validation: `ls internal/handler internal/service internal/repository`
- **MUST** pass `go vet ./...` with zero warnings. Rationale: `go vet` catches common programming errors. Validation: `go vet ./...`

---

## 2. constraints/development-guidelines.md — Development & Version Control Guidelines

| Field | Content |
|-------|---------|
| Purpose | Coding standards, branching strategy, commit conventions, and release process |
| Sections | 1) Coding Standards (with examples) 2) Branching Strategy (naming format) 3) Commit Conventions (message template) 4) Release Process (includes rollback) |
| Content Guidelines | Coding standards must include concrete code examples. Branch naming must define the exact format with examples. Commit conventions must provide a message template. Release process must include rollback procedure. |
| Quality Checks | Coding standards have code examples; Branch naming format is defined; Commit message template is provided; Rollback procedure is documented |

### Coding Standards Example

#### Go Coding Standards

- Run `gofmt` before committing — no unformatted code
- Run `goimports` for import ordering — stdlib → internal → third-party
- Error handling: always check `err` return value, never discard with `_`
- Naming: exported names use PascalCase; unexported use camelCase; acronyms are all caps

```go
// Preferred
func (s *TicketService) RouteTicket(ctx context.Context, id string) (*Ticket, error) {
    ticket, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("route ticket: %w", err)
    }
    return ticket, nil
}

// Avoid
func (s ticketService) route_ticket(ctx context.Context, id string) (*Ticket, error) {
    ticket, _ := s.repo.FindByID(ctx, id) // Missing error check
    return ticket, nil
}
```

### Branch Naming Example

#### Branch Naming Convention

Format: `<type>/<ticket-id>-<short-description>`

Types: `feat/`, `fix/`, `chore/`, `refactor/`, `docs/`

Examples:
- `feat/csp-123-add-ticket-routing`
- `fix/csp-456-auth-middleware-error`
- `chore/csp-789-update-go-version`

### Commit Message Template

#### Commit Conventions

Format:

```
<type>(<scope>): <subject>

<body>
```

Types: `feat`, `fix`, `chore`, `refactor`, `docs`, `test`, `ci`

Example:

```
feat(handler): add ticket creation endpoint

Implement POST /api/tickets endpoint with request validation
and automatic routing upon creation.
```

### Release Process Example

#### Release Process

1. Create release branch from `main`: `release/v<version>`
2. Run full test suite: `go test ./... -race`
3. Run lint: `golangci-lint run ./...`
4. Update version in code if applicable
5. Merge release branch to `main` via MR
6. Tag the merge commit: `v<version>`

#### Rollback Procedure

1. Identify the problematic commit or release tag
2. Create a revert branch from `main`: `fix/revert-v<version>`
3. Run `git revert <commit-sha>` on the revert branch
4. Merge revert MR with fast-track review
5. Verify rollback in staging before production deploy

---

## 3. constraints/constraint-validation.md — Constraint Validation Guide

| Field | Content |
|-------|---------|
| Purpose | Checklist and commands to validate constraint compliance |
| Sections | 1) Automated Checks (exact commands) 2) Manual Checklist 3) CI Validation 4) Remediation Guide |
| Content Guidelines | Automated checks must provide exact copy-paste commands. Manual checklist must cover constraints not easily automated. CI validation must describe how the pipeline enforces constraints. Remediation guide must provide step-by-step fixes for common violations. |
| Quality Checks | Every constraint has a validation step; Exact commands are provided; Remediation guide covers common violations |

### Automated Checks Example

#### Dependency Management Validation

| Check | Command | Expected Result |
|-------|---------|-----------------|
| Go Modules initialized | `test -f go.mod && echo PASS` | PASS |
| Go version meets minimum | `go version | grep -E '1\.(2[1-9]|[3-9])'` | Match found |
| GOPRIVATE configured | `go env GOPRIVATE | grep code.byted.org` | Match found |
| Dependencies clean | `go mod tidy && git diff --exit-code go.mod go.sum` | No diff |
| Dependencies verified | `go mod verify` | All modules verified |
| No unused imports | `goimports -l .` | No output |

#### Lint and Format Validation

| Check | Command | Expected Result |
|-------|---------|-----------------|
| Code formatted | `gofmt -l .` | No output (all files formatted) |
| Imports sorted | `goimports -l .` | No output |
| Lint passes | `golangci-lint run ./...` | Exit code 0 |
| Vet passes | `go vet ./...` | Exit code 0 |

### Manual Checklist Example

#### Manual Checklist

- [ ] No hardcoded secrets or credentials in source code
- [ ] CI pipeline includes lint, test, and build stages
- [ ] Environment variables are used for all configuration
- [ ] Branch naming follows `<type>/<ticket-id>-<description>` format
- [ ] Commit messages follow the conventional commit template
- [ ] Error handling follows the `fmt.Errorf("context: %w", err)` pattern

### CI Validation Example

#### CI Validation

The CI pipeline enforces constraints at merge request time:

1. **Lint Stage**: Runs `golangci-lint run ./...` — fails on lint errors
2. **Test Stage**: Runs `go test ./... -race -coverprofile=coverage.out` — fails on test failures
3. **Build Stage**: Runs `go build ./...` — fails on build errors
4. **Coverage Check**: Verifies coverage meets minimum thresholds

### Remediation Guide Example

#### Common Violations and Remediation

##### GOPRIVATE not configured

1. Run `go env GOPRIVATE` to check current value
2. Set in shell profile: `export GOPRIVATE=code.byted.org`
3. Also set: `export GONOSUMCHECK=code.byted.org`
4. Run `go mod download` to verify internal packages are accessible
5. Verify: `go env GOPRIVATE` returns `code.byted.org`

##### go.mod is not tidy

1. Run `go mod tidy` to clean up unused dependencies
2. Review the diff: `git diff go.mod go.sum`
3. Commit the updated files
4. Verify: `go mod tidy && git diff --exit-code go.mod go.sum`

##### Code not formatted with gofmt

1. Run `gofmt -w .` to format all files in place
2. Run `goimports -w .` to sort imports
3. Verify: `gofmt -l .` returns no output
