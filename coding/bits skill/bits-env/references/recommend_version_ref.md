# Recommend Version Reference — 分支推荐接口参考

> 本文件定义了分支与版本推荐的完整逻辑流程与输出格式。
> Plan Skill 在 **RESOLVE** 阶段缺少 branch/version 时执行。

**CLI 路径**: `bitscli env`（`scm-repo`、`scm-latest-version` 子命令），其余为 Git 原生命令。

---

## 概述

分支推荐**不是单一 CLI 命令**，而是一套由多个命令组成的决策流程：

1. 检测本地 Git 上下文
2. 匹配服务仓库
3. 若用户指定了 branch，通过 `scm-latest-version` 查询该分支最新构建版本
4. 若无本地上下文则回退到 PPE/Prod 基准版本

---

## 核心原则

> **最终部署参数必须使用 `--scm-version`，除非满足唯一例外条件。**
>
> **例外条件（情况 A）**: 用户在 Git 仓库中 + 仓库匹配目标服务 → 使用 `--branch`，**不查版本**。
>
> | 场景 | 部署参数 | 原因 |
> |------|---------|------|
> | 用户在 Git 仓库中，部署当前分支 | `--branch` | 用户明确意图部署本地开发分支 |
> | **其他所有场景** | **`--scm-version`** | version 精确锁定构建产物，消除分支→构建的不确定性 |

---

## 决策流程图

```
用户请求部署（缺少 branch/version）
        │
        ▼
  用户指定了 version？ ─── 是 ─── 直接使用 --scm-version，结束
        │
        否
        ▼
  ┌─────────────────────────────────────────────────────┐
  │ 优先检查: 当前目录是 Git 仓库 + 匹配目标服务？         │
  │                                                     │
  │   是 → ★ 情况 A: --branch（STOP，不查任何版本）       │
  │   否 → 继续往下                                      │
  └─────────────────────────────────────────────────────┘
        │ (否)
        ▼
  用户指定了 branch？（不在对应仓库中）
        │          │
        是         否
        ▼          ▼
  scm-repo +     情况 B: PPE Baseline
  scm-latest-       │
  version 查版本     ▼
        │       输出 --scm-version
        ▼
  有构建版本？
    │      │
   是     否
    ▼      ▼
  --scm-version  回退 prod 基准
```

> **关键判断顺序**: Git 仓库检查 **优先于** branch 指定检查。
> 一旦确认情况 A，立即输出 `--branch` 并终止，**禁止**继续执行任何版本查询。

> **关键**: 情况 A（本地 Git 仓库 + 匹配目标服务）**直接使用 `--branch`**，
> 不执行任何版本查询。这允许用户部署尚未编译的分支，由 SCM 触发编译。

---

## 情况 A: 用户在 Git 仓库中，部署当前分支

> **这是唯一允许最终使用 `--branch` 部署的场景。**
>
> **为什么不查版本？** 用户在本地修改代码并 push 后，SCM 编译可能尚未完成，
> 或旧版本号对应的是修改前的代码。使用 `--branch` 部署会触发 SCM 平台基于最新
> commit 编译，确保部署的是用户刚刚 push 的代码。查询版本 → 拿到旧构建产物 → 无法测试最新代码。

### ⛔ 情况 A 禁止行为（HARD STOP）

| # | 禁止行为 | 原因 |
|---|---------|------|
| 1 | ❌ 调用 `scm-repo` 查询版本 | 情况 A 不需要版本信息 |
| 2 | ❌ 调用 `scm-latest-version` | 情况 A 不需要版本信息 |
| 3 | ❌ 使用 `--scm-version` 部署 | 情况 A 必须使用 `--branch` |
| 4 | ❌ 从 `Env SCM Dependencies` 提取 Version | 即使看到版本数据也必须忽略 |
| 5 | ❌ 比较当前版本与目标版本 | 情况 A 没有"目标版本"概念 |

> **情况 A 的唯一输出**: `--branch <local_branch>`。不查版本，不比较版本，由 SCM 平台触发编译。

### Step A1: 提取上下文

