# Architecture Invariants

本文件记录项目的架构不变量——刻意不做的事情和必须维持的性质。
Agent 在编写代码时必须遵守这些约束。

编写方法：每条 invariant 必须包含具体的代码路径或模块边界示例（不能是空泛描述），从代码模式采样和模块依赖图中提取。Enforcement 列必须反映当前实际的强制方式（code review / lint rule / CI check），而非理想状态。

## Repository-level Invariants

| ID | Invariant | Scope | Enforcement | Status |
|---|---|---|---|---|
| INV-001 | <!-- HARNESS-GEN: 从 Phase 1 模块依赖图中提取具体的 import 方向约束。必须包含模块路径示例，如 "packages/X/ 不应 import 来自 apps/Y/ 的任何模块" --> | repo-wide | <!-- HARNESS-GEN: 标注当前实际的强制方式（code review / lint rule / CI check），不要虚标 --> | active |

## Architecture-level Invariants

| ID | Invariant | Scope | Enforcement | Status |
|---|---|---|---|---|
| INV-101 | <!-- HARNESS-GEN: 从 Phase 1 模块依赖图中提取具体的 import 方向约束。必须包含模块路径示例，如 "packages/X/ 不应 import 来自 apps/Y/ 的任何模块" --> | cross-module | <!-- HARNESS-GEN: 标注当前实际的强制方式（code review / lint rule / CI check），不要虚标 --> | active |

## Promotion Rule

当某条不变量被违反 3 次以上：

    口头约定 → 写入文档 → 写入 lint 规则 → 写入 pre-commit hook → 写入 CI

每次提升增加强制力。目标是让机器而非人来执行检查。
