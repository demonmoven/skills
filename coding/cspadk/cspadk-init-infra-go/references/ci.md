# Go CI Pipeline Configuration

## 1. Codebase CI Pipeline

CSP Go projects use Codebase CI as the continuous integration platform.

### 1.1 Standard Pipeline Structure

Define in `.codebase/pipelines/go-ci.yaml`:

```yaml
name: go-ci
trigger:
  change:
    branches: [main]
  push:
    branches: [main]

jobs:
  lint:
    name: Lint
    image: hub.byted.org/codebase/ci_go_1_23
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: golangci-lint
        commands:
          - golangci-lint run ./...

  test:
    name: Test
    image: hub.byted.org/codebase/ci_go_1_23
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Run Tests
        commands:
          - go test ./... -race -coverprofile=coverage.out -covermode=atomic
      - name: Check Coverage
        commands:
          - |
            COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
            echo "Total coverage: ${COVERAGE}%"

  build:
    name: Build
    image: hub.byted.org/codebase/ci_go_1_23
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Build
        commands:
          - go build ./...
```

### 1.2 Key Configuration Requirements

- **Trigger**: MR to main branch and push to main branch
- **Job Order**: lint → test → build (add `depends` if sequential execution needed)
- **Dependency Cache**: Recommend configuring Go Module cache to speed up builds
- **Coverage**: Test stage must output coverage report

### 1.3 With Cache (Recommended)

```yaml
name: go-ci
trigger:
  change:
    branches: [main]
    paths: &paths
      - "**/*.go"
      - "go.mod"
      - "go.sum"
    patchset-paths: *paths

jobs:
  lint:
    name: Lint
    image: hub.byted.org/codebase/ci_go_1_23
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Cache Go Modules
        uses: actions/cache
        inputs:
          path: ~/go/pkg/mod
          key: go-${{ hashFiles('go.sum') }}
      - name: golangci-lint
        commands:
          - golangci-lint run ./...

  test:
    name: Test
    image: hub.byted.org/codebase/ci_go_1_23
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Cache Go Modules
        uses: actions/cache
        inputs:
          path: ~/go/pkg/mod
          key: go-${{ hashFiles('go.sum') }}
      - name: Run Tests
        commands:
          - go test ./... -race -coverprofile=coverage.out -covermode=atomic

  build:
    name: Build
    image: hub.byted.org/codebase/ci_go_1_23
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Cache Go Modules
        uses: actions/cache
        inputs:
          path: ~/go/pkg/mod
          key: go-${{ hashFiles('go.sum') }}
      - name: Build
        commands:
          - go build ./...
```

### 1.4 CI Images Reference

| Image | Base OS | Status |
|-------|---------|--------|
| hub.byted.org/codebase/ci_go_1_24 | Debian 11 | ✅ Latest |
| hub.byted.org/codebase/ci_go_1_23 | Debian 11 | ✅ Recommended |
| hub.byted.org/codebase/ci_go_1_22 | Debian 11 | ✅ Available |

**Note**: Image name is `ci_go_xxx`, NOT `ci_golang_xxx`.

## 2. Verification Checklist

- [ ] `.codebase/pipelines/` contains Go CI configuration file
- [ ] Pipeline includes lint, test, build stages
- [ ] Test stage configured with coverage output
- [ ] MR trigger condition configured
- [ ] Using correct CI image (`ci_go_1_23` or newer)

## 3. Tech Stack Recommendations

| Category | Recommended | Alternative | When to Use Alternative |
|----------|-------------|-------------|------------------------|
| CI Platform | Codebase Pipeline | GitHub Actions | Open-source or non-ByteDance projects |
| Pipeline Format | YAML (.codebase/pipelines/) | Makefile targets | Projects without Codebase access |
| Go CI Image | ci_go_1_23 | ci_go_1_22 | Projects requiring Go 1.22 specifically |
| Dependency Cache | Go Module cache | No cache | Very small projects |

**Priority**: Keep current repo's existing choice. Only recommend alternatives when capability is missing.

## 4. Auto-Completion Procedure

### Detecting CI Configuration Status

1. Check for YAML files under `.codebase/pipelines/` → CI config exists
2. Check for `Makefile` with CI targets → alternative CI approach
3. Check for `.github/workflows/` → GitHub Actions used
4. If no CI config → full auto-completion

### Codebase Pipeline Auto-Completion Steps

1. Create `.codebase/pipelines/` directory
2. Generate `go-ci.yaml` using the Standard Pipeline Structure from Section 1.1
3. Select appropriate CI image based on project's `go.mod` version:
   - Go 1.24+ → `ci_go_1_24`
   - Go 1.23 → `ci_go_1_23`
   - Go 1.22 → `ci_go_1_22`
4. Validate YAML syntax
5. Verify: Pipeline appears in Codebase CI dashboard

### Acceptance Criteria

- [ ] `.codebase/pipelines/` contains at least one YAML file
- [ ] YAML has lint, test, build stages
- [ ] Test stage configured with coverage output
- [ ] CI image matches Go version requirement
