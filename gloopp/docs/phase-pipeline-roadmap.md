# Phase 2 阶段管道化 — 后续规划

> 状态：**已完成核心 + 集成测试，v0.1.2 补充单点 design→execute auto-spawn；增强项继续暂缓**
> 更新时间：2026-06-22
> 负责人：lihuanyu.0w0

---

## 一、Phase 2 已完成（核心）

Phase 2 目标：用通用 **Phase Pipeline** 替代硬编码的两阶段（剑士→法师）。

### 1.1 核心成果

| 模块 | 文件 | 说明 |
|---|---|---|
| 领域层 | `internal/domain/quest/phase.go` | `PhaseDef`、`PhaseRun`、`PhaseRole` 类型 |
| 领域层 | `internal/domain/quest/pipeline.go` | `Pipeline` 类型 + 默认管道 + 辅助方法 |
| 领域层 | `internal/domain/quest/repository.go` | `QuestMeta` 扩展 `Pipeline` / `CurrentPhaseIdx` / `Phases` |
| 领域层 | `internal/domain/quest/service.go` | 7 个状态迁移方法都加上了 phase 管道同步 |
| 存储层 | `internal/fsstore/quests.go` | `Phases []PhaseTask` 双写双读，从 WarriorID/MageID 反推 |
| 存储层 | `internal/fsstore/quest_repository_adapter.go` | `QuestMetaToDomain` / `QuestMetaFromDomain` phase 转换 |
| 编排层 | `internal/orchestrator/pipeline.go` | 统一 `runAgentPhase` + `buildPhaseConfig`，按索引分发 |
| 编排层 | `internal/orchestrator/macro_loop.go` | 用 `runAgentPhase` 替代 `runWarriorPhase`/`runMagePhase` |
| 测试 | `internal/domain/quest/service_phase_test.go` | 18 个集成测试，覆盖完整生命周期 |

### 1.2 设计原则

1. **数据驱动，非接口多态** — 所有 Agent 阶段共享 `runPhase` 引擎，差异通过 `PhaseDef`/`phaseConfig` 注入
2. **双写双读，向后兼容** — 新 `Phases` 列表与旧 `WarriorID`/`MageID` 共存，API 和前端零改动
3. **状态机映射** — `running` ↔ phase 0, `reviewing` ↔ phase 1（先保持 1:1，未来可扩展）
4. **线性管道 + 返工循环** — 按顺序执行，返工回到 phase 0

### 1.3 验证结果

- ✅ 编译通过
- ✅ domain/quest 测试全绿（含 18 个新增 phase 测试）
- ✅ fsstore / orchestrator / server 等原有测试全绿
- ✅ 端到端 `engine_e2e_test` 正常

### 1.4 v0.1.2 增量

- ✅ design quest 成功后可通过 `auto_spawn_execute` 自动创建 execute child。
- ✅ 父子关系通过 `child_execute_quest_id` / `parent_quest_id` 持久化，重复触发时优先返回既有 child，避免重复创建。
- ✅ quick design 第一版不自动 spawn，避免 quick 跳过评审语义与 design review 边界混淆。
- ⚠️ 这仍不是 Phase 3 事件溯源或完整 workflow automation；它是单个 design quest 的后置动作。

---

## 二、暂缓项（待合适时机再做）

### 2.1 #45 工具与技能的阶段感知

**目标**：工具注册和技能注册支持按 phase 索引查询，而不是硬编码 warrior/mage 分类。

**当前状态**：
- 工具注册：`WarriorAvailable`/`MageAvailable` 标签，执行时通过 `adv.Tools` 和 `phaseConfig.AllowedTools` 实际控制
- 技能注册：按 `AdventurerClass` 分类，加载时按冒险者职业匹配
- 实际执行已经能正确工作（因为 `phaseConfig` 里设置了 `AllowedTools` 和 `ReadOnly`）

**为什么暂缓**：
- 功能上已经够用，注册层的分类标签不影响运行时行为
- 改了之后也没有明显的业务收益，只是"更优雅"
- 等真的需要加第 3 个阶段时再一起改更合理

**未来实施要点**：
- `internal/platformtools/registry.go` 加 `ListForPhase(phaseIdx int) []ToolDef`
- `internal/skills/registry.go` 加 `ListForPhase(phaseIdx int) []Skill`
- 内部维护 `phaseIdx → tools/skills` 映射
- 保留 `WarriorAvailable`/`MageAvailable` 作为配置兼容（加载时转换为 phase 映射）
- 预期改动：2-3 个文件，中等复杂度

