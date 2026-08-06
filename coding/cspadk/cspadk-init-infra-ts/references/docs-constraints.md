# Constraints Documentation Templates

## 1. constraints/repository-constraints.md — Repository Constraint Specifications

| Field | Content |
|-------|---------|
| Purpose | Document mandatory low-level constraints: dependency management, directory structure, CI/CD, and security |
| Sections | 1) Dependency Management 2) Directory Structure Rules 3) CI/CD Requirements 4) Security Constraints 5) Repository-Specific Rules |
| Content Guidelines | Every constraint must be prefixed with MUST / MUST NOT / SHOULD. Each constraint must include a rationale explaining why it exists. Each constraint must provide a validation command that can be run to check compliance. |
| Quality Checks | Every constraint has a MUST/MUST NOT/SHOULD prefix; Rationale is provided for each constraint; Validation command is provided for each constraint |

### TS/React-Specific Constraint Examples

```markdown
### Dependency Management

- **MUST** use `emo` for all dependency operations. Rationale: emo manages the monorepo workspace and ensures consistent hoisting. Validation: `which emo && cat eden.monorepo.json`
- **MUST NOT** have `devDependencies` in root `package.json`. Rationale: All dev deps must be centralized in `infra/package.json` for monorepo consistency. Validation: `cat package.json | jq '.devDependencies'`
- **MUST** have `shamefully-hoist=false` in `.npmrc`. Rationale: Prevents implicit dependency access and ensures explicit dependency declarations. Validation: `grep shamefully-hoist .npmrc`

### Directory Structure Rules

- **MUST** place shared dev dependencies in `infra/`. Rationale: Centralized toolchain management prevents version drift across sub-projects. Validation: `ls infra/package.json`
- **MUST NOT** have `node_modules/` in the repository root. Rationale: emo hoists dependencies via `infra/node_modules/`. Validation: `test -d node_modules && echo 'FAIL' || echo 'PASS'`
- **MUST** keep `pnpm-lock.yaml` under `infra/`. Rationale: emo manages the lock file location. Validation: `ls infra/pnpm-lock.yaml`

### CI/CD Requirements

- **MUST** include lint, test, and build stages in CI pipeline. Rationale: Prevents merging code that fails quality checks. Validation: `grep -E 'lint|test|build' .codebase/pipelines/*.yml`
- **MUST** cache `node_modules` in CI. Rationale: Reduces build times significantly. Validation: `grep cache .codebase/pipelines/*.yml`

### Security Constraints

- **MUST NOT** commit secrets, tokens, or credentials to the repository. Rationale: Prevents credential leakage. Validation: `git log --all --diff-filter=A -- '*.env' '*.key' '*.pem'`
- **MUST** use environment variables for configuration. Rationale: Separates config from code. Validation: `grep -r 'hardcoded' src/ || echo 'PASS'`

### Repository-Specific Rules

- **SHOULD** prefix sub-project package names with `@i18n-cs/`. Rationale: Ensures consistent namespace across the monorepo. Validation: `jq '.name' packages/*/package.json`
```

---

## 2. constraints/development-guidelines.md — Development & Version Control Guidelines

| Field | Content |
|-------|---------|
| Purpose | Coding standards, branching strategy, commit conventions, and release process |
| Sections | 1) Coding Standards (with examples) 2) Branching Strategy (naming format) 3) Commit Conventions (message template) 4) Release Process (includes rollback) |
| Content Guidelines | Coding standards must include concrete code examples. Branch naming must define the exact format with examples. Commit conventions must provide a message template. Release process must include rollback procedure. |
| Quality Checks | Coding standards have code examples; Branch naming format is defined; Commit message template is provided; Rollback procedure is documented |

### Coding Standards Example

```markdown
### TypeScript Coding Standards

- Use `interface` for object types, `type` for unions and intersections
- Prefer `enum` over string literal unions for fixed sets
- Use `readonly` for immutable properties

Example:

```typescript
// ✅ Preferred
interface UserConfig {
  readonly name: string;
  readonly role: UserRole;
}

// ❌ Avoid
type UserConfig = {
  name: string;
  role: string;
};
```
```

### Branch Naming Example

