# Backend TD Writing Style Guide

Load this file in Phase 4 alongside `template.md`. Covers presentation priority, writing tone, section headings, tables, diagrams, captions, and spec-codable mode.

---

## Presentation priority — pick the leanest form, pair diagram + pseudo-code when spec-driven coding benefits

Reviewers scan in this order; start at the top form that works. **Diagram and pseudo-code are encouraged to coexist** — the diagram shows *shape*, the pseudo-code shows *execution*. That pairing is what lets a downstream coding agent (or human) implement directly from the TD without re-deriving intent.

| Priority | Form | Use for | When to escalate down |
|---|---|---|---|
| 1 | **Diagram** (native Lark whiteboard; Mermaid by default, PlantUML reserved for sequence diagrams) | Architecture, data flow, call graph, sequence of interactions, state machine, ER / table relationships, deployment topology | Diagram can't express the constraint (e.g. ordering invariants, precise branching, formulas) |
| 2 | **Pseudo-code in the repo's language** (Go / Java / Rust / …) with short comments on the non-obvious steps | Every module-level flow; method signatures of new/changed APIs; control flow, idempotency keys, retry loops, lock / transaction boundaries, cache key construction, MQ ack patterns | Idea is a policy / trade-off / fact with no execution to show |
| 3 | **Bullets** — one fact per bullet, direct voice, no hedging | Constraints, requirements, assumptions, risks, invariants, scope in/out, rollout steps | You're comparing ≥2 options on multiple axes |
| 4 | **Table** | Side-by-side comparison: decision alternatives, failure matrix, consistency trade-offs, timeout/retry budget per downstream, tier classification | N/A (tables are the terminal form for comparisons) |

**Rules:**
- Every architecture / data-flow section leads with a diagram. The diagram may be immediately followed by pseudo-code implementing the flow it depicts — that combo is explicitly desired.
- Pseudo-code is **strongly encouraged** for every new/changed flow: real method signatures (package, receiver, types, return) from the repo, with short bodies showing the key steps. This is what turns the TD into a spec — downstream agents can code directly from it.
- Pseudo-code must use real types / method names from the repo — not `Foo.doThing()`. Read the existing code first; match its conventions (Go idioms in Go repos, Spring annotations in Spring repos, etc.).
- Keep pseudo-code snippets focused (~5–30 lines each). If it grows longer, split by responsibility or lift sub-steps into bullet commentary.
- **No duplicate *prose* re-describing a diagram or code block.** The forms should be complementary (diagram shows shape, code shows steps), never redundant.
- Prose paragraphs are the form of last resort. If you catch yourself writing 3+ sentences to explain flow, re-cast as diagram + code.

## Spec-codable TD — optional but recommended

When the TD is intended to drive spec coding (i.e., a downstream agent will implement from it), include these extras so the TD *is* the spec:
- Full method signatures for every new/changed public method, with param/return types and (for Thrift) IDL field numbers.
- Complete IDL definitions for ALL interfaces involved (new, modified, and existing-but-called) — every request/response field with number, type, required/optional, and description, nested structs expanded to leaf level. This is critical for frontend teams who lack prior context on the interfaces.
- Pseudo-code bodies that name the exact downstream PSM methods, RDS tables, cache keys, and MQ topics being touched.
- Error / retry behavior written as code branches, not prose ("on X, do Y" becomes an `if`/`switch`).
- A short "Acceptance" bullet per flow: the observable outcome an implementation must satisfy.

## Writing style — concise, scannable

- Use bullet points, not paragraphs. One fact per bullet. Reviewers skim — narrative prose costs them time.
- Short, direct sentences. Drop hedges ("it might be", "we could consider"). State the design.
- Only use prose when a bullet actually loses meaning (rare — usually the "why" of a decision).
- Keep Background to ≤5 bullets; skip a multi-paragraph intro.

## Section heading convention — no arbitrary numbering

