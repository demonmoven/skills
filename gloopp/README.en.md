# Gloop

> Agents reincarnate as adventurers, then fall into the loop.

[中文](README.md) | English

Gloop is a **local-first Loop Engineering workbench**. It orchestrates ACP/CLI coding agents into a role-based adventurer party, but the platform owns the loop end-to-end: goal definition, decomposition, execution, acceptance, rework, and final human review are all managed explicitly by Gloop.

The point is simple: **one agent must never interpret a goal AND declare itself done**. Gloop keeps goals, phase outputs, events, reviews, and stats outside the model context, persisted to `~/.gloop`.

---

## Quick Start (recommended: install via npm)

### Prerequisites
- Runtime: Node `>=18` (only for `npm install` to download the prebuilt native binary; Gloop itself is a single Go binary, no Node required at runtime).
- Optional: any of `aiden`, `relay`, `pi`, `codex`, `traex`, or an ACP agent on `PATH`. The Mock executor is retained for diagnostics and tests, but active adventurers must bind to a real, enabled agent.
- For source builds: Go `1.26.1+`.

### Install & start in one go
```bash
npm i -g @bytedance-dev/gloop --registry https://bnpm.byted.org
gloop start
```

`gloop start` ships with three defaults on (each has a `--no-*` flag to turn it off):

| Default | What it does | How to disable |
|---|---|---|
| 🚀 **Detached / background** | `setsid` on Unix / `CREATE_NO_WINDOW` on Windows; stdout/stderr redirected to `~/.gloop/server.log` | `--no-detach` or `-nd` |
| 🌐 **Auto-open dashboard** | Waits until `/healthz` passes, then opens the token-authenticated Dashboard URL with `open` / `xdg-open` / `rundll32` | `--no-open` or `-no` |
| 🔁 **Register login-item autostart** | User-level, **no sudo required**: launchd (macOS) / systemd `--user` (Linux) / `Startup\gloop.vbs` (Windows) | `--no-autostart` or `-na` |

When started you'll see:
```text
🟢 Gloop started in background
   Listen:      http://0.0.0.0:37317
   Data Dir:    /home/you/.gloop
   Dashboard:   http://10.x.x.x:37317/dashboard?t=<24-byte bind token>
   Log:         /home/you/.gloop/server.log
   Pid:         1733333  (use `gloop stop` to shut down)
```

The `?t=` query parameter in the Dashboard URL is a per-machine bind token that prevents unrelated processes on the same dev box from reading/writing `~/.gloop`. **Strip it before sharing screenshots or URLs.** If interface auto-detection picks the wrong IP, pin it explicitly:
```bash
gloop start --host 0.0.0.0 --external-host 10.4.4.239 --port 37317
```
If the default port is taken, use `--port 0` to auto-scan 20 ports starting from 37317.

### Day-to-day operations
```bash
gloop status        # pid / Dashboard URL / autostart state in one line
gloop dashboard     # open the current Dashboard
gloop dashboard --print
                    # print the Dashboard URL only, useful in remote terminals
gloop logs -n 50    # last 50 lines of server.log
gloop logs -f       # follow mode (tail -f)
gloop restart       # stop + start; pass the same flags as start
gloop stop          # SIGTERM → 15s graceful window → SIGKILL fallback
```

