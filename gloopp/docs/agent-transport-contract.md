# Gloop Agent Transport Contract

> Status: draft
> Scope: official native agent execution, advanced protocol integrations, and
> the boundary between execution identity and optional adventurer persona.

## 1. Decision

Gloop is a Loop Engineering System. Its core objects are Quest, Run, Phase,
Workspace, Policy, Evidence, Report, Event, and Human Exception. Agent runtime
support must serve those objects; it must not turn Gloop into an IM-style bot
router.

Official native agents are CLI-first. A native agent is eligible for the
official path only when its local CLI can satisfy this transport contract and
pass readiness checks. ACP remains a first-class advanced/protocol integration
path, especially for external agents and for cases where ACP provides stronger
session or streaming semantics. ACP is not a deprecated hack.

Adventurers are not the execution primitive. The execution primitive is an
Agent Profile selected for a Phase Role. Adventurers remain as an optional
persona/progression overlay and as a backward-compatible explicit binding.

## 2. Terms

- Agent Profile: A runnable local agent identity, such as `traex_cli`,
  `codex_cli`, `claude_cli`, `relay_cli`, or an ACP profile. It owns command,
  transport, readiness, model, permission, and capability metadata.
- Transport: The machine interface used by Gloop to drive the agent. Supported
  families are CLI, ACP, and future protocol transports such as local HTTP/SSE.
- Phase Role: The phase-level responsibility, such as execute or review.
- Adventurer: An optional persona/progression overlay that can bind a phase to
  an agent profile and customize prompt/model/statistics.
- Readiness: The observed runtime state of an agent profile on this machine.
- Headless execution: Running under the Gloop daemon without an interactive TTY
  approval loop.

## 3. Transport Contract

An official transport MUST satisfy all of the following:

1. Non-interactive start: The agent can be launched by the daemon without TTY
   prompts.
2. Structured output: The transport emits JSONL, JSON-RPC, event stream, or an
   equivalent stable machine-readable protocol.
3. Workspace control: The transport can be pointed at a quest workspace and
   supports explicit policy for additional writable directories.
4. Error attribution: Version, auth, configuration, model, permission, process,
   and protocol failures can be classified by doctor/readiness checks.
5. Resource and lifecycle control: Gloop can interrupt, kill, wait for, and
   clean up the process without orphaned execution.

An agent capability contract is separate. It covers whether the agent can use
the Gloop ABI, such as `gloop phase done`, `gloop review`, and allowed command
execution. A transport can be valid while a specific agent still lacks a
required capability.

## 4. Official CLI Policy

Official native agents SHOULD use CLI transport by default once the CLI passes
readiness checks.

CLI profile seed files MUST be created disabled by default. Startup discovery
MAY detect installed official CLIs and MAY populate readiness metadata, but it
MUST NOT silently enable execution or elevate permissions.

Readiness states:

- `ready`: Connectivity smoke and execution smoke both pass under the selected
  policy.
- `reachable_unverified`: The binary starts and structured output is parseable,
  but execution smoke has not passed.
- `needs_auth`: Login or entitlement is missing.
- `needs_config`: Permission, model, sandbox, or profile configuration is not
  headless-compatible.
- `unsupported`: Required CLI features are unavailable.
- `installed_but_unverified`: The binary exists, but no smoke has completed.

`installed_but_unverified` means no smoke has run yet. `reachable_unverified`
means connectivity smoke ran successfully, but execution smoke has not passed.

Doctor MUST distinguish two smoke levels:

- Connectivity smoke: Low-risk, normally read-only. It verifies binary startup,
  structured output parsing, and session/thread id extraction. Passing this
  only proves the agent is reachable.
- Execution smoke: Runs a minimal real tool/action under the intended execution
  sandbox and permission policy. Only this can mark the profile `ready`.

Connectivity success MUST NOT be reported as ready-for-use.

## 5. ACP Policy

