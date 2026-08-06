# TS/React Lint Rules Configuration

## 1. ESLint

CSP TS/React projects use ESLint as the unified code checker.

### 1.1 Standard Configuration (.eslintrc.cjs)

```javascript
module.exports = {
  root: true,
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:react/recommended',
    'plugin:react-hooks/recommended',
    'prettier',
  ],
  parser: '@typescript-eslint/parser',
  parserOptions: {
    ecmaVersion: 'latest',
    sourceType: 'module',
    ecmaFeatures: { jsx: true },
  },
  plugins: ['@typescript-eslint', 'react', 'react-hooks'],
  rules: {
    'react/react-in-jsx-scope': 'off',
    '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
    '@typescript-eslint/explicit-function-return-type': 'off',
    '@typescript-eslint/no-explicit-any': 'warn',
  },
  settings: {
    react: { version: 'detect' },
  },
};
```

### 1.2 Custom Rules

Projects may add additional rules based on business needs, but **must not remove** any rules from the standard set above.

## 2. Prettier

### 2.1 Standard Configuration (.prettierrc)

```json
{
  "semi": true,
  "singleQuote": true,
  "trailingComma": "all",
  "printWidth": 100,
  "tabWidth": 2,
  "arrowParens": "always"
}
```

### 2.2 ESLint Integration

Use `eslint-config-prettier` to disable ESLint rules that conflict with Prettier.

## 3. TypeScript Strict Mode

`tsconfig.json` should enable strict mode:

```json
{
  "compilerOptions": {
    "strict": true,
    "noImplicitAny": true,
    "strictNullChecks": true
  }
}
```

## 4. Verification Checklist

- [ ] `.eslintrc.cjs` exists in project root or `infra/`
- [ ] `npx eslint .` executes successfully
- [ ] `.prettierrc` exists
- [ ] `tsconfig.json` has `strict` mode enabled
- [ ] CI pipeline includes Lint stage

## 5. Tech Stack Recommendations

| Category | Recommended | Alternative | When to Use Alternative |
|----------|-------------|-------------|------------------------|
| Linter | ESLint | — | None (ESLint is standard) |
| Formatter | Prettier | — | None (Prettier is standard) |
| ESLint Config | eslint-config-prettier | — | Required to disable conflicting rules |
| TypeScript Parser | @typescript-eslint/parser | — | Required for TS projects |

**Priority**: Keep current repo's existing choice. Only recommend alternatives when capability is missing.

## 6. Auto-Completion Procedure

### Detecting Lint Configuration Status

1. Check for `.eslintrc.*` or `eslint.config.*` → lint config exists
2. Check for `.prettierrc` or `.prettierrc.*` → prettier config exists
3. Check for `eslint` in `package.json` devDependencies → deps installed
4. If config exists but deps missing → install missing deps only
5. If no config → full auto-completion

### ESLint + Prettier Auto-Completion Steps

1. Install dependencies: `emo add -D eslint @typescript-eslint/parser @typescript-eslint/eslint-plugin eslint-plugin-react eslint-plugin-react-hooks eslint-config-prettier prettier`
2. Create `.eslintrc.cjs` using the Standard Configuration template from Section 1.1
3. Create `.prettierrc` using the Standard Configuration template from Section 2.1
4. Verify: `npx eslint .` → must exit 0 (no configuration errors)
5. Verify: `npx prettier --check .` → must exit 0

### Acceptance Criteria

- [ ] `.eslintrc.*` exists with valid configuration
- [ ] `.prettierrc` exists
- [ ] `npx eslint .` exits 0 (no config errors)
- [ ] ESLint and Prettier deps in package.json devDependencies
