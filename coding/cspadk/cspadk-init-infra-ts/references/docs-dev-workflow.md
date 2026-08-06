# Development Documentation Templates

## 1. development/getting-started.md — Quick Start Guide

| Field | Content |
|-------|---------|
| Purpose | Help new contributors set up environment and make first contribution |
| Sections | 1) Prerequisites (tools + versions) 2) Environment Setup (copy-paste commands) 3) First Run 4) First Contribution |
| Content Guidelines | Every step must have a copy-paste command. Prerequisites must include version requirements. Common pitfalls must be documented with workarounds. |
| Quality Checks | Every step has a copy-paste command; Prerequisites include version requirements; Common pitfalls are documented |

### TS/React Example — Prerequisites

```markdown
### Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Node.js | >= 18.x | `nvm install 18` |
| emo | latest | `npm install -g @i18n-cs/emo` |
| Git | >= 2.x | System package manager |

### Environment Setup

```bash
# 1. Clone the repository
git clone <repo-url> && cd <repo-name>

# 2. Install dependencies
emo install

# 3. Build all packages
npm run build

# 4. Start development
npm run dev
```

### First Run

```bash
# Verify the setup
node --version    # >= 18.x
emo --version     # emo is available
npm run build     # Build succeeds
npm run test      # Tests pass
npm run lint      # Lint passes
```

### Common Pitfalls

| Pitfall | Cause | Fix |
|---------|-------|-----|
| `emo: command not found` | emo not installed globally | `npm install -g @i18n-cs/emo` |
| Build fails with missing modules | Dependencies not installed | `emo install && npm run build` |
| `shamefully-hoist` error | `.npmrc` misconfigured | Ensure `shamefully-hoist=false` in `.npmrc` |
| Type errors in IDE | TypeScript version mismatch | Use workspace TypeScript version |
```

---

## 2. development/workflow.md — Development Workflow Description

| Field | Content |
|-------|---------|
| Purpose | Describe development workflow: feature development, testing, review, and deployment |
| Sections | 1) Development Flow (numbered sequence) 2) Local Development (hot reload) 3) Debugging Guide 4) Testing Workflow (unit + integration) 5) Deployment Flow (includes rollback) |
| Content Guidelines | Development flow must be a clear numbered sequence from feature request to production. Local development must describe hot reload setup. Debugging guide must cover common debugging scenarios. Testing workflow must distinguish unit and integration tests. Deployment flow must include rollback procedure. |
| Quality Checks | Development flow is a numbered sequence; Hot reload configuration is described; Debugging scenarios are covered; Unit and integration tests are distinguished; Rollback procedure is included |

### Development Flow Example

```markdown
### Development Flow

1. **Create Feature Branch**: `git checkout -b feat/<ticket-id>-<description>`
2. **Implement Feature**: Write code and tests following TDD when applicable
3. **Run Local Checks**: `npm run lint && npm run typecheck && npm run test`
4. **Push and Create MR**: Push branch and create merge request
5. **Code Review**: Address review feedback
6. **CI Validation**: Ensure all pipeline stages pass
7. **Merge**: Squash merge to main after approval
8. **Deploy**: Follow deployment flow for staging and production

### Local Development

```bash
# Start dev server with hot reload
npm run dev

# Watch mode for tests
npm run test -- --watch

# Type check in watch mode
npm run typecheck -- --watch
```

### Debugging Guide

| Scenario | Tool | Steps |
|----------|------|-------|
| Component rendering issue | React DevTools | Inspect component tree and props |
| State management bug | Browser debugger | Set breakpoints in store actions |
| Build error | Vite error overlay | Read error message, check import paths |
| Test failure | Vitest UI | `npm run test -- --ui` for interactive debugging |

### Testing Workflow

1. **Write Unit Tests**: Test individual functions and components in isolation
2. **Write Integration Tests**: Test feature workflows across multiple modules
3. **Run Full Suite**: `npm run test`
4. **Check Coverage**: `npm run test -- --coverage`
5. **Fix Failures**: Debug and fix before pushing

### Deployment Flow

1. Code is merged to `main` via approved MR
2. CI pipeline runs full validation (lint, test, build)
3. Staging deployment is triggered automatically
4. QA verification in staging environment
5. Production deployment via manual promotion
6. Post-deployment smoke test

#### Rollback Procedure

1. Identify the deployment that introduced the issue
2. Revert the merge commit on `main`
3. Trigger a new deployment from the reverted state
4. Verify the fix in staging before promoting to production
5. Document the incident and root cause
```