ACP remains supported as an advanced/protocol integration path. Gloop owns the
generic ACP client and should continue to harden it with stderr tail capture,
control-plane timeouts, process cleanup, and readiness diagnostics.

ACP profiles MAY be used as explicit phase bindings and MAY be recommended for
workloads where ACP has verified advantages, such as stateful incremental
prompts or richer streaming updates.

Official CLI-first does not authorize removing or silently migrating existing
ACP agent configs.

## 6. TraeX CLI Contract

TraeX CLI has a stateful execution path and MUST use it for official long-running
work.

Observed contract:

- `traex exec` persists a thread by default.
- JSONL contains `thread.started.thread_id`.
- `traex exec resume <thread_id>` resumes that exact thread.
- `traex exec resume --last` is global and MUST NOT be used by Gloop because
  concurrent quests can cross wires.

Implementation requirements:

1. On first turn, parse `thread.started.thread_id`.
2. Persist the TraeX thread id in Gloop quest/session state at the same level of
   durability as other transport session ids.
3. On subsequent turns, invoke `traex exec resume <thread_id>`.
4. Do not use `--ephemeral` for official long-running execution.
5. Use `--ephemeral` only for explicit short-task, diagnostic, or connectivity
   smoke flows.

## 7. Model Contract

Model ids are transport-specific. ACP model ids MUST NOT be blindly reused as
CLI `--model` values.

Observed TraeX behavior:

- `traex exec --model auto` fails with `code=4001`.
- `traex exec --model gpt-5.5/xhigh` fails with `code=4001`.
- `traex exec --model GPT-5.5` succeeds.
- Omitting `--model` succeeds and uses local TraeX configuration.
- ACP `session/new` can expose ids such as `gpt-5.5/xhigh`, which are not CLI
  argument contracts.

TraeX CLI rules:

1. If model is empty, `auto`, or `default`, Gloop MUST omit `--model`.
2. If the user explicitly configures a TraeX CLI model, Gloop MAY pass it only
   after doctor validates it with CLI smoke.
3. ACP-returned model ids are display/ACP metadata unless separately validated
   for CLI.

## 8. Permission And Sandbox Contract

Headless CLI execution cannot rely on interactive approval. Permission handling
MUST be explicit and conservative.

TraeX CLI rules:

1. If permission mode is unset, Gloop MUST omit `--permission-mode` and let
   TraeX use its headless default.
2. Explicit `default` or `plan` modes MUST be rejected or marked `needs_config`
   for Gloop headless execution, because TraeX exec cannot ask approvals.
3. `bypass_permissions` MUST require explicit policy authorization. It MUST NOT
   be selected by readiness probing, fallback, or migration.
4. `custom` MAY be used only after doctor verifies it is headless-compatible.
5. A transport fallback MUST NOT upgrade permissions.

Sandbox/workspace rules:

1. Official CLI execution MUST set the quest workspace as cwd.
2. Additional writable directories MUST be empty by default and granted only by
   policy.
3. The intended sandbox mode MUST be part of the agent profile readiness result.
4. Doctor MUST report which permission and sandbox were used for connectivity
   and execution smoke.

The default sandbox is agent-profile configuration, not hard-coded transport
behavior. Milestone 1 TraeX CLI should use `workspace-write` for execution
smoke and real execution unless policy overrides it; connectivity smoke MAY use
`read-only`.

## 9. Workspace And Internal State Boundaries

There are two different boundaries.

User workspace boundary is P0:

- Gloop MUST run official CLI work in the quest workspace.
- Gloop MUST NOT grant additional project directories unless policy explicitly
  allows them.
- Workspace-external user project writes are not allowed by default.

Agent internal state is separately auditable:

- Some CLIs may write internal state under their home/config directory, such as
  `$TRAE_HOME`.
- TraeX CLI has been observed writing `~/.trae/cli/memories/...` under
  `workspace-write`.