- Top-level headings (`#`) use the section names from the Document Template verbatim — **no leading numbers** (never `# 1. Background`, never `# 7. Feature Design`). The template's section order IS the doc's order; numbers add nothing and rot on reorder.
- Subsection headings under `# Feature Design` (or any other section) use `##` + a short descriptive name (e.g. `## Chatbot — Session + Multi-turn`). **Do not prefix with numeric tags** like `7.1`, `7.2` — those numbers have no anchor (the outer section isn't numbered) and become stale the moment you reorder or insert a new subsection.
- Picking a numbered style is only allowed when ALL top-level sections AND their subsections are numbered consistently throughout the doc (rare; mostly reserved for TDs driven by a numbered requirement list in the PRD). Mixing numbered and unnumbered headings in the same doc is a hard no.
- For cross-references, link by **section title** (e.g. "see the Chatbot — Output Templates subsection"), never by a number like "see 7.2". Titles survive reorders; numbers don't.
- One scoped exception: if the PRD itself enumerates requirements `1.`, `2.`, `3.` and the TD maps one-to-one, mirror the PRD's numbering inside Feature Design subsections AND state up front "numbers match PRD requirement list" so reviewers know the stability guarantee comes from the PRD, not this doc.

## Tables — prefer GitHub-flavored markdown

- Default to `| col | col |` markdown tables. Lark parses them correctly and they're far easier to edit later.
- Use `<lark-table>` XML only when you need specific column widths, merged cells (`colspan`/`rowspan`), header-column styling, or multiline cells with nested markdown. For those cases the XML is worth the verbosity.
- Do NOT mix both styles in the same section — pick one per table.

## Diagrams — native Lark whiteboard by default; ASCII only for trivial inline shapes

- Every TD needs an architecture / data-flow diagram. Default to a **native Lark whiteboard** — Lark renders Mermaid/PlantUML into a real, editable whiteboard, which reviewers can zoom and comment on.
- **Diagram dialect — Mermaid is the default; PlantUML is reserved for sequence diagrams.**
  - Empirically, Lark's server-side parser returns `code:2891001 "server internal error"` on medium-sized PlantUML (10+ components mixing `package` / `database` / `cloud` / stereotyped rectangles). The equivalent Mermaid `graph LR` renders first try. Default to Mermaid to avoid the class of parser failures.
  - Use **Mermaid** for: architecture / data flow (`graph LR`), state (`stateDiagram-v2`), ER (`erDiagram`), class (`classDiagram`), deployment/topology.
  - Use **PlantUML** only for: sequence diagrams where PlantUML's `group`/`alt`/`loop`/`par` blocks are meaningfully richer than Mermaid's. For sequences with ≤6 participants and no nested alt blocks, Mermaid's `sequenceDiagram` is fine too.
- Flow (see Phase 5 for the full publish sequence):
  1. When the markdown is published via `lark-cli docs +create` / `+update`, include a **paired** `<whiteboard type="blank"></whiteboard>` placeholder at the spot where the diagram should appear. A **self-closing** `<whiteboard/>` is read-only (it's the form returned by `+fetch` for existing boards) and will be stripped with a `BOARD_TOKEN_NOT_SUPPORTED` warning. Repeat the paired tag once per diagram. The response returns `data.board_tokens` — one token per placeholder, in document order.
  2. Populate each whiteboard with the `lark-whiteboard` skill: `cd /tmp && lark-cli whiteboard +update --whiteboard-token <board_token> --input_format mermaid --source @./arch.mmd --overwrite --yes`. Flags: `--whiteboard-token` (not `--token`); `--source` takes `@./<relative-file>` (path MUST be relative to cwd — `cd` into the source dir first); `--overwrite` replaces existing content (first-time population usually needs it too); `--yes` confirms the "unsafe_operation" gate.
  3. For layouts Mermaid/PlantUML can't express (custom swimlanes, arbitrary node shapes), emit raw OpenAPI JSON via the `lark-whiteboard-cli` workflow — see the `lark-whiteboard` skill docs.
- Fall back to ASCII only for trivial inline sketches that belong inline with prose (≤5 boxes, no crossing edges) — e.g., a 3-box request path in a single subsection. Do NOT use ASCII for the top-level architecture diagram.
- Do NOT paste raw mermaid/plantuml fenced code blocks into the doc markdown — put the source in a whiteboard instead.
- Minimum required `lark-cli` version for the `whiteboard` subcommand is **1.0.8** (introduced with `whiteboard +query` / `+update` Mermaid+PlantUML support). Earlier versions silently strip `<whiteboard type="blank">` tags and `whiteboard +update` fails with `unknown command`. Check with `lark-cli --version`; upgrade via `npm install -g @larksuite/cli` before Phase 5 if needed.
- **Caption rule — after a `<whiteboard>` tag, either nothing or ONE italic caption line, never a paragraph.** A caption is a short (≤1 sentence) figure label like `*Fig — request flow across quality_bench + downstreams.*`. Do NOT write a prose paragraph re-describing what's in the diagram (node list, edge list, per-arrow narration) — that duplicates the diagram, violates the "no prose re-description" rule, and grows stale when the diagram evolves. If reviewers truly need a text index of the components, put it inside the whiteboard as a legend node, not as doc prose.
- `<sheet token="..."/>` placeholders for embedded spreadsheets if the user maintains one.
