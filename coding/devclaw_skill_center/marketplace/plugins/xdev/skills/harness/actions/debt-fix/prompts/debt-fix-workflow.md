# Harness Debt Fix — 技术债识别与治理

> **本文件是 prompt 指令**，由 `<skill_dir>/actions/debt-fix/debt-fix.md` 加载。原属于 `harness-bootstrap` 项目的 `harness-debt-fix` 演进 skill，已合并对应的 `references/debt-fix-workflow.md` 详细参考（见文件末尾"附录"段）。

本技能不仅识别架构级技术债，还会 **实际修改代码完成修复**。每次运行最多处理 3 个技术债条目，逐个修复、逐个 commit、逐个验证，确保每次变更都是安全的、可回滚的。

与旧的 tech-debt-checker（仅识别 + 记录）不同，harness-debt-fix 覆盖从扫描到修复的完整闭环：发现问题 → 选择目标 → 修改代码 → 运行测试 → 提交变更 → 更新 debt-log。

## 何时使用

| 场景 | 触发方式 | 说明 |
|------|----------|------|
| 定期触发 | 用户手动调用（触发词：`技术债治理`、`debt fix`、`tech debt`） | 对仓库做周期性技术债盘点和修复 |
| 开发过程中 | 开发新功能时发现与已有逻辑重复 | 在当前分支顺手修复，避免债务累积 |
| ExecPlan 完成后 | ExecPlan 执行完毕，回头清理引入的技术债 | 新功能落地后立即收敛，防止技术债沉淀 |
| 用户主动指定 | 用户指定具体文件或模块要求检查和修复 | 针对性治理已知的问题区域 |

## 检查规则

以下规则与语言无关，描述的是架构级问题模式。检测时需要 Agent **理解代码语义**，不能仅靠正则匹配。

### 规则 1：重复持久化（duplicate-persistence）

**严重级别**：高

多个结构体/类承载了语义相同或高度重叠的持久化数据，导致数据源不唯一、更新时容易遗漏。

**检测方式**：
1. 扫描所有持久化结构体（带序列化标注的 struct/class/dataclass），提取字段名和字段类型
2. 查找不同结构中语义相同字段的重叠情况（字段名相似度 > 70% 或语义等价）
3. 查找同一目录下多个数据文件的读写入口，对比存储数据是否有交集

**修复方向**：合并为单一结构体和单一文件，作为该数据域的唯一事实来源（Single Source of Truth）。保留一个主结构体，将其他结构体改为引用或派生。

### 规则 2：同一功能多处独立实现（divergent-impl）

**严重级别**：高

同一个业务功能存在多个独立实现，函数签名不同但最终目的相同，维护成本翻倍且行为容易不一致。

**检测方式**：
1. 搜索功能语义相似的函数名（如 `loadConfig` / `readConfig` / `getConfiguration`）
2. 搜索对同一文件路径/数据表/API 的读写是否分散在多个不相关的函数中
3. 搜索是否有回退逻辑（fallback）——通常暗示两个数据源本应是同一个

**修复方向**：收敛为单一入口函数，其他调用点改为调用该入口。如果存在细微差异，通过参数化或策略模式统一。

### 规则 3：跨层数据模型分裂（model-divergence）

**严重级别**：中-高

不同层（前后端、服务间、存储层与业务层）的数据模型承载同一份数据，但字段命名、类型、可选性不一致，导致层间转换逻辑脆弱。

**检测方式**：
1. 对比不同层的数据结构字段是否一一对应（名称、类型、可选性）
2. 查找是否有层自行定义了另一层已暴露的返回类型（而非复用）
3. 查找类型断言（`as any`、`type: ignore`、强制类型转换）的使用——通常是模型不一致的症状

**修复方向**：尽可能使用自动生成的类型或共享的类型定义。如果层间确有差异需求，通过明确的 mapper/transformer 函数桥接，而非隐式忽略类型不匹配。

### 规则 4：配置/状态散落多文件（scattered-state）

**严重级别**：中

配置或状态信息散落在多个独立文件中，没有明确的边界划分，读取决策需要跨多个文件聚合信息。

**检测方式**：
1. 搜索文件读写涉及的路径，绘制 文件→结构→函数 的映射关系
2. 查找是否有函数需要读取多个状态文件才能做出单一决策
3. 查找是否存在非原子操作序列（先写 A 文件、再写 B 文件，中间失败会导致状态不一致）

