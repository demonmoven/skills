# 2.1.4 Slash Command vs Skill

> **本节目标**：理解 Slash Command 和 Skill 的核心区别、各自适用场景，以及它们共享的逐层揭示（Progressive Disclosure）机制。

---

## 概念与历史

**Slash Command 是一段 Prompt 的"快捷键"，Skill 是"工具箱"。** 前者由人手动输入 `/xxx` 触发，Agent 不会自主调用；后者 Agent 会根据任务语义自动翻找并召回。两者共享同一套逐层揭示机制——启动时只加载名称和描述，调用时才加载完整指令，但在"谁来触发"这个关键问题上截然不同。

### 什么是 Slash Command

Slash Command 分为两类：

1. **内置 Slash Command**：Claude Code CLI 硬编码的操作，如 `/clear`、`/compact`、`/help`、`/model`
   - 直接执行固定逻辑，不经过 AI 推理
   - 不可自定义
2. **自定义 Slash Command**：用户定义的 `.claude/commands/*.md` 文件
   - 每个 `.md` 文件自动变成一个 `/command-name` 命令
   - 本质上是给 Agent 注入一段指令
   - **必须由用户手动输入 `/xxx` 触发，Agent 不会自主调用**

### 什么是 Skill

Skill 是自定义 Slash Command 的升级版，**两大核心区别：一是 Agent 可以根据 description 自动召回，二是支持目录结构，可附带辅助脚本和参考文档**：

```text
Custom Slash Command:                    Skill:
.claude/commands/deploy.md              .claude/skills/deploy/
（单文件）                                ├── SKILL.md          ← 主指令
                                        ├── helper.sh         ← 辅助脚本
                                        └── REFERENCE.md      ← 参考文档
                                        （目录结构，可附带任意辅助文件）
```

### 核心对比

| 维度 | Slash Command | Skill |
|------|--------------|-------|
| **文件结构** | 单个 `.md` 文件 | 目录（`SKILL.md` + 可选辅助文件） |
| **存放位置（项目级）** | `.claude/commands/*.md` | `.claude/skills/*/SKILL.md` |
| **存放位置（个人全局）** | `~/.claude/commands/*.md` | `~/.claude/skills/*/SKILL.md` |
| **调用方式** | 仅手动 `/xxx` | 手动 `/xxx` + **Agent 自动召回** |
| **Agent 自动召回** | 不支持 | 支持（基于 description 语义匹配） |
| **辅助文件** | 不支持 | 支持（脚本、参考文档、模板等） |
| **Frontmatter** | 支持 | 支持（多一个 `disable-model-invocation` 控制） |
| **逐层揭示** | 支持（启动时只加载 name+description） | 支持（同） |
| **定位** | 用户的快捷操作入口 | Agent 的可发现能力包 |

---

## 核心机制：逐层揭示

两者共享的核心设计优势是**逐层揭示（Progressive Disclosure）**：

### 加载流程

```text
阶段 1：启动时
  → 只加载 frontmatter 中的 name + description
  → 每个 Skill/Command ~30-50 tokens
  → 100 个也只占 ~5000 tokens

阶段 2：匹配时（仅 Skill）
  → Claude 的 LLM 推理判断当前任务是否匹配某个 Skill 的 description
  → 纯 transformer 注意力机制，无正则/分类器
  → Slash Command 跳过此阶段——必须用户手动触发

阶段 3：调用时（用户 /xxx 或 Agent 自主触发）
  → 从文件系统读取完整内容
  → 注入对话上下文
  → Skill 的辅助脚本按需执行

阶段 4：执行后
  → 完整指令留在上下文中
  → 不会重复读取
```

### 为什么这很重要

```text
传统方式（CLAUDE.md）：
  所有指令始终在上下文中 → 100个指令 = 持续消耗大量 tokens

Skill / Slash Command 方式：
  100个 → 启动时只消耗 ~5000 tokens
        → 使用哪个才加载哪个
        → 未使用的几乎零成本
```

---

## 最佳实践：以 speckit 为例

