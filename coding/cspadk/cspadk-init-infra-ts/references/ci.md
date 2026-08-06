# TS/React CI Pipeline Configuration

## 1. Codebase CI Pipeline

### 1.1 Standard Pipeline Structure

Define in `.codebase/pipelines/ts-ci.yaml`:

```yaml
name: ts-ci
trigger:
  change:
    branches: [main]
  push:
    branches: [main]

jobs:
  lint:
    name: Lint
    image: hub.byted.org/codebase/ci_nodejs_20
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Install emo
        commands:
          - |
            command -v emo >/dev/null 2>&1 || npm install -g @ies/eden-monorepo
      - name: Install Dependencies
        commands:
          - emo install
      - name: ESLint
        commands:
          - npx eslint .
      - name: Type Check
        commands:
          - npx tsc --noEmit

  test:
    name: Test
    image: hub.byted.org/codebase/ci_nodejs_20
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Install emo
        commands:
          - |
            command -v emo >/dev/null 2>&1 || npm install -g @ies/eden-monorepo
      - name: Install Dependencies
        commands:
          - emo install
      - name: Run Tests
        commands:
          - npm test -- --coverage

  build:
    name: Build
    image: hub.byted.org/codebase/ci_nodejs_20
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Install emo
        commands:
          - |
            command -v emo >/dev/null 2>&1 || npm install -g @ies/eden-monorepo
      - name: Install Dependencies
        commands:
          - emo install
      - name: Build
        commands:
          - npm run build
```

### 1.2 Key Configuration Requirements

- **Trigger**: MR to main branch and push to main branch
- **Job Order**: lint → test → build (add `depends` if sequential execution needed)
- **Dependency Installation**: Must check and install `emo` first, then run `emo install`
- **Type Check**: Lint stage must include `tsc --noEmit`
- **Coverage**: Test stage must output coverage report

### 1.3 With Cache (Recommended)

```yaml
name: ts-ci
trigger:
  change:
    branches: [main]
    paths: &paths
      - "src/**/*.{ts,tsx}"
      - "package.json"
      - "pnpm-lock.yaml"
    patchset-paths: *paths

jobs:
  lint:
    name: Lint
    image: hub.byted.org/codebase/ci_nodejs_20
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Cache node_modules
        uses: actions/cache
        inputs:
          path: node_modules
          key: node-${{ hashFiles('pnpm-lock.yaml') }}
      - name: Install emo
        commands:
          - |
            command -v emo >/dev/null 2>&1 || npm install -g @ies/eden-monorepo
      - name: Install Dependencies
        commands:
          - emo install
      - name: ESLint
        commands:
          - npx eslint .
      - name: Type Check
        commands:
          - npx tsc --noEmit

  test:
    name: Test
    image: hub.byted.org/codebase/ci_nodejs_20
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Cache node_modules
        uses: actions/cache
        inputs:
          path: node_modules
          key: node-${{ hashFiles('pnpm-lock.yaml') }}
      - name: Install emo
        commands:
          - |
            command -v emo >/dev/null 2>&1 || npm install -g @ies/eden-monorepo
      - name: Install Dependencies
        commands:
          - emo install
      - name: Run Tests
        commands:
          - npm test -- --coverage

  build:
    name: Build
    image: hub.byted.org/codebase/ci_nodejs_20
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Cache node_modules
        uses: actions/cache
        inputs:
          path: node_modules
          key: node-${{ hashFiles('pnpm-lock.yaml') }}
      - name: Install emo
        commands:
          - |
            command -v emo >/dev/null 2>&1 || npm install -g @ies/eden-monorepo
      - name: Install Dependencies
        commands:
          - emo install
      - name: Build
        commands:
          - npm run build
```

### 1.4 CI Images Reference

| Image | Base OS | Status |
|-------|---------|--------|
| hub.byted.org/codebase/ci_nodejs_20 | Debian 10 | ✅ Recommended |
| hub.byted.org/codebase/ci_nodejs_18 | Debian 10 | ✅ Recommended |
| hub.byted.org/codebase/ci_nodejs_16 | Debian 10 | ✅ Available |
| hub.byted.org/codebase/ci_nvm | Debian 10 | ✅ Multi-version |

**Note**: CI images come with internal npm mirror and eden pre-configured.

## 2. Verification Checklist

- [ ] `.codebase/pipelines/` contains TS CI configuration file
- [ ] Pipeline includes lint, test, build stages
- [ ] Lint stage includes ESLint and TypeScript type check
- [ ] Test stage configured with coverage output
- [ ] MR trigger condition configured
- [ ] Using correct CI image (`ci_nodejs_20` or newer)

## 3. Tech Stack Recommendations

| Category | Recommended | Alternative | When to Use Alternative |
|----------|-------------|-------------|------------------------|
| CI Platform | Codebase Pipeline | GitHub Actions | Non-ByteDance projects |
| CI Image | ci_nodejs_20 | ci_nodejs_18 | Legacy Node.js 18 projects |
| Cache | actions/cache | None | Small projects with fast installs |

**Priority**: Keep current repo's existing choice. Only recommend alternatives when capability is missing.

## 4. Auto-Completion Procedure

### Detecting CI Pipeline Status

1. Check for YAML files under `.codebase/pipelines/` → CI exists
2. Check for GitHub Actions under `.github/workflows/` → alternative CI exists
3. If CI exists but missing stages → add missing stages
4. If no CI → full auto-completion

### Codebase Pipeline Auto-Completion Steps

1. Create `.codebase/pipelines/` directory if missing
2. Generate `ts-ci.yaml` using the Standard Pipeline Structure template from Section 1.1
3. Adjust Node.js image version to match project's `.nvmrc` or `package.json` engines field
4. Verify trigger branches match project's main branch name
5. Validate YAML syntax: `node -e "require('js-yaml').load(require('fs').readFileSync('.codebase/pipelines/ts-ci.yaml','utf8'))"` → must exit 0

### Acceptance Criteria

- [ ] At least one YAML exists under `.codebase/pipelines/`
- [ ] Pipeline includes lint, test, and build stages
- [ ] MR trigger condition configured
- [ ] YAML syntax is valid
