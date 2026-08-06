# JavaScript & TypeScript Language-Specific Prompt

This file supplies the JS/TS details required by `SKILL.md`: execution unit, extraction, filters, test organization, mandatory dependency preflight, verification commands, and scheduling. Framework APIs, component testing patterns, mock examples, engineering environment details, and error fixes are retrieved from RAG only when needed.

## Target Selection

### Minimum Execution Unit

The minimum execution unit is the **source file**. Process all selected functions in the same source file together so framework detection, imports, mocks, and test setup can be shared.

### Function Extraction

Extract regular functions, exported arrow/function expressions, class methods, React/Vue/Lynx components, and custom hooks. Record at least `function` and `line`; add optional `kind`, `class`, `export_type`, `signature` when useful.

Use `rg` for quick discovery and TypeScript Compiler API / Babel parser / IDE LSP when accurate symbols, signatures, or JSX/TSX parsing are needed.

### JS/TS Filters

Apply common `references/target-filter/AGENT.md` rules plus:

- Skip: `dist/`, `build/`, `coverage/`, `.next/`, `.nuxt/`, `.output/`, `.svelte-kit/`, `node_modules/`
- Skip: `*.d.ts`, `*.config.*`, `vite.config.*`, `jest.config.*`, `vitest.config.*`, `rspack.config.*`, `webpack.config.*`
- Skip: `*.stories.*`, `*.story.*`, `*.demo.*`, `__fixtures__/`, `fixtures/`
- Skip barrel files that only re-export symbols
- Skip type-only declarations, simple getter/setters, constant wrappers, pure style/config objects, and private inline callbacks without standalone logic

## Processing Contract

For each selected source file, track the source path, selected functions, starting lines, and optional symbol metadata such as `kind`, `class`, `export_type`, and `signature`. Use that information directly when generating tests and when reporting coverage, verification commands, failures, and defect mappings.

## Scheduling

Process source-file units serially:

1. Determine the selected functions for the current source file.
2. Writer processes all `functions[]` in that file together, sharing framework detection, imports, mocks, and test setup.
3. Fixer verifies and repairs the corresponding test file, then records the command, status, failures, and defect mapping in the final response.
4. Move to the next source file only after the current file is done.

Fixer may run at most **10** verify/fix rounds per test file.

## Test Organization

| Item | Convention |
|---|---|
| Test file | Follow existing style; otherwise use `*.test.ts(x)` / `*.test.js(x)` or project `*.spec.*` convention |
| Location | Prefer existing test location; otherwise source directory or a path matching `testMatch` / `include` |
| Test structure | `describe('<symbol>', ...)` + `it/test('<scenario>', ...)` |
| Components | React/Lynx: Testing Library style; Vue: `@vue/test-utils` style |

After Dependency Preflight is complete, incrementally supplement usable existing tests; rewrite only when existing tests are broken or obsolete. Test infrastructure edits are allowed when needed for JS/TS verification, including `package.json`, lockfiles, package-manager metadata, `node_modules` installation, and test config files such as `vitest.config.*`, `jest.config.*`, or `rstest.config.*`. Never modify production/business source code.

## Knowledge Retrieval

JS/TS framework and troubleshooting knowledge is maintained in AgentKnow and should be retrieved through RAG according to `${SKILL_ROOT}/references/rag.md`. Do not read local JS reference files for framework, component, mock, engineering, or troubleshooting knowledge.

Use RAG only when the current project context and existing tests do not already provide enough information. Recommended query triggers:

- Framework API or mock pattern is needed: Jest, Vitest, Rstest, ESM mock, partial mock, timer mock, mock hoisting.
- Component or hook testing pattern is needed: React, Testing Library, Vue, Lynx/ReactLynx.
- Arco Design React or Arco-based component library such as `@flux/design` is imported.
- Monorepo, package manager, dependency install, test config, path alias, or TypeScript config is unclear.
- Verification fails with non-trivial errors such as `Cannot find module`, `No tests found`, transform/ESM syntax errors, `document is not defined`, mock TDZ, timeout, or Rush/raw pnpm conflicts.

Useful RAG queries include:

- `"Jest mock ES module default export"`
- `"Vitest vi.hoisted mock factory external variable"`
- `"React Testing Library renderHook async hook"`
- `"Arco React Input onChange callback signature"`
- `"Rush monorepo JS unit test dependency preflight"`
- `"Jest Cannot find module path alias moduleNameMapper"`