以 X-Dev 的 `speckit`（SDD 后端引擎）为例，拆解一个成熟 Skill 的内部运作方式。

### 目录结构

```text
speckit/
├── SKILL.md                    ← 入口：frontmatter + action 路由
├── USAGE.md                    ← 用户手册
├── actions/                    ← 各 action 的执行指令（编排层）
│   ├── specify.md              ← 生成功能规格说明书
│   ├── review-spec.md          ← 审阅规格（HITL 交互式）
│   ├── tech-guidance.md        ← 技术指导输入
│   ├── tech-design.md          ← 技术方案设计
│   ├── review-tech-design.md   ← 审阅技术方案
│   ├── dev.md                  ← 任务拆分 + 逐 Phase 开发
│   └── run.md                  ← 全流程串联
├── agents/                     ← SubAgent 定义
│   ├── codebase-analyzer.md
│   ├── codebase-locator.md
│   └── codebase-pattern-finder.md
├── prompts/                    ← 各步骤的 prompt 模板（13 个）
│   ├── specify_prompt.md
│   ├── plan_prompt.md
│   ├── phase_dev_prompt.md
│   └── ...
└── resources/                  ← 参考资料和模板
    ├── spec_template.md
    ├── contracts_standard.md
    └── subtasks_template.md
```

### SKILL.md 的三大块

speckit 的 SKILL.md 由三大块组成：**Frontmatter → Action 路由表 → 执行规则**。

```yaml
# ━━━ 第一块：Frontmatter（Agent 启动时加载，~50 tokens）━━━
---
name: speckit
description: "Spec Kit - SDD 后端引擎，规格驱动开发全流程。
             支持 specify / review-spec / tech-guidance /
             tech-design / review-tech-design / dev / run。"
argument-hint: "<action> [参数]"
---

# ━━━ 第二块：Action 路由表 ━━━
# 从 $ARGUMENTS 中提取第一个词作为 action，路由到对应文件：
#
#   action = "specify"            → actions/specify.md
#   action = "review-spec"        → actions/review-spec.md
#   action = "dev"                → actions/dev.md
#   action = "run"                → actions/run.md（全流程串联）
#   ...

# ━━━ 第三块：执行规则 ━━━
# 1. 读取 actions/{action}.md 作为当前任务的执行指令
# 2. 所有 <skill_dir> 指向本 SKILL.md 所在目录
# 3. 剩余参数传递给 action 的执行逻辑
```

要点：
- `description` 要写清 Skill 做什么 + 支持哪些 action——这是 Agent 自动召回的唯一依据
- SKILL.md **本身不包含业务逻辑**，只做路由和规则定义

### 从用户输入到执行完成：全链路图

```text
用户: /speckit dev my-feature
         │
         ▼
┌─────────────────────────────────────────────────────────┐
│  SKILL.md（路由层）                                       │
│  解析 action="dev", 参数="my-feature"                     │
│  → 读取 actions/dev.md                                   │
└─────────────────┬───────────────────────────────────────┘
                  ▼
┌─────────────────────────────────────────────────────────┐
│  actions/dev.md（编排层）                                  │
│                                                          │
│  Step 0: 读取 prompts/resolve_feature_name.md            │
│          → 解析出 FEATURE_NAME、定位 specs 目录             │
│                                                          │
│  Step 1: task-split 任务拆分                               │
│          ├─ 读取 prompts/task_split_prompt.md             │
│          ├─ 读取 resources/subtasks_template.md           │
│          ├─ 读取已有的 spec.md + research.md + 5个技术方案   │
│          └─ 产出: subtasks_1.md, subtasks_2.md, ...       │
│                                                          │
│  Step 2: phase-dev 逐 Phase 串行开发                       │
│          ├─ 读取 prompts/phase_dev_prompt.md              │
│          ├─ 每个 Phase 启动一个 SubAgent 执行               │
│          │   └─ SubAgent(agents/codebase-analyzer.md)     │
│          │       分析代码库 → 实现代码 → commit + push       │
│          └─ Phase 1 完成 → Phase 2 → ... → 全部完成        │
└─────────────────────────────────────────────────────────┘
```

