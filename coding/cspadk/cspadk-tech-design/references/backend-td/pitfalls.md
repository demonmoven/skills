# Quality Checklist + Common Pitfalls

Load this file before publishing (or when doing a Phase 6 update). Combines the pre-publish Quality Checklist with the anti-patterns that recur in reviews. Pitfalls that have a dedicated reference (DECC, best-practices) point there — don't inline those rules.

---

## Pre-publish Quality Checklist

- [ ] Dependency manifest was explicitly confirmed by the user (Phase 2)
- [ ] Every downstream PSM has its current IDL version pinned
- [ ] Every IDL change has a backward-compatibility statement
- [ ] Complete IDL definitions for ALL interfaces (new, modified, and existing-but-called) are listed with full request/response parameter details (field number, type, required/optional, description, nested structs expanded) — the frontend team depends on this to formulate their technical plan
- [ ] Every DDL has a migration + rollback plan
- [ ] Data Model / IDL / Cache / MQ / Config sections were sanity-checked against `bytedance-best-practices.md` (sibling reference; no `UNIQUE KEY` on MySQL, no `FOREIGN KEY`, no `AUTO_INCREMENT` PK, cache keys have TTL, MQ consumers have DLQ, Thrift field numbers not reused, etc.)
- [ ] If the feature crosses regions: `decc.md` (sibling reference) was loaded and the DECC section completed against that checklist
- [ ] Feature-classification Gate was applied — conditional sections match the flagged axes
- [ ] Every significant decision has an alternatives comparison
- [ ] Code snippets use real type names from the repo (no placeholder `Foo`/`Bar`)
- [ ] Presentation priority respected: architecture/data-flow shown as diagram first; every new/changed flow has pseudo-code in the repo language with real method signatures; constraints/risks as bullets; comparisons as tables; prose only where nothing lighter carries the "why"
- [ ] Diagram and pseudo-code pair cleanly where both appear — no prose re-description of either
- [ ] If TD is marked spec-codable: full signatures, named downstreams/tables/topics, error branches as code, and an "Acceptance" bullet per flow
- [ ] Writing is bullet-driven, not paragraph-heavy (prose only where it genuinely carries the "why")
- [ ] Tables use GitHub-flavored markdown by default; `<lark-table>` only where column widths / merged cells / multiline markdown cells are needed
- [ ] A top-level architecture / data-flow diagram is present as a native Lark whiteboard (populated via `lark-cli whiteboard +update`), not an ASCII block
- [ ] Every `<whiteboard type="blank"></whiteboard>` placeholder in the markdown has a matching populated whiteboard (no empty boards in the published doc); self-closing `<whiteboard/>` was NOT used for creation
- [ ] No raw mermaid/plantuml fenced blocks leaked into the doc markdown — source belongs inside the whiteboard
- [ ] Inline ASCII sketches are reserved for trivial request-path illustrations (≤5 boxes); `<sheet>` placeholders used only when the user maintains a real sheet
- [ ] No filler content in conditional sections — if flagged, it's substantive; if not flagged, it's omitted entirely
- [ ] Benefits section is grounded in actual feature gains, not generic phrasing
- [ ] Publish artifacts (doc_id, doc_url, parent node, PRD) saved to auto-memory so future "update the TD" requests work without re-gathering inputs
- [ ] Swimlane section: exactly one boe and one ppe line; slug matches `^(boe|ppe)_[a-z][a-z0-9_]{1,29}$`; the suffix after the env prefix is BYTE-IDENTICAL between boe and ppe; suffix is a feature slug NOT a PSM name; no `prod_*` line
- [ ] Section headings: no leading numbers on top-level `#` headings; no `7.1` / `7.2` numeric prefixes on `##` subsections; cross-refs point to titles not numbers; numbering (if any) is consistent doc-wide, not a mix
- [ ] Every `<whiteboard>` tag is followed by either nothing or a single italic caption line — never a prose paragraph re-describing the diagram's nodes/edges
- [ ] Version History section exists at the top of the doc; on first publish, it contains a `0.1.0 — Initial draft — AI · csp-backend-td-generator` row; on every Phase 6 edit, a new row was appended with today's date, a one-line Changes summary, and `AI · csp-backend-td-generator` as Author (see `update-flow.md` Step E.5)

---

## Common Pitfalls

0. **Dispatching subagents before Phase 0 auth.** bytedcli/lark-cli require valid SSO sessions. Launching 4 parallel context-gathering subagents against dead sessions wastes tokens and returns nothing. Always run Phase 0 (`bytedcli auth status`, `lark-cli auth status`, per-site login if needed) in the main agent first; device-code login cannot be done inside a subagent. Every subagent prompt must include a fail-fast auth-check line.

1. **Skipping the manifest confirmation step.** Auto-discovered PSMs and stores are often incomplete or stale. The user knows the real dependency list — ask before writing.

2. **Mandating distributed-system sections when the feature doesn't warrant them.** Empty "Consistency Guarantees" / "Failure Matrix" sections add noise and teach reviewers to skim. Apply the Gate rigorously — omit rather than fill with filler.

3. **Assuming backward compatibility without checking.** Thrift changes that look safe can break older clients with pinned codegen. Check caller repos or get owner sign-off.

4. **Treating middleware as infinite.** RDS connection limits, MQ partition caps, TOS quotas — check current utilization before projecting new load.

5. **Hand-waving failures.** "We retry on failure" is not a design. Specify timeout, retry count, backoff, circuit breaker, and what the caller sees when retries exhaust.