```bash
git rev-parse --is-inside-work-tree   # 必须返回 true
git remote -v                          # 获取本地仓库地址
git branch --show-current              # 获取当前分支
```

### Step A2: 匹配验证（仅用 Git 命令，不调用 scm-repo）

1. 获取本地仓库远端地址:
   ```bash
   git remote -v
   ```
   提取 `origin` 的 URL（如 `git@code.byted.org:webarch/debugging_panel.git`）。

2. 获取目标服务的仓库地址:
   ```bash
   bitscli env scm-repo --psm <PSM>
   ```
   **⚠️ 仅读取 `Main Repository Git Info` 表中的 `Git URL` 字段用于比对。**
   **⚠️ 完全忽略 `Env SCM Dependencies` 表中的所有 Version/Branch 数据。**

3. 比对两个 Git URL:
   - **匹配** → Step A3（直接用 `--branch`）
   - **不匹配** → 进入 **情况 B**

### Step A3: 检查远端分支（可选，仅用于提示）

```bash
git ls-remote --heads origin <local_branch>
```

- **有结果** → 推荐 `--branch <local_branch>`，无警告
- **无结果** → 推荐 `--branch <local_branch>` + ⚠️ 警告"远程分支不存在，请先 push"

### 输出

```
部署参数: --branch <local_branch>
推荐来源: Local Branch
```

> **再次强调**: 情况 A 的输出永远是 `--branch`，永远不是 `--scm-version`。
> 即使当前分支没有任何构建产物，也使用 `--branch`，部署系统会自动触发 SCM 编译。


## 情况 B: 非 Git 仓库 / 服务不匹配 / 用户指定 branch 需转 version

> **本情况下最终输出必须是 `--scm-version`，不允许输出 `--branch`。**

### Step B1: 获取基准版本

```bash
bitscli env scm-repo --psm <PSM>
```

### Step B2: 解析输出 — 版本选择优先级

从 `Env SCM Dependencies` 表格中按以下优先级提取 **Version**：

| 优先级 | Env Type | 说明 |
|:-------|:---------|:-----|
| 1 | `prod` | 生产环境版本（最稳定，**首选**） |
| 2 | `boe_base` | BOE 基准版本（降级方案） |
| 3 | 其他 | 选择最新的可用版本 |

### Step B3: 输出

部署参数: `--scm-version <version>`
推荐来源: `PPE Baseline (prod)` / `PPE Baseline (boe_base)`

> **⚠️ 禁止行为**: 在情况 B 中提取到 branch 后直接用 `--branch` 部署。
> 必须提取对应的 **version** 字段。若 version 为空，则向上/向下查找最近的有 version 的 Env Type。

---

## 用户指定 branch 的转换逻辑（仅限非本地仓库场景）

> **适用条件**: 用户指定了 branch，但**不在对应 Git 仓库中**（已排除情况 A）。
> 此时不直接使用 `--branch`，而是通过以下两种方式查询该 branch 对应的最新构建版本。
>
> **⚠️ 如果用户在对应 Git 仓库中且部署当前分支 → 走情况 A，直接 `--branch`，不进入此流程。**

### 方式一：通过 scm-repo 查询

```bash
bitscli env scm-repo --psm <PSM>
```

从输出的 `Env SCM Dependencies` 中找到与用户指定 branch **匹配**的行，提取 Version。

### 方式二：通过 scm-latest-version 查询（推荐）

> 当方式一未找到匹配行、或找到匹配行但 Version 为空时，使用此命令直接查询指定 branch 的最新构建版本。

**前置条件**: 需要先通过 `scm-repo --psm <PSM>` 获取 `repo_id`（输出中的 `Main Repository Git Info` → `Repo ID`）。

```bash
bitscli env scm-latest-version --repo-id <repo_id> --branch <branch_name>
```

**参数说明**:

| Parameter | Flag | Required | Description |
|:--|:--|:--:|:--|
| repo_id | `--repo-id` | YES | 仓库 ID，从 `scm-repo` 输出中获取 |
| branch | `--branch` | YES | 目标分支名 |

