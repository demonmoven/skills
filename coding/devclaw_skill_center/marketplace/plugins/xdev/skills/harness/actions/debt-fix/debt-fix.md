# Action: debt-fix —— 技术债扫描与修复

扫描架构级技术债，**实际修改代码完成修复**。每次运行最多处理 3 个条目，逐个修复、逐个 commit、逐个验证。

## 何时使用

| 场景 | 触发方式 | 说明 |
|------|---------|------|
| 周期性盘点 | 用户手动调用 `/harness debt-fix` | 对仓库做技术债盘点和修复 |
| 开发过程中 | 开发新功能时发现与已有逻辑重复 | 在当前分支顺手修复 |
| ExecPlan 完成后 | ExecPlan 执行完毕，回头清理引入的技术债 | 防止技术债沉淀 |
| `/harness evolve` 路由 | evolve 检测到 invariant 违反，建议调用 | 用户确认后执行 |

## 执行方式

读取并按照 `<skill_dir>/actions/debt-fix/prompts/debt-fix-workflow.md` 中的完整流程执行：

1. **阶段 1：扫描与识别**（5 条规则：duplicate-persistence / divergent-impl / model-divergence / scattered-state / unabstracted-flow）
2. **阶段 2：选择修复目标**（最多 3 个，向用户确认；低置信度需显式批准）
3. **阶段 3：逐个修复**（分析 → 改代码 → 测试 + lint → commit → 更新 debt-log；失败立即回滚）
4. **阶段 4：分支策略**（按触发场景选择独立分支或跟随当前分支）

## 安全约束（必须遵守）

1. 每次最多修复 3 条
2. 每修复一条独立 commit（可独立回滚）
3. 每个 commit 前必须通过测试和 lint 检查
4. 验证失败立即回滚该条所有改动，跳过该条目
5. 低置信度条目需用户**显式批准**才可修复
6. 本 action **只修改代码文件**，不修改文档（`debt-log.md` 除外）

## 汇报

完成后向用户汇报：
- 扫描到的总条目数（按规则分类）
- 选中修复的条目（≤3 个）
- 每条修复的 commit hash 和验证结果
- debt-log.md 的更新摘要
