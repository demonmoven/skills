# Cursor Plugin Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Cursor plugin support alongside existing Claude Code plugin configuration, enabling the same skills to be distributed on both platforms.

**Architecture:** Parallel `.cursor-plugin/` config directories mirroring the existing `.claude-plugin/` structure. Skills and symlinks are fully reused — only config files and docs are added.

**Tech Stack:** JSON config files, Markdown documentation

---

### Task 1: Create root marketplace.json for Cursor

**Files:**
- Create: `.cursor-plugin/marketplace.json`

- [ ] **Step 1: Create the `.cursor-plugin/` directory and `marketplace.json`**

```json
{
  "name": "gdpa-skills",
  "owner": {
    "name": "chenglinfeng",
    "email": "chenglinfeng@bytedance.com"
  },
  "metadata": {
    "description": "GDPA SKILLS",
    "version": "2.0.0",
    "pluginRoot": "plugins"
  },
  "plugins": [
    {
      "name": "backend",
      "source": "./plugins/backend",
      "description": "后端开发全栈技能集。涵盖 GDP 框架、RAL 资源访问、Kitex RPC、Hertz HTTP、Overpass 代码生成、存储/配置/监控 SDK、TikTok 开发规范、GDPA 工具集等。"
    },
    {
      "name": "backend-knowledge",
      "source": "./plugins/backend-knowledge",
      "description": "后端知识文档技能集。涵盖 GDP 框架、RAL、Kitex、Hertz、Overpass、存储/配置/监控 SDK、TikTok 开发规范、集成测试、可观测性等框架文档和规范指南。"
    },
    {
      "name": "backend-tools",
      "source": "./plugins/backend-tools",
      "description": "后端开发工具集。涵盖 GDPA CLI、BAM API 查询与测试、Overpass 代码生成、Repotalk 代码分析、Argos 日志查询、RDS/TCC 查询、SCM 版本管理等工具。"
    },
    {
      "name": "gdp",
      "source": "./plugins/gdp",
      "description": "GDP 服务开发技能集。涵盖 GDP 框架核心知识和 RAL 资源访问层。"
    }
  ]
}
```

- [ ] **Step 2: Verify the file is valid JSON**

Run: `cat .cursor-plugin/marketplace.json | python3 -m json.tool > /dev/null && echo "Valid JSON"`
Expected: `Valid JSON`

- [ ] **Step 3: Commit**

```bash
git add .cursor-plugin/marketplace.json
git commit -m "feat(cursor): add root marketplace.json for Cursor plugin support"
```

---

### Task 2: Create plugin.json for each of the 4 plugins

**Files:**
- Create: `plugins/backend/.cursor-plugin/plugin.json`
- Create: `plugins/backend-knowledge/.cursor-plugin/plugin.json`
- Create: `plugins/backend-tools/.cursor-plugin/plugin.json`
- Create: `plugins/gdp/.cursor-plugin/plugin.json`

- [ ] **Step 1: Create `plugins/backend/.cursor-plugin/plugin.json`**

```json
{
  "name": "backend",
  "displayName": "GDPA Backend",
  "version": "1.0.0",
  "description": "后端开发全栈技能集",
  "author": {
    "name": "chenglinfeng",
    "email": "chenglinfeng@bytedance.com"
  },
  "keywords": ["gdpa", "backend", "gdp", "tiktok"]
}
```

- [ ] **Step 2: Create `plugins/backend-knowledge/.cursor-plugin/plugin.json`**

```json
{
  "name": "backend-knowledge",
  "displayName": "GDPA Backend Knowledge",
  "version": "1.0.0",
  "description": "后端知识文档技能集",
  "author": {
    "name": "chenglinfeng",
    "email": "chenglinfeng@bytedance.com"
  },
  "keywords": ["gdpa", "knowledge", "sdk", "framework"]
}
```

- [ ] **Step 3: Create `plugins/backend-tools/.cursor-plugin/plugin.json`**

```json
{
  "name": "backend-tools",
  "displayName": "GDPA Backend Tools",
  "version": "1.0.0",
  "description": "后端开发工具集",
  "author": {
    "name": "chenglinfeng",
    "email": "chenglinfeng@bytedance.com"
  },
  "keywords": ["gdpa", "tools", "cli", "query"]
}
```

