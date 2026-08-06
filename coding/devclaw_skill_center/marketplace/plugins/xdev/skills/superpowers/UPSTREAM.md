# Upstream Tracking

本目录是 [obra/superpowers](https://github.com/obra/superpowers) v5.0.7 的二次发行版，针对 `devclaw_skills_center` 的 plugin / skill 体系做了适配性修改。本文档记录 upstream 信息和所有本地改动。

## 来源

| 项 | 值 |
|----|----|
| 仓库 | https://github.com/obra/superpowers |
| 上游版本 | v5.0.7（`.claude-plugin/plugin.json` 声明） |
| 抓取时间 | 2026-04-08 |
| 抓取分支 | main |
| 抓取方式 | `git clone --depth 1 https://github.com/obra/superpowers.git` |
| 本机克隆位置 | `~/workspace/src/github/obra/superpowers/` |
| License | MIT（见 [./LICENSE](./LICENSE)） |
| 上游作者 | Jesse Vincent (jesse@fsck.com) / [Prime Radiant](https://primeradiant.com) |

## 本地修改清单

### 1. 形态：单 skill 容器（speckit 风格）

| 维度 | upstream | 本地 |
|------|----------|------|
| 入口形态 | 14 个独立 skill (`skills/{name}/SKILL.md`) | 单 skill `superpowers`，路由到 13 个 actions |
| 触发 | SessionStart hook 自动注入引导文本 | 显式 `/superpowers <action>` |
| 子 action 数 | 14 | 13（删除了 `using-git-worktrees`，详见第 2 条） |

### 2. 工作隔离：完全去掉 worktree

speckit / exec-plan 都没有"专门的切分支 skill" —— 切分支只是一行 `git checkout -b ...` 指令。本地版本与之对齐：

| 维度 | upstream | 本地 |
|------|----------|------|
| 切分支 skill | `skills/using-git-worktrees/SKILL.md` 用 `git worktree add` 创建独立工作目录 | **整个 `using-git-worktrees/` 目录删除** |
| 切分支方式 | `git worktree add ".worktrees/<name>" -b <name>` | 调用方在调用 action 之前手动 `git checkout -b {YYYYMMDDHHmmss}-{slug}` |
| 命名规则 | `<base-name>` 由 brainstorming 决定 | `{YYYYMMDDHHmmss}-{kebab-slug}` 与 speckit / exec-plan 一致 |

### 3. 目录结构：actions / prompts / resources / agents 平铺（speckit 风格）

upstream 用 `skills/{name}/` 子目录，每个子目录含一个 `SKILL.md` 加若干辅助文件。本地按 speckit 的形态重组：

```
superpowers/
├── SKILL.md                                       # 路由入口（本地新增）
├── USAGE.md                                       # 使用手册（本地新增）
├── UPSTREAM.md                                    # 本文件（本地新增）
├── LICENSE                                        # 来自 upstream LICENSE
├── actions/                                       # 13 个 action 入口
│   ├── brainstorm.md                              # ← brainstorming/SKILL.md
│   ├── plan.md                                    # ← writing-plans/SKILL.md
│   ├── execute.md                                 # ← subagent-driven-development/SKILL.md
│   ├── execute-batch.md                           # ← executing-plans/SKILL.md
│   ├── tdd.md                                     # ← test-driven-development/SKILL.md
│   ├── debug.md                                   # ← systematic-debugging/SKILL.md
│   ├── verify.md                                  # ← verification-before-completion/SKILL.md
│   ├── review.md                                  # ← requesting-code-review/SKILL.md
│   ├── handle-review.md                           # ← receiving-code-review/SKILL.md
│   ├── parallel.md                                # ← dispatching-parallel-agents/SKILL.md
│   ├── finish.md                                  # ← finishing-a-development-branch/SKILL.md
│   ├── intro.md                                   # ← using-superpowers/SKILL.md
│   └── write-skill.md                             # ← writing-skills/SKILL.md
├── prompts/                                       # 被 action dispatch 的 prompt 模板
│   ├── brainstorm-spec-document-reviewer.md       # ← brainstorming/spec-document-reviewer-prompt.md
│   ├── plan-document-reviewer.md                  # ← writing-plans/plan-document-reviewer-prompt.md
│   ├── execute-implementer.md                     # ← subagent-driven-development/implementer-prompt.md
│   ├── execute-spec-reviewer.md                   # ← subagent-driven-development/spec-reviewer-prompt.md
│   ├── execute-code-quality-reviewer.md           # ← subagent-driven-development/code-quality-reviewer-prompt.md
│   └── code-review-request-template.md            # ← requesting-code-review/code-reviewer.md
├── resources/                                     # 参考文档（reference docs）
│   ├── brainstorm-visual-companion.md             # ← brainstorming/visual-companion.md（默认禁用）
│   ├── tdd-anti-patterns.md                       # ← test-driven-development/testing-anti-patterns.md
│   ├── debug-condition-based-waiting.md           # ← systematic-debugging/condition-based-waiting.md
│   ├── debug-defense-in-depth.md                  # ← systematic-debugging/defense-in-depth.md
│   ├── debug-root-cause-tracing.md                # ← systematic-debugging/root-cause-tracing.md
│   ├── intro-codex-tools.md                       # ← using-superpowers/references/codex-tools.md
│   ├── intro-copilot-tools.md                     # ← using-superpowers/references/copilot-tools.md
│   ├── intro-gemini-tools.md                      # ← using-superpowers/references/gemini-tools.md
│   ├── write-skill-anthropic-best-practices.md    # ← writing-skills/anthropic-best-practices.md
│   ├── write-skill-persuasion-principles.md       # ← writing-skills/persuasion-principles.md
│   └── write-skill-testing-with-subagents.md      # ← writing-skills/testing-skills-with-subagents.md
└── agents/                                        # subagent 角色定义
    └── code-reviewer.md                           # ← upstream agents/code-reviewer.md
```

### 4. 产物路径：加 `docs/xdev/` 前缀

| 类型 | upstream | 本地 |
|------|----------|------|
| Spec | `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md` | `docs/xdev/superpowers/specs/YYYY-MM-DD-<topic>-design.md` |
| Plan | `docs/superpowers/plans/YYYY-MM-DD-<feature-name>.md` | `docs/xdev/superpowers/plans/YYYY-MM-DD-<feature-name>.md` |

> 与上游 `docs/` 前缀方向一致，但中间增加 `xdev` namespace 隔离，避免与业务仓库其他工具的 `docs/` 子目录冲突。

`docs/xdev/superpowers/specs/`、`docs/xdev/superpowers/plans/` 与 `docs/xdev/speckit/{FEATURE_NAME}/`、`docs/xdev/exec-plan/{FEATURE_NAME}/` 并列在使用方仓库的 `docs/xdev/` 下，互不重叠。

> 历史注：本仓库初次迁移时曾去掉 `docs/` 前缀（直接 `superpowers/specs/`），后续为了与其他 skill 统一并避免污染业务仓库根目录，又恢复并加上了 `xdev` namespace。

### 5. 触发：去 hook

| 维度 | upstream | 本地 |
|------|----------|------|
| 触发机制 | `hooks/hooks.json` + `hooks/session-start` 在 SessionStart 自动注入引导文本 | 剥离 hooks，改为显式 `/superpowers <action>` 入口 |

### 6. 剥离的内容

#### 仓库基础设施层（整体剥离）

- `.codex/` `.cursor-plugin/` `.opencode/` `.claude-plugin/` `.version-bump.json` `gemini-extension.json` `GEMINI.md`：各 harness 接入清单 / plugin manifest
- `commands/`：upstream 已 deprecated（只有 brainstorm/write-plan/execute-plan 三个文件，每个内容都是"请改用 skill"）
- `hooks/`：见第 5 条
- `tests/` `scripts/` `docs/` `.github/` `.gitattributes` `.gitignore` `package.json` `README.md` `CHANGELOG.md` `RELEASE-NOTES.md` `CODE_OF_CONDUCT.md` `CLAUDE.md` `AGENTS.md`：仓库基础设施

#### skill 子目录里的运行时依赖（剥离）

- `skills/brainstorming/scripts/`：依赖本地 Node 服务的 visual companion 服务端
- `skills/systematic-debugging/find-polluter.sh`：shell 脚本
- `skills/systematic-debugging/condition-based-waiting-example.ts`：独立 TypeScript 代码示例
- `skills/systematic-debugging/CREATION-LOG.md`：项目元信息
- `skills/systematic-debugging/test-pressure-1.md` ~ `test-pressure-3.md`、`test-academic.md`：eval 文档
- `skills/writing-skills/render-graphs.js`：Node 脚本
- `skills/writing-skills/graphviz-conventions.dot`：独立工具配置
- `skills/writing-skills/examples/`：示例目录（含 CLAUDE_MD_TESTING.md 等）

### 7. 修改的内容清单

行为塑形内容（Red Flags、rationalization 表、"human partner" 措辞、TDD 强约束等）**未修改**，仅调整工程胶水：

#### 路径替换（精确字符串）

- `docs/superpowers/specs/`（上游） → `docs/xdev/superpowers/specs/`（本地，加 xdev namespace）
- `docs/superpowers/plans/`（上游） → `docs/xdev/superpowers/plans/`（本地，加 xdev namespace）
- 影响文件：`SKILL.md`、`USAGE.md`、`actions/brainstorm.md`、`actions/plan.md`、`actions/execute.md`、`actions/review.md`、`prompts/brainstorm-spec-document-reviewer.md`

#### worktree → branch 改写

- `actions/finish.md`（原 `finishing-a-development-branch/SKILL.md`）：
  - "Cleanup Worktree" 步骤改写为 "Return to Base Branch"
  - 移除所有 `git worktree list` / `git worktree remove` 命令
  - "Pairs with: using-git-worktrees" 改写为"调用方手动 `git checkout -b`"的说明
- `actions/plan.md`（原 `writing-plans/SKILL.md`）：`in a dedicated worktree` → `on a dedicated feature branch`
- `actions/execute.md`、`actions/execute-batch.md`：`superpowers:using-git-worktrees - REQUIRED: Set up isolated workspace` → `Pre-condition (REQUIRED): Caller must git checkout -b ...`
- `resources/intro-codex-tools.md`：`using-git-worktrees` 引用 → 通用描述（保留 codex 自身环境的 worktree 检测逻辑，因为这是 codex 平台的运行环境判断）

#### 跨 action 引用替换

upstream 用 `superpowers:{old-skill-name}` 的形式跨 skill 引用，本地全部改为 `superpowers {new-action-name}`：

| upstream | 本地 |
|----------|------|
| `superpowers:brainstorming` | `superpowers brainstorm` |
| `superpowers:writing-plans` | `superpowers plan` |
| `superpowers:subagent-driven-development` | `superpowers execute` |
| `superpowers:executing-plans` | `superpowers execute-batch` |
| `superpowers:test-driven-development` | `superpowers tdd` |
| `superpowers:systematic-debugging` | `superpowers debug` |
| `superpowers:verification-before-completion` | `superpowers verify` |
| `superpowers:requesting-code-review` | `superpowers review` |
| `superpowers:receiving-code-review` | `superpowers handle-review` |
| `superpowers:dispatching-parallel-agents` | `superpowers parallel` |
| `superpowers:finishing-a-development-branch` | `superpowers finish` |
| `superpowers:using-superpowers` | `superpowers intro` |
| `superpowers:writing-skills` | `superpowers write-skill` |
| `superpowers:code-reviewer` | **不变**（这是 agent 名引用，对应 `agents/code-reviewer.md`） |

#### 内部文件路径替换

upstream 在每个 `skills/{name}/SKILL.md` 中用相对路径 `./xxx-prompt.md` 引用本子目录的辅助文件。重组到 actions/prompts/resources 平铺后，所有相对路径改为绝对的 `<skill_dir>/prompts/...` 或 `<skill_dir>/resources/...`：

| upstream（相对 skills/{name}/） | 本地（相对 superpowers/） |
|------|------|
| `./implementer-prompt.md` | `<skill_dir>/prompts/execute-implementer.md` |
| `./spec-reviewer-prompt.md` | `<skill_dir>/prompts/execute-spec-reviewer.md` |
| `./code-quality-reviewer-prompt.md` | `<skill_dir>/prompts/execute-code-quality-reviewer.md` |
| `./spec-document-reviewer-prompt.md`（在 brainstorming） | `<skill_dir>/prompts/brainstorm-spec-document-reviewer.md` |
| `./plan-document-reviewer-prompt.md`（在 writing-plans） | `<skill_dir>/prompts/plan-document-reviewer.md` |
| `requesting-code-review/code-reviewer.md` 或 `code-reviewer.md`（review 模板） | `<skill_dir>/prompts/code-review-request-template.md` |

## 同步上游的策略

后续 upstream 发布新版本时，**禁止直接覆盖**本目录的文件——必须人工 review。流程：

1. **更新本机 upstream 副本**：
   ```bash
   cd ~/workspace/src/github/obra/superpowers
   git pull
   ```

2. **diff 对比**：用 `diff` 工具对比 upstream 的 14 个 skill 与本地 13 个 actions，找出 upstream 的真实变更（区分"行为塑形内容修订"和"工程胶水改动"）。

3. **逐项 review**：检查 upstream 的每一处变更是否影响本地适配点：
   - 是否新增了 worktree 相关的硬要求？→ 需要本地化为 branch 模式
   - 是否新增了 `docs/superpowers/...` 路径？→ 需要本地化为 `superpowers/...`
   - 是否新增了 hook 配置？→ 需要剥离
   - 是否新增了 Node 脚本依赖？→ 需要剥离
   - 是否新增了 skill 子目录文件？→ 需要按"同名 prefix"规则迁到 prompts/ 或 resources/
   - 是否变更了某个 SKILL.md 的核心 "behavior shaping" 内容？→ 直接采纳到对应的 actions/{name}.md

4. **更新本文件**：把新版本号、抓取时间、变更摘要写入"来源"和本节"修改清单"。

> 上游在 `CLAUDE.md` 中明确反对"compliance"式重写。本地的所有改动都是**工程胶水层面**的适配（路径、目录组织、触发方式、依赖剥离），未改任何"行为塑形"内容（Red Flags 表、rationalization lists、"human partner" 措辞、TDD 强约束等）。同步时也应严格保持这一底线。

## 与本仓库其它 skill 的关系

| skill | 定位 | 产物根 | 适用场景 |
|-------|------|--------|----------|
| `speckit` | 字节内 SDD 后端开发流程 | `docs/xdev/speckit/{FEATURE_NAME}/` | 后端 API/功能开发 |
| `exec-plan` | 复杂任务的执行计划编写与执行 | `docs/xdev/exec-plan/{FEATURE_NAME}/` | 跨模块重构 / 架构变更 |
| `superpowers` | 通用 Agent 工作流引擎（强 TDD / subagent 派发） | `docs/xdev/superpowers/specs/`、`docs/xdev/superpowers/plans/` | 通用开发流 / 强调 TDD 和 subagent |

三者职责互补，可以根据任务性质选用。

## 已知限制

1. **subagent 派发依赖**：`actions/execute.md`（subagent-driven-development）依赖 Claude Code 的 `Agent`/Task tool 派发能力。在 Trae 上是否可用尚未 e2e 验证，不能保证全功能可用。
2. **Visual Companion 不可用**：`resources/brainstorm-visual-companion.md` 描述的浏览器 UI 组件依赖一个 Node 服务（`brainstorming/scripts/server.cjs`），运行时依赖被剥离了，本地版本默认禁用。
3. **flying with 5.0.6/5.0.7 没有 CHANGELOG**：upstream `CHANGELOG.md` 截止 5.0.5（2026-03-17），但 `package.json` 声明的版本是 5.0.7。5.0.6/5.0.7 的具体变更只在 `RELEASE-NOTES.md` 内（57 KB），未在本次迁移中全文展开。本地以仓库 main 分支当前文件状态为准。
