# superpowers 用户使用手册

本文档面向 superpowers skill 的使用者。

> Upstream: [obra/superpowers](https://github.com/obra/superpowers) v5.0.7（MIT）
> 详细的本地修改清单见 [./UPSTREAM.md](./UPSTREAM.md)

---

## 一句话理解 superpowers

superpowers 是一个**通用 Agent 工作流引擎**，由 13 个组合式 action 组成，覆盖从设计 → 计划 → 执行 → 验证 → 收尾的完整开发闭环。它的核心价值在于"防止 Agent 一上来就闷头写代码"——每个 action 都是经过 eval 调过的"行为塑形指令"。

---

## 快速开始

### 前置准备

1. **必须先建立 feature 分支**（这是与 upstream 最大的不同）：

   ```bash
   cd /path/to/your-repo
   git checkout -b 20260408143000-user-login-optimization
   ```

   命名规则：`{YYYYMMDDHHmmss}-{kebab-slug}`，与 `speckit` / `exec-plan` 一致。

2. 启动 Claude Code：
   ```bash
   claude
   ```

### 完整流程示例

```
/superpowers brainstorm 用户登录流程优化
   ↓ （Agent 通过对话澄清需求 + 生成 spec.md）
/superpowers plan
   ↓ （Agent 把 spec 拆成 2-5 分钟粒度的 task plan）
/superpowers execute
   ↓ （Agent 派发 subagent 逐 task 实现 + 两阶段审查）
/superpowers finish
   ↓ （选择 merge / PR / discard / 保留分支）
```

---

## 参数说明

`/superpowers <action> [参数]`

| action | 说明 | 适用场景 |
|--------|------|---------|
| `brainstorm` | 需求澄清 + spec 生成 | 任何"创造性工作"前都强制使用 |
| `plan` | spec → task 级 plan | 拿到 spec 后、动手前 |
| `execute` | subagent 派发执行 | 有 plan 后的主流执行模式（**推荐**） |
| `execute-batch` | 批量执行 + 人工 checkpoint | 无 subagent 支持时的备选 |
| `tdd` | RED-GREEN-REFACTOR | 写测试和实现时 |
| `debug` | 4 阶段根因调试 | 遇到 bug 时 |
| `verify` | 完成前验证 | 宣称完成前 |
| `review` | 发起 code review | 任务间或合并前 |
| `handle-review` | 接收 review 反馈 | 收到 review 意见后 |
| `parallel` | 并行 subagent 编排 | 多个独立任务可并行时 |
| `finish` | 结束 feature 分支 | 全部 task 完成后 |
| `intro` | 入门导览 | 新用户或新会话 |
| `write-skill` | 编写新 skill 方法论 | 想给 superpowers 加新能力时 |

---

## 场景化使用指南

### 场景 1：完整新 feature 开发（最常用）

```bash
# 第一步：本地切分支（**手动**，跟 speckit/exec-plan 一致）
git checkout -b 20260408143000-add-2fa

# 第二步：进入 Claude Code，跑全流程
claude
```

```
/superpowers brainstorm 加上手机短信二次验证
```

Agent 会通过对话澄清需求（一次问一个问题），讨论 2-3 个备选方案，最后生成 `docs/xdev/superpowers/specs/2026-04-08-add-2fa-design.md` 并要求你 review。

```
/superpowers plan
```

Agent 把 spec 拆成 task 级 plan，存到 `docs/xdev/superpowers/plans/2026-04-08-add-2fa.md`。每个 task 都是 2–5 分钟的具体步骤（写测试 → 看测试失败 → 写最小实现 → 看测试通过 → commit）。

```
/superpowers execute
```

Agent 派发 fresh subagent 逐 task 实现，每个 task 完成后做 spec 合规审查 + 代码质量审查。

```
/superpowers finish
```

呈现 4 个选项：merge / PR / 保留分支 / 丢弃。

### 场景 2：只用 debug

正在 debug 一个棘手的 bug？

```
/superpowers debug
```

Agent 进入 4 阶段根因法：现象收集 → 假设排序 → 假设验证 → 修复 + 验证。期间会用到 `resources/debug-root-cause-tracing.md`、`resources/debug-defense-in-depth.md` 等参考文档。

### 场景 3：单独 review 一段已完成的代码

```
/superpowers review
```

Agent 会派发 `superpowers:code-reviewer` agent（定义在 `agents/code-reviewer.md`）来 review 你的当前 commit 范围。

### 场景 4：续接已有 feature

切回原分支，再调用对应 action：

```bash
git checkout 20260408143000-add-2fa
claude
```

```
/superpowers execute
```

Agent 会从 `docs/xdev/superpowers/plans/2026-04-08-add-2fa.md` 中找到未完成的 task 继续执行。

### 场景 5：写新 skill

```
/superpowers write-skill
```

引导你按照 superpowers 的"skill = TDD applied to documentation"哲学写一个新 skill。

---

## 与 speckit / exec-plan 的差异和选用建议

| 维度 | speckit | exec-plan | superpowers |
|------|---------|-----------|-------------|
| 核心定位 | 字节内 SDD 后端开发流程 | 复杂任务的执行计划编写与执行 | 通用 Agent 工作流引擎（强 TDD / subagent 派发） |
| 文档风格 | 多文档分层（spec/prd/tech-design/dev） | 单文件 13 章节自包含 ExecPlan | 任务级 checkbox（2–5 min 粒度） |
| 子流程 | actions: specify/review-spec/tech-design/dev/run | 单 SKILL.md 含 4 操作模式 | actions: brainstorm/plan/execute/.../write-skill |
| 强约束 | spec → tech-design → 任务拆分 → 编码 | 各章节必填（Purpose/Progress/Decision Log 等） | TDD 强制 / subagent 派发 / 两阶段审查 |
| 飞书 MCP | ✅ 支持飞书文档链接 | ✅ 支持飞书文档链接 | ❌ 不支持 |
| 产物根 | `docs/xdev/speckit/{FEATURE_NAME}/` | `docs/xdev/exec-plan/{FEATURE_NAME}/` | `docs/xdev/superpowers/specs/`、`docs/xdev/superpowers/plans/` |

**选择建议**：

- **后端 API/功能开发** → speckit（字节定制流程，飞书 MCP 集成最好）
- **跨模块重构 / 架构变更** → exec-plan（自包含 ExecPlan 文档最适合长任务）
- **通用开发流 / 强调 TDD 和 subagent 派发** → superpowers
- **不确定** → 先看一眼 [./UPSTREAM.md](./UPSTREAM.md) 的差异说明，再决定

---

## 切分支约定（重要差异）

> **本地版本与 upstream 最大的差异**

upstream `obra/superpowers` 用 `git worktree` 隔离工作目录。**本地版本完全去掉了 worktree 概念**，遵循 speckit / exec-plan 的做法：

```bash
# 创建 feature 分支（手动，由你做）
git checkout -b 20260408143000-add-2fa
```

不再有任何"切分支 skill" —— 切分支只是一行 git 命令。

---

## 产物路径

```
<项目仓库根>/
├── superpowers/                       # superpowers 的产物根
│   ├── specs/                         # spec / design docs
│   │   └── 2026-04-08-add-2fa-design.md
│   └── plans/                         # implementation plans
│       └── 2026-04-08-add-2fa.md
├── speckit/                           # speckit 的产物根（与本 skill 并列）
│   └── 20260408143000-xxx/
└── exec-plan/                         # exec-plan 的产物根（与本 skill 并列）
    └── 20260408143000-xxx/
```

---

## 常见问题

### Q: 流程中断了怎么办？
切回 feature 分支，重新调对应 action。Agent 会从 plan 文件中找到未完成的 task 继续。

### Q: 必须用 subagent 吗？
不一定。`execute` 是推荐路径（subagent driven），但如果当前 harness 不支持 subagent（如某些精简版的 Claude Code），可以用 `execute-batch`（在主 session 内批量执行）。

### Q: 为什么不再有 `using-git-worktrees`？
因为 speckit / exec-plan 都没有专门的"切分支 skill"。本地版本对齐 follow 这个做法 —— 切分支由调用方手动一行 `git checkout -b ...` 完成。

### Q: visual companion（浏览器组件）能用吗？
不能。它依赖一个本地 Node 服务（`brainstorming/scripts/server.cjs`），与本仓库零运行时依赖原则冲突，已剥离。如需使用，请到 upstream 仓库自行启用。

### Q: 如何同步上游的新版本？
不要直接覆盖本目录！见 [./UPSTREAM.md](./UPSTREAM.md) 的"同步上游策略"。

---

## Action 速查表

| 我想... | 命令 |
|---------|------|
| 完整流程（新建 feature） | 先 `git checkout -b ...`，然后 `/superpowers brainstorm <需求>` |
| 续接已有 feature | `git checkout <feature-branch>` 然后 `/superpowers execute` |
| 只做需求澄清 | `/superpowers brainstorm <需求>` |
| 只做计划拆分 | `/superpowers plan` |
| 只做执行 | `/superpowers execute` |
| 单独 debug | `/superpowers debug` |
| 单独 review | `/superpowers review` |
| 收尾 | `/superpowers finish` |
