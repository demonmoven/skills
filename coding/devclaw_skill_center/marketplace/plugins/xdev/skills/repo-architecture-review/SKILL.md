---
name: repo-architecture-review
description: "Analyze an entire code repository for architecture-level problems. Phase 1: static tool analysis (layering violations, god packages, code clones, coupling, dead code, doc drift) producing a Markdown report. Phase 2: parallel multi-agent structural exploration — starting from Phase 1 symptoms, traces root causes by open-ended codebase exploration. Discovers missing architectural decisions (not code-level issues): anemic layers, implicit wiring, absent cross-cutting strategies, structural untestability, convention divergence, etc. Draws causal chains from symptoms to root causes. Supports Go and Python with full toolchain; other languages get graph-only coverage. Use when the user asks to 'analyze the architecture', 'review the codebase structure', 'find architecture smells', '仓库级架构分析', '架构体检', '结构性问题', or similar. NOT for function-level bugs, code style, naming, or performance review — those are out of scope."
---

# Repo Architecture Review Skill

This skill produces a repository-level architecture problem report for Go and Python repos.

## How to invoke (agent-mediated, recommended)

The recommended approach uses a **two-phase split**: the CLI runs all
deterministic analysis (L0-L2), then the agent processes LLM-dependent
groups using its own subagent capability, and finally the CLI renders the
report. This avoids spawning external `claude` processes entirely.

### Step 1: Run tools phase (no LLM)

> All commands below reference `dist/cli.js` **relative to this SKILL.md file's
> directory**. Before running, resolve `SKILL_DIR` to the absolute path of the
> directory containing this SKILL.md (e.g. `~/.claude/skills/repo-architecture-review`).

```bash
node "$SKILL_DIR/dist/cli.js" analyze . --phase tools
```

This runs L0 prescan (with fallback layering convention), L1 tool adapters,
L2 graph metrics, grouping, and signal crossing. It produces:

- `.architecture-review/tmp/repo-profile.json` — detected languages, packages, configs
- `.architecture-review/tmp/tool-runs.json` — per-tool results
- `.architecture-review/tmp/l2-violations.json` — graph-level violations
- `.architecture-review/tmp/deterministic-findings.json` — findings from metric-hard-evidence groups (cycles, god packages, code clones, layering violations, horizontal coupling) — these need no LLM
- `.architecture-review/tmp/pending-llm-groups.json` — violation groups that need LLM judgment and synthesis
- `.architecture-review/tmp/layer-discovery-inputs.json` — inputs for optional LLM-based layering discovery
- `.architecture-review/tmp/run-meta.json` — timing metadata

### Step 2: Agent processes pending LLM groups via subagents

Read `.architecture-review/tmp/pending-llm-groups.json`. Each entry has:

```json
{
  "id": "grp:...",
  "category": "doc-drift",
  "commonFile": "path/to/file.go",
  "commonDir": "path/to/",
  "violations": [ { "source": "...", "ruleId": "...", "locations": [...], "evidence": {...}, "message": "..." } ],
  "crossed": { "independentSources": ["l1-tool", "l2-graph"], "baseConfidence": "high" }
}
```

For each pending group, launch a **Task subagent** (`subagent_type: general`)
with a prompt that includes the group data and asks it to produce a Finding.
Launch groups in parallel (up to 5-8 concurrent subagents).