If source imports `@arco-design/web-react` or an Arco-based component library such as `@flux/design`, query RAG for Arco React callback signatures before writing tests unless adjacent tests already show the exact API. Use actual installed package type declarations as the source of truth when RAG/docs and local package types conflict. For Arco popups and static APIs (`Message`, `Notification`, `Modal.confirm`), account for portal rendering or mock static methods when the source only triggers feedback.

Defect severity calibration and verification remain local skill policy: use `assets/javascript/references/defect-scoring.md` and `assets/javascript/references/defect-verifier.md` when analyzing JS/TS defects.

Context analysis is manual for JS/TS: **do not use `utree context`**. Read the source file, direct imports, existing tests, and adjacent tests as needed.

## Dependency Preflight [MANDATORY — BEFORE writing any test code]

> ⚠️ **BLOCKING PREREQUISITE**: This section MUST be fully completed BEFORE generating or writing any test code. If dependencies cannot be installed or test framework config cannot be established, do NOT proceed to test generation — resolve the environment first. The execution order is: **install deps → setup config → write tests → verify**.

### Repository-Local Execution Boundary [MANDATORY]

Do not create external runnable test projects/sandboxes. Tests, test config, dependency setup, and verification must stay inside `PROJECT_ROOT` or the owning package; if that cannot work, record `FAILED_REASON` instead.

Before writing or verifying tests, ensure the owning package is runnable:

1. Find the nearest `package.json`; in monorepos, prefer that package unless workspace tooling requires the root.
2. Detect workspace tooling before lockfiles. If `rush.json` is present, the repo is a Rush repo even when it uses pnpm internally. If `eden.monorepo.json` is present, treat it as an Eden/EMO repo. Do not classify these as raw pnpm workspaces just because `pnpm-lock.yaml` exists.
3. Detect the test framework from nearest config files, `package.json` dependencies/scripts, and adjacent tests. Config files take precedence over root-level dependency hints in monorepos. If no framework is detected, use the fallback in step 6.
4. In Rush repos, complete Rush tool preflight before any install/add/test command: read `rushVersion` and `nodeSupportedVersionRange`, confirm the current Node is supported, check whether Rush launcher scripts exist, and use either a matching global `rush` or an existing repo launcher such as `common/scripts/install-run-rush.js`. Do not run `rush install` through a missing or mismatched Rush CLI. Query RAG for Rush-specific details if the local repo conventions are unclear.
5. **Install dependencies FIRST**: If `node_modules` is missing or the detected test framework cannot resolve, run the appropriate install command immediately — do not defer to the verification phase.
   - In Rush repos, use Rush commands; do not run raw `pnpm install` / `pnpm add`, and do not create `pnpm-workspace.yaml` as a workaround.
   - In Eden/EMO repos, use EMO commands; do not bypass EMO with raw package-manager installs unless project docs explicitly require it.
   - Honor project registry config (`.npmrc`, Rush/EMO config, internal registry settings). Do not switch private/internal package installs to a public registry; only use a public fallback when all missing packages are public and no project registry is configured.
   - If install fails due to unrelated lifecycle scripts, first use the project/package-manager supported script-skip mechanism. Do not replace Rush/EMO with raw pnpm to bypass a postinstall.
6. **Setup test config**: If no test framework/script exists, add the minimal test setup needed to run generated tests: prefer Vitest for TS/ESM/TSX/Vite/component/path-alias projects, Jest for plain CommonJS JS. Add `jsdom` only for DOM tests. This includes creating/updating `vitest.config.*` or `jest.config.*` and adding the `test` script to `package.json`.
7. **Verify framework resolves**: After install and config, confirm the test framework is runnable using the package/workspace command surface (for example, `npm exec -- vitest --version` in npm projects or `rush-pnpm exec vitest --version` in Rush projects). Only proceed to test generation after this succeeds.
8. During Dependency Preflight, only change test infrastructure (`package.json`, lockfile, package-manager metadata, `node_modules`, test config files); never change production/business source for verification.
9. If setup fails after all attempts, record the command and error in the final report with `FAILED_REASON`; do not report generation as complete just because test files were written.

## Mock / Framework Rules

Detected framework decides API:

- Jest: `jest.fn/mock/spyOn`, partial mock with `jest.requireActual`
- Vitest: `vi.fn/mock/spyOn`, partial mock with `importOriginal` / `vi.importActual`
- Rstest: `rs.fn/mock/spyOn`, follow Rstest-specific import-actual syntax
- Assertions: use the detected framework's `expect` and project-installed matchers; do not introduce new assertion libraries.

