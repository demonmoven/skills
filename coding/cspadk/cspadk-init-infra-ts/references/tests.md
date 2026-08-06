# TS/React Testing Framework Configuration

## 1. Test Organization

### 1.1 Directory Structure

```
src/
├── components/
│   ├── Button/
│   │   ├── Button.tsx
│   │   └── Button.test.tsx
│   └── Modal/
│       ├── Modal.tsx
│       └── Modal.test.tsx
├── hooks/
│   ├── useAuth.ts
│   └── useAuth.test.ts
└── utils/
    ├── format.ts
    └── format.test.ts
```

- Test files are in the same directory as source files, named `*.test.ts(x)`
- Can also use `__tests__/` directory for centralized management

### 1.2 Jest Configuration (jest.config.cjs)

```javascript
module.exports = {
  testEnvironment: 'jsdom',
  transform: {
    '^.+\\.tsx?$': 'ts-jest',
  },
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
  },
  collectCoverageFrom: [
    'src/**/*.{ts,tsx}',
    '!src/**/*.d.ts',
    '!src/**/*.test.{ts,tsx}',
  ],
  coverageThresholds: {
    global: { branches: 60, functions: 60, lines: 60 },
  },
};
```

### 1.3 Vitest Configuration (vitest.config.ts) (Optional Alternative)

```typescript
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'jsdom',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
    },
  },
});
```

## 2. React Testing Library

### 2.1 Dependencies

```json
{
  "devDependencies": {
    "@testing-library/react": "^14.0.0",
    "@testing-library/jest-dom": "^6.0.0",
    "@testing-library/user-event": "^14.0.0",
    "jsdom": "^22.0.0"
  }
}
```

### 2.2 Test Template

```tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from './Button';

describe('Button', () => {
  it('renders with text', () => {
    render(<Button>Click me</Button>);
    expect(screen.getByText('Click me')).toBeInTheDocument();
  });

  it('handles click', async () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Click</Button>);
    await userEvent.click(screen.getByText('Click'));
    expect(onClick).toHaveBeenCalledOnce();
  });
});
```

## 3. Coverage Requirements

- **Minimum Coverage**: Components ≥ 60%, utility functions ≥ 80%
- Coverage report: `npm test -- --coverage`

## 4. Verification Checklist

- [ ] `jest.config.*` or `vitest.config.*` exists
- [ ] `@testing-library/react` dependency configured
- [ ] `npm test` executes successfully
- [ ] Coverage report can be generated
- [ ] CI includes test stage

## 5. Tech Stack Recommendations

| Category | Recommended | Alternative | When to Use Alternative |
|----------|-------------|-------------|------------------------|
| Test Runner | Vitest | Jest | Legacy projects already using Jest |
| Component Testing | @testing-library/react | Enzyme | Never (deprecated) |
| Coverage | @vitest/coverage-v8 | @vitest/coverage-istanbul | Never (v8 is faster) |
| Mocking | vi.mock (built-in) | sinon | Never (built-in sufficient) |
| API Mocking | msw | nock | Existing nock setup in project |

**Priority**: Keep current repo's existing choice. Only recommend alternatives when capability is missing.

## 6. Auto-Completion Procedure

### Detecting Test Framework Status

1. Check for `vitest.config.*` or `jest.config.*` → framework exists
2. Check for test scripts in `package.json` → scripts configured
3. Check for `src/**/*.test.*` files → tests written
4. If framework exists but no tests → only generate smoke test
5. If no framework → full auto-completion

### Vitest Auto-Completion Steps

1. Install: `emo add -D vitest @vitest/coverage-v8`
2. Create `vitest.config.ts` with coverage config:
   ```typescript
   import { defineConfig } from 'vitest/config';

   export default defineConfig({
     test: {
       environment: 'jsdom',
       coverage: {
         provider: 'v8',
         reporter: ['text', 'lcov'],
       },
     },
   });
   ```
3. Add to `package.json` scripts: `"test": "vitest"`, `"test:coverage": "vitest --coverage"`
4. Create smoke test file (e.g., `src/smoke.test.ts`):
   ```typescript
   import { describe, it, expect } from 'vitest';

   describe('smoke test', () => {
     it('runs without errors', () => {
       expect(true).toBe(true);
     });
   });
   ```
5. Run: `npx vitest run` → must exit 0

### Jest Auto-Completion Steps (only if project already uses Jest)

1. Install: `emo add -D jest ts-jest @types/jest`
2. Create `jest.config.cjs` with coverage config
3. Add to `package.json` scripts: `"test": "jest"`, `"test:coverage": "jest --coverage"`
4. Create smoke test file
5. Run: `npx jest` → must exit 0

### Acceptance Criteria

- [ ] `vitest` (or `jest`) in package.json devDependencies
- [ ] Config file exists with coverage config
- [ ] `npm test` exits 0
- [ ] At least one .test.ts file exists and passes