```markdown
### Branch Naming Convention

Format: `<type>/<ticket-id>-<short-description>`

Types: `feat/`, `fix/`, `chore/`, `refactor/`, `docs/`

Examples:
- `feat/cspadk-123-add-skill-pull`
- `fix/cspadk-456-lint-config-error`
- `chore/cspadk-789-update-deps`
```

### Commit Message Template

```markdown
### Commit Conventions

Format:

```
<type>(<scope>): <subject>

<body>

Co-Authored-By: ...
```

Types: `feat`, `fix`, `chore`, `refactor`, `docs`, `test`, `ci`

Example:

```
feat(cli): add skill pull command

Implement `cspadk skill pull <name>` to copy skill templates
from the skill registry to the local project.

Co-Authored-By: Developer <dev@example.com>
```
```

### Release Process Example

```markdown
### Release Process

1. Create release branch from `main`: `release/v<version>`
2. Run full test suite: `npm run test`
3. Update version in `package.json` files
4. Merge release branch to `main` via MR
5. Tag the merge commit: `v<version>`

### Rollback Procedure

1. Identify the problematic commit or release tag
2. Create a revert branch from `main`: `fix/revert-v<version>`
3. Run `git revert <commit-sha>` on the revert branch
4. Merge revert MR with fast-track review
5. Verify rollback in staging before production deploy
```

---

## 3. constraints/constraint-validation.md — Constraint Validation Guide

| Field | Content |
|-------|---------|
| Purpose | Checklist and commands to validate constraint compliance |
| Sections | 1) Automated Checks (exact commands) 2) Manual Checklist 3) CI Validation 4) Remediation Guide |
| Content Guidelines | Automated checks must provide exact copy-paste commands. Manual checklist must cover constraints not easily automated. CI validation must describe how the pipeline enforces constraints. Remediation guide must provide step-by-step fixes for common violations. |
| Quality Checks | Every constraint has a validation step; Exact commands are provided; Remediation guide covers common violations |

### Automated Checks Example

```markdown
### Dependency Management Validation

| Check | Command | Expected Result |
|-------|---------|-----------------|
| emo is installed | `which emo` | Path to emo binary |
| Monorepo config exists | `test -f eden.monorepo.json && echo PASS` | PASS |
| No root devDependencies | `jq '.devDependencies \| length' package.json` | 0 |
| shamefully-hoist=false | `grep shamefully-hoist=false .npmrc` | Match found |
| Lock file in infra/ | `test -f infra/pnpm-lock.yaml && echo PASS` | PASS |
| No root node_modules | `test -d node_modules && echo FAIL \|\| echo PASS` | PASS |
```

### Manual Checklist Example

```markdown
### Manual Checklist

- [ ] No hardcoded secrets or credentials in source code
- [ ] CI pipeline includes lint, test, and build stages
- [ ] Environment variables are used for all configuration
- [ ] Branch naming follows `<type>/<ticket-id>-<description>` format
- [ ] Commit messages follow the conventional commit template
```

### CI Validation Example

```markdown
### CI Validation

The CI pipeline enforces constraints at merge request time:

1. **Lint Stage**: Runs `npm run lint` — fails on ESLint errors
2. **Type Check Stage**: Runs `npm run typecheck` — fails on TypeScript errors
3. **Test Stage**: Runs `npm run test` — fails on test failures
4. **Build Stage**: Runs `npm run build` — fails on build errors
5. **Constraint Check Stage**: Runs custom validation script — fails on constraint violations
```

### Remediation Guide Example

```markdown
### Common Violations and Remediation

#### Root package.json has devDependencies

1. Move each devDependency to `infra/package.json`
2. Remove the `devDependencies` key from root `package.json`
3. Run `emo install` to reinstall dependencies
4. Verify: `jq '.devDependencies | length' package.json` returns `0`

#### shamefully-hoist not set to false

1. Open `.npmrc`
2. Add or update the line: `shamefully-hoist=false`
3. Run `emo install` to re-resolve dependencies
4. Verify: `grep shamefully-hoist=false .npmrc`

#### pnpm-lock.yaml in wrong location

1. Move `pnpm-lock.yaml` to `infra/pnpm-lock.yaml`
2. Run `emo install` to regenerate if needed
3. Verify: `test -f infra/pnpm-lock.yaml && echo PASS`
```