- [ ] **Step 4: Create `plugins/gdp/.cursor-plugin/plugin.json`**

```json
{
  "name": "gdp",
  "displayName": "GDPA GDP",
  "version": "1.0.0",
  "description": "GDP 服务开发技能集",
  "author": {
    "name": "chenglinfeng",
    "email": "chenglinfeng@bytedance.com"
  },
  "keywords": ["gdpa", "gdp", "ral"]
}
```

- [ ] **Step 5: Verify all 4 files are valid JSON**

Run: `for f in plugins/backend/.cursor-plugin/plugin.json plugins/backend-knowledge/.cursor-plugin/plugin.json plugins/backend-tools/.cursor-plugin/plugin.json plugins/gdp/.cursor-plugin/plugin.json; do echo -n "$f: "; cat "$f" | python3 -m json.tool > /dev/null && echo "OK"; done`
Expected:
```
plugins/backend/.cursor-plugin/plugin.json: OK
plugins/backend-knowledge/.cursor-plugin/plugin.json: OK
plugins/backend-tools/.cursor-plugin/plugin.json: OK
plugins/gdp/.cursor-plugin/plugin.json: OK
```

- [ ] **Step 6: Commit**

```bash
git add plugins/backend/.cursor-plugin/plugin.json plugins/backend-knowledge/.cursor-plugin/plugin.json plugins/backend-tools/.cursor-plugin/plugin.json plugins/gdp/.cursor-plugin/plugin.json
git commit -m "feat(cursor): add plugin.json for all 4 plugins"
```

---

### Task 3: Update README.md

**Files:**
- Modify: `README.md`

Changes needed:
1. Add Cursor marketplace install section alongside the existing Claude Code section in 快速开始
2. Add `.cursor-plugin/` entries to the 目录结构 tree

- [ ] **Step 1: Add Cursor marketplace install to 快速开始 section**

After the existing `### Claude Code` block (line 21) and before `### Cursor / Trae / Gemini` (line 23), replace the `### Cursor / Trae / Gemini` section header and its Cursor entry to separate Cursor marketplace install from the npx approach:

Replace lines 23-37 with:

```markdown
### Cursor

**方式一：Marketplace 安装（推荐）**

Cursor 已支持原生 Plugin Marketplace，与 Claude Code 类似：

1. 在 Cursor 中打开 Plugin Marketplace
2. 搜索 `gdpa-skills`
3. 选择需要的插件（推荐 `backend`）进行安装

**方式二：npx 安装**

```bash
npx skills add git@code.byted.org:tiktok/gdpa_skills.git --agent cursor --skill '*'
```

### Trae / Gemini CLI

```bash
# Trae（国际版）
npx skills add git@code.byted.org:tiktok/gdpa_skills.git --agent trae --skill '*'

# Trae CN（国内版）
npx skills add git@code.byted.org:tiktok/gdpa_skills.git --agent trae-cn --skill '*'

# Gemini CLI
npx skills add git@code.byted.org:tiktok/gdpa_skills.git --agent gemini-cli --skill '*'
```
```

- [ ] **Step 2: Add `.cursor-plugin/` to 目录结构 section**

In the directory tree (around line 214), add `.cursor-plugin/` entries. After line 214 (`├── .claude-plugin/marketplace.json`), add:

```
├── .cursor-plugin/marketplace.json   # Cursor plugin marketplace config
```

And within each plugin directory in the tree, the `.cursor-plugin/` is auto-discovered so no explicit listing is needed. But for clarity, add a comment at the top of the plugins section. Change line 147:

```
├── plugins/                          # Plugin 分组（通过软链接引用 skills，支持 Claude Code 和 Cursor）
```

- [ ] **Step 3: Verify README renders correctly**

