---
name: cspadk-tech-design
description: Unified router for Technical Design generation supporting both Frontend and Backend TD. Uses Lark CLI skills for document operations. Use this skill when the user asks for a technical design, or says "write TD", "create technical design", "tech design document", "generate TD", "write a tech doc", "TD for this feature", "frontend TD", "FE TD", "backend TD", "BE TD", "API design", "service TD", or "server-side TD".
version: 1.6.0
metadata:
  patterns:
    - generator
  domain: cspadk
  i18n_level: 0
  prompt_version: "1.6.0"
  agent_support:
    - claude-code
    - cursor
    - trae
  language:
    - zh-CN
    - en
---

# Technical Design Document Generator

Unified router for generating Technical Design (TD) documents from product requirements, design specs, and codebase analysis. Supports both **Frontend TD** and **Backend TD** with a normalized workflow.

## TD Type Detection

| Trigger Words | TD Type | Reference Guide |
|---------------|---------|-----------------|
| "frontend TD", "FE TD", "UI design", "Figma TD" | **Frontend** | [references/frontend-td.md](references/frontend-td.md) |
| "backend TD", "BE TD", "API design", "service TD", "server-side TD" | **Backend** | [references/backend-td.md](references/backend-td.md) |
| Other TD-related words | Ask via `AskUserQuestion` | — |

If the user doesn't specify, use `AskUserQuestion` to ask: "Are you generating a Frontend or Backend technical design?"

## Requirement Status Tracking

This skill updates the requirement status via CLI commands:

| Field | Value | CLI Command |
|-------|-------|-------------|
| `currentPhase` | `tech-design` | `cspadk req-update-phase tech-design` |
| `documents.techDesign` | `<local-td-path>` | `cspadk req-update-document techDesign <path>` |
| `documents.techDesignRemote` | `<remote-td-url>` | `cspadk req-update-document techDesignRemote <url>` |
| `extraContext[]` | `{ path, label, remote? }` | `cspadk req-extra-context-add <path> --label <label> [--remote <url>]` |

**Status file location**: `.ttadk/requirements-status/<requirement-name>.json`

## Lark Operations

This skill uses `lark-cli` commands for all Lark document operations:

| Operation | Command | Purpose |
|-----------|---------|---------|
| Authentication | `lark-cli auth status` | Check Lark authentication status |
| Document Read | `lark-cli docs +fetch` | Fetch Lark document content |
| Document Create | `lark-cli docs +create` | Create a new Lark document |
| Document Update | `lark-cli docs +update` | Update an existing Lark document |
| Whiteboard | `lark-cli whiteboard +update` | Create architecture diagrams, flowcharts |
| Wiki Node | `lark-cli wiki +node-create` | Create a wiki node under a parent |

**Prerequisite**: `lark-cli` must be installed and authenticated. Run `lark-cli auth status` to verify.

## Inputs

This skill can work with either:

1. **Brainstorming artifacts** (preferred):
   - Design document referenced in `documents.design` field of requirement status
   - Read it through `cspadk req-current --json`

2. **Requirement source documents from requirement status**:
   - Local synced PRD/TD files in `documents.prd` and `documents.td`
   - Original Feishu/Lark source links in `documents.prdRemote` and `documents.tdRemote`

3. **Extra context documents from requirement status**:
   - Additional technical documents in `extraContext` array (e.g., backend tech designs from other teams, data warehouse designs, infrastructure plans)
   - Each entry has `path`, `label`, and optional `remote` fields

4. **Direct inputs** (when no prior artifacts exist):
   - PRD link, Figma link, code path, output wiki URL
   - Implementation ideas, scope definition

## Configuration

This skill reads configuration from `.ttadk/csp-config.json`:

| Field | Description | Default |
|-------|-------------|---------|
| `techDesign.wikiParentUrl` | Lark wiki parent node URL | (optional) |
| `techDesign.localPath` | Local path to store TD files | `docs/tech-design` |
| `bits.space` | BITS deploy space ID | (optional) |
| `bits.deployEnabled` | Whether to enable BITS deployment | `false` |

## Normalized Workflow