**Subagent prompt template** (paste the group's JSON where indicated):

```
You are an architecture reviewer for a code repository. You will analyze a
group of related static-analysis violations and determine whether they
represent a real architecture problem.

The repo root is: {REPO_ROOT}

Here is the violation group:
{GROUP_JSON}

Base confidence from signal crossing: {crossed.baseConfidence}
Independent sources: {crossed.independentSources}

Instructions:
1. Read the files mentioned in the violations' locations to understand the
   actual code context. Use the Read tool on the relevant files.
2. Determine if this is a real architecture problem, noise, or insufficient
   evidence.
3. If real, write a finding in Chinese (for a senior engineer audience).
4. Return ONLY a JSON object (no markdown fences) matching this schema:

{
  "verdict": "real" | "noise" | "insufficient",
  "title": "one-line Chinese title, max 30 chars, specific not generic",
  "rootCause": "1-3 paragraphs in Chinese explaining why this happens, citing specific files",
  "impact": "1 paragraph in Chinese: how this hurts maintenance, performance, onboarding, etc.",
  "actions": ["Chinese action items, verb-first, cite specific files/packages"],
  "confidence": "high" | "medium" | "low"
}

Writing rules:
- title must be specific. Bad: "耦合问题". Good: "biz/dal/init.go 充当全应用启动总线"
- rootCause must read like paragraphs, not bullet points
- impact must be understandable by a PM, avoid jargon like "best practices"
- actions must start with verbs, cite file paths or package names
- FORBIDDEN phrases: "consider refactoring", "best practices", "may want to",
  "should probably", "建议关注", "值得重视"
- If evidence is insufficient, return verdict="insufficient" with empty
  title/rootCause/impact/actions
```

After all subagents return, build the `llm-findings.json` array. For each
subagent result with `verdict === "real"`, construct a Finding:

```json
{
  "id": "<group.id>",
  "category": "<group.category>",
  "title": "<from subagent>",
  "rootCause": "<from subagent>",
  "impact": "<from subagent>",
  "actions": ["<from subagent>"],
  "confidence": "<from subagent>",
  "confidenceScore": 0.85,
  "severity": "<worst severity from group.violations>",
  "signals": [{"source": "<v.source>", "description": "<v.message>"}],
  "locations": ["<all locations from group.violations>"],
  "violationIds": ["<all violation ids>"]
}
```

For groups where the subagent returned `verdict !== "real"`, create a
low-signal passthrough finding:

```json
{
  "id": "<group.id>",
  "category": "<group.category>",
  "title": "[LOW-SIGNAL] <first violation message>",
  "rootCause": "Agent subagent classified as <verdict>. Raw evidence: ...",
  "impact": "Low signal — reported for transparency.",
  "actions": ["Manually inspect the listed locations"],
  "confidence": "low",
  "confidenceScore": 0.2,
  "severity": "low",
  "signals": [...],
  "locations": [...],
  "violationIds": [...]
}
```

Write the array to `.architecture-review/tmp/llm-findings.json`.

### Step 3: Render report

```bash
node "$SKILL_DIR/dist/cli.js" analyze . --phase report \
  --output architecture-analysis-$(date +%Y-%m-%d).md
```

This reads `deterministic-findings.json` + `llm-findings.json`, merges and
ranks them, then renders the full Markdown report.

### Alternative: full pipeline (requires `claude` CLI)

If the `claude` CLI is available in PATH, the original single-command mode
still works:

```bash
node "$SKILL_DIR/dist/cli.js" analyze . \
  --output architecture-analysis-$(date +%Y-%m-%d).md \
  --no-baseline        # disable incremental filtering
  --mode deep          # default; alternatives: quick
```

This runs the entire L0-L4 pipeline internally, spawning `claude -p` for LLM
calls. Use `--phase all` (or omit `--phase`) for this mode.

## What it does

1. **L0 pre-scan**: detects languages, build tools, CI, Dockerfiles; captures Go module path. **NEW: uses LLM to discover the repo's actual layering convention** (e.g. handler → service → dao, or cmd → app → domain → infra) instead of assuming a fixed convention.
2. **L1 tools (parallel)**: runs import-linter, grimp, pydeps, arch-go, **go-arch** (built-in), goda, gocyclo, go-list, hadolint, actionlint, code-maat, scc. Auto-installs missing ones and auto-generates starter configs if absent. Tools that crash are reported as FAILED and excluded from health scoring — never silently collapsed to "0 violations".
3. **L2 unified graph (parallel)**: tree-sitter + git + L1 imports → one graph. Computes modularity, centrality, Alves per-repo percentile thresholds, co-change, hotspot, bus factor, orphan, directory Gini, doc-drift. **NEW: god-file thresholds are adaptive** — computed from the repo's own file-size distribution (p90/p97) instead of hardcoded.
4. **L3 LLM fusion**: groups violations by category + location; multi-signal crossing (≥2 independent sources = high confidence); **metric-hard-evidence bypass** for deterministic findings (cycles, god packages, code clones, layering, horizontal coupling) — these skip LLM triage and go straight to the main report with a hand-written narrative; semantic LLM judgment still applies to fuzzy categories (doc-drift, etc.). In **two-phase mode**, L3 is handled by the agent's own subagents instead of spawning external `claude` processes — see "How to invoke" above.
5. **L4 report**: single Markdown file with **Dashboard Summary** → **Action Roadmap** → **Findings** (grouped by severity, card-style) → **Panoramic Metrics** → Known Issues (folded) → **Execution Audit** (table format). Low-signal items are collapsed by category.

### Dynamic layering convention discovery

Instead of hardcoding a single layering convention, the skill now uses LLM to
analyse the repo's directory structure and infer its actual layering pattern
during L0 pre-scan. This means it works correctly on:

- ByteDance web-services: `handler → consumer → facade → service → repository → dao → dal`
- DDD-style projects: `cmd → application → domain → infrastructure`
- Clean Architecture: `delivery → usecase → repository → entity`
- Or any custom layering convention the repo follows

If the LLM is unavailable or the repo has no clear layering, it falls back to
the classic ByteDance convention. The discovered convention is shown in the
report's metrics section.

### Adaptive thresholds

Thresholds for God Package detection, God File detection, and code clone
detection are now computed from the repo's own statistical distribution:

- **God Package fan-in**: uses the repo's fan-in p90 percentile (floor: 5)
- **God File LOC**: uses p90 for medium severity, p97 for high (floors: 300, 800)
- **Code clones**: uses 40% of the median file size (floor: 50 lines)

The actual thresholds used are transparently shown in evidence signals and
the panoramic metrics section, so the reader knows exactly why a finding
was flagged.

### Built-in `go-arch` adapter — what it detects

The skill ships a pure-TypeScript `go-arch` adapter that does not rely on
third-party binaries and cannot silently crash. It derives five
deterministic signals from `go list -deps -json`:

1. **Package cycles** (Tarjan SCC) — true architectural cycles.
2. **Layering violations** — inner layer importing outer under the **repo's
   discovered layering convention**. Each (srcLayer → depLayer) pair becomes
   one aggregated finding with all violating edges as evidence.
3. **Horizontal coupling** — sibling modules under business layers importing
   each other. Infra layers are excluded to avoid false positives.
4. **God packages** — fan-in above repo's p90 percentile AND instability
   `I = fan_out/(fan_in+fan_out) > 0.5`.
5. **Cross-package code clones** — above the adaptive identical-line threshold
   AND ≥ 50% overlap ratio between two files in different package dirs.

Each signal fires its own Chinese narrative finding at high confidence.

Run time: 2-8 minutes on medium repos (500-2000 files); map-reduce chunking kicks in above 5k files.

### Report structure

The report is designed to be read in 3 minutes or 30 minutes:

1. **Dashboard summary** — health score, severity distribution, top problems (3-minute read)
2. **Action roadmap** — prioritized P0/P1/P2 table with first suggested action
3. **Detailed findings** — grouped by severity; each finding is a card with root cause, impact, actions, and collapsible evidence
4. **Panoramic metrics** — language stats, layering convention, adaptive thresholds
5. **Low-signal reference** — collapsed by category, minimal visual weight
6. **Execution audit** — tool status table with timing

## External contract

The only user-facing output is `architecture-analysis-YYYY-MM-DD.md` at the repo root (or `--output` path). All intermediate artifacts live in `.architecture-review/tmp/` and can be inspected or deleted.

## Limitations

- Languages supported with full toolchain: Go, Python.
- Other languages get graph-only coverage (tree-sitter + git), reported as "not-scanned" for tool-dependent categories.
- Java/Kotlin: ArchUnit cannot be auto-installed (it's a test-framework library); the skill recommends integration but will not write test files.
- In two-phase mode (`--phase tools` + agent + `--phase report`), layer discovery uses the fallback ByteDance convention. The agent can optionally do LLM-based layer discovery by reading `layer-discovery-inputs.json` and updating `repo-profile.json`.
- In full pipeline mode (`--phase all`), LLM layer discovery requires the `claude` CLI in PATH. If unavailable, falls back to the default convention.

## Artifacts

After running `analyze .`, the report directory `.architecture-review/` contains:

- **`architecture-analysis-YYYY-MM-DD.md`**: Single Markdown report (external contract; safe to share).
- **`baseline.json`**: Fingerprints of all findings from the last run.
- **`tmp/`**: Internal working directory (safe to delete; next run rebuilds).

All `.architecture-review/` content is safe to delete; the next `analyze` run rebuilds from scratch.

## Phase 2: Structural Problem Exploration

The CLI tool (Phase 1) detects import-graph level issues: layering violations, god packages, code clones, cyclomatic complexity. But there is a class of **structural problems** that static tools cannot catch — they require semantic understanding of the codebase's runtime behavior, design intent, and missing abstractions.

**After the tool-based report is generated**, the agent SHOULD proactively offer to run Phase 2: a parallel multi-agent deep exploration of the codebase to surface structural problems. This is especially valuable for business repositories with 200+ files.

### What are structural problems (vs code-level issues)

Structural problems are **systemic pattern deficiencies** that affect the entire codebase's ability to evolve. They are NOT:
- ❌ "This function has a hardcoded value" (that's a CR-level issue)
- ❌ "This line swallows an error" (that's a CR-level issue)
- ❌ "This variable name is unclear" (that's code style)

They ARE:
- ✅ "The codebase has no error propagation strategy — 99% of errors are returned naked"
- ✅ "The service layer is systematically bypassed — handlers do business orchestration directly"
- ✅ "There is no async safety infrastructure — every developer invents their own goroutine pattern"
- ✅ "The dependency graph is wired at runtime through mutable globals with no compile-time guarantees"

The key test: **would fixing one instance solve the problem, or does it require an architectural decision that changes how the whole team writes code?** If the latter, it's structural.

### Methodology: How to discover structural problems

Structural problems are unique to each repo. Do NOT apply a fixed checklist. Instead, use these **thinking tools** to guide open-ended exploration.

#### Thinking tool 1: The intended-vs-actual gap

Every codebase has two architectures (cf. Sonar's "architecture gap", SEI's architecture erosion research):
- **The intended architecture**: what the directory names, README, or lead developer's mental model says it should be.
- **The actual architecture**: what the import graph, call patterns, and runtime behavior reveal it truly is.

The gap between them IS the structural problem. Phase 1 tools detect the actual architecture's shape. Phase 2 asks: **where does the actual diverge from the intended, and why?**

Approach: First, reconstruct the intended architecture from directory structure, naming conventions, any existing docs. Then, compare against what Phase 1 revealed. Every divergence is a candidate structural finding — but only if it's systemic (affects a whole layer or pattern), not incidental (one developer made a shortcut).

#### Thinking tool 2: Decision archaeology

Inspired by Architecture Decision Records (ADRs) and SEI/CMU's ATAM: every structural problem is a **decision that was never explicitly made**. The team didn't decide to have bad error handling — they simply never decided how errors should be handled. The absence of a decision is the root cause; the inconsistency is the symptom.

When you find a structural pattern, ask: **what question was never answered?**
- "Which layer is responsible for wrapping errors with context?"
- "How should a new feature module share logic with existing modules?"
- "What happens to an in-flight request when a downstream dependency is unreachable?"
- "How does a new developer know which of the 4 handler patterns to use for a new endpoint?"

If the codebase cannot answer these questions — through docs, conventions, or code structure — that's a missing decision.

#### Thinking tool 3: Scenario probing

From ATAM (Architecture Tradeoff Analysis Method): the best way to expose structural weaknesses is to ask **"what happens when..."** questions. These are not hypothetical — they are concrete scenarios that the codebase must handle:

- **Change scenario**: "A new developer needs to add a feature similar to an existing one. What do they copy? What can they reuse? Where do they get lost?" — This reveals missing abstractions, convention divergence, and documentation gaps.
- **Failure scenario**: "The review service is down for 5 minutes. What does this service do?" — This reveals fail-mode ambiguity.
- **Scale scenario**: "Request volume doubles. Which goroutine/connection/queue patterns break first?" — This reveals concurrency architecture gaps.
- **Onboarding scenario**: "Someone reads the code for the first time. Can they understand the request flow from entry point to database without reading every file?" — This reveals layering integrity.

These scenarios are repo-specific. Design them based on what the repo does and what Phase 1 symptoms suggest.

#### Thinking tool 4: The "three instances" rule

From the ARC methodology: "Don't abstract until you have three concrete implementations." Apply this in reverse for analysis — **don't call something a structural problem until you see it in three or more independent places.**

- One fat handler is a code review issue.
- Three fat handlers following the same anti-pattern is a structural problem (missing service layer convention).
- One swallowed error is a bug.
- Swallowed errors in 20+ files across all layers is a structural problem (no error handling strategy).

This rule prevents over-reporting and keeps findings at the architectural level.

#### Thinking tool 5: Trace actual behavior, not just static structure

Static analysis (Phase 1) sees import edges. But structural problems often live in **runtime behavior** that imports don't capture:

- A function that's 500 lines long isn't visible in the import graph, but it reveals a missing decomposition.
- Init functions that must run in a specific order aren't visible in imports, but they reveal implicit wiring.
- A goroutine that outlives its parent request isn't visible in imports, but it reveals a missing lifecycle model.
- A response that silently succeeds when a dependency fails isn't visible in imports, but it reveals fail-mode ambiguity.

Design subagent prompts that ask about **behavior**: "what actually happens when a request arrives?", "how does the system start up?", "what happens when this fails?", not just "what imports what?"

### How to run the exploration

Launch **3-5 parallel subagents** (Task tool, `subagent_type: explore`). Each investigates one broad question **derived from this repo's specific Phase 1 symptoms and context**.

**Three sources for designing exploration questions:**

1. **Symptom-driven**: Start from a Phase 1 finding and ask WHY.
   - "The tool report shows 30 code clones between service modules. Explore WHY — is there a missing shared abstraction? Is the architecture structured in a way that makes sharing impossible?"

2. **Scenario-driven**: Pick a scenario relevant to this repo's domain and trace it.
   - "Trace what happens when a video generation request arrives, from handler to queue to completion. How many layers does it cross? Where does business logic actually live?"

3. **Surprise-driven**: Follow a dependency edge or pattern that seems wrong and ask what forced it.
   - "The import graph shows dal importing handler. Explore what dal/init.go actually does — is it a god-init? How are dependencies connected?"

**Key instruction for each subagent:**

> You are looking for STRUCTURAL patterns, not individual code issues. Apply the "three instances" test: don't report something unless you see it in 3+ independent places.
>
> A structural pattern is one that:
> - Affects an entire architectural layer or cross-cutting concern
> - Cannot be fixed by changing one file — requires a team-level design decision
> - Is the result of a MISSING convention or decision, not a developer mistake
>
> For each pattern found, report: (1) what the pattern is, (2) how pervasive it is, (3) what architectural decision is missing, (4) what downstream consequences it causes.
>
> Do NOT report individual instances. DO report systemic patterns.

### How to synthesize findings

After all subagents return, the most valuable insight is NOT a list of problems — it's the **causal structure** between them.

#### Finding root causes

1. Collect all structural findings from subagents.
2. Apply the "fix one instance" test — discard anything that's really code-level.
3. Apply the "three instances" test — discard anything that's incidental, not systemic.
4. For each remaining finding, ask: "is this caused by another finding?" Draw arrows.
5. Find the roots of the causal graph — these are the true structural problems. Everything else is a downstream consequence.

Most repos have **1-3 root causes** that cascade into many symptoms. The tool report's separate findings (code clones, layering violations, god packages) often trace back to the same root.

#### Framing as missing decisions

Each structural problem should be named as a **missing architectural decision**, not as a bad pattern. This shifts the conversation from blame to architecture:

- ❌ "Errors are handled inconsistently" (describes symptoms)
- ✅ "No error propagation strategy has been decided — the team has never chosen which layer wraps errors, what error types to use, or how to preserve context" (names the missing decision)

- ❌ "There are too many global variables" (describes symptoms)
- ✅ "No dependency injection approach was chosen — components are connected through runtime mutable state with no compile-time guarantees about initialization order" (names the missing decision)

For each missing decision, consider what an ADR (Architecture Decision Record) would look like if the team made this decision today — what's the context, what are the options, what are the tradeoffs? This helps frame actionable recommendations.

### What to include in the report

1. **Each structural problem**: named as a missing architectural decision, with evidence of pervasiveness (not file:line lists)
2. **The causal graph**: showing root causes vs downstream consequences, connecting back to Phase 1 symptoms
3. **A leverage-ordered roadmap**: fix root causes first. "If you only do one thing, do X — it unblocks Y and Z."
4. **Explicit separation** from Phase 1: Phase 1 = WHAT the architecture looks like. Phase 2 = WHY it looks that way.

### Vocabulary (reference, not checklist)

Terms for naming structural problems you discover. A repo may have none of these, or may have problems not on this list. Use these as vocabulary, not as a rubric.

- **Intended-actual gap**: the directory structure declares an architecture the code doesn't follow.
- **Anemic layer**: a layer exists in name but is systematically bypassed in practice.
- **Missing shared abstraction**: modules duplicate logic because no sharable interface exists (possibly blocked by circular deps or layering rules).
- **Implicit wiring**: components connected via global state, init ordering, or import side effects rather than explicit construction.
- **Absent cross-cutting strategy**: a universal concern (errors, retries, timeouts, auth) has N incompatible implementations because no convention was chosen.
- **Convention divergence**: multiple patterns coexist for the same concern, indicating conventions were never established or enforced.
- **Structural untestability**: code can't be tested not because tests are missing, but because dependency structure prevents isolation.
- **Fail-mode ambiguity**: critical-path behavior on dependency failure is neither designed nor documented.
- **Architecture erosion**: the intended architecture was once followed but has degraded through accumulated shortcuts — distinct from "architecture drift" where the documentation is simply out of date (cf. Li et al., 2023).
