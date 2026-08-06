# Go Dependency Management Best Practices

## 1. Go Module Management

### 1.1 Initialization

```bash
go mod init <module-path>
```

- `module-path` should follow company naming conventions, e.g., `code.byted.org/ies/csp-xxx-service`
- Minimum Go version: **1.21+**

### 1.2 Adding and Updating Dependencies

```bash
go get <package>@<version>
go mod tidy
```

- Must run `go mod tidy` after adding new dependencies to clean up indirect dependencies
- Do not commit dependencies with `// indirect` that are not actually used

### 1.3 Dependency Source Constraints

- **Prefer internal Proxy**: Configure `GOPRIVATE=code.byted.org` and `GONOSUMCHECK=code.byted.org`
- Do not introduce third-party libraries from unknown sources
- Confirm license compatibility before introducing new dependencies

### 1.4 go.sum Integrity

- `go.sum` must be committed with code
- Do not manually modify `go.sum`, only update via `go mod tidy` or `go get`
- Run `go mod verify` to verify dependency integrity

## 2. Common Internal Dependencies

| Dependency | Purpose | Import Method |
|------------|---------|---------------|
| `code.byted.org/kitex` | RPC Framework | `go get code.byted.org/kitex` |
| `code.byted.org/hertz` | HTTP Framework | `go get code.byted.org/hertz` |
| `code.byted.org/gopkg` | Common Utilities | Import sub-packages as needed |

## 3. Verification Checklist

- [ ] `go.mod` exists and module-path follows conventions
- [ ] Go version ≥ 1.21
- [ ] `go mod tidy` runs without errors
- [ ] `go mod verify` passes
- [ ] `GOPRIVATE` configured
- [ ] No redundant indirect dependencies

## 4. Tech Stack Recommendations

| Category | Recommended | Alternative | When to Use Alternative |
|----------|-------------|-------------|------------------------|
| Dep Management | Go Modules | — | None (Go Modules is standard) |
| Internal Proxy | GOPRIVATE=code.byted.org | — | Required for all ByteDance projects |
| Dep Tooling | go mod (built-in) | dep | Never (deprecated) |
| Vendoring | go mod vendor | — | Highly regulated environments requiring audit |

**Priority**: Keep current repo's existing choice. Only recommend alternatives when capability is missing.

## 5. Auto-Completion Procedure

### Detecting Dependency Management Status

1. Check for `go.mod` → Go Modules initialized
2. Check `go version` output → Go version meets minimum (≥ 1.21)
3. Check `GOPRIVATE` env var → internal proxy configured
4. If `go.mod` missing → full auto-completion
5. If `GOPRIVATE` not set → configure proxy

### Go Modules Auto-Completion Steps

1. Run `go mod init <module-path>` (prompt user for module path if not inferrable)
   - Module path should follow company naming: `code.byted.org/ies/<service-name>`
2. Set `GOPRIVATE=code.byted.org` and `GONOSUMCHECK=code.byted.org` in shell profile
3. Run `go mod tidy` to resolve dependencies
4. Run `go mod verify` to verify integrity
5. Verify: `go build ./...` exits 0

### Acceptance Criteria

- [ ] `go.mod` exists with correct module path
- [ ] Go version ≥ 1.21
- [ ] `GOPRIVATE` configured
- [ ] `go mod tidy` runs without errors
- [ ] `go mod verify` passes
- [ ] `go build ./...` exits 0
