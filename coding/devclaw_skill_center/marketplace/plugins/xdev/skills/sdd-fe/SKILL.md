---
name: sdd-fe
description: "CoStudio SDD 前端引擎 - 规格驱动开发全流程。支持 run / review-feature-spec / mining / plan / split-tasks / development。"
argument-hint: "[action] [参数...]"
---

# devclaw-sdd-fe

CoStudio SDD（Specification-Driven Development）前端引擎，统一入口。

## 用户输入

```text
$ARGUMENTS
```

## 参数格式

`$ARGUMENTS` 格式：`[action] [args...] [参数...]`

第一个词为 `action`（可选），决定执行哪个 SDD 步骤。如果没有指定，则默认使用 `run`。后续分为两部分：
- `args...`：用户透传的 args 内容，会无损透传到所有 action 内
- `[参数...]`：该 action 的业务参数

所有的操作和参数优先解析用户的 args 内容。

| action | 后续参数 | 说明 | 类型 |
|--------|---------|------|------|
| `run` | `[RepoPath] <PrdContent> <ConstitutionFile>` | 生成功能规格说明书 | 自动化 |
| `review-feature-spec` | `<InputsDir> <SpecsDir> <ConstitutionFile>` | 审阅规格（HITL） | 交互式 |
| `mining` | `<SpecsDir> <InputsDir> [RepoDir] <ConstitutionFile>` | 技术挖掘 | 自动化 |
| `plan` | `<SpecsDir> [RepoDir] [InputsDir] <ConstitutionFile>` | 生成技术方案 | 自动化 |
| `split-tasks` | `<SpecsDir> [RepoDir] <ConstitutionFile>` | 拆分任务 | 自动化 |
| `development` | `<SpecsDir> [PhaseNumber] [RepoDir] <ConstitutionFile>` | 执行开发（省略 PhaseNumber 则自动扫描全部 Phase 连续执行） | 自动化 |

---

## 执行方式

> **`<skill_dir>` 约定**：以下路径中 `<skill_dir>` 指代本 SKILL.md 所在目录的绝对路径。

### 1. 解析 action

从 `$ARGUMENTS` 中提取第一个词作为 `action`，剩余部分作为该 action 的参数。

**如果 `$ARGUMENTS` 为空或第一个词不是有效的 action，则默认 `action = "run"`，所有参数作为 `run` 的业务参数。**

### 2. 路由分派

根据 `action` 的值，读取对应的 action 文件并执行：

```
action = "run" → 读取并执行 <skill_dir>/actions/run.md
action = "review-feature-spec"     → 读取并执行 <skill_dir>/actions/review-feature-spec.md
action = "mining"                   → 读取并执行 <skill_dir>/actions/mining.md
action = "plan"                     → 读取并执行 <skill_dir>/actions/plan.md
action = "split-tasks"              → 读取并执行 <skill_dir>/actions/split-tasks.md
action = "development"              → 读取并执行 <skill_dir>/actions/development.md
其他                                 → 输出下方的帮助信息
```

### 3. 执行 action

读取对应的 `actions/{action}.md` 文件后：

1. 将文件内容作为当前任务的执行指令
2. 文件中所有 `<skill_dir>` 均指向本 SKILL.md 所在目录
3. 将剩余参数（去掉 action 后的部分）传递给该 action 的执行逻辑

### 4. 帮助信息（action 未匹配时输出）

```
devclaw-sdd-fe - CoStudio SDD 前端引擎

用法：devclaw-sdd-fe [action] [参数...]

可用 actions：
  run                       生成功能规格说明书（默认）
  review-feature-spec        交互式审阅规格
  mining                     技术挖掘
  plan                       生成技术方案
  split-tasks                拆分任务
  development                执行开发

示例：
  devclaw-sdd-fe /path/to/repo "需求内容" /path/to/constitution.md        # 默认使用 run
  devclaw-sdd-fe "需求内容" /path/to/constitution.md                          # 默认使用 run，RepoPath 默认使用 pwd
  devclaw-sdd-fe run /path/to/repo "需求内容" /path/to/constitution.md
  devclaw-sdd-fe run "需求内容" /path/to/constitution.md  # RepoPath 默认使用 pwd
  devclaw-sdd-fe plan /path/to/specs /path/to/repo /path/to/inputs /path/to/constitution.md
  devclaw-sdd-fe development /path/to/specs 1 /path/to/repo /path/to/constitution.md
  devclaw-sdd-fe development /path/to/specs              # 省略 PhaseNumber，自动扫描全部 Phase 连续执行
```
