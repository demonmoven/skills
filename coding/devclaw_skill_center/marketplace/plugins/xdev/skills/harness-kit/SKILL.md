---
name: harness-kit
description: "Harness Engineering 工具集 — 10 个独立细分 action：架构级技术债扫描、基线文档建立、文档持续维护、提交护栏、执行计划、Go 观测栈、本地孪生、深度分析、飞书同步、跨仓库文档引用。当用户提到 harness-kit、harness 工具集、harness kit、debt scan、doc-init、doc-gardener、hook-init、obs-init、local-twin、exec-plan、harness analysis、feishu-sync、setup-ref、跨仓库引用、workspace ref、本地观测栈、本地孪生、文档新鲜度时触发。"
argument-hint: "<debt-scan|doc-init|doc-gardener|hook-init|exec-plan|obs-init|local-twin|analysis|feishu-sync|setup-ref> [参数]"
---

# harness-kit

Harness Engineering 工具集统一入口。10 个独立 action 各自聚焦工程化的某一支柱（上下文工程 / 架构约束 / 持续演进 / 跨仓库引用），可单独调用。

## 用户输入

```text
$ARGUMENTS
```

## 参数格式

`$ARGUMENTS` 格式：`<action> [参数]`

第一个词为 `action`，决定执行哪个 harness-kit 子能力，剩余部分为该 action 的参数。

| action | 说明 | 入口文件 |
|--------|------|----------|
| `debt-scan` | 架构级技术债扫描（结构性问题，非 lint 级） | `<skill_dir>/actions/debt-scan/debt-scan.md` |
| `doc-init` | 仓库基线文档体系建立（AGENTS / ARCHITECTURE / docs/） | `<skill_dir>/actions/doc-init/doc-init.md` |
| `doc-gardener` | 文档新鲜度巡检与自动修复 | `<skill_dir>/actions/doc-gardener/doc-gardener.md` |
| `hook-init` | 交互式 pre-commit 质量护栏建立 | `<skill_dir>/actions/hook-init/hook-init.md` |
| `exec-plan` | 需求全生命周期执行计划 | `<skill_dir>/actions/exec-plan/exec-plan.md` |
| `obs-init` | Go 仓库本地观测栈（VictoriaLogs + VictoriaTraces） | `<skill_dir>/actions/obs-init/obs-init.md` |
| `local-twin` | 字节内场 Go 服务本地化分析 | `<skill_dir>/actions/local-twin/local-twin.md` |
| `analysis` | Harness 视角的仓库成熟度审计 | `<skill_dir>/actions/analysis/analysis.md` |
| `feishu-sync` | 飞书文档与仓库 Markdown 双向同步 | `<skill_dir>/actions/feishu-sync/feishu-sync.md` |
| `setup-ref` | 跨仓库文档引用关系初始化（主仓 + 子仓互相登记 AGENTS / ARCHITECTURE） | `<skill_dir>/actions/setup-ref/setup-ref.md` |

---

## 执行方式

> **`<skill_dir>` 约定**：以下路径中 `<skill_dir>` 指代本 SKILL.md 所在目录的绝对路径。

### 1. 解析 action

从 `$ARGUMENTS` 中提取第一个词作为 `action`，剩余部分作为该 action 的参数。

### 2. 路由分派

按上方表格找到对应的入口 md 文件。

### 3. 执行 action

读取对应的入口 md 文件后：

1. 将文件内容作为当前任务的执行指令
2. 文件中所有 `<skill_dir>` 均指向本 SKILL.md 所在目录
3. 将剩余参数（去掉 action 后的部分）传递给该 action 的执行逻辑
4. 各 action 自带 `references/` 目录（如有），通过 `<skill_dir>/actions/<action>/references/...` 访问

### 4. 帮助信息（action 未匹配时输出）

```
harness-kit — OpenAI Harness Engineering 工具集

用法：/xdev:harness-kit <action> [参数]

可用 actions：
  debt-scan       架构级技术债扫描（结构性问题，非 lint 级）
  doc-init        仓库基线文档体系建立（AGENTS / ARCHITECTURE / docs/）
  doc-gardener    文档新鲜度巡检与自动修复
  hook-init       交互式 pre-commit 质量护栏建立
  exec-plan       需求全生命周期执行计划
  obs-init        Go 仓库本地观测栈（VictoriaLogs + VictoriaTraces）
  local-twin      字节内场 Go 服务本地化分析
  analysis        Harness 视角的仓库成熟度审计
  feishu-sync     飞书文档与仓库 Markdown 双向同步
  setup-ref       跨仓库文档引用关系初始化（主仓 + 子仓互相登记）

典型使用场景：
  /xdev:harness-kit debt-scan                # 架构债盘点
  /xdev:harness-kit doc-init                 # 给新仓库建文档体系
  /xdev:harness-kit doc-gardener             # 文档巡检 + 修复
  /xdev:harness-kit hook-init                # 交互式建提交护栏
  /xdev:harness-kit obs-init                 # Go 仓库接观测栈
  /xdev:harness-kit local-twin               # 内场服务本地化可行性分析
  /xdev:harness-kit analysis                 # 仓库 harness 成熟度评估
  /xdev:harness-kit exec-plan                # 复杂任务执行计划
  /xdev:harness-kit feishu-sync              # 飞书文档同步
  /xdev:harness-kit setup-ref claude         # 在 workspace 内跨仓库建立文档引用关系
```

---

## 关于来源

本 skill 的 action 分两类：

1. **9 个移植 action**（`debt-scan` / `doc-init` / `doc-gardener` / `hook-init` / `exec-plan` / `obs-init` / `local-twin` / `analysis` / `feishu-sync`）：来自 OpenAI Harness Engineering 工具集，各 action 文件头的 callout 注明了原 skill 名、作者与版本。
2. **xdev 原生 action**（`setup-ref`）：在多 git 工作区里建立跨仓库的文档引用关系，与 `xdev workspace create / init` 配套使用。

在 harness-kit 内统一以 action 形式调用，与 Skills Center 既有的 `/xdev:harness` 是**两个独立 skill**，无相互引用。
