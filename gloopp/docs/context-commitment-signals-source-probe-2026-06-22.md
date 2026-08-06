# Commitment Signals Source Probe

> Date: 2026-06-22
> Scope: Phase -1 source validation for `context-commitment-signals-spec-v0.3.md`.
> Result: Phase 0 can proceed with Lark Task as the only main source.

## Summary

The Lark user identity is available and has the scopes needed for task, IM, and calendar read probes. Lark Task returned a small non-empty set of incomplete tasks, so it is viable as the Phase 0 source for candidate commitment signals. IM search and calendar are technically callable, but they should not be used as Phase 0 sources: IM returns sensitive raw message fields, and calendar events are only weak time-pressure hints.

Phase 0 should therefore update `auto_context_refresh` to place at most three task-derived candidate signals under `dim_lark_im` -> `Current Focus` -> `### Candidate Commitment Signals (Experimental)`. It must not create an independent `dim_commitment_signals.md` dimension yet.

## Probes

### Auth

Command:

```bash
lark-cli auth status --json
```

Observed:

- `identities.user.status`: `ready`
- `identities.user.tokenStatus`: `valid`
- User scopes include task read, IM message search/read, and calendar event read capabilities.
- Token expiry is time-bound; future refresh failures should be treated as source unavailability, not as empty commitments.

### Lark Task

Command:

```bash
lark-cli task +get-my-tasks --as user --complete=false --page-all --json
```

Observed shape:

- Envelope keys: `_notice`, `data`, `identity`, `ok`
- `data` keys: `has_more`, `items`, `page_token`
- Incomplete task count in this probe: `4`
- Sample item keys: `created_at`, `due_at`, `guid`, `summary`, `url`

Assessment:

- Viable Phase 0 source.
- `summary` can be generalized into an action summary.
- `due_at` can be used as a time hint.
- `guid` and `url` must never be written into context markdown.
- If the task list is empty, write a `Gaps` note instead of fabricating signals.

### Lark IM

Command:

```bash
lark-cli im +messages-search --as user --query Gloop --page-size 5 --json
```

Observed shape:

- Envelope keys: `_notice`, `data`, `identity`, `ok`
- `data` keys: `has_more`, `messages`, `page_token`, `total`
- Message count in this probe: `5`
- Sample message keys include `chat_id`, `chat_name`, `content`, `message_app_link`, `message_id`, `sender`, `thread_id`

Assessment:

- Technically callable, but not a Phase 0 source.
- Raw fields are sensitive and must not be copied into context.
- Future Phase 1 can use IM only as a supplementary source after precision review.
- Context retrieval entries should use keywords and time windows only, not `chat_id`, `message_id`, or message links.

### Lark Calendar

Command:

```bash
lark-cli calendar +agenda --as user --start 2026-06-22T00:00:00+08:00 --end 2026-06-23T00:00:00+08:00 --json
```

Observed shape:

- Envelope keys: `_notice`, `data`, `identity`, `meta`, `ok`
- `data` type: array
- Agenda item count in this probe: `3`
- Sample item keys include `app_link`, `event_id`, `summary`, `start_time`, `end_time`, `description`, `event_organizer`, `self_rsvp_status`

Assessment:

- Technically callable, but only a low-confidence time-pressure hint.
- Event IDs, links, attendees, exact event titles, and descriptions must not be written into context.
- Do not use calendar data to infer commitments unless another source provides an explicit deliverable.

## Phase 0 Rules

- Source: Lark Task only.
- Placement: `dim_lark_im` -> `Current Focus` -> `### Candidate Commitment Signals (Experimental)`.
- Maximum items: `3`.
- Required fields: action summary, time hint, source type, confidence, uncertainty reason, verification keywords.
- Source type: `lark_task`.
- Confidence: `高` for incomplete tasks with deadlines, `中高` for incomplete tasks without deadlines.
- Forbidden context fields: `guid`, `url`, `open_id`, raw task title, raw chat content, `chat_id`, `message_id`, message links, exact calendar event IDs, exact attendees.
- No independent `dim_commitment_signals.md` until Phase 2.
- No status, priority, completion state, or hidden task-store semantics in context.

## Follow-Up

After 3-5 successful refresh runs, review precision and usefulness. If the task-derived signals are empty or low value, remove the experiment. Only if Phase 0 has clear value should Phase 1 add chat signals as a supplementary source.