Mock hoisting rule: `jest.mock` / `vi.mock` / `rs.mock` factories run before `let`/`const` initialization. Never reference top-level test variables from a mock factory.

Safe patterns:

- Jest: define mocks inside factory, then read them with `jest.requireMock()`
- Vitest: use `vi.hoisted`
- All frameworks: lazy getter state, or non-hoisted `doMock` when supported
- Avoid TS annotations, `as` assertions, and JSX inside Jest mock factories when Babel parses them

## Verification Commands

Prefer project scripts and adjacent-test commands over global binaries.

Before verification, confirm Dependency Preflight has already been completed (it is mandatory before writing code). If skipped earlier, complete it now before running any test commands.

### Execution Timeout [MANDATORY]

Every JS/TS test execution command MUST use an outer shell timeout.

- Focused file/pattern: 120 seconds.
- Package-level, coverage, or slow bootstrap: 300 seconds.

```bash
if command -v timeout >/dev/null 2>&1; then
  timeout 120s sh -lc '<test command>'
else
  perl -e 'alarm shift; exec @ARGV' 120 sh -lc '<test command>'
fi
```

Use `300` / `300s` when needed. If timeout fires, verification failed; fix open handles/async leaks instead of relying on `--forceExit`. If the wrapper itself breaks the project command, remove the wrapper, run the original command, and record why.

Command selection:

1. Reuse commands from nearby tests, README, `AGENTS.md`, `CLAUDE.md`, or `package.json#scripts`.
2. In monorepo, run from the nearest package root with `package.json`.
3. Package-manager defaults:
   - Rush: run `rushx test -- <test-file>` from the package directory; if no `test` script exists, complete Dependency Preflight and query RAG when Rush-specific command details are unclear.
   - Eden/EMO: use package/project scripts through `emo run` / `emox` according to existing project docs.
   - npm: `npm test -- <test-file>`
   - pnpm: `pnpm test -- <test-file>` or `pnpm --filter <pkg> test -- <test-file>`
   - yarn: `yarn test <test-file>` or `yarn workspace <pkg> test <test-file>`
   - bun: `bun test <test-file>`
4. If scripts are too broad, run the detected framework through the package manager: Jest `<test-file>`, Vitest `run <test-file>`, or Rstest `<test-file>`.

Checks:

- Run tests first. Only then check TS/lint.
- Prefer IDE diagnostics for changed test files; otherwise use project `typecheck` / `tsc`.
- Use existing lint/format scripts; auto-fix when safe.
- Collect coverage only when requested or already part of the workflow.

## JS/TS-Specific Defect Signals

> For complete defect determination rules, see `references/detect-bugs.md`. Only JS/TS-specific supplementary signals are listed here. Each signal includes a context condition — judge in context, not mechanically.
>
> **Priority**: When deep mode is active and `invariants_path` is provided, declared invariants take precedence over the signals below. Use these signals as supplementary detection heuristics, not as overrides of explicitly declared behavioral contracts.

**Runtime Crash Signals:**

- Empty array/object access without length guard (e.g., `options[0].value` when array can be empty) → **Most likely a real defect** when the empty state is reachable in normal business flow (e.g., all items consumed, filter returns nothing)
- `TypeError: Cannot read properties of undefined/null` → Only counts when the null/undefined is **produced by the function's internal logic** (API returns null, map lookup misses, conditional assignment skipped); test-injected null props don't count
- `RangeError: Maximum call stack size exceeded` or useEffect triggering self-setState loop → Need to confirm whether the trigger path is reachable in normal usage

**Logic & Semantic Contradiction Signals:**

- Condition expression contradicts its control-flow semantics (e.g., `if (err) { saveData() }` should be `if (!err)`) → **Most likely a real defect**
- Comment/documentation explicitly states behavior X, but implementation does not perform X → **Most likely a real defect** when the missing logic would affect the function's return value or side effects
- API response `code`/`status` not checked before using `data` — when the response can legitimately fail and code continues with empty/null data as if success → **Likely a real defect** (silent failure path)
- Function implementation completely contradicts its name/signature semantics → **MUST report when**: the function's name/signature declares a dynamic semantic purpose (check, validate, verify, compute, fetch, ensure, guard, filter, etc.) but the implementation returns a fixed value without executing any logic corresponding to that purpose. This is a **mandatory detection** — does NOT require caller/doc/invariant corroboration. NOT caught: functions whose purpose IS to return a constant (`getDefaultTimeout`, `getMaxRetries`); functions that do perform some logic but produce a slightly different result than the name suggests.
- Operator precedence causing unintended grouping (e.g., `a || b ? x : y` parsed as `(a || b) ? x : y` instead of intended `a || (b ? x : y)`) → **Most likely a real defect** when it changes the return value or control flow
- Recursive/nested function call missing parameter pass-through (e.g., `omit(obj[key])` without forwarding `options`) → **Likely a real defect** when the omitted parameter affects behavior in nested levels

