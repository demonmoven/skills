# Defect Verifier Protocol (JS/TS)

> Lightweight post-review confidence check for JavaScript/TypeScript targets. Modeled after the Go approach: 3 validation gates + explicit false-positive pattern matching.

---

## Purpose

The Verifier is NOT a strict filter that aggressively drops findings. Its job is to:
- Apply a lightweight 3-gate confidence check (matching Go's approach)
- Recognize and drop **explicit false-positive patterns** (listed below)
- Deduplicate same-pattern reports
- Calibrate severity using `defect-scoring.md`

**Design principle**: Better to keep a borderline-real defect than to miss a genuine bug. The 3 gates are confidence signals, not hard barriers.

## When to Execute

Execute after defect detection (Step 4) produces candidate defects and before finalizing BUG_MAP.

---

## 3-Gate Confidence Check

For each candidate defect, sequentially evaluate:

### Gate 1: Business Scenario Gate

> "Would this issue be triggered in a real user scenario?"

- If the trigger condition is a normal business state (empty list, API error, user clicks fast) → **PASS**
- If the trigger requires extreme/malicious input that normal users wouldn't produce → **WEAK**
- If the trigger is practically impossible in the application context → **FAIL**

**Important**: Do NOT fail this gate just because the trigger requires a specific state. "User has no data" / "API returns error" / "all items filtered out" are NORMAL scenarios.

### Gate 2: Developer Acceptance Gate

> "If reported to the developer, would they acknowledge this as a bug worth fixing?"

- If a reasonable developer would say "yes, this needs fixing" → **PASS**
- If a developer would say "that's by design" or "not worth the effort" → **WEAK**
- If a developer would say "that's not a bug at all" → **FAIL**

**Heuristic**: Crashes, data corruption, duplicate submissions, silent failures on critical paths → developers always want to fix these.

### Gate 3: Evidence Conclusiveness Gate

> "Can I point to specific code lines showing a logic contradiction or missing critical logic?"

- If you can quote exact lines + explain the semantic contradiction → **PASS**
- If the function's name/signature declares a dynamic semantic purpose (check, validate, guard, compute, fetch, ensure, filter, etc.) but the implementation returns a fixed value without executing any corresponding logic → **PASS** — the mismatch between declared dynamic purpose and static implementation is provable from the function alone. This is a mandatory-detection pattern per `defect-scoring.md` P0 #17.
- If the issue is plausible but requires assumptions about runtime state → **WEAK**
- If the issue is purely hypothetical with no concrete code evidence → **FAIL**

### Decision Matrix

| Gate 1 | Gate 2 | Gate 3 | Decision |
|--------|--------|--------|----------|
| PASS | PASS | PASS | **KEEP** (high confidence) |
| PASS | PASS | WEAK | **KEEP** (note uncertainty in evidence) |
| PASS | WEAK | PASS | **KEEP** (may adjust severity down) |
| WEAK | PASS | PASS | **KEEP** (may adjust severity down) |
| PASS | WEAK | WEAK | **KEEP at P2 max** |
| WEAK | PASS | WEAK | **KEEP at P2 max** |
| WEAK | WEAK | PASS | **KEEP at P2 max** |
| FAIL | any | any | **DROP** |
| any | FAIL | any | **DROP** |
| any | any | FAIL | **DROP** |
| WEAK | WEAK | WEAK | **DROP** |

**Key difference from strict 6-gate**: Passing 2 of 3 gates is sufficient. Only drop when a gate is outright FAIL or all 3 are WEAK.

---

## Explicit False-Positive Patterns (JS/TS)

Apply all false-positive patterns defined in `defect-scoring.md` "False Positive Prevention Principle" section. If a candidate matches any pattern listed there, DROP immediately regardless of gate results.

Additionally, the following patterns specific to the verification phase also trigger immediate DROP:

1. **Optimistic UI patterns**: Deleting local state before confirming server success (optimistic updates are standard UX)
2. **Race conditions requiring sub-100ms user interaction**: If triggering the race requires clicking faster than humanly possible or requires specific network timing that self-resolves

---

## Deduplication Rules

- Same defect pattern in one file → keep only the most severe instance
- Same root cause manifesting in multiple locations → keep only 1
- Max **5** defects per source file after deduplication (relaxed from 3 to avoid over-filtering)

---

## Integration with Workflow

```
Step 4: Defect detection produces candidate defects
    ↓
Defect Verifier (THIS PROTOCOL)
    ├── Check against false-positive patterns → DROP matches
    ├── Apply 3-gate confidence check → DROP only clear FAILs
    └── Deduplicate same-pattern reports
    ↓
Finalized BUG_MAP (for kept defects)
    ↓
Step 5: Test generation with defect scenarios
```

### Key Principles

1. **Bias toward recall over precision** — a borderline-real defect kept at P2 is better than a genuine P0 dropped
2. **False-positive patterns are hard rules** — if a candidate matches an explicit pattern above, DROP regardless of gate results
3. **Evidence insufficiency ≠ automatic DROP** — if gates 1+2 pass but evidence is soft, keep at lower severity rather than dropping
4. **Do NOT add new defects** during verification — only filter/adjust existing candidates
5. **Can revise** description, severity, evidence wording — but cannot transform into a completely different defect