**修复方向**：明确每个文件的职责边界，合并强耦合的状态到同一文件。对于必须分开的状态，引入统一的读取层（facade）。

### 规则 5：相似流程未抽象（unabstracted-flow）

**严重级别**：中

多个代码路径执行高度相似的流程（步骤序列相同，仅参数或细节不同），但各自独立编写，修改一处时容易遗漏其他。

**检测方式**：
1. 对比不同模式/分支的初始化函数或处理流程，识别结构相似但细节不同的步骤序列
2. 查找多个函数中出现相同的"步骤模式"（如：验证→获取→转换→保存）
3. 查找 copy-paste 痕迹（大段相似代码块，仅个别变量名不同）

**修复方向**：提取公共流程为模板方法（Template Method）或 pipeline，将差异部分参数化或通过回调注入。

## 工作流

整个工作流分为 4 个阶段，严格按序执行。

### 阶段 1：扫描与识别

1. **确定扫描范围**：默认扫描主要源码目录（排除 vendor/node_modules/生成代码），用户可指定特定模块或文件
2. **逐规则分析**：对上述 5 条规则逐一执行检测，每条规则需要 Agent 理解代码语义
3. **生成扫描报告**：结构化报告，每个发现包含：
   - 规则编号和名称
   - 严重级别
   - 置信度（高/中/低）
   - 涉及的文件路径和行号
   - 问题描述
   - 建议修复方向
4. **写入 debt-log**：将所有发现追加到 `docs/quality/debt-log.md`（如果该文件存在），状态标记为 `open`

### 阶段 2：选择修复目标

1. **从扫描报告中选取最多 3 个条目** 进行修复，优先选择：
   - 严重级别高的
   - 置信度高的
   - 修复范围可控的（涉及文件数少、影响面明确）
2. **向用户确认**：列出选中的条目，说明修复方案和预期影响，等待用户确认
3. **低置信度条目需显式批准**：置信度为"低"的发现，必须在确认时明确提示用户该条目置信度较低，获得用户显式同意后才可修复

### 阶段 3：逐个修复

对每个选中的条目，执行以下循环：

```
┌─ 分析 ──→ 修改代码 ──→ 验证（测试+lint）──→ 提交 commit ──→ 更新 debt-log ─┐
│                              │                                                 │
│                         验证失败 → 回滚变更 → 记录失败原因 → 跳过该条目         │
│                                                                                │
└────────────────────── 下一个条目 ◄──────────────────────────────────────────────┘
```

具体步骤：

1. **分析**：深入理解问题代码的上下文和依赖关系，确定修改范围
2. **修改代码**：执行实际的代码修改（重构、合并、提取等）
3. **验证**：运行项目的测试套件和 lint 检查，确保修改未引入回归
4. **提交 commit**：使用规范的 commit message 格式提交（见本文件末尾"附录：debt-fix workflow 详细参考"段）
5. **更新 debt-log**：将 debt-log 中对应条目的状态从 `open` 更新为 `resolved`，记录修复方式和日期

**验证失败处理**：如果测试或 lint 未通过，立即回滚本次修改，记录失败原因，跳过该条目继续处理下一个。不允许提交未通过验证的代码。

### 阶段 4：分支策略

根据触发场景选择不同的分支策略：

| 触发场景 | 分支策略 | 说明 |
|----------|----------|------|
| 定期触发 | 创建独立分支 `debt-fix/{date}` + 提交 MR | 独立的技术债修复批次，方便 review |
| ExecPlan 完成后 | 跟随当前分支 | 作为 ExecPlan 收尾工作的一部分 |
| 开发过程中 | 跟随当前分支 | 顺手修复，与功能开发一起提交 |
| 用户主动指定 | 按用户要求 | 用户指定则遵从，未指定则创建独立分支 |

## 安全约束

1. **每次最多修复 3 个条目**：控制单次变更的范围，降低引入问题的风险
2. **逐条目独立 commit**：每修复一个技术债条目，单独提交一个 commit，确保可独立回滚
3. **测试验证门禁**：每个 commit 前必须通过测试和 lint 检查，未通过则回滚该条目的所有修改
4. **不修改文档**：本技能只修改代码文件，不修改文档文件（debt-log 除外）。文档更新由 harness-evolve 负责
5. **置信度标注**：扫描报告中的每个发现必须标注置信度（高/中/低），低置信度条目需用户显式批准才可修复

## 参考

详细的 commit 格式、扫描报告模板、debt-log 写入格式和完整工作流示例见本文件末尾"附录：debt-fix workflow 详细参考"段。