- Gloop cannot claim that `--cd` alone prevents internal agent state writes.

Gloop SHOULD:

1. Disclose this behavior in doctor for affected agents.
2. Parse structured output for file changes outside the quest workspace.
3. Persist an event such as `agent.external_state_write_observed` with agent,
   transport, quest id, session id, path, kind, and sandbox mode.

## 10. History And Context Cost Contract

Stateful CLI resume reduces the need to resend full conversation history, but it
does not automatically eliminate Gloop-side context injection cost.

Gloop MUST distinguish:

- delta mode: only the new user/phase delta is sent to an existing transport
  session;
- full-history mode: complete conversation or complete quest context is resent;
- context-pack refresh: Gloop sends a fresh context pack because platform state
  changed.

If a CLI executor resends full history or a complete context pack after the
configured threshold, it MUST record `agent.cli_full_history_warning` in the
event store. Documentation alone is not enough.

The warning payload SHOULD include agent, transport, quest id, session id,
turn count, estimated prompt bytes/tokens when available, and reason.

## 11. Event Contract

Execution-relevant transport facts MUST be persisted to the Gloop event store,
not only printed to stderr.

Required events:

- `agent.transport_selected`
- `agent.readiness_changed`
- `agent.cli_full_history_warning`
- `agent.external_state_write_observed`

`agent.transport_selected` is emitted once per session creation, when Gloop
binds a phase/session to a concrete transport. It is not emitted per turn.

`agent.readiness_changed` is emitted when readiness changes during startup
probe, explicit doctor re-run, or transport failure recovery probe. It is not a
per-turn heartbeat.

Future fallback event:

- `agent.transport_fallback`

Fallback payload MUST include agent, from transport, to transport, attempts,
reason code, stderr tail, quest id, session id, and timestamp.

## 12. Identity And Adventurer Contract

Adventurers are optional overlays. Agent Profile plus Phase Role Binding is the
execution foundation.

Resolution order for a phase:

1. Explicit adventurer binding, for backward compatibility and persona flows.
2. Explicit phase role binding.
3. Policy-selected ready official agent profile.

Milestone 1 does not change the existing adventurer-required quest path. Removing
that requirement is a later engine milestone because it affects executor lookup,
prompt building, phase configuration, recovery sanity checks, reports, stats,
and UI.

Future milestones should make `source_adventurer_id` optional and record
`agent_id`, `executor_id`, and `phase_role` as first-class report/event fields.

## 13. Milestones

### Milestone 1: Contract and TraeX CLI Experimental Profile

Scope:

- Add this contract.
- Add disabled `traex_cli` seed/profile metadata.
- Add doctor readiness for TraeX CLI as experimental.
- Verify and persist TraeX CLI thread id when executor support is enabled.
- Remove hard-coded `--ephemeral` and `--permission-mode bypass_permissions`
  from official TraeX CLI execution.
- Add connectivity/execution smoke distinction.
- Add event parsing for workspace-external file changes if TraeX JSONL exposes
  them.

Non-goals:

- Do not make TraeX CLI recommended until execution smoke passes.
- Do not remove existing `traex` ACP configs.
- Do not make adventurers optional in this milestone.
- Do not implement automatic transport fallback.

### Milestone 2: Agent-Direct Phase Binding

Scope:

- Add direct agent profile selection for execute/review phases.
- Add role-based prompt building independent of adventurer class.
- Add default phase role binding config.
- Keep adventurers as explicit overrides.

### Milestone 3: Adventurer Overlay

Scope:

- Make adventurers optional persona/progression overlays.
- Update UI and CLI onboarding so users can run quests without recruiting
  adventurers first.
- Keep old adventurer configs importable and runnable.

### Milestone 4: Explicit Transport Fallback

Scope:

- Implement same-agent transport fallback only.
- Default off.
- Persist `agent.transport_fallback`.
- Never perform cross-agent fallback silently.
