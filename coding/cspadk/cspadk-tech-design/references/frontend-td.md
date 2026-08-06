# Frontend Technical Design Reference

This reference provides FE-specific guidance for generating Frontend Technical Design documents.

## Input Requirements

**Note**: These input requirements apply only in the **lite workflow** (when no brainstorming artifacts exist). In the full workflow, inputs are derived from the brainstorming design document.

| Input | Required | Example |
|-------|----------|---------|
| PRD link | Yes | `https://bytedance.larkoffice.com/wiki/xxx` |
| Figma link | Yes | `https://www.figma.com/design/xxx?node-id=xxx` |
| Code path | Yes | `/path/to/monorepo/apps/my-app` |
| Output wiki URL | Yes | `https://bytedance.larkoffice.com/wiki/xxx` (parent node) |
| Backend TD link | No | `https://bytedance.larkoffice.com/wiki/xxx` |
| Implementation ideas | No | User's initial thoughts, constraints, performance concerns |
| Scope | No | Which phases/segments to cover |

If the user omits the output wiki URL, ask for it before Phase 3. Everything else can be gathered during the workflow.

## Context Gathering

### Fetch Figma Mockups

1. Parse the Figma URL to extract `file_key` and `node_id` (convert `-` to `:` in node_id):
   ```
   URL: figma.com/design/<file_key>/...?node-id=<node_id>
   ```

2. Get node structure:
   ```bash
   curl -s -H "X-Figma-Token: $FIGMA_OAUTH_TOKEN" \
     "https://api.figma.com/v1/files/<file_key>/nodes?ids=<node_id>&depth=2"
   ```

3. Collect all child frame IDs, then get screenshot URLs:
   ```bash
   curl -s -H "X-Figma-Token: $FIGMA_OAUTH_TOKEN" \
     "https://api.figma.com/v1/images/<file_key>?ids=<node_ids_comma_separated>&format=png&scale=2"
   ```

4. Download all screenshots:
   ```bash
   mkdir -p /tmp/figma-mockups
   curl -s -o "/tmp/figma-mockups/<frame_name>.png" "<s3_url>"
   ```

5. View each screenshot with the Read tool to understand the UI design.

If `$FIGMA_OAUTH_TOKEN` is not set, ask the user for their Figma Personal Access Token.

### Explore the Codebase

Use the Agent tool with `subagent_type: Explore` for thorough codebase analysis. Request:

- Project structure, build system (Webpack/Rspack/Vite), bundler config
- Routing system and route definitions
- Component architecture and key components
- State management (Redux/Zustand/MobX) and data fetching (React Query/SWR)
- Existing implementations that will be affected by the new feature
- Performance patterns, caching mechanisms
- Module federation / micro-frontend setup (if any)
- Key type definitions and interfaces
- Testing setup and patterns

The exploration should focus on areas relevant to the PRD requirements. Don't explore the entire codebase — focus on what matters.

## Document Template

Load the appropriate template file based on the detected output language:

| Language | Template File |
|----------|---------------|
| English (`en`) | [template-en.md](frontend-td/template-en.md) |
| Chinese (`zh-CN`) | [template-zh.md](frontend-td/template-zh.md) |

Both templates include: Version History, Requirement, Background & Problem Statement, Architecture Overview, **Swimlane**, Layout Restructuring, Feature Design Sections, Technology Selection & Key Decisions, State Management Design, API Integration, Error Handling & Edge Cases, Performance Budget & Optimization, Risk Assessment, Testing Strategy, and Timeline Estimate.

## FE-Specific Quality Checklist

- [ ] Every technical decision has a comparison table with alternatives
- [ ] Performance section has quantifiable targets with current baselines
- [ ] Timeline tasks are each ≤ 2 PD with explicit dependencies
- [ ] Code examples use real type names and API paths from the actual codebase
- [ ] No placeholder content — every section has substantive, specific content
- [ ] Architecture diagrams use ASCII art (Lark doesn't render mermaid)
- [ ] Figma screenshots are inserted as images
- [ ] Risk items have mitigation strategies
- [ ] Open questions are listed if any API or data model is unclear
- [ ] No stale references — if approach changed during writing, all sections are consistent
- [ ] Swimlane section: exactly one boe and one ppe line; slug matches `^[a-z][a-z0-9_]{1,29}$`; the suffix after the env prefix is BYTE-IDENTICAL between boe and ppe; **FE slug MUST match the backend TD's slug**

## FE-Specific Common Pitfalls

1. **Validate assumptions before designing.** The first plausible-sounding approach often has a fundamental flaw hidden in the underlying mechanism (browser process model, library internals, framework constraints). Research the "how it actually works" before committing to 10 pages of detailed design.
2. **Check library/framework compatibility with the project's actual version.** Third-party libraries often have breaking compatibility issues with specific React/framework versions, rendering modes (StrictMode, createRoot), or build systems. Always verify against the project's actual setup, not just the library's README.
3. **Uncover hidden implementation complexity.** Features that look simple in the UI mockup may require solving non-trivial infrastructure problems (state isolation, routing context, cross-component communication) that the PRD never mentions. Walk through the implementation mentally and ask at each step: "Does the current system actually support this?"
4. **Architectural changes propagate everywhere.** A single design decision change touches component tree, file structure, data flow, implementation code, performance model, risk assessment, and timeline. After any direction change, do a full-text search for terms related to the old approach and fix every occurrence.
