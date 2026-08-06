---
name: openspec
description: "OpenSpec 工作流引擎 - 基于 spec-driven schema 的纯 prompt SDD 流程,proposal → specs → design → tasks → apply → archive,产物落 docs/xdev/openspec/。支持 propose / explore / apply / archive / run(propose 内置自动 init + L3 续接,archive 内置 delta merge,run 支持断点续传)。"
argument-hint: "<action> [参数]"
---

# openspec

OpenSpec 工作流引擎 — fluid 多 change 的规格驱动开发,纯 prompt 实现,不依赖任何外部 CLI。

> **设计哲学**:fluid not rigid · iterative not waterfall · easy not complex · built for brownfield · scalable from personal to enterprise.
>
> **工作模式**:本 skill 由 Claude Agent 直接执行整个工作流——读取本 skill 内置的 4 个 artifact 模板(`templates/*.md`),按照 `spec-driven` schema 编排顺序生成 `proposal → specs → design → tasks`,然后实施 `apply` 与 `archive`。**不依赖** `npx openspec` 或上游 `@fission-ai/openspec` CLI。

## 用户输入

```text
$ARGUMENTS
```

## 参数格式

`$ARGUMENTS` 格式:`<action> [参数]`

第一个词为 `action`,决定执行哪个 OpenSpec 步骤。后续部分为该 action 的参数(可选)。

| action | 参数 | 说明 | 类型 |
|--------|------|------|------|
| `propose` | `[需求描述或 change-name]` | 一步生成完整 change(proposal + specs + design + tasks);内置 L1 自动 init + L2 解析 change + L3 续接已存在 artifact ⭐ | 自动化(默认) |
| `explore` | `[topic]` | 探索性对话,不创建 artifact | 交互式 |
| `apply` | `[change-name]` | 按 tasks.md 实施代码(自动续接未勾选的 task) | 混合 |
| `archive` | `[change-name]` | 归档已完成的 change(内置 delta merge,自动 sync 主 specs) | 混合 |
| `run` | `[需求描述]` | 全流程串联(propose → review → apply → archive);**不带参数 = 断点续传**:列出活跃 change 让用户选续接 ⭐ | 混合 |

**参数说明**:

- `propose`:带参数 = 描述新需求或 change-name;不带参数 = 让用户输入
- `run`:带参数 = 描述新需求(全流程);不带参数 = 列出活跃 change 续接(断点续传)
- `apply` / `archive`:带参数 = 指定 change-name;不带参数 = 自动从对话上下文推断或交互选择
- `explore`:带参数 = topic;不带参数 = 进入自由对话

> **旧 actions 已合并**:`init` / `new` / `continue` / `ff` / `sync` / `verify` 已合并到 `propose` / `archive` / `run`。
> - 想要 init → 直接 `propose`(自动 mkdir 骨架)
> - 想要 new + 续接 → 直接 `propose`(L2/L3 检测)
> - 想要 ff(批量生成 artifact) → 直接 `propose`(Step 6 循环就是批量)
> - 想要 sync(delta merge) → 直接 `archive`(内嵌)
> - 想要中断后续接 → 直接 `run`(不带参数,断点续传)
> - 想要 verify(对照 spec 检查代码) → 在对话里说"帮我对照 spec 检查代码"

---

## Schema 编排(默认 spec-driven)

本 skill 内置 **spec-driven** schema,按以下依赖图生成 artifacts:

```
proposal           (no deps)         → proposal.md
   ↓
   ├─→ specs       (requires: proposal) → specs/<capability>/spec.md
   ├─→ design      (requires: proposal, optional) → design.md
   ↓
tasks              (requires: specs, design) → tasks.md
   ↓
apply              (requires: tasks)         → 实施代码 + 勾选 tasks.md
```

每个 artifact 的"生成指令"集中在 `prompts/artifact_instructions.md`(被 `actions/propose.md` 在 Step 6 调用),不需要外部 CLI 提供。

模板文件位于 `<skill_dir>/templates/`:
- `proposal.md` — Why / What Changes / Capabilities / Impact
- `spec.md` — ADDED/MODIFIED/REMOVED/RENAMED Requirements + H4 Scenarios
- `design.md` — Context / Goals / Decisions / Risks
- `tasks.md` — checkbox 列表

> **想要自定义 schema?** 直接在对话里告诉 Claude 新的 artifact 编排逻辑(例如"propose 之后我想多加一步 risk-assessment"),Claude 会现场按你的描述调整流程,不需要改 skill 代码。

---

## 执行方式