---

# 附录：debt-fix workflow 详细参考

# Debt Fix Workflow — 详细工作流参考

本文档是 harness-debt-fix skill 的详细参考，包含 commit 格式、报告模板、debt-log 写入格式和完整工作流示例。

## Commit Message 格式

每个技术债修复使用以下 commit message 格式：

```
fix(debt): {简要描述修复内容} [TD-{nnn}]

Rule: {规则名称}
Area: {涉及的模块/目录}
Verification: {验证方式，如 "all tests pass", "lint clean"}
```

### 示例

```
fix(debt): merge overlapping Config and Settings structs into unified AppConfig [TD-012]

Rule: duplicate-persistence
Area: internal/config/
Verification: all tests pass, lint clean
```

```
fix(debt): consolidate loadUserProfile and fetchUserData into single entry point [TD-015]

Rule: divergent-impl
Area: src/services/user/
Verification: all tests pass, lint clean
```

### 字段说明

| 字段 | 说明 |
|------|------|
| `fix(debt):` | 固定前缀，标识这是技术债修复 commit |
| `{简要描述}` | 一句话说明做了什么修复，使用祈使语气 |
| `[TD-{nnn}]` | debt-log 中的条目编号，建立 commit 与 debt-log 的追溯关系 |
| `Rule:` | 触发的检查规则名称 |
| `Area:` | 修改涉及的主要目录或模块 |
| `Verification:` | 验证通过的方式 |

## 扫描报告格式

阶段 1 完成后，生成以下格式的扫描报告：

```markdown
# 技术债扫描报告

扫描时间: YYYY-MM-DD HH:MM
扫描范围: {目录或模块列表}
扫描者: harness-debt-fix

## 总览

| 规则 | 发现数 | 高严重 | 中严重 | 高置信 | 中置信 | 低置信 |
|------|--------|--------|--------|--------|--------|--------|
| duplicate-persistence | N | N | N | N | N | N |
| divergent-impl | N | N | N | N | N | N |
| model-divergence | N | N | N | N | N | N |
| scattered-state | N | N | N | N | N | N |
| unabstracted-flow | N | N | N | N | N | N |
| **合计** | **N** | **N** | **N** | **N** | **N** | **N** |

## 详细发现

### 发现 1: {简要标题}

- **规则**: {规则名称}
- **严重级别**: 高/中-高/中
- **置信度**: 高/中/低
- **涉及文件**:
  - `path/to/file1.go:42` — {文件在问题中的角色}
  - `path/to/file2.go:78` — {文件在问题中的角色}
- **问题描述**: {详细说明为什么这是技术债，当前代码有什么问题}
- **建议修复方向**: {具体的修复策略}
- **预估影响范围**: {修改会影响哪些模块/功能}

### 发现 2: ...

(每个发现重复上述格式)

## 修复建议

建议优先修复以下条目（按严重级别×置信度排序）：

1. 发现 {N}: {标题} — 严重级别 {X}, 置信度 {Y}
2. 发现 {M}: {标题} — 严重级别 {X}, 置信度 {Y}
3. 发现 {K}: {标题} — 严重级别 {X}, 置信度 {Y}
```

## debt-log.md 写入格式

### 新增条目（open 状态）

扫描发现的技术债写入 debt-log 时使用以下格式：

```markdown
| TD-{nnn} | {模块/目录} | {规则名称} | {问题简述} | {严重级别} | {建议修复方向} | open |
```

在 debt-log.md 的表格中追加新行。TD 编号通过查看已有条目的最大编号 +1 确定。

### 已解决条目（resolved 状态）

修复完成后，将对应条目更新为：

```markdown
| TD-{nnn} | {模块/目录} | {规则名称} | {问题简述} | {严重级别} | ~~{原建议}~~ 已修复: {实际修复方式} ({YYYY-MM-DD}) | resolved |
```

修改内容：
1. 在"Suggested Fix"列中，将原建议加删除线，追加实际修复方式和日期
2. 将 Status 列从 `open` 改为 `resolved`

### 跳过条目（skipped 状态）

如果修复失败被跳过，将对应条目更新为：

```markdown
| TD-{nnn} | {模块/目录} | {规则名称} | {问题简述} | {严重级别} | {原建议} — 修复失败: {失败原因} ({YYYY-MM-DD}) | skipped |
```

## 完整工作流示例

以下是一个端到端的示例，展示从扫描到修复的完整过程。

### 背景