**Explicit NOT-a-defect patterns (do NOT report these):**

- `if (!value) return '--'` or `if (!time) return placeholder` — this is a **sentinel value pattern** (treating 0/""/null as "no data"), NOT a boundary error. See `defect-scoring.md` False Positive Prevention.
- A function returning early for a subset of inputs (e.g., `if (config.type === 'X') return id`) is NOT "making subsequent code unreachable" — it is a valid early-return pattern. Only report when the early return **contradicts** the function's documented/named purpose.
- `JSON.stringify(obj, keysArray)` with a top-level-only replacer is NOT a defect unless the function's contract explicitly requires deep serialization.

**State & Lifecycle Signals:**

- Submit/action guard state (e.g., `submitLoading`, `isSubmitting`) hardcoded to false or never set to true, AND no other mechanism prevents duplicate submission → **Likely a real defect** (duplicate submission risk on critical operations like create/pay/delete)
- `form.validate()` error swallowed (`.catch()` returns undefined) and submission continues unconditionally → **Most likely a real defect**
- addEventListener/setInterval bound N times but cleanup only removes once → residual listeners accumulate; **likely a real defect when it causes functional side effects** (responding to events in disabled state, memory growing unboundedly). Also applies when: same listener registered twice in different code paths within one effect (e.g., lines 96-100 AND lines 102-105 both call addEventListener), but cleanup only removes once — the second listener persists after disable/unmount.
- useEffect/useCallback creating async operations (fetch, debounced callbacks, timers) without cleanup or AbortController → **Likely a real defect** when the async callback calls setState after unmount or when rapid re-renders cause stale responses to overwrite fresh data
- useEffect dependency array missing variables that are **read AND used to produce side effects** (API calls, state writes, event reports with stale data) → **Likely a real defect** when consequence is incorrect data submission or silent data corruption; missing deps that only affect re-render timing don't count
- Variable/closure captures stale value after async boundary → Only counts when consequence is **data corruption or incorrect submission**; UI flicker that self-heals doesn't count

**Props/Data Flow Signals:**

- Props spread (`...rest`) unintentionally forwards a prop that overwrites an internally-rendered element → **Likely a real defect** when the overwritten element has functional significance (e.g., navigation button disappears)
- Mutating a prop value in-place (e.g., `push` on a prop array) without cloning → **Likely a real defect** when it causes parent state corruption

**Conditional Rendering & Component Logic Signals:**

- `&&` short-circuit rendering falsy primitives (e.g., `{count && <List/>}` renders `"0"` when count=0 instead of rendering nothing) → **Most likely a real defect** when the rendered falsy value is visible to the user
- Key prop derived from array index on a reorderable/filterable/deletable list causing component state to persist incorrectly after reorder → **Likely a real defect** when stateful inputs (controlled/uncontrolled) are involved
- Route guard / permission check renders protected content before redirect completes (flash of unauthorized content) → **Likely a real defect** when sensitive data is exposed in the brief render

**Data Transformation & Request Signals:**

- `Array.sort()` / `Array.reverse()` / `Array.splice()` called directly on reactive state or props without cloning first → **Most likely a real defect** when operating on React state, Vue reactive data, or props (mutates source of truth in place)
- Pagination / infinite-scroll request race condition: rapid page changes cause stale response to overwrite fresh data, with no AbortController or request-id comparison → **Likely a real defect** when no cancellation or sequencing mechanism exists
- `useMemo` / `useCallback` dependency array missing a variable that changes and affects the memoized result, OR dependency array containing an unstable reference (inline object/array/function) causing memo to recompute every render → **Likely a real defect** when consequence is stale data used in submission/API call, or unbounded re-computation causing UI freeze

### Defect Scoring (JS/TS Specific)