### Phase 0: Load Configuration, Check Artifacts, and Authenticate

1. **Load CSP Configuration**:
   - Read `.ttadk/csp-config.json` if exists
   - Extract `techDesign.wikiParentUrl` as the default wiki parent URL
   - Extract `techDesign.localPath` and `bits` settings

2. **Get Current Requirement Status**:
   ```bash
   cspadk req-current --json
   ```
   Parse the output to get document paths from requirement status.

3. **Check for Brainstorming Artifacts**:
   - Check `documents.design` in the requirement status first
   - If exists, read the design document for context
   - Also read any documents referenced in `extraContext` array for supplementary context
   - Skip to Phase 2 for analysis if artifacts exist

4. **Check Lark Authentication**:
   - Run `lark-cli auth status` to check authentication status
   - If not authenticated, guide the user to authenticate

5. **Detect Output Language**:
   - Analyze the user's input language
   - If the user writes primarily in Chinese, set `output_lang = "zh-CN"`
   - If the user writes primarily in English, set `output_lang = "en"`
   - If mixed or unclear, default to `"en"` unless the user explicitly requests Chinese
   - This `output_lang` variable determines which template to load in later phases

### Phase 1: Determine TD Type and Gather Context

1. **Determine TD Type**:
   - Detect TD type from trigger words (Frontend/Backend)
   - If ambiguous, use `AskUserQuestion` to clarify

2. **Load TD Type Reference**:
   - **Frontend TD**: Load [references/frontend-td.md](references/frontend-td.md), then load the appropriate template based on `output_lang`:
     - `output_lang = "en"` → [references/frontend-td/template-en.md](references/frontend-td/template-en.md)
     - `output_lang = "zh-CN"` → [references/frontend-td/template-zh.md](references/frontend-td/template-zh.md)
   - **Backend TD**: Load [references/backend-td.md](references/backend-td.md)

3. **Gather Context**:
   - Fetch Lark documents using `lark-cli docs +fetch`
   - Read any documents from `extraContext` array in requirement status
   - Explore Codebase using Agent tool with `subagent_type: Explore`
   - **Backend only**: Load `references/backend-td/context-gathering.md`

4. **Backend TD Specific**:
   - Determine sizing tier (XS/S/M/L)
   - Run classification gate for L tier
   - Build and confirm dependency manifest

### Phase 2: Analyze and Design

**If brainstorming artifacts exist**: Use the brainstorming output as the foundation.

**If no brainstorming artifacts**: Think through from scratch:
1. What are the real technical challenges?
2. What can be reused?
3. What are the alternatives?
4. Where are the risks?

**Backend TD**: Load `references/backend-td/bytedance-best-practices.md` before designing.

### Phase 3: Write the TD Document

Write a comprehensive markdown file at `/tmp/td.md`:

- **Frontend TD**: Use the template loaded in Phase 1 (based on `output_lang`)
- **Backend TD**: Load the appropriate template based on `output_lang`:
  - `output_lang = "en"` → [references/backend-td/template.md](references/backend-td/template.md)
  - `output_lang = "zh-CN"` → [references/backend-td/template-zh.md](references/backend-td/template-zh.md)
  - Also load [references/backend-td/style-guide.md](references/backend-td/style-guide.md) regardless of language

Every section must be grounded in the codebase exploration and PRD analysis.

### Phase 4: Decide Whether to Create a Remote Lark Doc

1. Use `AskUserQuestion` to ask whether to create a remote Lark document
2. If user declines, skip to Phase 7
3. If user agrees:
   - **If `techDesign.wikiParentUrl` is already configured**: Use it directly
   - **If not configured**: Ask for the wiki parent URL via `AskUserQuestion`
     - After user provides the URL, update `.ttadk/csp-config.json`:
       ```json
       {
         "techDesign": {
           "wikiParentUrl": "<user-provided-url>"
         }
       }
       ```
     - Merge with existing config, don't overwrite other fields

### Phase 5: Publish to Lark Wiki (Optional)

Use `lark-cli` to create the document:

1. Create the document:
   ```bash
   lark-cli docs +create --title "<Feature Name> Technical Design" --markdown "$(cat /tmp/td.md)"
   ```

2. If `techDesign.wikiParentUrl` is set, create a wiki node:
   ```bash
   lark-cli wiki +node-create --parent-node-token <wiki-parent-token> --title "<Feature Name> Technical Design"
   ```

3. Save returned `doc_id` and `doc_url`

4. **For diagrams**: Create whiteboards:
   ```bash
   lark-cli whiteboard +update --whiteboard-token <whiteboard_token> --input_format mermaid --source "$(cat diagram.mmd)" --overwrite --yes
   ```

5. **For Frontend TD**: Insert Figma screenshots with captions

5. Return the document URL to the user

### Phase 6: Update Existing Doc (when user requests changes)

When the user asks to modify an existing TD:

1. **Always re-read the entire document first**:
   ```bash
   lark-cli docs +fetch --doc "<doc_url>"
   ```

2. **For targeted section updates**:
   ```bash
   lark-cli docs +update --doc "<doc_url>" --mode overwrite --markdown "$(cat /tmp/td-updated.md)"
   ```

3. **For multiple section changes**:
   - Use overwrite mode with full corrected markdown

4. **Backend TD**: Load `references/backend-td/update-flow.md`

### Phase 7: Update Status via CLI

- Update the phase:
  ```bash
  cspadk req-update-phase tech-design --json
  ```
- Record the generated local TD file path:
  ```bash
  cspadk req-update-document techDesign <local-td-path> --json
  ```
- If remote Lark TD was created:
  ```bash
  cspadk req-update-document techDesignRemote <wiki-url> --json
  ```
- If additional context documents were discovered during the TD process (e.g., related tech designs from other teams, data warehouse schemas), add them:
  ```bash
  cspadk req-extra-context-add <path> --label "<label>" [--remote <url>] --json
  ```

## Quality Checklist

Before publishing, verify:

- [ ] TD type correctly determined (Frontend/Backend)
- [ ] All required sections included per TD type
- [ ] Every technical decision has a comparison table with alternatives
- [ ] Code examples use real type names and API paths from the actual codebase
- [ ] No placeholder content — every section has substantive, specific content
- [ ] No stale references — all sections are consistent
- [ ] Swimlane section present (both FE and BE TDs) with boe/ppe entries; slug byte-identical across envs; FE slug matches BE slug

**Backend TD additional checks**:
- [ ] TD tier correctly sized
- [ ] Dependency manifest confirmed
- [ ] All conditional sections included per Classification Gate
- [ ] Complete IDL definitions for ALL interfaces (new, modified, and existing-but-called) with full request/response parameters — frontend team depends on this

## Next Step

After completing this skill:

- **Recommended**: Use `/cspadk-continue` to automatically proceed to the next workflow phase.
- Use `/cspadk-requirement-breakdown` to split complex requirements into logical modules.
- Use `/cspadk-openspec-flow` to generate OpenSpec artifacts.
- Share `documents.techDesignRemote` with reviewers when a remote Lark TD was created.

Update the requirement status via CLI before proceeding.

## Output Language

The output language is determined by the user's input language and controls which template is used:
- **Chinese input** → `output_lang = "zh-CN"` → All TD content (section headings, instructions, descriptions) is written in Chinese using the `-zh` template variant.
- **English input** → `output_lang = "en"` → All TD content is written in English using the default template.
- **Mixed/unclear** → Default to `"en"` unless the user explicitly requests Chinese.

**Technical terms that ALWAYS remain in English** regardless of `output_lang`:
- Platform names: PSM, IDL, RDS, TOS, BMQ, Abase/Redis, TCC, APM, DECC, etc.
- Technology names: Thrift, Kitex, React, TypeScript, SQL, GraphQL, etc.
- Code: variable names, function names, class names, SQL statements
- PSM identifiers and service names (e.g., `data.ies.csp_quality_consumer`)
- Tool names: lark-cli, bytedcli, etc.

This language setting applies to:
- The document template used for generation
- All section headings and descriptive text in the TD
- Quality checklist feedback and suggestions