**输出**: 直接返回版本号字符串（如 `1.0.1.922`），无额外格式。

### 综合判断流程

```
用户指定 branch
    │
    ▼
Step 1: scm-repo --psm <PSM>
    │
    ├── 找到匹配行且 Version 非空 → 使用 --scm-version <version> ✅
    │
    ├── 找到匹配行但 Version 为空 ──┐
    │                               │
    └── 未找到匹配行 ──────────────┐│
                                   ││
                                   ▼▼
                    Step 2: scm-latest-version --repo-id <repo_id> --branch <branch>
                               │
                               ├── 返回版本号 → 使用 --scm-version <version> ✅
                               │
                               └── 无结果/报错 → 回退到 prod 基准 version，并提示用户
```

| 查询结果 | 动作 |
|---------|------|
| scm-repo 找到匹配行且 Version 非空 | 使用 `--scm-version <version>` |
| scm-repo 未找到 / Version 为空 → scm-latest-version 返回版本号 | 使用 `--scm-version <version>` |
| 两种方式均未获取到版本 | 回退到 prod 基准 version，并提示用户 |

---

## Output Format

```text
### Version Recommendation
| Item | Value |
| :--- | :--- |
| Service | {psm} |
| Recommended | `--scm-version {version}` 或 `--branch {branch}` |
| Source | Local Branch / PPE Baseline (prod) / PPE Baseline (boe_base) |

### Warnings
> ⚠️ {warning_message}  (如有)
```

### 示例 — 本地 Git 分支（情况 A）

```text
### Version Recommendation
| Item | Value |
| :--- | :--- |
| Service | inf.hae.boe |
| Recommended | `--branch feat/new-ui` |
| Source | Local Branch |

### Warnings
> ⚠️ 远程分支不存在，请先 push。
```

### 示例 — PPE Baseline（情况 B）

```text
### Version Recommendation
| Item | Value |
| :--- | :--- |
| Service | inf.hae.boe |
| Recommended | `--scm-version 1.0.1.810` |
| Source | PPE Baseline (prod) |
```

### 示例 — 用户指定 branch 转 version（scm-repo 匹配）

```text
### Version Recommendation
| Item | Value |
| :--- | :--- |
| Service | inf.hae.boe |
| Recommended | `--scm-version 1.0.2.903` |
| Source | Branch "feat/api-v2" → version 1.0.2.903 (via scm-repo) |
```

### 示例 — 用户指定 branch 转 version（scm-latest-version 查询）

```text
### Version Recommendation
| Item | Value |
| :--- | :--- |
| Service | inf.hae.boe |
| Recommended | `--scm-version 1.0.1.922` |
| Source | Branch "cm_dev/env_xiaoxi" → version 1.0.1.922 (via scm-latest-version) |
```

---

## Plan 接收格式

| Field | Type | Description | Example |
|:------|:-----|:------------|:--------|
| deploy_param | string | 最终部署参数 | `--scm-version 1.0.1.810` 或 `--branch feat/x` |
| version | string \| null | 精确版本号 | `1.0.1.810` |
| branch | string \| null | 分支名（仅情况 A） | `feat/new-api` |
| source | enum | 推荐来源 | `local_git` / `ppe_baseline_prod` / `ppe_baseline_boe` / `scm_latest_version` |
| warning | string \| null | 警告信息 | "远程分支不存在" |

---

## 优先级总览

| 优先级 | 条件 | 部署参数 |
|:-------|:-----|:--------|
| **1** | 用户明确指定 version | `--scm-version <user_value>` |
| **2** | 用户指定 branch + 查到对应 version | `--scm-version <resolved>` |
| **3** | 用户在 Git 仓库中部署当前分支（情况 A）—— **不查版本，直接触发编译** | `--branch <local_branch>` |
| **4** | PPE/Prod 基准版本（情况 B） | `--scm-version <prod_version>` |
| **5** | 以上均不可用 | `--scm-version <boe_base_version>`，再不可用 → 询问用户 |

> **禁止兜底到 `--branch master`**。如果无法确定版本，必须询问用户，不可猜测。