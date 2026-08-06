# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **Claude Code Skills Plugin Collection** for ByteDance's GDPA team. It provides structured knowledge and best practices for GDP service development, deployment, testing, and observability.

**This is NOT a compiled project** - it's a documentation/knowledge base repository distributed through Claude Code and Cursor plugin marketplaces.

## Repository Structure

All skills are flat in `skills/`, each with a `SKILL.md`. Each plugin directory under `plugins/` contains a `plugin.json` and symlinks to skills.

```
gdpa_skills/
├── skills/                   # All skills flat (single source of truth)
│   ├── gdp-knowledge/
│   ├── kitex-knowledge/
│   ├── bam-api/
│   └── ...
├── plugins/                  # Plugin grouping
│   ├── backend/
│   │   ├── .claude-plugin/plugin.json
│   │   ├── .cursor-plugin/plugin.json
│   │   └── skills/           # Symlinks to ../../skills/<name>
│   ├── backend-knowledge/
│   └── backend-tools/
├── .claude-plugin/           # Plugin marketplace config
├── .cursor-plugin/           # Cursor plugin marketplace config
├── README.md
└── CLAUDE.md
```

## Plugin System

Plugins are defined in `.claude-plugin/marketplace.json` (Claude Code) and `.cursor-plugin/marketplace.json` (Cursor). Both contain the same 4 plugins: `backend`, `backend-knowledge`, `backend-tools`, `gdp`. Claude Code lists skills explicitly; Cursor auto-discovers from `skills/` directories.

**Adding a new skill (complete checklist):**

1. **Create skill directory**: `skills/<skill-name>/` with kebab-case naming
2. **Add SKILL.md** with frontmatter: `name`, `description` (start with "Use when..." or "当...时使用")
3. **Update marketplace.json**: Add `"./skills/<skill-name>"` to appropriate plugin(s) in `.claude-plugin/marketplace.json`
4. **Update plugin.json**: Add `"./skills/<skill-name>"` to `plugins/<plugin>/.claude-plugin/plugin.json`
5. **Create symlinks**: Run for each target plugin:
   ```bash
   cd plugins/<plugin>/skills && ln -s ../../../skills/<skill-name> <skill-name>
   ```
   > **Cursor note:** No Cursor config changes needed when adding skills — Cursor auto-discovers from `skills/` directories. Only update `.cursor-plugin/marketplace.json` and `plugins/<plugin>/.cursor-plugin/plugin.json` when adding a **new plugin**.
6. **Update README.md**:
   - Update skill counts in Plugins table
   - Add skill to backend-tools table (or appropriate section)
   - Add skill to directory structure tree

**Example: Adding `my-new-tool` to backend and backend-tools:**

```bash
# 1. Create skill
mkdir -p skills/my-new-tool
# Add SKILL.md with proper frontmatter

# 2. Update marketplace.json - add to backend and backend-tools plugins

# 3. Update plugin.json files
# plugins/backend/.claude-plugin/plugin.json
# plugins/backend-tools/.claude-plugin/plugin.json

# 4. Create symlinks
cd plugins/backend/skills && ln -s ../../../skills/my-new-tool my-new-tool
cd plugins/backend-tools/skills && ln -s ../../../skills/my-new-tool my-new-tool

# 5. Update README.md counts and tables
```

## Skill File Format

### Knowledge Skills (non-invocable)

Knowledge skills (`*-knowledge`) are auto-loaded based on context, not manually invoked:

```yaml
---
name: xxx-knowledge
description: Use when [triggering conditions]. [Brief context].
user-invocable: false
---

# Skill Name

## Instructions
[Core guidance]

## Examples
[Concrete examples]

## Related Resources
[Links to Feishu docs, wikis]
```

### Tool Skills (invocable)

Tool skills can be manually invoked by users via `/skill-name`:

```yaml
---
name: skill-name
description: Use when [triggering conditions]. [Brief context].
---

# Skill Name

## Instructions
[Core guidance]

## Examples
[Concrete examples]
```

## Conventions

- **Language:** Documentation in Chinese, code examples in Go/Thrift
- **References:** Internal Feishu wikis and ByteDance platform links
- **Naming:** Use hyphens in skill names, verb-first active voice preferred
- **Content:** Focus on GDP framework patterns, internal tooling, and ByteDance-specific practices
