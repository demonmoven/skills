---
name: superpowers
description: "Superpowers - 通用 Agent 工作流引擎，13 个组合式 action 覆盖 brainstorm/plan/execute/review/debug 全流程。Upstream: obra/superpowers v5.0.7（MIT）。"
argument-hint: "<action> [参数]"
---

# superpowers

Superpowers — 通用 Agent 工作流引擎。从 [obra/superpowers](https://github.com/obra/superpowers) v5.0.7 (MIT) 迁移并适配 devclaw_skills_center。

> Upstream 的 14 个独立 skill 在迁移后被组织为单一 skill 容器 + 13 个 actions（去掉了 `using-git-worktrees` —— 详见 ./UPSTREAM.md）。

## 用户输入

```text
$ARGUMENTS
```

## 参数格式

`$ARGUMENTS` 格式：`<action> [参数]`

第一个词为 `action`，决定执行哪个 action 文件。

| action | 说明 |
|--------|------|
| `brainstorm` | 设计前的需求澄清 + spec 生成（强制 design-first，含 HARD-GATE） |
| `plan` | 把 spec 拆成 task 级 plan（2–5 分钟粒度的 TDD checkbox） |
| `execute` | subagent 派发执行（**推荐**）：fresh subagent per task + 两阶段审查 |
| `execute-batch` | 备选执行模式：批量执行 + 人工 checkpoint（无 subagent 时用） |
| `tdd` | RED-GREEN-REFACTOR 流程的强约束 |
| `debug` | 4 阶段根因调试（含 root-cause-tracing/defense-in-depth 等技巧） |
| `verify` | 完成宣称前的必经验证 |
| `review` | 任务间发起 code review（dispatch superpowers:code-reviewer agent） |
| `handle-review` | 接收 code review 反馈并处理 |
| `parallel` | 并行 subagent 编排 |
| `finish` | 结束 feature 分支（merge / PR / discard / 保留四选一） |
| `intro` | 整套 superpowers 系统的入门导览 |
| `write-skill` | 编写新 skill 的方法论（含 eval/pressure 测试） |
| `help` | 显示帮助信息 |

## 执行方式

> **`<skill_dir>` 约定**：以下路径中 `<skill_dir>` 指代本 SKILL.md 所在目录的绝对路径（即 `marketplace/plugins/xdev/skills/superpowers/`）。

### 1. 解析 action

从 `$ARGUMENTS` 中提取第一个词作为 `action`，剩余部分为该 action 的参数（可选）。

### 2. 路由分派

根据 `action` 的值，读取对应的 action 文件并执行：

```
action = "brainstorm"      → 读取并执行 <skill_dir>/actions/brainstorm.md
action = "plan"            → 读取并执行 <skill_dir>/actions/plan.md
action = "execute"         → 读取并执行 <skill_dir>/actions/execute.md
action = "execute-batch"   → 读取并执行 <skill_dir>/actions/execute-batch.md
action = "tdd"             → 读取并执行 <skill_dir>/actions/tdd.md
action = "debug"           → 读取并执行 <skill_dir>/actions/debug.md
action = "verify"          → 读取并执行 <skill_dir>/actions/verify.md
action = "review"          → 读取并执行 <skill_dir>/actions/review.md
action = "handle-review"   → 读取并执行 <skill_dir>/actions/handle-review.md
action = "parallel"        → 读取并执行 <skill_dir>/actions/parallel.md
action = "finish"          → 读取并执行 <skill_dir>/actions/finish.md
action = "intro"           → 读取并执行 <skill_dir>/actions/intro.md
action = "write-skill"     → 读取并执行 <skill_dir>/actions/write-skill.md
其他                       → 输出下方帮助信息
```

### 3. 执行 action

读取对应的 `actions/{action}.md` 文件后：

1. 将文件内容作为当前任务的执行指令
2. 文件中所有 `<skill_dir>` 均指向本 SKILL.md 所在目录
3. 文件中可能引用 `<skill_dir>/prompts/...` 或 `<skill_dir>/resources/...` 下的辅助文件，按需读取
4. 文件中跨 action 的引用（如 `superpowers execute`、`superpowers tdd`）表示"调用方应使用对应的 action"，由调用方在合适的时机用 `/superpowers <name>` 触发

## 切分支约定（与 speckit / exec-plan 对齐）

speckit 和 exec-plan 都没有"专门的切分支 skill"。superpowers **同样不再** 提供独立的切分支 skill —— 上游的 `using-git-worktrees` 已经被移除。

**调用方必须在执行任何 action 之前** 手动建立 feature 分支：

```bash
git checkout -b {YYYYMMDDHHmmss}-{slug}
# 例如:
git checkout -b 20260408143000-user-login-optimization
```

命名规范与 `speckit` / `exec-plan` 一致：
- `{YYYYMMDDHHmmss}` —— 时间戳
- `{slug}` —— 短小的 kebab-case 描述（推荐 ≤ 4 个英文单词）

**续接已有 feature**：直接 `git checkout {existing-branch}`，再调用对应 action。

## 产物路径约定

| 类型 | 路径 |
|------|------|
| Spec / Design Doc | `docs/xdev/superpowers/specs/YYYY-MM-DD-<topic>-design.md` |
| Implementation Plan | `docs/xdev/superpowers/plans/YYYY-MM-DD-<feature-name>.md` |

> 这两个根目录与 `docs/xdev/speckit/{FEATURE_NAME}/`、`docs/xdev/exec-plan/{FEATURE_NAME}/` 并列在 `docs/xdev/` 之下，三者互不重叠。`docs/xdev/superpowers/` 路径相对于使用方项目仓库的根目录。

## 与上游的关键差异

详见 ./UPSTREAM.md。简要说明：

1. **形态**：上游是 14 个独立 skill；本地是单 skill 容器 + 13 个 actions
2. **结构**：`skills/{name}/SKILL.md` → `actions/{name}.md` + `prompts/...` + `resources/...`
3. **隔离方式**：上游用 `git worktree`；本地完全去掉 worktree，由调用方手动 `git checkout -b`
4. **产物路径**：本地与上游都使用 `docs/` 前缀，但中间增加 `xdev` namespace 隔离 → `docs/xdev/superpowers/{specs,plans}/`（详见 UPSTREAM.md 第 4 节）
5. **触发**：上游用 SessionStart hook 自动注入；本地用显式 `/superpowers <action>`
6. **剥离**：commands/、hooks/、tests/、docs/、Node 脚本依赖、各 harness 配置等

## 帮助信息（action 未匹配时输出）

```
superpowers - 通用 Agent 工作流引擎

用法：/superpowers <action> [参数]

可用 actions：
  brainstorm      需求澄清 + spec 生成（强制 design-first）
  plan            spec → 2-5 分钟粒度的 task plan
  execute         subagent 派发执行（推荐）
  execute-batch   备选：批量执行 + 人工 checkpoint
  tdd             RED-GREEN-REFACTOR 流程
  debug           4 阶段根因调试
  verify          完成前验证
  review          发起 code review
  handle-review   接收 code review 反馈
  parallel        并行 subagent 编排
  finish          结束 feature 分支
  intro           整套系统入门导览
  write-skill     编写新 skill 的方法论
  help            显示此帮助

示例：
  /superpowers brainstorm 用户登录流程优化
  /superpowers plan
  /superpowers execute
  /superpowers debug
  /superpowers finish

切分支：
  在调用任何 action 之前，先手动建立 feature 分支：
  git checkout -b 20260408143000-user-login-optimization
```
