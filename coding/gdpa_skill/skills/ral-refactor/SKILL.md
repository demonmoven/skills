---
name: ral-refactor
description: Use when refactoring RPC calls from hard-coded IDC routing (WithIDC) to RAL framework. Covers pattern analysis, code refactoring, and RAL initialization for GDP/Non-GDP projects.
---

# RAL Refactor

RAL 代码重构工具，用于将硬编码的 IDC 路由（WithIDC）重构为 RAL 框架。

## Workflow

完成 RAL 重构需要按以下步骤进行：

### Step 1: Pattern Analysis

分析 RPC 调用中的 WithIDC 路由模式并分类，**输出到 `pattern_analysis.md` 文档**。

**参考文档执行任务：[rpc_check_ral_pattern.md](./rpc_check_ral_pattern.md)**

### Step 2: Code Refactoring

**读取 `pattern_analysis.md` 文档**，自动遍历所有 WithIDC 点位进行重构。

**参考文档执行任务：[rpc_refactor_agent.md](./rpc_refactor_agent.md)**

### Step 3: RAL Initialization

为 GDP/Non-GDP 项目添加 RAL 初始化逻辑。

**参考文档执行任务：[rpc_refactor_init_agent.md](./rpc_refactor_init_agent.md)**

## Utilities

### Get RAL Info by Package Path

根据 package path 自动获取 PSM 信息。

**参考文档：[get_ral_info_by_package_path.md](./tools/get_ral_info_by_package_path.md)**
