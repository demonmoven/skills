# Go Lint Rules Configuration

## 1. golangci-lint Configuration

CSP Go projects use `golangci-lint` as the unified lint tool.

### 1.1 Installation

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### 1.2 Standard Configuration Template (.golangci.yml)

```yaml
run:
  timeout: 5m
  modules-download-mode: readonly

linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - typecheck
    - gocritic
    - gofmt
    - goimports
    - misspell
    - nilerr
    - prealloc
    - revive
    - unconvert
    - unparam

linters-settings:
  gocritic:
    enabled-tags:
      - diagnostic
      - style
  revive:
    rules:
      - name: exported
      - name: unused-parameter
      - name: unreachable-code
      - name: context-as-argument

issues:
  max-issues-per-linter: 50
  max-same-issues: 5
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
```

### 1.3 Custom Rules

Projects may add additional rules based on business needs, but **must not remove** any rules from the standard set above.

## 2. Code Formatting

- Use `gofmt` as the base formatting tool
- Use `goimports` for import ordering
- Recommend configuring IDE to auto-format on save

## 3. Verification Checklist

- [ ] `.golangci.yml` exists in project root
- [ ] `golangci-lint run` executes successfully
- [ ] CI pipeline includes Lint stage
- [ ] No CSP standard rules are disabled

## 4. Tech Stack Recommendations

| Category | Recommended | Alternative | When to Use Alternative |
|----------|-------------|-------------|------------------------|
| Linter | golangci-lint | revive | Projects with very specific revive rules |
| Formatter | gofmt (built-in) | — | None (gofmt is standard) |
| Import Sorter | goimports | — | None (goimports is standard) |
| Static Analysis | staticcheck (via golangci-lint) | — | None (built into golangci-lint) |

**Priority**: Keep current repo's existing choice. Only recommend alternatives when capability is missing.

## 5. Auto-Completion Procedure

### Detecting Lint Configuration Status

1. Check for `.golangci.yml` or `.golangci.yaml` → lint config exists
2. Check for `golangci-lint` in `$GOBIN` or `$PATH` → tool installed
3. If config exists but tool not installed → install tool only
4. If no config → full auto-completion

### golangci-lint Auto-Completion Steps

1. Install: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
2. Create `.golangci.yml` using the Standard Configuration Template from Section 1.2
3. Verify: `golangci-lint run --max-issues-per-linter=0 ./...` → must exit 0 (no config errors)
4. If existing project has lint errors: run with `--max-issues-per-linter=50` to see all issues, then address

### Acceptance Criteria

- [ ] `.golangci.yml` exists with valid configuration
- [ ] `golangci-lint` binary available in `$PATH`
- [ ] `golangci-lint run` exits without configuration errors
- [ ] No CSP standard rules are disabled in config
