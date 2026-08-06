# Development Documentation Templates

## 1. development/getting-started.md — Quick Start Guide

| Field | Content |
|-------|---------|
| Purpose | Help new contributors set up environment and make first contribution |
| Sections | 1) Prerequisites (tools + versions) 2) Environment Setup (copy-paste commands) 3) First Run 4) First Contribution |
| Content Guidelines | Every step must have a copy-paste command. Prerequisites must include version requirements. Common pitfalls must be documented with workarounds. |
| Quality Checks | Every step has a copy-paste command; Prerequisites include version requirements; Common pitfalls are documented |

### Go Example — Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | >= 1.21 | https://go.dev/dl/ |
| golangci-lint | latest | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |
| make | — | System package manager |
| Git | >= 2.x | System package manager |

### Environment Setup

```bash
# 1. Clone the repository
git clone <repo-url> && cd <repo-name>

# 2. Configure internal proxy (ByteDance projects)
export GOPRIVATE=code.byted.org
export GONOSUMCHECK=code.byted.org

# 3. Download dependencies
go mod download

# 4. Build the project
make build

# 5. Run tests
go test ./...
```

### First Run

```bash
# Verify the setup
go version          # >= 1.21
golangci-lint version  # golangci-lint is available
make build          # Build succeeds
go test ./...       # Tests pass
golangci-lint run ./...  # Lint passes
```

### Common Pitfalls

| Pitfall | Cause | Fix |
|---------|-------|-----|
| `GOPRIVATE` not set | Internal packages unreachable | `export GOPRIVATE=code.byted.org` |
| `go mod download` fails | Proxy configuration missing | Check `GOPROXY` and `GOPRIVATE` settings |
| `golangci-lint: command not found` | Not in PATH | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |
| `undefined: xxx` build errors | Dependencies not downloaded | `go mod tidy && go mod download` |
| Permission denied on `go install` | GOBIN not writable | Check `go env GOBIN` and ensure directory exists |

---

## 2. development/workflow.md — Development Workflow Description

| Field | Content |
|-------|---------|
| Purpose | Describe development workflow: feature development, testing, review, and deployment |
| Sections | 1) Development Flow (numbered sequence) 2) Local Development (hot reload) 3) Debugging Guide 4) Testing Workflow (unit + integration) 5) Deployment Flow (includes rollback) |
| Content Guidelines | Development flow must be a clear numbered sequence from feature request to production. Local development must describe hot reload setup. Debugging guide must cover common debugging scenarios. Testing workflow must distinguish unit and integration tests. Deployment flow must include rollback procedure. |
| Quality Checks | Development flow is a numbered sequence; Hot reload configuration is described; Debugging scenarios are covered; Unit and integration tests are distinguished; Rollback procedure is included |

### Development Flow Example

#### Development Flow

1. **Create Feature Branch**: `git checkout -b feat/<ticket-id>-<description>`
2. **Implement Feature**: Write code and tests following TDD when applicable
3. **Run Local Checks**: `make lint && make test && make build`
4. **Push and Create MR**: Push branch and create merge request
5. **Code Review**: Address review feedback
6. **CI Validation**: Ensure all pipeline stages pass
7. **Merge**: Squash merge to main after approval
8. **Deploy**: Follow deployment flow for staging and production

### Local Development

```bash
# Start the service locally
make run

# Or run directly
go run ./cmd/server/

# Watch mode for tests (using air for hot reload)
air

# Run specific test
go test ./internal/ticket/... -run TestRouteTicket -v
```

### Debugging Guide

| Scenario | Tool | Steps |
|----------|------|-------|
| HTTP handler error | curl + logs | `curl -v localhost:8080/api/tickets` and check server logs |
| RPC call failure | Kitex debug logs | Set `LOG_LEVEL=debug` and reproduce the call |
| Database query issue | MySQL client | Connect to DB and run the query directly |
| Race condition | `go test -race` | Run tests with race detector enabled |
| Memory leak | pprof | Add `import _ "net/http/pprof"` and check `/debug/pprof/` |

### Testing Workflow

1. **Write Unit Tests**: Test individual functions and methods in isolation
2. **Write Integration Tests**: Test across layers (handler → service → repository)
3. **Run Full Suite**: `go test ./... -race`
4. **Check Coverage**: `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out`
5. **Fix Failures**: Debug and fix before pushing

### Deployment Flow

1. Code is merged to `main` via approved MR
2. CI pipeline runs full validation (lint, test, build)
3. Docker image is built and pushed to registry
4. Staging deployment is triggered automatically
5. QA verification in staging environment
6. Production deployment via manual promotion
7. Post-deployment smoke test

#### Rollback Procedure

1. Identify the deployment that introduced the issue
2. Revert the merge commit on `main`
3. Trigger a new deployment from the reverted state
4. Verify the fix in staging before promoting to production
5. Document the incident and root cause
