---
name: harness
description: "Harness Engineering 一键落地 — 为任意代码仓库安装文档体系、hooks、lint、架构约束与持续演进能力。语言无关（Go/Python/Rust/Java/TypeScript 等任意技术栈）。当用户提到 'harness 初始化'、'harness 化'、'仓库工程实践初始化'、'安装 harness'、'技术债治理'、'文档对齐'、'知识演进'、'规范 lint 化' 时触发。"
argument-hint: "<action> [参数]"
---

# harness

Harness Engineering 工程实践一键落地引擎。统一入口。

> 关于 Harness Engineering 的理念和三大支柱（Context Engineering / Architecture Constraints / Garbage Collection），参见 `<skill_dir>/references/harness-overview.md` 与 `<skill_dir>/references/harness-engineering.md`。

## 用户输入

```text
$ARGUMENTS
```

## 参数格式

`$ARGUMENTS` 格式：`<action> [参数]`

第一个词为 `action`，决定执行哪个 harness 子能力。

| action | 说明 | 类型 |
|--------|------|------|
| `init` | 全流程初始化（执行 6 个 phase：分析 → hooks → docs → AGENTS → ARCHITECTURE → 验证） | 混合（含交互） |
| `debt-fix` | 技术债扫描与修复（每次最多 3 项，逐 commit 验证） | 自动化 |
| `doc-fix` | 以代码为准修复文档（解决知识缺口，文档/代码对齐） | 自动化 |
| `evolve` | 知识演进编排（检测 + 健康检查报告 + 路由建议，**不直接修改任何代码或文档**） | 混合 |
| `lint-promote` | 把反复违反的文档级规范提升为自动化 lint 规则 | 自动化 |

> **本 skill 不包含**：
> - **`exec-plan`**：Skills Center 已有独立的 `/exec-plan` skill，搭配使用即可
> - **`codex-audit`**：依赖 OpenAI Codex CLI（小众），如需使用请单独安装原始 skill

---

## 执行方式

> **`<skill_dir>` 约定**：以下路径中 `<skill_dir>` 指代本 SKILL.md 所在目录的绝对路径。

### 1. 解析 action

从 `$ARGUMENTS` 中提取第一个词作为 `action`，剩余部分作为该 action 的参数。

### 2. 路由分派

| action 值 | 入口文件 |
|-----------|---------|
| `init` | `<skill_dir>/actions/init/init.md` |
| `debt-fix` | `<skill_dir>/actions/debt-fix/debt-fix.md` |
| `doc-fix` | `<skill_dir>/actions/doc-fix/doc-fix.md` |
| `evolve` | `<skill_dir>/actions/evolve/evolve.md` |
| `lint-promote` | `<skill_dir>/actions/lint-promote/lint-promote.md` |
| 其他 | 输出下方的帮助信息 |

### 3. 执行 action

读取对应的入口 md 文件后：

1. 将文件内容作为当前任务的执行指令
2. 文件中所有 `<skill_dir>` 均指向本 SKILL.md 所在目录
3. 将剩余参数（去掉 action 后的部分）传递给该 action 的执行逻辑

### 4. 帮助信息（action 未匹配时输出）

```
harness — Harness Engineering 工程实践一键落地

用法：/harness <action> [参数]

可用 actions：
  init               全流程初始化新仓库（6 phase：分析 → hooks → docs → AGENTS → ARCHITECTURE → 验证）
  debt-fix           技术债扫描与修复（每次最多 3 项，逐 commit 验证）
  doc-fix            以代码为准修复文档（解决知识缺口，文档/代码对齐）
  evolve             知识演进编排（健康检查报告 + 路由建议，不直接修改）
  lint-promote       把反复违反的文档级规范提升为自动化 lint 规则

典型使用场景：
  /harness init                          # 第一次给一个新仓库做 harness 化
  /harness debt-fix                      # 周期性技术债盘点
  /harness doc-fix                       # 文档与代码对齐修复
  /harness evolve                        # 文档健康检查 + 演进建议
  /harness lint-promote                  # 把文档约束升级成 lint 规则

搭配使用：
  /exec-plan                             # 复杂任务前的设计文档（独立 skill，非 harness 一部分）
```
