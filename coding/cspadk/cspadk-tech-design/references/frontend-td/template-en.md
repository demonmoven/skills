# Frontend TD Document Template (English)

Load this file when writing the FE TD markdown in Phase 3. This template is the skeleton for a Frontend Technical Design document.

Write the doc to `/tmp/td.md`.

---

```markdown
## Version History

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0.0 | {today's date} | Initial draft | FE |

## Requirement

| Item | Link |
|------|------|
| PRD | {prd_link} |
| Backend TD | {backend_td_link} |
| Figma | {figma_link} |
| Meego | |

## 1. Background & Problem Statement

### Current Pain Points
{Analyze REAL issues from the codebase — not hypothetical problems. Reference specific files, components, and patterns that cause issues. Use data from exploration.}

### Scope
{Define what's in scope for this TD vs out of scope. Use a table with Segment/Goal/FE Scope columns if the work has multiple phases.}

## 2. Architecture Overview

### 2.1 Core Design Principles
{Table with columns: Principle | Decision | Rationale. Each row is a key architectural choice.}

### 2.2 Data Flow
{Step-by-step description of the primary user flow. Number each step. Include what component handles it, what API is called, and what state changes.}

## Swimlane

> **Swimlane = a routing header** (`X-TT-ENV` / RPC env tag) attached to HTTP & RPC calls so that boe/ppe traffic lands on the correct isolated test instance. Convention: `{env}_{feature_slug}` where `env ∈ {boe, ppe}` and `feature_slug` is a short snake_case identifier describing the **feature**, NOT the PSM (PSM names are too long and are shared across features).
>
> **Hard rules — violations break test-env routing:**
> 1. The `{feature_slug}` MUST be **byte-identical** across envs (same slug on `boe_` and `ppe_`). `boe_ai_chatbot` + `ppe_ai_summary` is broken.
> 2. Slug regex: `^[a-z][a-z0-9_]{1,29}$` — starts with a letter, snake_case, ≤30 chars total (the env prefix counts against the header's routing-label length cap).
> 3. Full swimlane regex: `^(boe|ppe)_[a-z][a-z0-9_]{1,29}$`.
> 4. Do NOT list a `prod_*` swimlane — prod has no swimlane (it's the default un-tagged route). Only boe and ppe get entries.
> 5. One slug per TD. Don't fragment the feature across multiple swimlanes unless the feature legitimately spans independently-tested subsystems (rare).
> 6. Frontend swimlane MUST match the backend TD's `feature_slug` — both FE and BE share the same lane for a feature.

- boe_<feature_slug>
- ppe_<feature_slug>

## 3. Layout Restructuring

### 3.1 Key Layout Change
{One paragraph explaining what changes at the highest level.}

### 3.2 New Component Tree
{Code block showing the new component hierarchy with comments. Use plaintext, not JSX — Lark renders it better.}

### 3.3 New / Modified File Structure
{Directory tree with inline comments explaining each new/modified file. Mark NEW vs MODIFIED vs DEPRECATE.}

## 4-N. Feature Design Sections

{One numbered section per major feature. Each should include:}

### N.1 Data Model
{TypeScript interfaces for key data structures}

### N.2 Component Design
{How the component works, DOM structure, CSS strategy}

### N.3 Implementation
{Key code — component implementation, hooks, state management. Use REAL type names and API paths from the codebase.}

### N.4 Performance Comparison (if relevant)
{Table comparing old vs new approach: Aspect | Current | New}

## Technology Selection & Key Decisions

{For EVERY significant technical decision, include:}

### Decision: {Name}
{Comparison table of all evaluated options}

| Approach | How it works | Pros | Cons | Score |
|----------|-------------|------|------|-------|

{Why the chosen approach wins — specific, technical reasons}
{Why each alternative was rejected — specific, technical reasons}
{Verification plan if there's uncertainty}

This is the most important section of the TD. Reviewers will focus here. Every decision must be justified with evidence, not opinion.

## State Management Design

### Store Architecture
{ASCII diagram or code block showing store structure, slices, and what React Query handles vs what Zustand handles.}

### Initialization Flow
{Numbered steps from app mount to ready state.}

## API Integration

### New API Endpoints
{Table: FE Endpoint | Backend RPC | Method}

### Request/Response Types
{TypeScript interfaces matching the backend TD's Thrift/Protobuf definitions}

### Event Tracking / Activity Recording
{How user actions are tracked — fire-and-forget patterns, MQ integration.}

## Error Handling & Edge Cases

### Error Boundaries
{Which components need error boundaries, fallback UI behavior.}

### API Error Handling
{Table: Error Type | User-Facing Message | Recovery Action | Retry Policy}

### Edge Cases
{List non-obvious edge cases: empty states, offline, rate limiting, concurrent mutations, partial failures.}

## Performance Budget & Optimization

{Table: Metric | Target | Current Baseline | Strategy}

{List optimization techniques with brief explanation of each.}

{Memory management model — what's the expected memory footprint and how is it bounded.}

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
{Identify technical risks from the design. Include: dependency risks, migration risks, performance risks, compatibility risks.}

## Testing Strategy

### Unit Tests
{Table: Module | What to test. Focus on extractable business logic, not UI.}

### Integration Tests
{Table: Scenario | Validation criteria}

## Timeline Estimate

{Table: Task | Effort (PD) | Dependency}

Guidelines for timeline:
- Group tasks by feature area with section headers (bold row, no effort)
- Each task should be ≤ 2 PD. If larger, break it down.
- Include unit tests and integration tests as separate line items
- Include a "Performance profiling & tuning" line item (typically 1-2 PD)
- Add a Total row at the bottom
- Dependencies should reference other task names, "—" for no dependency, or "Backend API ready" for external deps
```
