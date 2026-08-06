# TS/JS Defect Risk Scoring Standard v1.0

> This document provides JS/TS-specific defect scoring criteria for the Code Reviewer agent. It supplements the generic `detect-bugs.md` with frontend-specific patterns, trust boundary rules, and noise reduction principles.

---

## Core Principles

### Trust Boundary Degradation Principle

When a defect's trigger condition depends on "upstream component passing abnormal props" or "backend API returning dirty data", apply severity degradation. Frontend code reasonably trusts same-project upstream components and own backend APIs — such issues are "defensive programming gaps" rather than "certain defects".

Degradation rules:
- Originally P0 (certain crash) but trigger depends on upstream/backend abnormal data → downgrade to **P2**
- Originally P1 (high risk) but trigger depends on upstream/backend abnormal data → downgrade to **P2**
- Only maintain original severity when data-flow analysis **confirms** the variable is necessarily abnormal on reachable paths within the current code

**IMPORTANT EXCEPTION — Do NOT apply degradation when:**
- The "abnormal" state is actually a **normal business scenario** (empty search results, empty filter results, API rate-limit/timeout response, user hasn't created any data yet)
- The function itself is responsible for producing the empty/null state (e.g., internal filter logic can produce empty array, then code accesses `[0]`)
- The response code/status indicates failure and is a documented API behavior (not "dirty data")

### False Positive Prevention Principle

The following situations **MUST NOT** be reported as defects:
- Design choices matching user expectations (e.g., component API allows overriding specific params via generics)
- When it cannot be confirmed whether upstream methods accept null values, do not assume null necessarily causes errors
- Read-only mode retaining copy functionality — matches common product design
- Initialization logic that only fires once (e.g., `useEffect(fn, [])` only runs on mount) — unless proven that data changes post-initialization actually cause problems
- Industry-common "could be better but doesn't affect functionality" patterns (e.g., React state update after unmount — no longer produces warnings in React 18+)
- Pure TypeScript type refinement issues that have no runtime impact (e.g., `as` casts between compatible types)
- Error handling that swallows errors on non-critical paths (metrics, logging, analytics)
- **Sentinel value patterns**: Using `0`, `""`, `null` as "no data" indicators and returning placeholder text (e.g., `if (!time) return '--'` where time=0 means "not set")
- **Intentional action availability**: Buttons/actions enabled in specific states by design (e.g., allowing delete on RUNNING tasks for abort-and-cleanup workflows)
- **Behavioral parameter differences between code paths**: Different paths providing slightly different params to callbacks — unless it causes data loss or crash
- **Read-then-write non-atomicity in single-user frontend**: `check existence → create/update` without DB locks — in browser context with single user, concurrent execution is practically impossible
- **Filter/search granularity choices**: Returning broader results than theoretically optimal — this is a product/UX decision

### Same-Pattern Deduplication Principle

The same defect pattern (e.g., useEffect missing cleanup, async setState missing unmount guard) in a single file should report **at most the most severe 1 instance**. If the same pattern appears across multiple files, report at most 1 per file. Do not template-apply the same pattern to every useEffect / every async function.

---

## Severity Levels (JS/TS Specific)

### P3: Style & Code Smell (Do NOT Report)

These are NOT defects — do not report or include in results:

1. Implicit `any` on function params/returns (unless `noImplicitAny` is confirmed enabled)
2. Naming convention violations (camelCase, PascalCase, UPPER_SNAKE)
3. Unused imports, unreachable code, unused variables
4. `var` without hoisting side-effects
5. Magic numbers or repeated string literals not extracted as constants
6. `as` type assertions between compatible parent-child types
7. Nested control flow > 4 levels (maintainability concern only)
8. Function body > 80 lines
9. Duplicated code blocks (AST similarity)
10. Array index as React/Vue list key in `.map()` callbacks

### P2: Boundary Risk (Report only when evidence is strong)

Specific input/timing conditions trigger the issue:

1. Direct property access on union type containing `null | undefined` without optional chaining or guard
2. Implicit type coercion: `==`/`!=` (non `== null` idiom), unary `+` for conversion, string concat with non-string
3. `any`-typed variable with direct `.prop` or `[key]` access
4. Async function call without try/catch, or `.then()` without `.catch()`
5. Array access via variable index without bounds check
6. Switch on union/enum missing `default` and not exhaustive
7. Type guard narrowing used after `await` boundary (value may have changed)
8. `as unknown as T` double assertion (incompatible type coercion)
9. Argument type mismatch detectable through type inference
10. **Trust boundary gap** (defensive programming): Missing null/abnormal defense on upstream props or backend response, but cannot confirm upstream necessarily passes abnormal values — **report at P2 maximum, usually skip**
11. Async race condition whose consequence is only UI flicker/toast flash, self-heals with no data side-effects
12. Async data loading timing issue causing brief wrong UI state, auto-recovers when data arrives
13. `useEffect(fn, [])` initialization logic that won't re-execute on subsequent changes, but cannot confirm actual problems
14. Controlled component not syncing external props changes (`useState(props.value)` without sync `useEffect`), when parent behavior is unconfirmed

### P1: High Risk (Probable runtime error or silent data corruption)

1. `any` value in arithmetic operations (+-*/) or assigned to strongly-typed variable without runtime typeof/schema validation
2. `fetch().json()` / `JSON.parse()` / `URLSearchParams.get()` / `localStorage.getItem()` return used without try/catch or schema validation when data source is untrusted
3. addEventListener without removeEventListener / setInterval without clearInterval / useEffect without cleanup — **P1 when**: (a) leak accumulates continuously (e.g., re-binds every render), OR (b) same listener registered multiple times in one effect but cleanup only removes one instance, OR (c) async operation (fetch/debounce/timer) created in effect without abort/cancel mechanism and the callback writes state. If only a single residual listener on unmount with no functional side effect → P2
4. useEffect/lifecycle async callback calling setState without unmount check — **only P1 when async is long-running AND race consequence involves data submission or unrecoverable state**; if consequence is only console warning or UI flicker → P2; React 18+ projects (check `package.json` for `react` version >= 18) → downgrade to P2 when no data submission consequence
5. Regex with nested quantifiers (`(a+)+`) or optional-repeat overlay — ReDoS risk
6. innerHTML / dangerouslySetInnerHTML / v-html with input from backend API without escaping → P1 (user-controlled input → P0)
7. Generic function without `extends` constraint, body accesses properties on generic param
8. `Object.keys(obj)` return used directly as `obj[key]` index (TS returns `string[]` not `(keyof T)[]`)
9. Custom type guard `is` implementation doesn't actually guarantee type correctness
10. `eval()` / `new Function()` / `setTimeout(string)` executing dynamic code
11. `obj[dynamicKey] = value` where dynamicKey from external input without filtering `__proto__`/`constructor`/`prototype` (prototype pollution)
12. Closure capturing outer let/var that is modified after closure creation (including loop variable capture)
13. useEffect referencing reactive variables not in dependency array (stale closure / infinite re-render)
14. Comparing object/array references in useMemo/useEffect deps or `===` (new reference every render → infinite effect / memo bypass)
15. Event handler setState after await without AbortController/cancel mechanism, AND race consequence involves data submission
16. Non-whitelist fields bypassing validation being auto-saved/submitted (unauthorized auto-operation)
17. Form submission without debounce/throttle causing duplicate orders/payments/submissions — P1 because backend may have idempotency control; escalate to P0 only when backend idempotency is confirmed absent
18. Sensitive operations (payment, refund, delete, export) without confirmation or permission check — P1 because this is a UX safety-net gap, not a certain data error; escalate to P0 only when the operation is irreversible AND no backend authorization exists
19. Concurrent submission without idempotency control causing duplicate charges/oversell/data inconsistency — P1 because backend idempotency may exist; escalate to P0 only when confirmed no server-side dedup

### P0: Critical Defect (Certain crash, data error, or security incident)

1. Data-flow confirmed variable is necessarily null/undefined on a reachable path, followed by unguarded property access (certain NPE)
2. Calling function/method on value confirmed to be null/undefined (frontend white-screen #1 cause)
3. `while(true)` without break/return exit; recursion without termination; useEffect deps triggering self-setState loop (infinite loop/recursion)
4. Condition expression contradicts its control flow semantics (e.g., `if(err) { saveData() }` should be `if(!err)`)
5. Floating-point used directly for currency/amount calculation without integer cents or Decimal library
6. setState called directly in render function/component body (infinite re-render)
7. `delete obj.id` then `save(obj)` where save accesses `obj.id`
8. Hard-coded API Key/Token/Secret strings (high-entropy + sensitive keywords)
9. Frontend URL concatenating userId/roleId controlling data access without backend verification (privilege escalation); innerHTML with user-controlled input without escaping
10. Calculation errors in amount/quantity/inventory/discount leading to over/under-charge, sign reversal
11. Time/timezone/timestamp errors causing timeout misjudgment, validity period errors, statistics offset
12. Pagination logic errors causing data duplication, omission, or page overflow
13. Critical business state check missing or incorrect, allowing cancelled/completed/disabled flows to continue
14. Function implementation completely contradicts its name/signature semantics (e.g., `checkNeedLogin` unconditionally returns false) — **MANDATORY DETECTION**: When a function's name/signature declares a dynamic semantic purpose (check, validate, verify, compute, fetch, ensure, guard, filter, etc.) but the implementation returns a fixed value without executing any logic corresponding to that purpose. The key criterion: the semantic purpose demands dynamic evaluation, yet the implementation is entirely static. MUST be reported as P0 regardless of whether caller/doc/invariant evidence is available. NOT caught: functions whose purpose IS to provide a constant (`getDefaultTimeout`, `getMaxRetries`).
15. Form validation failure not blocking submission (e.g., `form.validate().catch()` swallows error then continues submit)

### P0 Evidence Standard (mandatory self-check before assigning P0)

P0 means **certain** — not "likely", not "possible under some conditions". Before marking any defect as P0, it must pass this gate:

> **Can you prove the defect is triggered using ONLY facts visible within the current function and its immediate code context, without assuming anything about callers, runtime environment, or upstream behavior?**

If your conclusion depends on ANY of the following assumptions, the defect is **at most P1**:

- "If the caller passes null / empty / 0 / invalid input" — unless you can trace a concrete in-scope code path that produces that input
- "If the runtime environment is SSR / Node / lacks window / lacks specific API" — unless the project explicitly targets that environment (check package.json, tsconfig, or build config)
- "If the upstream doesn't validate" — unless you can read the upstream code in the current diff/TARGETS and confirm it doesn't
- "If the function is called when state is X" — unless you can show a concrete trigger sequence from a user action or lifecycle hook
- "The function should throw/reject instead of returning a fallback" — unless the function's name, signature, or doc explicitly promises to throw (functions named `safe*`, `try*`, `fallback*`, `compat*`, `default*` are presumed to intentionally return fallback values)

**P0 requires determinism**: the defect fires on a normal code path with inputs the function itself produces or is contractually guaranteed to receive. Anything that requires an assumption to trigger is P1 at most.

---

## Scoring Application Rules

### When to Use This Document

This scoring standard is applied:
1. **During defect detection** (Step 4 of SKILL workflow) — to calibrate the severity of discovered JS/TS defects
2. **During Defect Verification** (post-review filtering) — to validate that reported defects meet the minimum severity threshold and are not false positives

### Minimum Reporting Threshold

- Only report defects at **P1 and above** (P0, P1)
- P2 defects may be reported only when evidence is conclusive and the defect clearly affects correctness
- P3 and below: NEVER report

### Quantity Control

> **优先级说明**：当本文件的数量限制与 `detect-bugs.md` Section 5 的通用限制冲突时，以本文件（语言特定规则）为准。

Per source file, typically **0-5** defects are worth reporting. If you find more than 7, re-examine — but do not artificially cap at a low number if the defects are genuine and pass the 3-gate check.

### Pre-Report Self-Check (Every Defect Must Pass)

1. **Reachability**: Can the trigger input/state actually occur in normal usage? Remember: empty arrays, API errors, users with no data — these ARE normal. Only drop if trigger requires malicious input or impossible state combinations.
2. **Impact clarity**: What specifically happens when triggered? Must be crash, data error, functional failure, or security issue. "Could be improved" is insufficient.
3. **Bug vs design choice**: Is this a code error or an intentional design? Check the false-positive patterns in `defect-verifier.md` — if it matches, it's not a bug.
4. **Template application check**: Same pattern max 1-2 reports with the most severe instance.
