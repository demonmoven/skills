# Backend TD Document Template

Load this file when writing the TD markdown in Phase 4. The template below is the skeleton for a TD — conditional sections are tagged with the tier / classification-gate axis that unlocks them.

Write the doc to `/tmp/backend-td.md`.

Section headings are **verbatim** — do NOT prefix with numbers (`# 1. Background` is wrong). See `style-guide.md` for heading conventions.

---

```markdown
# Version History

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 0.1.0 | {YYYY-MM-DD} | Initial draft | AI · csp-backend-td-generator |

> **Update discipline:** every published edit MUST append a new row before `docs +update` is called. AI-driven edits use `AI · csp-backend-td-generator` in the Author column so reviewers can distinguish agent changes from human ones at a glance. Versioning: bump patch for typo/link/wording fixes, minor for new sections, major for tier or scope changes. See `update-flow.md` Step E.5.

# Background

| Meego | PRD | Expected Benefits |
|---|---|---|
| {meego_link} | <mention-doc token="{prd_token}" type="wiki">{prd_title}</mention-doc> | {1-3 short bullets} |

- {Current state — one bullet per fact. Cite specific services/methods/files.}
- {Pain point — what's broken or missing today.}
- {What the PRD asks for — one-line summary.}
- {In-scope for this TD — bullet list.}
- {Out-of-scope — bullet list.}

# Architecture / Data Flow

<whiteboard type="blank"></whiteboard>

*Fig — {one-line caption describing what the diagram shows, optional.}*

> **Template-only note (delete before publishing):** the Mermaid source below is NOT included in the published TD. It goes into the whiteboard via `lark-cli whiteboard +update` in Phase 5. Do NOT paste it as a fenced code block in the doc.
>
> ```mermaid
> graph LR
>   FE[FE] -->|GraphQL| BFF[BFF]
>   BFF -->|Thrift| PSM[our PSM]
>   PSM -->|Thrift| DOWN[downstream PSM]
>   PSM --> RDS[(RDS)]
>   PSM --> ABASE[(Abase)]
>   PSM --> TCC{{TCC}}
> ```

For trivial inline request-path sketches inside a subsection (≤5 boxes, no crossing edges), an ASCII fenced block is still acceptable.

# Swimlane

> **Swimlane = a routing header** (`X-TT-ENV` / RPC env tag) attached to HTTP & RPC calls so that boe/ppe traffic lands on the correct isolated test instance. Convention: `{env}_{feature_slug}` where `env ∈ {boe, ppe}` and `feature_slug` is a short snake_case identifier describing the **feature**, NOT the PSM (PSM names are too long and are shared across features).
>
> **Hard rules — violations break test-env routing:**
> 1. The `{feature_slug}` MUST be **byte-identical** across envs (same slug on `boe_` and `ppe_`). `boe_ai_chatbot` + `ppe_ai_summary` is broken.
> 2. Slug regex: `^[a-z][a-z0-9_]{1,29}$` — starts with a letter, snake_case, ≤30 chars total (the env prefix counts against the header's routing-label length cap).
> 3. Full swimlane regex: `^(boe|ppe)_[a-z][a-z0-9_]{1,29}$`.
> 4. Do NOT list a `prod_*` swimlane — prod has no swimlane (it's the default un-tagged route). Only boe and ppe get entries.
> 5. One slug per TD. Don't fragment the feature across multiple swimlanes unless the feature legitimately spans independently-tested subsystems (rare).

- boe_<feature_slug>
- ppe_<feature_slug>

# Dependency Overview  [M+]

One markdown table per category; omit empty categories.

**Downstream PSMs**

| PSM | Methods used | IDL source | Change | Owner |
|---|---|---|---|---|

**Upstream PSMs**

| Caller PSM | Methods called | BC impact | Owner |
|---|---|---|---|

**Data Stores**

| Type | Cluster / ID | Usage | Change | Owner |
|---|---|---|---|---|

# IDL / API Changes  [M+, omit if no IDL change]

**Interface IDL Full Definitions** — For every interface involved in this requirement (new, modified, or existing-but-called), list the complete request/response parameter IDL so the frontend team can formulate their technical plan. Use the following format per interface:

```thrift
// PSM: <psm>
service <ServiceName> {
    // Method description
    <Response> <methodName>(1: <Request> req)
}

struct <Request> {
    1: required i64 id          // field description
    2: optional string name     // field description
    3: required list<i64> ids   // field description
}