6. **Forgetting the rollback story.** A feature gate flips code paths; it doesn't reverse a backfill. State explicitly what's reversible.

7. **Assuming the vendored SPI IDL is current.** Local-first saves time for known downstreams but the SPI module can lag master by weeks. When the TD proposes using a recently-added field or method, drift-check that specific method against Overpass rather than trusting the vendored copy.

8. **Reaching for `<lark-table>` by default.** Markdown tables render fine in Lark and are far easier to edit in Phase 6. Use the XML form only when you need explicit column widths, merged cells, or multiline-markdown cells.

8a. **Wall-of-text Background / Feature Design.** Reviewers skim. Default to bullets; use prose only when a bullet loses the meaning.

8b. **Skipping the architecture diagram.** Every TD needs a populated whiteboard call graph — emit a **paired** `<whiteboard type="blank"></whiteboard>` placeholder in markdown (NOT self-closing `<whiteboard/>` — that form is read-only and is silently stripped with a `BOARD_TOKEN_NOT_SUPPORTED` warning, leaving `board_tokens` empty) and run `lark-cli whiteboard +update --whiteboard-token <token> --input_format mermaid --source @./arch.mmd --overwrite --yes` against the returned `board_token`. Default to Mermaid; reserve PlantUML for sequence diagrams because Lark's server-side parser returns `code:2891001` on medium-sized component PlantUML. Never ship a TD with just prose or an empty whiteboard.

8c. **Publishing without saving the handoff memory.** When a user later says "update the TD to add X", the skill must be able to find the `doc_id` / parent node / original inputs without asking from scratch. Always write the `td_{feature_slug}` memory entry in Phase 5 step 4.

8d. **Editing the stale local `/tmp/backend-td.md` instead of re-fetching.** Between sessions the wiki copy may have been hand-edited or commented on. Always `lark-cli docs +fetch` before an update.

8e. **Describing flow in prose when a diagram or pseudo-code would be shorter — or skipping pseudo-code entirely.** Prefer diagram + pseudo-code (they complement, not compete). Every new/changed flow should show real method signatures and a short body in the repo's language so a downstream agent can spec-code directly from the TD. If you wrote 3+ sentences explaining how requests flow, re-cast as diagram + code. If there's no code snippet for a flow that has real execution, add one.

8f. **Fake pseudo-code.** `Foo.doThing()` / `handler(req)` placeholders defeat the point. Use real package paths, real receiver types, real downstream PSM method names from the IDL, and real table/column names from the RDS schema. If you haven't read the code yet, read it first — the snippet is the spec the implementer (or coding agent) will lean on.

9. **DECC pitfalls (cross-region only).** Load `decc.md` (sibling reference) — it enumerates the full list (label-version drift, missing caller-side code changes, binary over DES-RPC, TT user-data in forbidden directions, HTTP skipping BAM registration, RDS/MQ sync path confusion). Don't try to recall these from memory.

10. **Best-practices violations.** Load `bytedance-best-practices.md` (sibling reference) when writing Data Model / IDL / Cache / MQ / Config sections. Common repeat-offenders: MySQL `UNIQUE KEY` on synced tables, `FOREIGN KEY`, `AUTO_INCREMENT` primary keys, cache keys without TTL, MQ consumer without DLQ.

11. **Broken swimlane naming.** Swimlanes are routing headers attached to HTTP/RPC calls that send boe/ppe traffic to the correct test instance. The suffix after the env prefix MUST be byte-identical across envs (`boe_ai_chatbot` + `ppe_ai_chatbot`, not `boe_quality_bench` + `ppe_om_api_gateway`), MUST match `^[a-z][a-z0-9_]{1,29}$`, and MUST be a feature slug — not a PSM name. Never include a `prod_*` entry (prod is the untagged default). A mismatched slug silently routes test traffic to prod or drops it, breaking the entire test-environment story. **FE and BE TDs MUST use the same `feature_slug`** — a mismatch causes FE and BE to route to different test instances.

12. **Section-numbering inconsistency.** Never prefix top-level `#` headings with numbers (`# 1. Background` is wrong; `# Background` is right). Never prefix Feature Design subsections with numeric tags like `7.1`, `7.2` when no outer section is numbered — those numbers have no anchor and rot on reorder. Stick to descriptive `##` subsection titles and cross-reference by title. The template's section order IS the doc's order.

13. **Prose paragraph under a `<whiteboard>` tag.** A re-description of the diagram's nodes and edges ("Whiteboard shows: FE → BFF → PSM → (a) RDS, (b) Coze, ...") violates the "no prose re-describing a diagram" rule, duplicates the figure, and rots the moment the diagram is edited. Allowed forms immediately after a `<whiteboard>` tag: (a) nothing, (b) ONE italic caption line like `*Fig — request flow.*`. If reviewers need a legend, add it as a node inside the whiteboard, not as doc prose.

14. **`replace_range` on a section containing a whiteboard reference orphans the board.** See `update-flow.md` (sibling reference) "Gotcha" section for the full diagnosis and fix.

15. **Only listing IDL diffs without full parameter definitions.** A before→after diff shows what changed, but the frontend team has no prior context on the interface — they need the complete request/response struct with every field's number, type, required/optional, and description. Existing interfaces that are called but unchanged also need full IDL listed. Without this, the frontend cannot design their integration and must chase backend owners for clarification, causing delays.
