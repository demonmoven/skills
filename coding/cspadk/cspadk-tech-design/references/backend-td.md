# Backend Technical Design Guide

This document provides the workflow and guidelines for generating Backend Technical Design documents. It is loaded by `cspadk-tech-design` when Backend TD type is detected.

## Overview

Generate a backend TD grounded in real codebase + internal-platform data (IDL via Overpass, schemas via RDS, baselines via APM), published to a Feishu/Lark wiki.

The TD structure is layered (IDL → Data Model → Feature Design → Decisions) because that matches how backend reviewers read. Distributed-systems sections (failure matrix, idempotency, consistency, capacity, observability, rollout, security, cross-team) are **conditional** — include them only when the feature's blast-radius warrants it.

## References — Load on Demand

Load these reference files from `references/backend-td/` when the corresponding phase fires:

| Reference | Load When |
|-----------|-----------|
| `checkpoints.md` | Any decision gate — full `AskUserQuestion` catalog |
| `context-gathering.md` | Phase 1 — sub-phase bash commands |
| `bytedance-best-practices.md` | Phase 3 — writing Data Model / IDL / Cache / MQ sections |
| `decc.md` | Phase 3 — cross-region data transfer |
| `template.md` | Phase 4 — the TD document template (English) |
| `template-zh.md` | Phase 4 — the TD document template (Chinese) |
| `style-guide.md` | Phase 4 — presentation priority, diagrams, tables |
| `update-flow.md` | Phase 6 — editing a published TD |
| `pitfalls.md` | Pre-publish + review — Quality Checklist |

## Input Requirements

| Input | Required | Example |
|-------|----------|---------|
| PRD / Meego link | Yes | `https://bytedance.larkoffice.com/wiki/xxx` |
| Service repo path | Yes | `/path/to/service-repo` |
| Own PSM | Yes | `data.ies.csp_quality_consumer` |
| Output wiki URL | Auto (from config) | Parent wiki node for publishing. Read from `.ttadk/csp-config.json` → `techDesign.wikiParentUrl`. Ask user only if not configured, then save to config. |
| Downstream PSMs | No (auto-discover) | List of PSMs this service calls |
| Upstream PSMs | No (user-supplied) | List of PSMs that call this service |
| Data stores touched | No (auto-discover) | RDS, TOS, MQ, Abase/Redis, ES |

## TD Size Tier (First Gate)

Before writing, pick a size tier. This prevents 15-section bloat on a one-flow feature.

| Tier | When | Sections Included |
|------|------|-------------------|
| **XS** | No IDL change, no DDL, no new downstream | Background, Swimlane, Feature Design, Benefits, Schedule |
| **S** | ≤1 new flow, ≤1 downstream PSM, no DDL | XS + Effort Estimation, Test Cases, Open Questions |
| **M** | Multiple modules, IDL changes, DDL | S + IDL/API Changes, Data Model, Dependencies, Decisions |
| **L** | Risky/distributed features | M + distributed-concerns sections |

Decide the tier from the PRD + repo exploration. Use `AskUserQuestion` with the 4 tiers. Include reasoning. Do not proceed until the user picks.

## Feature Classification Gate (Second Gate — L tier only)

Skip this entirely for XS/S/M-only features.

| Axis | Question | If YES, Include |
|------|----------|-----------------|
| Write path | Writes to any persistent store? | Idempotency subsection |
| Money/permissions/compliance | Touches billing, auth, PII? | Security & Compliance section |
| Cross-region data transfer | Data moves across RoW/EU TTP/US TTP? | DECC section (load `decc.md`) |
| External calls | Calls ≥1 downstream PSM? | Timeout/Retry Budget + Failure Matrix |
| Traffic delta | Adds ≥10% QPS? | Capacity section with APM baseline |
| New observability | New metric/alert/dashboard? | Observability section |
| Risky rollout | Irreversible migration? | Rollout Plan + Kill Switch |
| Cross-team dependency | Requires other PSM owner changes? | Cross-Team Dependencies table |

## Workflow

### Phase 1: Gather Context

**Load `references/backend-td/context-gathering.md`** — sub-phases 1a–1g:
- 1a. Fetch PRD / Meego
- 1b. Explore the service repo (Agent subagent)
- 1c. Resolve IDL (local-first, Overpass fallback)
- 1d. Inventory middleware dependencies (RDS / TOS / MQ / Abase / ES)
- 1e. Observability baseline (only if traffic delta)
- 1f. Upstream caller discovery (user-supplied)
- 1g. Prior art & internal tooling research

### Phase 2: Build & Confirm the Dependency Manifest

Render the manifest and pause for user confirmation:

```
Own PSM: <psm>
Repo: <path>
Owner team: <team>

Downstream PSMs (we call them):
  <psm> | methods=[...] | IDL=<version> | owner=<team> | changes=<none|modify>
Upstream PSMs (they call us):
  <psm> | calls=[our_methods] | owner=<team> | impact=<bc|breaking>
Data Stores:
  <type> | <id> | usage=<...> | change=<NEW|MODIFY|NONE>
```

Use `AskUserQuestion` to confirm — do not skip this step.

### Phase 3: Analyze and Design

**Before designing, load `references/backend-td/bytedance-best-practices.md`.**

Think through:
1. **IDL changes** — Backward-compatible? Are full IDL definitions (all request/response fields) available for every interface involved, so the frontend team can formulate their technical plan?
2. **Data model** — DDL statements? Migration plan?
3. **For each Classification Gate axis**, work out section content
4. **Alternatives** for every significant decision
5. **Open questions & risks**

### Phase 4: Write the TD Document

Write markdown to `/tmp/backend-td.md`.

**Load both:**
- `references/backend-td/template.md` (English) or `references/backend-td/template-zh.md` (Chinese) — based on the detected `output_lang`
- `references/backend-td/style-guide.md` — presentation priority

### Phase 5: Publish to Lark Wiki

Use `lark-cli` to create the document:

1. Create the document:
   ```bash
   lark-cli docs +create --title "<Feature Name> Backend Technical Design" --markdown "$(cat /tmp/backend-td.md)"
   ```

2. If wiki parent URL is configured (from `.ttadk/csp-config.json`), create a wiki node:
   ```bash
   lark-cli wiki +node-create --parent-node-token <wiki-parent-token> --title "<Feature Name> Backend Technical Design"
   ```

3. Save returned `doc_id` and `doc_url`

4. **For diagrams**: Create whiteboards:
   ```bash
   lark-cli whiteboard +update --whiteboard-token <whiteboard_token> --input_format mermaid --source "$(cat diagram.mmd)" --overwrite --yes
   ```

5. Return the document URL to the user

### Phase 6: Update Existing TD

**Load `references/backend-td/update-flow.md`** when the user asks to edit a published TD.

Entry signals:
- "update the TD to …", "add a section about …"
- Existing Lark doc URL with change requests

## Quality Checklist

**Before publishing, load `references/backend-td/pitfalls.md`.**

Verify:
- [ ] TD tier correctly sized
- [ ] Dependency manifest confirmed
- [ ] All conditional sections included per Classification Gate
- [ ] Backward-compatible IDL changes
- [ ] Complete IDL definitions for ALL interfaces (new, modified, and existing-but-called) with full request/response parameters — frontend team depends on this
- [ ] Migration + rollback plan for DDL
- [ ] No empty conditional sections
