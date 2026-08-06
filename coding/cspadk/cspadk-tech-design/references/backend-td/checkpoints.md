# AskUserQuestion Checkpoint Catalog

Load this file whenever you hit a decision gate and need to confirm with the user. Backend TDs are high-blast-radius — confidently-wrong defaults waste reviewer cycles.

**Rule:** any decision the user should own → `AskUserQuestion` with explicit options. Free-form narration ("I'm about to write the IDL section now") is fine between checkpoints. If a checkpoint has only one real choice, proceed without asking — don't ask performative questions. Prefer fewer options that are meaningfully different over many near-duplicates (the tool allows up to 4 structured options plus free-form).

---

## Mandatory checkpoints

| # | When | Question | Options to offer |
|---|------|----------|------------------|
| 1 | Missing required input (PSM, wiki URL) | "I need X before proceeding" | Free-text via `openEndedSuggestions` |
| 2 | After exploration, before writing | "Sizing this as **{tier}** because {reasons}. Confirm?" | XS / S / M / L / Describe different sizing |
| 3 | Dependency manifest draft | "Is this manifest correct?" | Confirm as-is / Add missing / Remove incorrect / Major revision needed |
| 4 | L tier only — each Classification Gate axis | "Does this feature touch {axis}?" | Yes / No / Uncertain — skip subsection |
| 5 | Significant design decision with ≥2 viable approaches | "Which approach for {decision}?" | Option A / Option B / Option C / Let me explain a 4th |
| 6 | IDL drift check needed | "Vendored IDL for {psm} is N days old — check Overpass for drift?" | Yes check / Trust vendored / I'll check manually |
| 7 | Before publishing to wiki | "Ready to publish at {title} under {parent_node}?" | Publish / Change title / Change parent / Review markdown first |
| 8 | After publish, for architecture diagram | "Populate the embedded whiteboard with the Mermaid diagram now?" | Populate now / Leave blank for manual drawing / Skip (ASCII only) |
| 9 | Phase 6 targeted edit | "Apply change to section **{Section}** only?" | Apply / Show diff first / Cancel |
| 10 | Phase 6 structural change | "This update rewrites the whole doc. Ready?" | Rewrite / Cancel / Keep as targeted edit instead |

## Worked examples

**Sizing checkpoint (#2) — include reasoning in the question body:**
> "Sizing this as **M** — IDL change on our PSM + new Abase key + 2 downstream calls. Confirm?"

**Manifest confirmation (#3) — render the manifest first, then ask.**

**Design decision (#5) — do not silently pick.** When the skill itself is uncertain between approaches, lay out the top 2–3 with tradeoffs and let the user choose. If you catch yourself writing "we'll go with X" without asking, stop and convert to a checkpoint.
