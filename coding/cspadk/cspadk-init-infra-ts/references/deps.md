# TS/React Dependency Management Best Practices

## 1. emo Monorepo Management

This repository uses [Eden Monorepo (emo)](https://emo.bytedance.net/) for dependency management.

### 1.1 Core Rules

- **Must use `emo`**: All dependency operations must be executed via `emo`
  - ✅ `emo install`
  - ✅ `emo add <package>`
  - ❌ `pnpm install`, `npm install`, `yarn add`

- **Infra Mode**: Common dev dependencies are centralized in `infra/` directory
  - Sub-projects reference `@i18n-cs/infra-config` via `workspace:*`
  - ESLint, Prettier, TypeScript and other toolchain are managed by `infra/`

- **Root Directory Constraints**:
  - Root `package.json` cannot have `devDependencies`
  - Root directory cannot have `node_modules/`
  - `.npmrc` must have `shamefully-hoist=false`

### 1.2 Common Commands

```bash
emo install              # Install all dependencies
emo add <package>        # Add dependency to current sub-project
emo add -D <package>     # Add dev dependency
emo run <script>         # Run script
emox <tool>              # Run tool at root level
```

### 1.3 Lock File Management

- `pnpm-lock.yaml` is managed by `emo` automatically, located in `infra/` directory
- **Do not manually modify or delete** lock file

## 2. Dependency Version Management

### 2.1 Sub-project package.json

```json
{
  "name": "@i18n-cs/<skill-name>",
  "version": "1.0.0",
  "dependencies": {
    "@i18n-cs/infra-config": "workspace:*"
  }
}
```

### 2.2 Common Dependencies

| Dependency | Purpose | Version Management |
|------------|---------|-------------------|
| `react` / `react-dom` | UI Framework | Via infra unified management |
| `typescript` | Type System | Via infra unified management |
| `@i18n-cs/infra-config` | Infra Config | workspace:* |

## 3. Verification Checklist

- [ ] `eden.monorepo.json` exists
- [ ] Root `package.json` has no `devDependencies`
- [ ] Root directory has no `node_modules/`
- [ ] `.npmrc` has `shamefully-hoist=false`
- [ ] `pnpm-lock.yaml` is in `infra/`
- [ ] Sub-project package name starts with `@i18n-cs/`

## 4. Tech Stack Recommendations

| Category | Recommended | Alternative | When to Use Alternative |
|----------|-------------|-------------|------------------------|
| Package Manager | emo (Eden Monorepo) | pnpm | Non-monorepo projects without emo |
| Monorepo Config | `eden.monorepo.json` | `pnpm-workspace.yaml` | Non-emo monorepos |
| Lock File | `pnpm-lock.yaml` in `infra/` | `pnpm-lock.yaml` at root | Non-emo projects |

**Priority**: Keep current repo's existing choice. Only recommend alternatives when capability is missing.

## 5. Auto-Completion Procedure

### Detecting Dependency Management Status

1. Check for `eden.monorepo.json` → emo configured
2. Check for `pnpm-workspace.yaml` → pnpm workspace configured
3. Check for root `package.json` `devDependencies` → rule violation
4. Check for root `node_modules/` → rule violation
5. Check for `.npmrc` with `shamefully-hoist=false` → constraint met
6. If emo missing but pnpm available → configure emo
7. If rule violations detected → fix violations

### emo Configuration Auto-Completion Steps

1. Install emo if missing: `npm install -g @ies/eden-monorepo`
2. Initialize emo: `emo init` (if `eden.monorepo.json` does not exist)
3. Move root `devDependencies` to `infra/package.json`:
   - Identify all root devDependencies
   - Add them to `infra/package.json` devDependencies
   - Remove from root `package.json`
   - Run `emo install` to regenerate lock file
4. Verify `.npmrc` has `shamefully-hoist=false`:
   ```bash
   grep -q "shamefully-hoist=false" .npmrc || echo "shamefully-hoist=false" >> .npmrc
   ```
5. Delete root `node_modules/` if it exists (emo hoists dependencies)
6. Verify: `emo install` exits 0

### Acceptance Criteria

- [ ] `eden.monorepo.json` exists
- [ ] Root `package.json` has no `devDependencies`
- [ ] Root directory has no `node_modules/`
- [ ] `.npmrc` has `shamefully-hoist=false`
- [ ] `pnpm-lock.yaml` is in `infra/`
- [ ] `emo install` exits 0
