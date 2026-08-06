---
name: gstack
description: "gstack 角色编排引擎 - 移植自 garrytan/gstack v0.15.1.0 的 AI 工程团队 sprint 流程。33 个 action 覆盖 Sprint 全链路：构思 / 三审 / 设计 / 实现审查 / 浏览器 QA / 安全审计 / 发布部署 / 复盘 / 安全防护 / 会话智能。统一入口 /gstack <action>。"
argument-hint: "<action> [参数]"
---

# gstack

gstack — 移植自 [garrytan/gstack](https://github.com/garrytan/gstack) v0.15.1.0 的 AI 工程团队角色编排引擎。**33 个 action 完整覆盖**上游 skill 集合（不含 worktree 的并行工作区编排——上游本身没有专门的 worktree skill，仅 plan-eng-review 内嵌一段并行化策略，已随该 action 一并迁移）。

> **本仓库版本与上游差异**：
> - 去除了 telemetry / proactive 配置、`~/.gstack/` 状态管理、自我升级机制
> - 上游 33 个 skill 全量迁移为 actions（保留原工作流逻辑、删除私有基础设施）
> - 切分支 / 工作区解析对齐 speckit / exec-plan / openspec，复用 `prompts/resolve_workspace.md`
> - 中间产物统一落到 `{PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}/{action}.md`

## 用户输入

```text
$ARGUMENTS
```

## 参数格式

`$ARGUMENTS` 格式：`<action> [参数]`

第一个词为 `action`，决定派发哪个角色或流程。后续部分为该 action 的参数（可选）。

### Action 全集（按职能分组，共 33 个）

#### Sprint 全流程入口

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `run` | `[需求描述或链接]` | 全流程串联（office-hours → autoplan → review → cso → qa → ship） |

#### 构思与计划（5）

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `office-hours` | `[需求或 idea]` | YC Office Hours / Builder mode（产品构思） |
| `plan-ceo-review` | `[FEATURE_NAME]` | CEO / 创始人级审查（"Brian Chesky 模式"） |
| `plan-eng-review` | `[FEATURE_NAME]` | 工程经理架构审查（含 worktree 并行化策略） |
| `plan-design-review` | `[FEATURE_NAME]` | 高级设计师 7 维度审查 |
| `autoplan` | `[FEATURE_NAME]` | 一键串联 CEO + Eng + Design 三审 |

#### 设计与视觉（4）

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `design-consultation` | `[FEATURE_NAME]` | 设计合伙人（从零构建设计系统） |
| `design-shotgun` | `[FEATURE_NAME]` | 设计探索器（多变体并行生成） |
| `design-html` | `[FEATURE_NAME]` | 设计工程师（设计稿 → 生产级 HTML） |
| `design-review` | `[url]` | 80 项视觉审计 + 修复循环 ⚠️ 需 browse |

#### 调查与审查（4）

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `investigate` | `[bug 描述]` | 调试专家根因分析（铁律：调查先于修复） |
| `review` | `[base_branch]` | 7 路并行 code review |
| `codex` | `[模式]` | OpenAI Codex 跨模型审查 ⚠️ 需 codex CLI |
| `cso` | `[scope]` | 首席安全官（OWASP + STRIDE） |

#### QA 与浏览器（5）

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `qa` | `[url]` | 真实浏览器 QA + 自动修复 ⚠️ 需 browse |
| `qa-only` | `[url]` | QA 报告（不改代码） ⚠️ 需 browse |
| `browse` | `[command]` | 持久化 Chromium 自动化（gstack 核心二进制） ⚠️ 需 browse |
| `connect-chrome` | （无） | 连接真实 Chrome（带侧边栏扩展） ⚠️ 需 browse |
| `setup-browser-cookies` | （无） | 从真实浏览器导入 Cookie ⚠️ 需 browse |

#### 发布与运维（6）

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `ship` | （无） | 发布全流程（test + review + push + PR） |
| `land-and-deploy` | （无） | 合并 PR + 等 CI + 部署 + 验证生产 |
| `document-release` | （无） | 自动更新所有项目文档以匹配刚发布内容 |
| `canary` | （无） | 部署后监控循环（控制台错误、性能回归） |
| `benchmark` | `[scope]` | 性能基准（页面加载、Core Web Vitals 等） |
| `setup-deploy` | （无） | 配置部署链路（首次设置） |

#### 复盘（1）

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `retro` | `[period]` | 工程经理周复盘 |

#### 安全防护（4）

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `careful` | （无） | 破坏性命令前警告 ⚠️ 仅供参考（需 hook） |
| `freeze` | `[directory]` | 锁定文件编辑范围 ⚠️ 仅供参考（需 hook） |
| `guard` | `[directory]` | careful + freeze 一键开启 ⚠️ 仅供参考 |
| `unfreeze` | （无） | 解除 freeze ⚠️ 仅供参考 |

#### 会话智能（3）

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `checkpoint` | `[name]` | 保存 / 恢复工作状态快照 ⚠️ 需上游 gstack 安装 |
| `health` | （无） | 代码健康度仪表板 ⚠️ 需上游 gstack 安装 |
| `learn` | （无） | 跨会话学习记忆管理 ⚠️ 需上游 gstack 安装 |

#### 自管理（1）

| action | 参数 | 说明 |
|--------|------|------|
| `gstack-upgrade` | （无） | 升级 gstack ⚠️ 不适用 devclaw（按 plugin marketplace 升级） |

### 标记说明

- ⚠️ **需 browse**：依赖 gstack 上游的 `browse` 二进制。详见 USAGE.md "前置条件 — browse 二进制"
- ⚠️ **需 codex CLI**：依赖 `npm install -g @openai/codex`
- ⚠️ **需 hook**：上游通过宿主进程 hook 实现命令拦截 / 文件锁定，devclaw 不提供 hook 机制，本 action 仅作为内容参考
- ⚠️ **需上游 gstack 安装**：依赖 `~/.gstack/` 状态目录，迁移版本仅供内容参考

---

## 执行方式

> **`<skill_dir>` 约定**：以下路径中 `<skill_dir>` 指代本 SKILL.md 所在目录的绝对路径。

### 1. 解析 action

从 `$ARGUMENTS` 中提取第一个词作为 `action`，剩余部分作为该 action 的参数。

### 2. 路由分派

根据 `action` 的值，读取对应的 action 文件并执行：

```
<skill_dir>/actions/{action}.md
```

支持的 action 名（共 33 个）：

```
run
office-hours plan-ceo-review plan-eng-review plan-design-review autoplan
design-consultation design-shotgun design-html design-review
investigate review codex cso
qa qa-only browse connect-chrome setup-browser-cookies
ship land-and-deploy document-release canary benchmark setup-deploy
retro
careful freeze guard unfreeze
checkpoint health learn
gstack-upgrade
```

未匹配 → 输出下方帮助信息。

### 3. 执行 action

读取对应的 `actions/{action}.md` 文件后：

1. 将文件内容作为当前任务的执行指令
2. 文件中所有 `<skill_dir>` 均指向本 SKILL.md 所在目录
3. 将剩余参数（去掉 action 后的部分）传递给该 action 的执行逻辑

### 4. 帮助信息（action 未匹配时输出）

```
gstack - AI 工程团队角色编排引擎（33 个 action）

用法：/gstack <action> [参数]

⭐ 推荐入口：
  /gstack run [需求描述]            全流程串联（最常用）

构思与计划：
  /gstack office-hours [需求]
  /gstack plan-ceo-review [feature]
  /gstack plan-eng-review [feature]
  /gstack plan-design-review [feature]
  /gstack autoplan [feature]

设计与视觉：
  /gstack design-consultation [feature]
  /gstack design-shotgun [feature]
  /gstack design-html [feature]
  /gstack design-review [url]                ⚠️ 需 browse

调查与审查：
  /gstack investigate [bug 描述]
  /gstack review [base_branch]
  /gstack codex [模式]                       ⚠️ 需 codex CLI
  /gstack cso [scope]

QA 与浏览器：
  /gstack qa [url]                           ⚠️ 需 browse
  /gstack qa-only [url]                      ⚠️ 需 browse
  /gstack browse [command]                   ⚠️ 需 browse
  /gstack connect-chrome                     ⚠️ 需 browse
  /gstack setup-browser-cookies              ⚠️ 需 browse

发布与运维：
  /gstack ship
  /gstack land-and-deploy
  /gstack document-release
  /gstack canary
  /gstack benchmark [scope]
  /gstack setup-deploy

复盘：
  /gstack retro [period]

安全防护（仅参考）：
  /gstack careful / freeze / guard / unfreeze

会话智能（需上游 gstack）：
  /gstack checkpoint / health / learn

自管理：
  /gstack gstack-upgrade                     ⚠️ devclaw 不适用

完整文档见 USAGE.md。
```

---

## 与 speckit / exec-plan / openspec 的关系

| 维度 | speckit | exec-plan | openspec | **gstack** |
|------|---------|-----------|----------|-----------|
| 工作模型 | SDD（spec-first） | 自包含执行计划 | fluid 多 change | **角色编排（33 个专家）** |
| 切分支 | 必须 | 必须 | 必须 | **run 模式必须，单 action 不切** |
| 中间产物目录 | `docs/xdev/speckit/{FEATURE_NAME}/` | `docs/xdev/exec-plan/{FEATURE_NAME}/` | `docs/xdev/openspec/changes/{name}/` | **`docs/xdev/gstack/{FEATURE_NAME}/`** |
| Action 数量 | 7 | 1 | 11 | **33** |
| 浏览器 QA | 无 | 无 | 无 | **有（依赖 browse 二进制）** |
| 跨模型 review | 无 | 无 | 无 | **有（codex action）** |

四个 skill 共用同一份「git 仓库扫描 + 主仓库选择」逻辑：`prompts/resolve_workspace.md`（4 份物理副本，gstack 是第 4 份）。