仓库是一个 Go 后端服务，用户执行 `技术债治理` 命令触发定期扫描。

### 阶段 1：扫描与识别

Agent 扫描 `internal/` 目录，逐规则检测，发现 4 个技术债条目：

```
扫描结果：

发现 1 [duplicate-persistence, 高, 置信度: 高]
  internal/config/app_config.go:AppConfig 与 internal/config/settings.go:Settings
  两个结构体都包含 DatabaseURL, CacheAddr, LogLevel 等 12 个重叠字段
  → 合并为单一 AppConfig 结构体

发现 2 [divergent-impl, 高, 置信度: 高]
  internal/user/loader.go:LoadUser() 与 internal/api/handlers/user.go:fetchUserFromDB()
  两个函数都查询 users 表并组装 User 对象，但字段处理逻辑略有差异
  → 收敛为 internal/user/loader.go:LoadUser()，handler 调用该函数

发现 3 [unabstracted-flow, 中, 置信度: 中]
  internal/order/create.go:CreateOrder() 与 internal/order/refund.go:CreateRefund()
  两个流程都执行：验证→查库存→计算金额→写事务→发通知，80% 步骤相同
  → 提取公共流程为 OrderPipeline，差异部分通过策略回调注入

发现 4 [scattered-state, 中, 置信度: 低]
  internal/runtime/state.go 和 internal/runtime/flags.go
  运行时状态分散在两个文件中，但各自职责尚算清晰
  → 建议合并，但置信度低，可能是合理的职责分离
```

Agent 将 4 个发现写入 `docs/quality/debt-log.md`：

```markdown
| TD-023 | internal/config/ | duplicate-persistence | AppConfig 与 Settings 结构体 12 字段重叠 | 高 | 合并为单一 AppConfig | open |
| TD-024 | internal/user/, internal/api/ | divergent-impl | LoadUser 和 fetchUserFromDB 功能重复 | 高 | 收敛为 LoadUser 单一入口 | open |
| TD-025 | internal/order/ | unabstracted-flow | CreateOrder 和 CreateRefund 流程 80% 重复 | 中 | 提取 OrderPipeline | open |
| TD-026 | internal/runtime/ | scattered-state | 运行时状态分散在 state.go 和 flags.go | 中 | 合并为单一状态文件 | open |
```

### 阶段 2：选择修复目标

Agent 从 4 个发现中选择 3 个（最多 3 个限制）：

```
建议修复目标（按优先级排序）：

1. TD-023 [duplicate-persistence, 高, 置信度: 高]
   合并 AppConfig 和 Settings 结构体
   影响范围：internal/config/ 及所有引用 Settings 的文件

2. TD-024 [divergent-impl, 高, 置信度: 高]
   收敛 LoadUser 和 fetchUserFromDB
   影响范围：internal/user/loader.go, internal/api/handlers/user.go

3. TD-025 [unabstracted-flow, 中, 置信度: 中]
   提取 OrderPipeline 统一订单和退款流程
   影响范围：internal/order/

跳过：
- TD-026 [散落状态, 中, 置信度: 低] — 置信度低，当前职责分离可能是合理的

是否确认修复以上 3 个条目？
```

用户确认后进入阶段 3。

### 阶段 3：逐个修复

#### 修复 TD-023：合并重复持久化结构体

```
步骤 1 — 分析：
  - AppConfig 有 18 个字段，Settings 有 15 个字段，12 个重叠
  - Settings 独有的 3 个字段（FeatureFlags, ABTestConfig, DebugMode）可并入 AppConfig
  - 有 8 个文件引用了 Settings 结构体

步骤 2 — 修改代码：
  - 将 Settings 独有的 3 个字段移入 AppConfig
  - 删除 settings.go 文件
  - 更新 8 个引用文件：Settings → AppConfig
  - 更新 config 加载逻辑：合并两个文件的读取为单一入口

步骤 3 — 验证：
  $ go test ./internal/config/... → PASS (12 tests)
  $ go test ./... → PASS (247 tests)
  $ golangci-lint run → clean

步骤 4 — 提交：
  fix(debt): merge overlapping Config and Settings structs into unified AppConfig [TD-023]

  Rule: duplicate-persistence
  Area: internal/config/
  Verification: all tests pass (247), lint clean

步骤 5 — 更新 debt-log：
  TD-023 状态: open → resolved
  修复方式: "已修复: 合并 Settings 字段到 AppConfig, 删除 settings.go (2026-04-02)"
```