Run: `head -50 README.md`
Expected: Updated 快速开始 section with separate Cursor marketplace install section.

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: add Cursor plugin install instructions to README"
```

---

### Task 4: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

Changes needed:
1. Update project overview to mention dual-platform support
2. Add `.cursor-plugin/` to repository structure tree
3. Add Cursor maintenance note to the "Adding a new skill" checklist

- [ ] **Step 1: Update Project Overview**

Replace line 7:
```markdown
**This is NOT a compiled project** - it's a documentation/knowledge base repository distributed through Claude Code's plugin marketplace.
```
with:
```markdown
**This is NOT a compiled project** - it's a documentation/knowledge base repository distributed through Claude Code and Cursor plugin marketplaces.
```

- [ ] **Step 2: Add `.cursor-plugin/` to Repository Structure tree**

After line 28 (`├── .claude-plugin/           # Plugin marketplace config`), add:
```
├── .cursor-plugin/          # Cursor plugin marketplace config
```

Within each plugin listing, after `.claude-plugin/plugin.json` lines, add the cursor counterpart. Update the tree to:

```
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

- [ ] **Step 3: Update Plugin System description**

Replace line 35:
```markdown
Plugins are defined in `.claude-plugin/marketplace.json` (4 plugins: `backend`, `backend-knowledge`, `backend-tools`, `gdp`). Skills paths point to `./skills/<name>`.
```
with:
```markdown
Plugins are defined in `.claude-plugin/marketplace.json` (Claude Code) and `.cursor-plugin/marketplace.json` (Cursor). Both contain the same 4 plugins: `backend`, `backend-knowledge`, `backend-tools`, `gdp`. Claude Code lists skills explicitly; Cursor auto-discovers from `skills/` directories.
```

- [ ] **Step 4: Add Cursor maintenance note to "Adding a new skill" checklist**

After step 5 (Create symlinks, line 46) and before step 6 (Update README.md), insert a new note:

```markdown
   > **Cursor note:** No Cursor config changes needed when adding skills — Cursor auto-discovers from `skills/` directories. Only update `.cursor-plugin/marketplace.json` and `plugins/<plugin>/.cursor-plugin/plugin.json` when adding a **new plugin**.
```

- [ ] **Step 5: Verify CLAUDE.md is well-formed**

Run: `head -40 CLAUDE.md`
Expected: Updated overview mentioning Cursor, structure tree with `.cursor-plugin/`.

- [ ] **Step 6: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md with Cursor plugin maintenance guidance"
```

---

### Task 5: Final verification

**Files:** (no changes, read-only checks)

- [ ] **Step 1: Verify all new files exist**

Run: `ls -la .cursor-plugin/marketplace.json plugins/backend/.cursor-plugin/plugin.json plugins/backend-knowledge/.cursor-plugin/plugin.json plugins/backend-tools/.cursor-plugin/plugin.json plugins/gdp/.cursor-plugin/plugin.json`
Expected: All 5 files listed with non-zero size.

- [ ] **Step 2: Verify JSON validity of all new files**

Run: `for f in .cursor-plugin/marketplace.json plugins/backend/.cursor-plugin/plugin.json plugins/backend-knowledge/.cursor-plugin/plugin.json plugins/backend-tools/.cursor-plugin/plugin.json plugins/gdp/.cursor-plugin/plugin.json; do echo -n "$f: "; python3 -m json.tool "$f" > /dev/null 2>&1 && echo "OK" || echo "INVALID"; done`
Expected: All 5 files show `OK`.

- [ ] **Step 3: Verify existing Claude Code configs are untouched**

Run: `git diff HEAD .claude-plugin/ plugins/backend/.claude-plugin/ plugins/backend-knowledge/.claude-plugin/ plugins/backend-tools/.claude-plugin/ plugins/gdp/.claude-plugin/`
Expected: No output (no changes to existing configs).

- [ ] **Step 4: Verify no SKILL.md files were modified**

Run: `git diff HEAD -- '*.md' | grep -c 'SKILL.md'`
Expected: `0` (no SKILL.md changes).

- [ ] **Step 5: Review git log for clean commit history**

Run: `git log --oneline -5`
Expected: 4 new commits:
```
feat(cursor): add root marketplace.json for Cursor plugin support
feat(cursor): add plugin.json for all 4 plugins
docs: add Cursor plugin install instructions to README
docs: update CLAUDE.md with Cursor plugin maintenance guidance
```
