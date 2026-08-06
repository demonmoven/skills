# Gloop Architecture

Gloop should use DDD as a boundary discipline, not as a directory template.
The codebase keeps Go packages small and direct, while preserving one-way
dependencies and stable domain language.

## Domain Language

Use these terms consistently in code, API payloads, events, logs, and tests:

- `Quest`: the work request and its lifecycle.
- `Phase`: one execution/review step inside a quest.
- `Adventurer`: an agent persona assigned to a quest, with level, exp, and win/lose stats.
- `Review`: a verdict on a quest or phase output, with a 1–10 quality score.
- `Workspace`: the isolated filesystem where work happens.
- `Executor`: the southbound adapter that talks to an agent runtime.
- `Automation`: a trigger that creates and optionally starts/applies a quest.
- `Skill`: an instruction package injected into agent system prompts.
  Classified by `kind` into `capability` (能力型, tool/method-focused) and
  `orchestration` (编排型, lifecycle-driving). Each skill carries structured
  `related_skills` metadata for discovery (`depends_on` / `related` / `see_also`),
  and follows a unified CLI Contract format.
- `FailureAttribution`: structured diagnosis attached to a failed/blocked quest.
- `Policy`: a decision rule with auditable input hashes and decision IDs
  (review policy, recovery policy, hard guardrails).
- `AutoEvidence`: L0 commands auto-run by the platform before mage review to
  inject baseline evidence without agent effort.
- `WaitingInput`: a quest sub-state where the warrior has asked the user a
  question and is paused for an answer.

Do not introduce synonyms such as task/job/run for quest lifecycle concepts
unless they represent a distinct domain object.

## Package Roles

- `internal/model`: stable vocabulary and primitive domain values. It must not
  import other Gloop packages.
- `internal/domain/quest`: Quest domain aggregate. Pure in-memory state
  machine, budget validation, failure attribution, phase definitions, and
  pipeline logic. Owns the business rules; fsstore handles persistence.
- `internal/fsstore`: filesystem persistence and workspace mechanics:
  config, quest metadata, JSONL logs, git worktree/copy/diff/apply, safety
  checks, mage command allowlist, personal stats, and config package
  import/export. It stores state but should not decide lifecycle policy.
- `internal/orchestrator`: application/domain workflow. Quest state transitions,
  review handling, rework, scheduler behavior, automation, experience awards,
  failure attribution, agent recovery, and auto-evidence injection.
- `internal/policy`: policy engine. ReviewPolicy, RecoveryPolicy, and
  HardGuardrail decisions. Every decision carries a decision_id, input hash,
  and audit trail — all auto-decisions flow through here.
- `internal/executor`: southbound executor port and implementations. The
  interface is ISP-split into `ExecutorMeta`, `SessionLifecycle`,
  `ChatExecutor`, and `ContextExporter`; the combined `Executor` embeds all
  four. Implementations: ACP, TraeX (CLI), Aiden (3 variants), Relay, Pi,
  Codex, Mock.
- `internal/platformtools`: platform tool definitions and handlers. Agents invoke
  these via CLI syscalls (`gloop` CLI commands), not through a native tool ABI.
- `internal/events`: in-process event bus. Engine publishes, server/SSE consumes.
- `internal/prompt`: prompt templates that render skills and quest context
  into system / user prompts.
- `internal/skills`: skill registry, markdown asset loading, and frontmatter
  metadata parsing (kind, category, related_skills, requirements). Built-in
  skills are `go:embed`-ed from the top-level `skills/` directory.
- `internal/server`: HTTP adapter. REST endpoints, SSE streams, local token
  auth, and embedded dashboard static assets.
- `internal/cli`: CLI adapter. Command parsing, bootstrap, and all subcommands.
- `internal/auth`: local token generation and HTTP middleware.
- `internal/acp`: ACP (Agent Client Protocol) client SDK.
- `internal/notifications`: notification delivery (Lark, etc.) and event
  subscriber patterns.
- `internal/idl`: interface definition exports (tool / skill / agent schemas).
- `internal/arch`: import boundary tests that enforce dependency direction.
- `internal/version`: build-time version info.
- `internal/updatecheck`: version update check.
- `skills/`: official built-in agent skill assets embedded into the binary.
  These are product-facing instruction packages, not Go implementation internals.
- `cmd/*`: process entry points only.
- `web/`: React + Vite dashboard. `web/dist` is embedded into the Go binary.

## Dependency Direction

The intended direction is:

```text
cmd -> cli/server -> orchestrator -> domain/fsstore/executor/events/policy/prompt -> model
```

Additional low-level packages such as `version`, `acp`, `auth`, `notifications`,
`platformtools`, `skills`, and `idl` stay below adapters and must not import
workflow packages.

Allowed practical exceptions:

- `platformtools` can expose a small callback interface implemented by
  orchestrator. This keeps tool handlers independent from the concrete engine.
- `prompt` may depend on `platformtools` and `skills` to render ABI/skill
  manifests into system prompts.
- `fsstore` may import `executor` only for persisted session/message data
  shapes. It must not call executor implementations.
- `orchestrator` may use `domain/quest` service types and call into them as
  the domain layer; domain must not import orchestrator.

## Hard Rules

These are enforced by `internal/arch` tests:

- `model` and `version` do not import other Gloop internal packages.
- `domain/quest` does not import `orchestrator`, `server`, `cli`, `fsstore`,
  `executor`, `prompt`, `platformtools`, or `skills`. It may import `model`.
- `fsstore` does not import `orchestrator`, `server`, `cli`, `prompt`,
  `platformtools`, `skills`, `policy`, or `domain/quest`.
- `executor` does not import `orchestrator`, `server`, `cli`, `fsstore`,
  `prompt`, `platformtools`, `skills`, `policy`, or `domain/quest`.
- `policy` does not import `orchestrator`, `server`, `cli`, `fsstore`,
  `executor`, or `domain/quest`. It operates on `InputFacts` only.
- `orchestrator` does not import `server` or `cli`.
- `server` does not import `cli`.
- `prompt`, `platformtools`, and `skills` do not import `orchestrator`,
  `server`, `cli`, `policy`, or `domain/quest`.

When a rule feels inconvenient, prefer adding a small interface at the lower
level or moving the policy into orchestrator. Do not make a lower package import
a higher one to save a few lines.

## Design Guidance

- Keep domain rules close to the aggregate they protect. For now, Quest
  lifecycle rules live in orchestrator, with `fsstore.QuestMeta.TransitionTo`
  guarding valid state transitions.
- Do not add a repository/usecase/domain directory split just to look like DDD.
  Add packages only when they reduce real coupling or clarify ownership.
- Prefer explicit dependencies over globals. If a package only needs one method,
  define the narrow interface it needs.
- Keep adapters thin. HTTP handlers and CLI commands should validate inputs,
  call orchestrator, and render output.
- Put risky filesystem/git behavior behind focused helpers and tests. Workspace
  operations have higher blast radius than most formatting refactors.