#### 修复 TD-024：收敛重复用户加载实现

```
步骤 1 — 分析：
  - LoadUser() 返回完整 User 对象，包含所有关联数据
  - fetchUserFromDB() 只返回基础字段，缺少 roles 和 preferences
  - handler 中调用 fetchUserFromDB 后还有额外的 role 查询逻辑

步骤 2 — 修改代码：
  - 扩展 LoadUser() 增加 options 参数控制加载深度
  - handler 改为调用 LoadUser(ctx, userID, WithBasicFields())
  - 删除 fetchUserFromDB() 函数

步骤 3 — 验证：
  $ go test ./internal/user/... → PASS (18 tests)
  $ go test ./internal/api/... → PASS (34 tests)
  $ go test ./... → PASS (247 tests)
  $ golangci-lint run → clean

步骤 4 — 提交：
  fix(debt): consolidate fetchUserFromDB into LoadUser with field options [TD-024]

  Rule: divergent-impl
  Area: internal/user/, internal/api/handlers/
  Verification: all tests pass (247), lint clean

步骤 5 — 更新 debt-log：
  TD-024 状态: open → resolved
  修复方式: "已修复: 删除 fetchUserFromDB, 扩展 LoadUser 支持 options 控制加载深度 (2026-04-02)"
```

#### 修复 TD-025：提取公共订单流程

```
步骤 1 — 分析：
  - CreateOrder 和 CreateRefund 共享 5 个步骤中的 4 个
  - 差异仅在"计算金额"步骤（正向 vs 逆向）和"通知模板"

步骤 2 — 修改代码：
  - 创建 internal/order/pipeline.go，定义 OrderPipeline 结构和 Step 接口
  - 将公共步骤（validate, checkInventory, writeTransaction, notify）提取到 pipeline
  - CreateOrder 和 CreateRefund 改为构造 pipeline 并注入各自的 calculateAmount 实现

步骤 3 — 验证：
  $ go test ./internal/order/... → FAIL
    order_test.go:45: TestCreateOrder expected direct DB call, got pipeline invocation
  
  ⚠ 测试失败！回滚所有修改。
  $ git checkout -- internal/order/
  
  失败原因：测试中有对内部实现细节的断言（mock 了直接的 DB 调用），
  需要先更新测试才能重构。标记为 skipped。

步骤 5 — 更新 debt-log：
  TD-025 状态: open → skipped
  原因: "修复失败: 现有测试耦合了内部实现细节，需先重构测试 (2026-04-02)"
```

### 阶段 4：分支策略与收尾

本次为定期触发，因此使用独立分支策略：

```
1. 在开始前已创建分支：debt-fix/2026-04-02
2. 分支上共 2 个 commit（TD-023, TD-024 修复成功，TD-025 跳过）
3. 推送分支并创建 MR：
   标题: fix(debt): resolve 2 architecture-level tech debts (TD-023, TD-024)
   描述: 包含扫描报告摘要、修复详情、跳过原因

最终统计：
  - 扫描发现: 4 个技术债
  - 选择修复: 3 个
  - 成功修复: 2 个 (TD-023, TD-024)
  - 修复失败: 1 个 (TD-025, 测试耦合问题)
  - 未处理:   1 个 (TD-026, 置信度低，跳过)
```

## 验证失败处理

当修复某个条目后测试或 lint 未通过时，执行以下流程：

### 1. 立即回滚

```bash
# 回滚该条目的所有未提交修改
git checkout -- .
# 如果已有 staged 文件
git reset HEAD .
```

回滚必须将代码恢复到该条目修复前的状态，不留任何残余修改。

### 2. 记录失败原因

在 debt-log 中将对应条目标记为 `skipped`，详细记录：
- 具体哪个测试/lint 规则失败
- 失败的根本原因分析（不是简单地记录报错信息）
- 是否有前置条件需要先解决（如测试需要重构）

### 3. 继续处理下一个条目

回滚并记录后，立即继续处理选中列表中的下一个条目。不要因为一个条目失败就终止整个修复流程。

### 4. 最终报告中汇总

在修复流程结束后的报告中，将所有失败条目集中列出：

```markdown
## 修复失败条目

| TD 编号 | 规则 | 失败原因 | 建议后续操作 |
|---------|------|----------|-------------|
| TD-{nnn} | {规则} | {失败原因} | {下次如何处理} |
```

这些条目在 debt-log 中保持 `skipped` 状态，等待前置条件满足后在后续的 debt-fix 运行中重新处理。