### 2.2 #46 经验计算按 phase role

**目标**：经验发放从"warrior 发一份 + mage 发一份"改成"按 phase role 计算"。

**当前状态**：
- `calcExpGain` 函数硬编码分 warrior 和 mage 两种公式
- `awardExp` 在宏循环结束时一次性发放
- 经验写在 `personal_stats.json` 里

**为什么暂缓**：
- 改动面不小：要改经验计算函数、个人统计结构、事件 payload
- 当前两阶段场景下，按角色发和按 phase 发结果一样
- 需要同步改前端展示（如果有经验明细的话）
- 等 phase 数量增加后再改才有实际价值

**未来实施要点**：
- `calcExpGain(role PhaseRole, turns int, intensity, ...)` 按角色计算
- 每个 phase 结束时发放该 phase 的经验（而不是最后一起发）
- `personal_stats` 里按 phase role 分类统计
- 事件 `quest_success`/`quest_failed` 的 exp payload 改成按 phase 拆分
- 预期改动：4-5 个文件，中等复杂度

### 2.3 Phase 3 事件溯源

**目标**：事件成为状态的唯一真相源，`meta.json` 退化为快照/读模型。

**为什么暂缓**：
- 范式级重构，改动面极大（几乎所有模块）
- 与"不要覆盖其他人的修改"约束冲突
- 当前文件系统存储 + 内存事件总线的架构还够用，没有到必须换的程度
- 等数据量上来（>1000 quests）或并发需求上来后再评估

**未来实施要点**（高优先级先做的几个）：
1. 设计完整事件 schema（生命周期类、阶段类、评审类、工作区类、预算类）
2. 实现 `EventStore` 接口 + 文件系统实现
3. Quest 聚合支持从事件回放重建状态
4. 所有状态变更改为"应用事件 + 追加事件 + 更新快照"的三段式
5. SSE 断线重连按事件版本补发
6. meta.json 变成快照（可从事件重建）

预期：**独立的 Phase 3 大版本**，需要专门规划和排期。

---

## 三、什么时候启动这些暂缓项

### 启动 #45 + #46 的触发条件

满足以下任意一条即可启动：
- 需要加第 3 个 agent 阶段（如 QA 质检师、产品经理评审）
- 需要支持不同 quest type 走不同管道（如纯执行 quest 只有 1 个 phase）
- 工具/技能的分类逻辑变得复杂，warrior/mage 二元不够用

### 启动 Phase 3 的触发条件

满足以下任意两条以上再评估：
- 单用户 quest 数量 > 1000，列表查询性能明显下降
- 需要多进程/多节点部署，内存事件总线不够用
- SSE 断线重连丢失事件成为实际投诉点
- 崩溃后状态恢复成为刚需
- 有审计/合规要求，需要完整的不可变事件日志

---

## 四、相关文件速查

| 概念 | 文件 | 入口 |
|---|---|---|
| Phase 领域类型 | `internal/domain/quest/phase.go` | `PhaseDef` / `PhaseRun` / `PhaseRole` |
| Pipeline 默认值 | `internal/domain/quest/pipeline.go` | `DefaultPipeline()` |
| Quest 聚合 phase 方法 | `internal/domain/quest/repository.go` | `EnsurePhases()` / `StartCurrentPhase()` / `ReworkToPhase()` |
| 状态迁移 phase 同步 | `internal/domain/quest/service.go` | 搜索 `// ===== Phase 管道同步 =====` |
| 存储层双写 | `internal/fsstore/quests.go` | `EnsurePhases()` / `SyncPhaseIDs()` / `PhaseForIdx()` |
| 存储→领域转换 | `internal/fsstore/quest_repository_adapter.go` | `phaseTaskToDomain()` / `phaseTaskFromDomain()` |
| 统一 phase 执行 | `internal/orchestrator/pipeline.go` | `runAgentPhase()` / `buildPhaseConfig()` |
| 宏循环 phase 调用 | `internal/orchestrator/macro_loop.go` | 搜索 `runAgentPhase` |
| Phase 集成测试 | `internal/domain/quest/service_phase_test.go` | `TestPhasePipeline_*` |