For JS/TS defects, apply the scoring criteria in `assets/javascript/references/defect-scoring.md`:

- **Trust Boundary Degradation**: When trigger depends on upstream/backend abnormal data, downgrade. But **do NOT downgrade** when the empty/error state is a normal business scenario (e.g., empty search results, API rate-limit response, user has no data yet).
- **False Positive Prevention**: See the explicit "NOT a defect" list in `defect-scoring.md`. Key patterns: design choices, sentinel values, UX-level timing issues, frontend scenarios where concurrency is practically impossible.
- **Minimum Reporting Threshold**: Report P0 and P1 defects. P2 only when evidence is conclusive and impact is functional failure.

### Defect Verification (JS/TS Specific)

After discovering candidate defects, apply the 3-gate self-check from `assets/javascript/references/defect-verifier.md`. This is a **lightweight confidence check**, not a strict filter:

1. **Business Scenario Gate**: "Would this issue be triggered in a real user scenario?" — If yes or plausible, PASS.
2. **Developer Acceptance Gate**: "If reported to the developer, would they acknowledge this as a bug worth fixing?" — If a reasonable developer would say "yes, this needs fixing", PASS.
3. **Evidence Conclusiveness Gate**: "Can I point to specific code lines showing a logic contradiction or missing critical logic?" — If the code clearly does/omits something contradicting its declared intent, PASS.

**Passing 2 of 3 gates is sufficient to keep the defect** (at appropriate severity). Only drop when a candidate fails all 3, or when it matches an explicit false-positive pattern listed in the verifier.

**Calibration**: Reporting 0 defects is perfectly valid. But do not over-filter — if the code has an empty-array crash path or a submit-without-validation path that a developer would want to fix, report it.

### 降噪规则覆盖 (Noise Rule Overrides)

The following overrides apply to JS/TS defect analysis per `detect-bugs.md` Section 4.3. When a candidate matches an override condition below, the corresponding generic noise rule is bypassed.

| Rule | Override Condition | Override Behavior |
|------|-------------------|-------------------|
| A-泛化防御建议 | Empty array/object access without guard where the empty state is reachable in normal business flow (empty search results, all items filtered out, API returns empty list, user has no data) | Do NOT filter; assess per `defect-scoring.md` — typically P0 (confirmed crash path) or P1 |
| E-服务边界转换 | Frontend/JS/TS code has no RPC/protobuf service boundaries | Rule does not apply to JS/TS; never use this rule to filter JS/TS candidates |
| F-框架托管契约 | JS/TS framework guarantees: React `useState` setter reference stability; React 18+ no-warning on unmount setState; React Router matched `params` non-null; Testing Library auto-cleanup; bundler module resolution | Only filter when the specific framework guarantee is confirmed in the project's installed version; do not assume guarantees without version evidence |
| G-不可稳定触发 | Async race condition whose consequence is data submission, persistent state corruption, or duplicate critical operations (pay/delete/create); can be stably reproduced via `jest.useFakeTimers()` / controlled mock timing | Do NOT filter; report at P1 per `defect-scoring.md` P1 #15 when data submission is involved, P2 otherwise |

**Clarification — Rule 3 (仅凭语义猜测) and P0 #17:**
P0 #17 (function implementation contradicts name/signature semantics) is based on *code evidence* — a static implementation that contradicts a declared dynamic purpose is provable from the function body alone. This constitutes "数据流证据" under Rule 3 and is therefore never filtered by it.

**Execution precedence**: These overrides are evaluated BEFORE the generic noise rules from `detect-bugs.md` Section 4.2. If a candidate matches an override condition, the noise rule is bypassed and the candidate proceeds to severity assessment per `defect-scoring.md` and `defect-verifier.md`.

## Generation and Exit Rules

- Cover happy path, boundary values, and exception/error paths.
- Reuse project style and helpers.
- Do not weaken assertions to pass.
- If one approach fails repeatedly, rethink mock/test structure or rewrite the test file.
- If a case cannot be made stable, remove that case and keep passing useful coverage.
- Exit only when tests pass, TS has no test-file errors, and lint has no relevant errors.

## Local Policy References

| Document | Purpose |
|---|---|
| `defect-scoring.md` | JS/TS-specific severity scoring, trust boundary rules, false-positive patterns |
| `defect-verifier.md` | 3-gate confidence check + explicit false-positive pattern matching |

All other JS/TS reference knowledge is retrieved through RAG from AgentKnow when needed.