> **`<skill_dir>` 约定**:以下路径中 `<skill_dir>` 指代本 SKILL.md 所在目录的绝对路径。
>
> **产物路径约定**:本 skill 在使用方仓库下生成的所有产物都落在 `{PRIMARY_REPO}/docs/xdev/openspec/` 之下,与 `docs/xdev/speckit/`、`docs/xdev/exec-plan/`、`docs/xdev/gstack/`、`docs/xdev/superpowers/` 并列。

### 1. 解析 action

从 `$ARGUMENTS` 中提取第一个词作为 `action`,剩余部分作为该 action 的参数。

### 2. 路由分派

根据 `action` 的值,读取对应的 action 文件并执行:

```
action = "propose"  → 读取并执行 <skill_dir>/actions/propose.md
action = "explore"  → 读取并执行 <skill_dir>/actions/explore.md
action = "apply"    → 读取并执行 <skill_dir>/actions/apply.md
action = "archive"  → 读取并执行 <skill_dir>/actions/archive.md
action = "run"      → 读取并执行 <skill_dir>/actions/run.md
其他                → 输出下方的帮助信息
```

### 3. 执行 action

读取对应的 `actions/{action}.md` 文件后:

1. 将文件内容作为当前任务的执行指令
2. 文件中所有 `<skill_dir>` 均指向本 SKILL.md 所在目录
3. 将剩余参数(去掉 action 后的部分)传递给该 action 的执行逻辑

### 4. 帮助信息(action 未匹配时输出)

```
openspec - OpenSpec 工作流引擎(纯 prompt 实现)

用法:/openspec <action> [参数]

可用 actions:
  propose [需求描述]            一步生成完整 change(proposal + specs + design + tasks)
                                内置自动 init + L3 续接 ⭐ 推荐
  explore [topic]               探索性对话,不创建 artifact
  apply [change-name]           按 tasks.md 实施代码(自动续接未勾选的 task)
  archive [change-name]         归档已完成的 change(内置 delta merge)
  run [需求描述]                全流程串联(propose → review → apply → archive)
                                不带参数 = 断点续传,列出活跃 change 续接 ⭐ 推荐

参数均为可选——除 propose / run 外,未提供 change-name 时会从对话上下文推断或交互选择。

示例:
  /openspec propose 加一个 dark mode 切换           # 全新需求,自动 init + 生成完整 change
  /openspec apply add-dark-mode                     # 实施 tasks.md
  /openspec archive add-dark-mode                   # 归档(自动 delta merge)
  /openspec run 用户登录流程优化                    # 全流程一把梭
  /openspec run                                     # 不带参数 = 断点续传,选活跃 change 续接

旧 actions 已合并:init / new / continue / ff / sync / verify 已并入 propose / archive / run。
  - 想 init / new / 续接 → propose 自动处理
  - 想 sync delta → archive 自动处理
  - 想中断后续传 → run(不带参数)
  - 想 verify → 对话里说"帮我对照 spec 检查代码"
```

---

## 与 speckit / exec-plan 的关系

| 维度 | speckit | exec-plan | openspec |
|------|---------|-----------|----------|
| 工作模型 | SDD(spec-first) | 自包含执行计划 | fluid 多 change |
| 切分支 | 必须 | 必须 | 不强制(以 change 而非分支为单位) |
| 中间产物目录 | `docs/xdev/speckit/{FEATURE_NAME}/` | `docs/xdev/exec-plan/{FEATURE_NAME}/` | `docs/xdev/openspec/changes/{change-name}/` |
| Action 数量 | 7 | 1 | 5 |
| 实现形态 | 纯 prompt + Claude Agent | 纯 prompt + Claude Agent | **纯 prompt + Claude Agent**(本版本) |

四个 skill(speckit / exec-plan / gstack / openspec)共用同一份「git 仓库扫描 + 主仓库选择」逻辑:`prompts/resolve_workspace.md`(4 份物理副本,openspec 是第 4 份)。

> **历史注**:本 skill 早期版本是 [Fission-AI/OpenSpec](https://github.com/Fission-AI/OpenSpec) CLI 的包装层,需要 `npm install -g @fission-ai/openspec`。当前版本已重构为纯 prompt 实现:schema.yaml 中的 artifact instruction 直接嵌入到对应 action 文件,模板文件位于 `templates/`,所有"目录扫描 / 状态判断 / delta 合并"由 Claude Agent 直接完成。这样做的好处是:零外部依赖、产物路径完全可控、与其他 4 个 skill 形态一致。