struct <Response> {
    1: required bool success    // field description
    2: optional string message  // field description
}
```

Requirements:
- Every field must include its number, required/optional, type, and description
- Enum types list all values with their meanings
- Nested structs are expanded recursively to leaf fields
- Existing interfaces called by this requirement but unchanged must also have full IDL listed (the frontend lacks prior context)
- Modified interfaces annotate change type per field: `[ADDED]`, `[MODIFIED]`, `[DEPRECATED]`

**Thrift IDL Diff** — for each method added/modified/deprecated, show a before→after code block and one-line backward-compat statement.

**Error Codes**

| Code | Meaning | When thrown | Caller expected action |
|---|---|---|---|

**Backward Compatibility** — one-line: do old clients still work? If not, caller migration plan.

# Data Model Changes  [M+, omit if no DDL]

**Schema**

```sql
{DDL statements}
```

- {Per statement: online-DDL safe? row count? estimated migration time?}

**Migration / Backfill** — bullets for batch size, throttle, verification, rollback.

**Read/Write Path Impact** — bullets for new query patterns, hot keys, index coverage, cache invalidation.

# Feature Design

## {Flow / module name}

**Challenge** — one bullet per hard problem.

**Approach** — bullets tying to real repo classes/packages.

**Components**
- `ClassName` — role
- `OtherClass` — role

**Flow** (ASCII sequence if non-trivial):

```
FE ──SendMessage──▶ BFF ──▶ PSM
                            │
                            ├─ insert USER row
                            ├─ call downstream.Invoke()
                            └─ insert ASSISTANT(PENDING) row
```

**Code** — real Java/Thrift/SQL/JSON snippets, not pseudocode.

## {next flow}
{...}

# Technology Selection & Key Decisions  [M+, omit if no significant decisions]

## Decision: {name}

| Approach | How it works | Pros | Cons |
|---|---|---|---|

- **Chosen:** {approach} — {specific reasons grounded in exploration data}
- **Rejected:** {per alternative, one-line rejection reason}
- **Verify before merge:** {if uncertain}

# Distributed Concerns  [CONDITIONAL — include only subsections flagged by the Gate]

## Timeout & Retry Budget  [if external calls]

| Call | Timeout | Retries | Total budget | Caller-side timeout |
|---|---|---|---|---|

## Failure Matrix  [if external calls]

| Scenario | Detection | Behavior | Recovery | User impact |
|---|---|---|---|---|

## Idempotency  [if write path]
- Per write op: key strategy, dedup window, storage.

## Consistency Guarantees  [if cross-region / multi-writer]
- Read-after-write? Cache-DB ordering? What's eventual vs strong?

# Capacity & Performance  [CONDITIONAL — traffic delta]

## Current Baseline

| Endpoint | QPS | p50 | p99 | Error rate |
|---|---|---|---|---|

From APM/Slardar. No placeholders.

## Projected Load
- Estimate delta from PRD traffic + upstream caller patterns.

## Resource Ask
- Instances, DB conns, MQ partitions, cache memory, TOS bandwidth.

# Observability  [CONDITIONAL — new observability need]

## Metrics

| Metric | Type | Tags | Alert threshold |
|---|---|---|---|

## Logs
- Structured log points & field conventions.

## Dashboards & Alerts
- Grafana/Slardar dashboards to create/update — include links.

# Security & Compliance  [CONDITIONAL — money/permissions/compliance]

{Auth/authz changes, PII handling, data retention, audit logging.}

# Rollout Plan  [CONDITIONAL — risky rollout]

## Feature Gate
{Settings/TCC key, default values, progressive rollout steps.}

## Canary & Ramp
{% canary → bake time → ramp steps. Success criteria per stage.}

## Kill Switch & Rollback
{What flips the feature off instantly. Data rollback story if needed after backfill.}

# DECC / Cross-Region Compliance  [MANDATORY if feature moves data across RoW / EU TTP / US TTP — including CN<->RoW, BOE<->RoW]

> **Load `decc.md` (sibling reference) before writing this section** — full required-fields table, design-time actions, label plan guidance, and DECC pitfalls live there. Do not rewrite from memory.

Summary line (keep in the published TD, even when the full section is elaborated from the reference):

- Scenario + Direction: `{e.g. Texas: RoW-TT -> CN}`
- Callee PSM + service type: `{rpc-psm | http-psm | http-domain}`
- Methods/fields crossing boundary + label-plan status: `{link to BAM / DECC UI}`
- Caller-side code changes required: `{kitex callopt / HTTP headers / Mesh switch}`
- Approval path: `{auto | manual}` — US is always manual

Fill in the full required-fields table and design-time checklist from `decc.md` (sibling reference).

# Benefits

- **Maintainability:** {...}
- **Correctness:** {...}
- **Scalability:** {...}
- **Extensibility:** {...}
- **Testability:** {...}

# Effort Estimation  [S+]

| Task | Effort (PD) | Dependency |
|---|---|---|

# Overall Schedule

| Milestone | Owner | Target Date | Status |
|---|---|---|---|

# Showcase Test Cases  [S+]

| Scenario | Setup | Expected Outcome | Status |
|---|---|---|---|

# Starling Keys  [OPTIONAL — only if i18n text added]

| Key | en-US | zh-CN | other locales |
|---|---|---|---|

# Open Questions & Risks  [S+, omit if none]

- {Question / risk} — owner: {name} — resolution path: {...}

# Technical Review TODO  [M+]

| Reviewer | Concern / Suggestion | Resolution |
|---|---|---|
```
