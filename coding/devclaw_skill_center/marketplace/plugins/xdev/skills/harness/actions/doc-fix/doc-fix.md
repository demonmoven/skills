# Action: doc-fix —— 文档治理

以代码为唯一真相来源，修改文档中与代码不符的部分。**只改文档，绝不改代码**。

## 何时使用

| 场景 | 触发方式 | 分支策略 |
|------|---------|---------|
| 定期触发 | 用户手动调用 `/harness doc-fix` | 创建独立分支，提独立 MR |
| ExecPlan 完成后 | Agent 将 ExecPlan 标记为 completed | 跟随当前用户分支 |
| 结构变更 | 新增/删除/重命名顶层目录、package 或核心模块 | 跟随当前用户分支 |
| `/harness evolve` 路由 | evolve 健康检查发现文档过时 | 按用户指示决定 |

## 执行方式

读取并按照 `<skill_dir>/actions/doc-fix/prompts/doc-fix-workflow.md` 中的完整流程执行。

### 检查的 6 个维度

1. ARCHITECTURE.md vs 实际目录结构（幽灵模块 / 未记录模块）
2. code-patterns.md vs 实际代码模式
3. invariants.md vs 实际代码约束
4. AGENTS.md 导航完整性
5. knowledge-gaps.md 中的 open 条目
6. ADR 决策与当前实现是否一致

## 安全约束

1. **只修改文档文件**，不修改任何源代码
2. 每次更新使用 HTML 注释标注触发原因和来源
3. 无法以代码为准确认的更新，标记为 Knowledge Gap 而非猜测填充

## 汇报

完成后向用户汇报：
- 6 个维度的检查结果
- 修复的文档清单（每个文档的差异摘要）
- knowledge-gaps.md 中关闭/新增的条目
