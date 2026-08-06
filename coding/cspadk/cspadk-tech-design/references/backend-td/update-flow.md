# Phase 6 — Update an Already-Published TD

Load this file when the user asks to edit a published TD (see entry signals below). TDs evolve through reviewer comments, scope changes, and architecture pivots. This phase supports editing a published doc without regenerating from scratch.

**Entry signals** — invoke this phase (not Phase 1–5) when the user says things like:
- "update the TD to …", "add a section about …", "revise the TD …", "the reviewer asked for …"
- pastes an existing Lark doc URL and asks for changes
- references a TD that was previously published in this repo / memory

---

## Step A — Locate the target document

1. If the user provided a doc URL, extract `doc_id` from the URL (last path segment).
2. Else, read the auto-memory for a `td_*` entry matching the feature; confirm with `AskUserQuestion` before editing.
3. Else, ask the user for the doc URL — do not guess.

## Step B — Re-fetch the current doc (always, before any edit)

```bash
lark-cli docs +fetch --doc "<doc_url>" > /tmp/backend-td-current.json
```

Parse `parsed.data.markdown` and `parsed.data.title`. This is the source of truth — the on-disk `/tmp/backend-td.md` from the original run is stale and must not be used.

## Step C — Decide edit mode

| Situation | Mode | Command |
|---|---|---|
| Change one section's content (e.g. rewrite "Observability" after adding metrics) | Targeted replace | `cd /tmp && lark-cli docs +update --doc <id> --mode replace_range --selection-by-title "<Section>" --markdown @./section.md` |
| Add a new section that didn't exist | Append or insert-after | `cd /tmp && lark-cli docs +update --doc <id> --mode insert_after --selection-by-title "<Anchor Section>" --markdown @./new-section.md` (new markdown may embed a paired `<whiteboard type="blank"></whiteboard>` — the response's `data.board_tokens` gives the new token to populate via `lark-cli whiteboard +update`) |
| Multiple section edits or structural changes (e.g. tier promoted M→L) | Full overwrite | `cd /tmp && lark-cli docs +update --doc <id> --mode overwrite --markdown @./backend-td.md` (rebuild the full doc locally first; any existing whiteboard tokens in the old doc are lost — re-populate new ones from the response's `board_tokens`) |
| Small in-place fixes (typo, link, one row of a table) | Targeted replace of the smallest enclosing section | as above |
| Update an existing diagram only (no doc content change) | `lark-whiteboard` only | `cd /tmp && lark-cli whiteboard +update --whiteboard-token <board_token> --input_format mermaid --source @./arch.mmd --overwrite --yes` — reuse the `board_tokens` saved in the `td_*` memory; don't touch `docs +update`. Prefer Mermaid; reserve PlantUML for sequence diagrams. |

## Step D — Before writing, verify freshness

- If the edit references a downstream PSM, IDL method, DDL, or metric name: re-check the repo / Overpass for drift since the original TD was written. Stale references in an update are worse than in a first draft — reviewers trust them.
- If the architectural direction changed (e.g. switched from polling to SSE), scan the *entire* existing doc for stale references (old PSM names, old IDL signatures, old decision rationale) and fix every one in the same update. A half-updated TD is a worse artifact than the old version.

## Step E — Confirm with `AskUserQuestion` before writing

- For targeted edits: "Apply change to section **{Section}** only?" → Apply / Show diff first / Cancel.
- For structural changes (tier change, major section removed): "This update rewrites the whole doc. Ready?" → Rewrite / Cancel / Keep as targeted edit instead.

## Step E.5 — Append a Version History row (mandatory for every edit)

Every published edit must add a new row to the Version History table at the top of the doc. This is what makes AI-agent edits visible to reviewers — without it, reviewers can't tell what changed between visits.

Row format:

| Field | Value |
|---|---|
| Version | Bump from the previous row. Patch for typo/link/wording; minor for new section or non-trivial rewrite; major for tier change, scope shift, or reversal of an earlier decision. |
| Date | Today, `YYYY-MM-DD`. |
| Changes | One line describing what changed. Reference section names: `"Reworked Observability section to add new alert thresholds"`. |
| Author | **`AI · csp-backend-td-generator`** for any change driven by this skill. Do NOT substitute a human name unless the user explicitly states they made the edit manually outside the agent. |

**How to apply per edit mode (mode chosen in Step C):**

- **Targeted replace / insert-after** that doesn't touch Version History → run a SECOND `lark-cli docs +update --doc <id> --mode replace_range --selection-by-title "Version History" --markdown @./version-history.md` after the main edit, with the full updated table (re-fetched + appended row) in `version-history.md`.
- **Full overwrite** → the new row is part of the rebuilt full markdown; nothing extra to do.
- **Whiteboard-only update** → still append a row (Changes column: `"Updated <diagram name> whiteboard"`) so the change isn't invisible.

If Version History doesn't exist in the doc yet (TD predates this discipline), create it as a new top section via `insert_after` against the first heading, populated with one row covering this edit.

## Step F — After update, refresh the auto-memory entry

Update the `td_{feature_slug}` memory with the new state (e.g., resolved TODOs, added sections, bumped tier). Keep the `doc_id` / `doc_url` stable; only the metadata changes.

---

## Common update scenarios — treat these as first-class entry points

- "reviewer X said Y" → add a row to Technical Review TODO; fix the flagged section; update memory with the resolution.
- "the feature scope grew" → re-run the sizing gate via `AskUserQuestion`; if tier bumps, add the new conditional sections and re-scan for consistency.
- "IDL got renamed" → search the doc markdown for every mention of the old name; replace all; re-emit the full sections that touched the rename.
- "link the TD to a Meego ticket / PRD that didn't exist yet" → targeted edit of the Background table only.

## Gotcha — `replace_range` on a section containing a whiteboard reference orphans the board

`lark-cli docs +fetch` returns existing whiteboards as read-only `<whiteboard token="..."/>` self-closing tags. If you copy that form into your replacement markdown and `docs +update --mode replace_range` it back, Lark strips the tag (`BOARD_TOKEN_NOT_SUPPORTED` warning) and the whiteboard vanishes from the doc — the underlying board still exists but is no longer embedded.

**Fix:** in the replacement markdown, use `<whiteboard type="blank"></whiteboard>` instead; capture the fresh `board_tokens[i]` from the response; repopulate via `lark-cli whiteboard +update`. Update the `td_*` memory with the NEW token. The old orphaned token becomes dead weight; don't reference it again.