### 各目录的职责

| 目录 | 职责 | 被谁调用 |
|------|------|----------|
| `SKILL.md` | 入口路由：解析 action → 分派到 actions/ | Agent 框架 |
| `actions/` | 编排层：定义每个 action 的执行步骤和资源引用 | SKILL.md 路由 |
| `prompts/` | 指令模板：每个步骤的具体 prompt（如任务拆分、代码生成） | actions/*.md 引用 |
| `agents/` | SubAgent 定义：独立上下文的专精 Agent（如代码分析器） | actions/*.md 启动 |
| `resources/` | 参考资料：模板、标准、约定（如 spec 模板、接口标准） | prompts/*.md 引用 |

调用链：`SKILL.md` → `actions/*.md` → `prompts/*.md` + `agents/*.md` + `resources/*.md`

### description 怎么写

```yaml
# Bad: 太模糊，Agent 不知道何时触发
description: "一个有用的工具"

# Good: 明确触发条件和能力范围
description: "Spec Kit - SDD 后端引擎，规格驱动开发全流程。支持 specify / review-spec /
             tech-guidance / tech-design / review-tech-design / dev / run（全流程）。"

# Good: 列出触发关键词
description: "当用户要求创建新的 API 接口、需要从 IDL 生成代码、
             或提到 thrift/hz 代码生成时使用"
```

### `disable-model-invocation` 的使用

对于有副作用的 Skill（部署、发送消息、清理数据等），设置 `disable-model-invocation: true` 防止 Agent 自动触发。此时 Skill 退化为"增强版 Slash Command"——只能手动 `/xxx` 调用，但保留目录结构的优势。

---

## Good Case vs Bad Case

### Good Case

- speckit 用 action 路由模式支持 7 个子命令，单个 SKILL.md 只做路由，业务逻辑分散到 `actions/*.md`
- `description` 列出所有支持的 action，Agent 看到"写个 spec"就自动召回
- 部署操作设为 `disable-model-invocation: true`，防止 Agent 误触发
- 简单的"跑一下 lint"用 Slash Command，不需要 Agent 自动召回

### Bad Case

| 错误 | 后果 |
|------|------|
| 把 Skill 的全部指令写进 CLAUDE.md | 上下文始终被占满，其他任务受影响 |
| description 不写或太短 | Agent 永远不会自主触发，Skill 等于白做 |
| 该用 Slash Command 的用了 Skill | Agent 可能自动触发有副作用的操作 |
| 该用 Skill 的用了 Slash Command | 失去 Agent 自动召回能力和辅助文件支持 |
| 一个 SKILL.md 里塞几百行逻辑 | 应该用 action 路由 + 分文件，保持可维护性 |
| Skill 之间有隐式依赖 | 单独调用时缺少前置上下文 |

---

## Skill 的共享和分发

Skill 写好之后如何让更多人用到？业界已形成一批 Skill Hub 平台，分为公域和字节域两类。

### Skill Hub 一览

| 平台 | 类型 | 地址 | 特点 |
|------|------|------|------|
| **skills.sh** (Vercel) | 公域 | [skills.sh](https://skills.sh) | Vercel 出品，开源标准（agentskills.io），`npx skills add` 一键安装，87K+ skills |
| **SkillsMP** | 公域 | [skillsmp.com](https://skillsmp.com) | 社区聚合，从 GitHub 自动同步，425K+ skills，搜索发现为主 |
| **LobeHub Skills** | 公域 | [lobehub.com/skills](https://lobehub.com/skills) | 产品化体验好，增长快，适合非技术用户浏览 |
| **SkillHub** | 公域 | [skillhub.club](https://www.skillhub.club) | AI 评分机制，7K+ skills，质量筛选 |
| **Agent Skills Hub** | 公域 | [agentskillshub.dev](https://agentskillshub.dev) | 安全优先，自动代码扫描，信任评级 |
| **AI 扩展能力市场** | 字节域 | [skills.bytedance.net](https://skills.bytedance.net) | 字节统一 Skill 市场，支持 Web 上传 + CLI 安装，集成 Aime/Trae CN 一键试用 |
| **AI Skills Labs** | 字节域 | [ai-skills.bytedance.net](https://ai-skills.bytedance.net) | 字节 Skill/MCP/Plugin 三合一平台，CLI 管理，支持 Cursor/Trae 等多 Agent |

### 公域 Top 2：skills.sh & SkillsMP

#### skills.sh (Vercel)

**安装 Skill：**

```bash
# 从 GitHub 仓库安装
npx skills add vercel-labs/agent-skills --skill frontend-design

# 指定 Agent 类型
npx skills add owner/repo --skill my-skill -a claude-code

# 全局安装（所有项目可用）
npx skills add owner/repo --skill my-skill -g
```

**发布 Skill：**

将包含 `SKILL.md` 的目录推到 GitHub 仓库，skills.sh 会自动索引。也可在 skills.sh 网站提交仓库链接。

#### SkillsMP

SkillsMP 是搜索发现平台，不提供安装 CLI。在 [skillsmp.com](https://skillsmp.com) 搜索后，跳转到 GitHub 仓库，手动 clone 并复制到 `.claude/skills/` 或 `.codex/skills/`。

### 字节域：AI 扩展能力市场 & AI Skills Labs

#### AI 扩展能力市场 (skills.bytedance.net)

**安装 CLI：**

```bash
npm i skills -g --registry=https://bnpm.byted.org
```

**安装 Skill：**

```bash
# 从市场安装
npx -y skills@latest add <group> --skill <skill-name>

# 示例：安装 bytedance-log skill
npx -y skills@latest add skills.byted.org/chenyunpeng.1024.skills --skill bytedance-log

# 从 GitHub 安装
npx -y skills@latest add github.com/anthropics/skills --skill doc-coauthoring

# 安装整个 collection
npx -y skills@latest collection add <collection-id>
```

**发布 Skill：**

```bash
# CLI 发布（支持文件夹、zip、URL、git 地址）
npx -y skills@latest publish <skill-spec> --group

# Web 发布：skills.bytedance.net 首页 → "+ 创建 Skill"
# 支持手动创建、上传 ZIP、Git 仓库导入
```

**其他命令：**

```bash
npx -y skills@latest find <keyword>   # 搜索
npx -y skills@latest list             # 列出已安装
npx -y skills@latest remove --skill <name>  # 卸载
npx -y skills@latest login            # 登录认证
```

#### AI Skills Labs (ai-skills.bytedance.net)

三合一平台（Skill + MCP Server + Plugin），CLI 名称为 `ai`（包名 `@tiktok-fe/ai`），支持 **47 个 Coding Agent**（包括 Claude Code、Codex、Cursor、Trae、Trae CN、Coco、Gemini CLI 等）。

**安装 CLI：**

```bash
# 自动安装脚本
curl -fsSL https://ai-skills.bytedance.net/scripts/cli.sh | bash

# 或手动安装
pnpm add @tiktok-fe/ai -g && ai setup
```

**安装 Skill：**

```bash
# 从市场安装
ai skill add <owner>/<repo> --skill <name>

# 指定 Agent
ai skill add <owner>/<repo> --skill <name> -a claude-code

# 直接调用（无需全局安装）
npx @tiktok-fe/ai skill add <owner>/<repo> --skill <name>
```

**发布 Skill：**

```bash
# 初始化骨架
ai skill init

# 发布到市场
ai skill publish
```

**其他命令：**

```bash
ai skill list           # 列出已安装
ai skill remove <name>  # 卸载
ai mcp add <name>       # 安装 MCP Server
ai plugin add <name>    # 安装 Plugin
ai install              # 从 ai-package.json 批量安装
ai agents               # 列出支持的 47 个 Agent
```

> 完整文档：[ai-skills.bytedance.net/docs](https://ai-skills.bytedance.net/docs)

---

[上一节：SubAgent](021c-subagent.md) | [返回上级：上下文控制](021-context-control.md)