**When do I need to restart?** (bootstrap-time values don't hot-reload)
- Editing `~/.gloop/config.json` (ports, budgets, concurrency, limits).
- Adding/editing agents under `~/.gloop/agents/`.
- Upgrading the binary itself with `npm i -g @bytedance-dev/gloop --registry https://bnpm.byted.org`.

**When you do NOT need to restart**: adventurers, prompts, automation rules, individual quests — those are read at runtime.

### Build & run from source
```bash
go test ./...
go run ./cmd/gloop start --port 37317
```

### Update notifications
`@bytedance-dev/gloop` checks for updates in the background (cached for 24h, uses the standard `update-notifier` flow). To turn it off:
```bash
export GLOOP_NO_UPDATE_NOTIFIER=1
```

---

## The controlled loop

```text
Create quest
  → Warrior (Maker) understands the task, executes changes, runs `gloop phase done`
  → Mage (Checker) audits output, may run auditable validation commands,
     then runs `gloop review pass/request-changes/reject`
  → Acceptance Checker scores goal/check state
  → blocked / reviewing / needs_rework (up to MaxReworkPerQuest auto-reworks)
  → Human passes, requests changes, or rejects
        ├ pass    → apply to working dir (git apply --check preflight + backup snapshot)
        ├ changes → back to Warrior phase (rework counter +=1)
        └ reject  → discard artifacts, terminal state
```

The two-class model is **maker/checker by construction** — a warrior can never silently approve their own work. The mage runs in a discardable sandbox copy and validation commands remain allowlisted. This is the foundation of loop engineering here.

### Quest intensity

Intensity is an internal budget tier, not a semantic classifier. Gloop does not infer task difficulty; it only applies the tier explicitly chosen by the user or an automation rule.

| Intensity | UI label | Budget behavior |
|---|---|---|
| `quick` | Quick | 10 turns / 1 rework / 30-minute quest; skips mage review and goes straight to human review |
| `standard` | Standard | Follows global config, defaulting to 50 turns per phase / 3 reworks / 3-hour quest |
| `deep` | Deep | 80 turns / 4 reworks / 8-hour quest |
| `adversarial` | Strict Review | 100 turns / 5 reworks / 12-hour quest; mage should provide stronger evidence |

Mage verdicts are not rewritten by score thresholds. If a mage returns `pass` with a low score, or if rework progress looks stale, Gloop records an auditable `review.signal` for the Dashboard and human review instead of changing the mage's verdict or silently increasing intensity.

### Mage validation commands

The mage uses Bash to invoke the local `gloop ...` CLI for review protocol and validation actions. Review phases run in a discardable sandbox copy: `GLOOP_WORKSPACE_PATH` and `GLOOP_CONTEXT` point at the sandbox, so the real workspace path is not exposed to the mage process. Phase-end signals use `GLOOP_SIGNAL_WORKSPACE_PATH` to write back to the real quest workspace, so `gloop review ...` can be consumed by the orchestrator while normal file writes still stay in the sandbox. Direct workspace writes are captured as `mage_sandbox_writes_captured` and discarded at phase end.

Validation commands still execute only through audited command IDs listed in `mage_command_allowlist`. The Settings page can scan the current project and recommend test, build, lint, and typecheck commands from files such as `go.mod`, `package.json`, `pyproject.toml`, and `Cargo.toml`; recommendations are never enabled automatically and must be added to the allowlist explicitly.

Commands can declare `side_effect_level`: `L0` for read-only analysis, `L1` for local reversible changes, and `L2` for external side effects. L2 commands are rejected by default and only run when an automation explicitly enables `allow_l2` while `auto_apply` is disabled.

### Common quest commands
```bash
# Create an execute quest (blocks until reviewable)
gloop quest run --query "Translate the README to natural, technically accurate English"
gloop quest run --query "Strictly review this release config change" --intensity adversarial
# Create a design-only quest; later spawn an execute quest from its output
gloop design --query "Redesign the agent adapter layer to support multi-model fallthrough"

gloop quest list
gloop quest show qst_abcd1234 --events 20
gloop quest diff qst_abcd1234 --full
gloop quest review qst_abcd1234 pass --comment "LGTM, apply"
gloop quest apply qst_abcd1234
gloop quest comment qst_abcd1234 --comment "please also cover edge cases"
gloop quest answer qst_abcd1234 --answer "Use the existing API shape"
gloop quest backup list qst_abcd1234
gloop quest spawn --query "Split out the documentation follow-up" --group-id docs
gloop quest discard qst_abcd1234
gloop quest cancel qst_abcd1234
gloop quest resolve-blocked qst_abcd1234 --action continue
gloop quest spawn-execute qst_abcd1234 --start
gloop quest cleanup --dry-run   # purge terminal workspaces per workspace_retention_days

# Adventurers, config, skills, stats
gloop adventurer list
gloop config show
gloop skill list [--kind capability|orchestration] [--category ...]
gloop skill show gloop-quest-execution
gloop stats --range week

# Diagnostics and local benchmarks
gloop doctor agents --json
gloop doctor agents --smoke --json
gloop doctor notifications --json
gloop doctor e2e --json
gloop doctor e2e --agent relay --json

# User context (cross-quest global background knowledge)
gloop context list
gloop context show workspace
gloop context summary
gloop context refresh
gloop context export --format okf --out ./knowledge-bundle
gloop context write-dim workspace --content-file ./workspace-context.md
gloop context write-summary --summary-file ./summary.md

# Mage command allowlist management
gloop command list
gloop command run <command-id>

# Built-in skills
gloop skill list [--kind capability|orchestration] [--category ...]
gloop skill show gloop-quest-execution
gloop skill validate

# IDL export
gloop idl export [--only all|tools|skills|abi|classes|agent-schemas]
```

`doctor agents` checks configured agents, executable resolution, adapter support, versions, and optional CLI smoke calls. `doctor e2e` runs a local Mock-backed loop through quest creation, execution, human review, diff, and apply; pass `--agent <name>` to run a real-agent E2E contract check. The real-agent contract path pins the current Gloop binary through `$GLOOP_DOCTOR_CURRENT_BINARY` so the agent does not accidentally call an older installed binary.

A `.gloopignore` file at the project root is honored in copy/worktree modes; blank lines, `#` comments, exact paths, directory prefixes, and basic globs are supported.

---

## Data layout (`~/.gloop`)

**Everything runtime lives outside the repo.** Never commit any of this.

```text
~/.gloop/
├── config.json              # Global config: host/port/budgets/limits/allowlists
├── token.json               # Local bind token (chmod 0600)
├── server.pid               # Written by the real server process for stop/status
├── server.log               # Detached-mode stdout + stderr
├── .gitignore               # Ignores workspace/ to avoid accidental commits
│
├── agents/               # One JSON per ACP/CLI agent
│   ├── claude.json
│   ├── relay.json
│   └── mock.json
├── adventurers/             # Agent persona: class, model, level, win-rate
│   └── adv_<id>.json
├── prompts/                 # Optional custom prompt overrides
├── automation/              # Trigger rules
│   └── auto_<id>.json
├── skills/                  # User-imported 3rd-party skills; built-ins are embedded
│   └── <name>/SKILL.md
├── context/                 # Global user context (cross-quest background knowledge)
│   ├── summary.md           # High-level summary
│   ├── meta.json            # Metadata
│   └── dim_<name>.md        # Per-dimension details (workspace / lark_im / lark_doc, etc.)
│
└── workspace/               # Runtime scratch, retention-policy aware
    ├── quests/
    │   └── qst_<shortid>/
    │       ├── meta.json              # Lifecycle + user-visible status
    │       ├── goals.json
    │       ├── plan.json
    │       ├── sessions/              # turn summary / assistant / tool call / tool result / error
    │       ├── events.jsonl           # Dashboard & audit timeline
    │       ├── reviews.jsonl          # Human adjudication records
    │       ├── backups/               # Source-file snapshots taken before `apply`
    │       └── work/                  # Actual agent working copy (git worktree or copy)
    └── stats/                 # Per-adventurer + global runtime stats
        └── adv_<id>.json
```

### Persistence contracts (treat as internal public API)
Schema changes should be **additive + defaulted in fsstore**, never breaking: existing local workspaces must continue to load.

The four files with the highest contract value are `meta.json`, `sessions/*.jsonl`, `events.jsonl`, and `reviews.jsonl`.

### ContextPack and trace rows
Before each agent phase, Gloop builds a structured ContextPack rather than concatenating opaque prompt strings. Each block carries `source`, `trust`, `phase`, `role`, staleness, priority, user-control metadata, and a truncation summary. The same trimmed pack that is sent to the agent is persisted as `context_pack` rows in `sessions/*.jsonl`.

The Dashboard's Execution Trace and `GET /api/quests/{id}/trace` expose those context rows alongside semantic events, assistant messages, user comments, and observed native tool calls. Use `scope=current|all`, `round=<n>`, and `limit=<n>` when debugging a specific rework round.

---

## Dashboard philosophy

**Mobile-first.** Most human decision time happens on a phone — watching progress, not reading megabytes of logs.

High-priority mobile flows:
- At-a-glance "do I need to act?" (blocked/reviewing first).
- Review via goal/check state + evidence summary; don't force logs.
- Pass/changes/reject/pause/resume/abort in one short path.
- Surface failures, budget overruns, and blocked states *above* generic event noise.
- Detailed logs are collapsible, collapsed by default.

Desktop remains important but is the deep-debug surface: diff review, mage allowlist editing, DAG planning, tool-ABI inspection.

### Current frontend shape

`web/` is a standalone **React + Vite + TypeScript** project:
- Source in `web/src` (routing in `App.tsx`, 8 pages under `pages/*`, slime mascot in `components/Logo.tsx`).
- Tokens live in `web/src/index.css`.
- Built assets under `web/dist` are `go:embed`ded via `web/embed.go` + `internal/server/dashboard.go` → **zero extra deployment, one single binary**.
- REST + SSE clients (`EventSource` for global `/api/stream` and per-quest `/api/quests/{id}/stream`) live in `web/src/api/*`.
- Local frontend iteration: `cd web && npm run dev` (Vite dev server auto-proxies `/api` and `/healthz` to `127.0.0.1:37317`), or set `GLOOP_WEB_DIST=./web/dist` for the Go binary to read assets from disk.

**Always rebuild and commit `web/dist` after changing `web/src`**, otherwise other users' binaries will ship the stale dashboard:
```bash
cd web && npm run build    # required after any frontend source change
go test ./...
```

---

## Honest status of what ships today

✅ Shipped & tested:
- Single-machine single-user.
- Execute quests: Warrior build → Mage review → Human verdict.
- Design quests + `spawn-execute` to branch a follow-up execute quest.
- Git worktree / copy / readonly workspace modes; worktree is the primary path.
- Agent adapters: generic ACP + TraeX (ACP & CLI) + Aiden (base / X Claude / X Codex) + Relay / Pi / Codex CLIs + Mock diagnostics.
- Agent protocol is skill + Bash + `gloop ...` CLI syscalls: `gloop phase`, `gloop review`, `gloop quest`, `gloop note`, and `gloop command`.
- Diff view / apply (with `git apply --check` conflict preflight + backup snapshots) / discard.
- Blocked resolution (continue → human-review → cancel), plus per-quest budget top-ups.
- Stopping criteria: turn cap, rework cap, phase/quest duration caps, consecutive-agent-error stall, no-progress stall.
- Mage auditable command allowlist executor with `side_effect_level` L0/L1/L2 gating (safe local-testing defaults; E2E tools like bytedcli are opt-in).
- Automation + Inbox backend/CLI/scheduler/quest-flagging + built-in pages.
- **User Context**: cross-quest persistent global background knowledge, dimension-based storage with progressive disclosure; `auto_context_refresh` automation for daily refresh; built-in `gloop-user-context` skill and CLI contract.
- **Knowledge export**: OKF-format portable knowledge bundles (Markdown + YAML frontmatter) with preview and diff.
- **Failure Attribution**: structured failure diagnosis with stage/category/recoverable marking and suggested recovery actions.
- **Agent Recovery**: automatic non-interactive recovery for auth-expired agents (Relay `auth login --sso`, TraeX `login status`).
- **Built-in Skills** (9): `gloop-quest-execution`, `gloop-quest-review`, `gloop-quest-fanout`, `gloop-self-awareness`, `gloop-note-keeping`, `gloop-user-context`, `gloop-user-notification`, `gloop-code-exploration`, `gloop-inbox-triage` — all official skills use the `gloop-` prefix, classified into `capability` and `orchestration` kinds, with unified CLI Contract format and structured `related_skills` metadata.
- Levels, class titles, auto-assignment ordered by win-rate → level → age.
- Mage sandbox copy for review phases, plus allowlisted validation commands.
- `.gloopignore`, workspace retention cleanup (CLI/API/scheduler).
- SSE event stream + embedded React Dashboard in the Go binary (9 pages: Quests / QuestDetail / Inbox / Automations / Knowledge / Skills / Stats / Adventurers / Settings).
- **Full ops command suite**: `start / stop / restart / status / logs -f / dashboard`, plus user-level cross-platform autostart.
- **Doctor diagnostics**: `doctor agents --smoke` for agent smoke tests; `doctor e2e --agent X` for real-agent E2E contract verification.
- **Personal stats**: per-adventurer and global runtime statistics with time-range filtering.
- **Config package import/export**: portable configuration migration between machines.
- **Version update check**: background npm version polling with Dashboard upgrade banner.

🛣️ Roadmap (see `PRDv2.md`):
- CheckGraph replacing LLM-only acceptance.
- Discovery & triage loop.
- Parallel best-of-N execution.
- Full pause/resume UX.
- Learning loop + per-project memory.
- Deep polish on mobile review UX.

---

## Agent view: out-of-the-box deployment runbook

When a user hands you this repo and says "run this locally", follow this runbook. The contract is: verify deps → test → init → start → probe → return a usable Dashboard URL.

### 1. Environment & repo sanity
```bash
pwd
git status --short --branch
go version
```
Do **not** revert uncommitted user changes; explain whether they affect deployment.

### 2. Verify tests
```bash
go test ./...
```
No Node build is required for backend deploys — `web/dist` is already checked in and `go:embed`ded. Rebuild it only if frontend source changed.

### 3. Isolated data dir (never pollute user's real `~/.gloop`)
Prefer a repo-local ignored dir. `start` auto-initializes missing files; explicit init is optional:
```bash
export GLOOP_DATA_DIR="$PWD/.gloop-dev"
go run ./cmd/gloop init --data-dir "$GLOOP_DATA_DIR"   # optional
```
Use default `~/.gloop` **only** if the user explicitly asks for the real environment.

### 4. Start the service
```bash
go run ./cmd/gloop start \
  --no-detach --no-autostart --no-open \
  --host 0.0.0.0 --port 37317 --data-dir "$GLOOP_DATA_DIR"
```
`--port 0` scans for free ports. `--external-host` pins the advertised address.
Real humans can just `go run ./cmd/gloop start` for the default background+autostart+browser experience.

Success output:
```text
🟢 Gloop started in background
   Listen:      http://0.0.0.0:<port>
   Data Dir:    ...
   Dashboard:   http://<dev-ip>:<port>/dashboard?t=<token>
```
**Always return the full Dashboard URL including `?t=` — never just a port.**

### 5. Health + API checks
```bash
curl -s http://<dev-ip>:37317/healthz      # → {"ok":true,"service":"Gloop"}
# /api/* requires the token from the URL
curl -s -H "Authorization: Bearer <token>" http://<dev-ip>:37317/api/quests
```

### 6. (Optional) Mock quest smoke test
Mock is a diagnostic executor, not a production substitute. It can still walk the full local lifecycle:
```bash
go run ./cmd/gloop quest run \
  --data-dir "$GLOOP_DATA_DIR" \
  --query "Local deploy smoke: service can create a quest and reach a reviewable state"
```
Verify the quest appears in the Dashboard.

### 7. Report format
Success: the Dashboard URL (with token), data dir, `go test` result, `/healthz` result, which executors are real vs Mock, and **how to stop** (`Ctrl+C` for foreground, `gloop stop` for background).

Failure: never just say "failed". Report the stage, the key error, the completed steps, and the recommended next action.

---

## First principles (read before changing any code)

- **The platform owns the loop.** Agents execute phases; the engine controls lifecycle, acceptance, rework, and final adjudication.
- **Generator and evaluator stay separate.** Never allow an execution phase to silently approve its own output.
- **State lives on disk.** Critical progress must be recoverable from JSON/JSONL — never from in-memory or model-context state alone.
- **Human review is a first-class state.** Do not bypass `reviewing` for non-trivial outputs.
- **Mobile decisions matter.** Never make common review actions desktop-only.

---

## Code map

```text
cmd/
  gloop/              Primary CLI entrypoint (start/stop/restart/status/logs included)
  gloopd/             Compat entrypoint: merged into gloop start, kept only so old
                      scripts don't break; prints a deprecation notice.

internal/cli/         Flag parsing, bootstrap, agent detection, all subcommand impls
  cmd_start.go        start/stop/restart/status/logs + cross-platform autostart + detach
  cmd_quest.go        quest run/info/list/show/diff/review/apply/discard/cancel/cleanup
                      + comment/answer/backup/spawn/spawn-execute
  cmd_doctor.go       doctor agents (--smoke) + notifications + e2e (--agent)
  cmd_context.go      context list/show/summary/refresh/export/write-dim/write-summary
  cmd_command.go      command list/run (mage allowlist management)
  cmd_skill.go        skill list/show/validate
  cmd_stats.go        stats (--range)
  cmd_idl.go          idl export

internal/orchestrator/   Engine & loop control — THIS IS THE HEART
  engine.go              CreateQuest / StartQuest / ResolveUserReview
  quest_lifecycle.go     State transitions
  macro_loop.go          Warrior → Mage → Acceptance → Human outer loop
  micro_loop.go          Per-turn Think/Act/Observe
  scheduler.go / cron.go Quest scheduling, cleanup, automation triggers
  automation.go          Automation rule evaluation
  failure_attribution.go Structured failure diagnosis + caching
  agent_recovery.go      Auto-recovery registry (auth refresh for Relay/TraeX)
  command_runner.go      Mage allowlist command executor (L0/L1/L2 gating)
  skills.go / context.go

internal/executor/    Executor interface (ISP: Meta/Session/Chat/ContextExporter)
                      + ACP / TraeX / Aiden (3 variants) / Relay / Pi / Codex / Mock
internal/fsstore/     JSON/JSONL persistence + git worktree/copy/diff/apply + safety
                      + command allowlist + personal stats + config package
internal/server/      HTTP API (api_*.go), bind-token auth, SSE stream, embedded Dashboard
internal/model/       Stable vocabulary & primitive values (0 imports enforced by arch test)
internal/events/      In-process event bus (engine ↔ dashboard stream)
internal/prompt/      Class & role prompt templates (skill manifest + context rendering)
                      + ContextPack IR, budgeting, file-mode/capability-first helpers
internal/platformtools/  Platform mechanism handlers reused by CLI/API/tests; not exposed as agent native tools
internal/skills/      Built-in skill parser + user-skill loader
internal/idl/         IDL export: tools / skills / agent schemas / ABI / classes
internal/auth/        Token generation + HTTP middleware
internal/acp/         ACP protocol client SDK
internal/arch/        Import boundary tests (one-way deps are enforced here)
internal/version/     Build-time version info
internal/updatecheck/ Background version update polling

skills/               Official built-in skills — markdown assets embedded in the binary
  gloop-quest-execution/  gloop-quest-review/  gloop-quest-fanout/
  gloop-self-awareness/   gloop-note-keeping/  gloop-user-context/
  gloop-user-notification/ gloop-code-exploration/ gloop-inbox-triage/

web/                  React + Vite Dashboard
  src/App.tsx         Routing
  src/pages/*         QuestsBoard / QuestDetail / Inbox / Adventurers / Automations /
                      Knowledge / Settings / Stats / Skills
  src/components/*    Shared components (incl. slime mascot: Logo.tsx)
  src/api/*           REST + SSE clients
  src/index.css       Style tokens
  dist/               go:embed build artifacts; MUST be committed with source changes
```

Dependency direction (enforced by `internal/arch` tests):
```text
cmd → cli/server → orchestrator → fsstore/executor/events/prompt → model
Low-level packages (version/auth/acp/idl) stay below adapters, never reverse-import.
```

### Loop implementation hot spots (update state-transition tests when you touch these)

- `CreateQuest` — persists quest metadata.
- `StartQuest` — drives the MacroLoop.
- `runAgentPhase` / `runPhase` — PhaseDef-driven phase execution; execute phases end via `gloop phase done/fail`, review phases end via `gloop review pass/request-changes/reject`.
- `runMicroLoop` — per-turn Think/Act/Observe, writes events, session rows, and ContextPack trace rows.
- Agent syscalls (`gloop phase/review/note/quest/*/command/*`) communicate with the orchestrator via `.gloop/context.json` and `.gloop/signals/<sid>.json`.
- `ResolveUserReview` — applies human pass/changes/reject decisions.

High-risk surface: rework counters, review state, session persistence, acceptance outcomes.

---

## Repository hygiene

- After changing `web/src`, **always** `cd web && npm run build` and commit `web/dist` alongside. Other users' embedded dashboards depend on it.
- Never commit `web/node_modules`.
- Never commit runtime data from `~/.gloop` (`server.log`, `server.pid`, `workspace/*`, `sessions/*.jsonl` contain local execution history).
- JSON schema changes are **additive + defaulted in fsstore**. Older local workspaces must keep loading.
- Roadmap-level changes go into `PRDv2.md`. Operational, onboarding, and contributor guidance goes in this README.
- Platform-specific code lives in `*_unix.go` / `*_windows.go` with explicit `//go:build` tags. Don't put giant `runtime.GOOS` switches in portable files.

---

## Roadmap anchors

See `PRDv2.md` for the full blueprint. Near-term focus:
- CheckGraph (deterministic goal-validation DAGs) replacing LLM-only acceptance.
- Hardening worktree isolation (automatic copy fallback if isolation can't be guaranteed; no silent fallback to the source tree).
- Discovery & triage loop (requirements go through classification/assignment before execution).
- Parallel best-of-N execution.
- Finer mage allowlists, granular failure recovery, complete pause/resume UX.
- Learning loop (cross-quest project memory + adaptive adventurer profiles).
- Deep polish on mobile review & decision UX.
