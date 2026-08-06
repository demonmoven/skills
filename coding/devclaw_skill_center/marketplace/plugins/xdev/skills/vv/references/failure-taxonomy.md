# 失败分类定义（Failure Taxonomy）

## 六大失败类型

### 1. POLICY_VIOLATION — 策略违规

**定义**：违反了明确定义的编码策略或规范。

**判定规则**：
- 规则检查子系统中的 finding，除 category 包含 `scope` / `over-design` / `over-engineer` 的以外（这些归入 UNNECESSARY_COMPLEXITY）
- 硬编码密钥/token
- 禁用依赖引入
- 修改禁止目录
- go vet / go build 失败

**来源子系统**：规则检查（subsystem_1）

**示例**：
- 硬编码数据库密码
- 引入黑名单依赖
- go vet 报告 shadow variable

---

### 2. STRUCTURE_DRIFT — 结构偏移

**定义**：代码结构偏离了项目约定或最佳实践。

**判定规则**：
- 结构测试子系统中发现的所有 finding
- 配对文件未同步修改
- 新文件放置在错误目录
- 新 API 未注册到路由表
- 缺少 schema/migration 文件
- 缺少 telemetry 埋点

**来源子系统**：结构测试（subsystem_2）

**示例**：
- handler.go 修改了但 handler_test.go 未更新
- 新 handler 放在 service/ 目录
- 新 API 端点未注册路由

---

### 3. OUTCOME_FAIL — 产出失败

**定义**：Agent 产出未达成任务目标。

**判定规则**：
- 任务评测子系统中的 finding（除 category 包含 `efficiency` / `complexity` 的以外，这些归入 UNNECESSARY_COMPLEXITY）
- 回归门禁子系统中发现的所有 finding
- 编译失败
- 测试不通过
- 功能未正确实现
- 需求未满足
- 约束被违反

**来源子系统**：任务评测（subsystem_3）+ 回归门禁（subsystem_5）

**示例**：
- SPEC 要求分页但未实现
- 编译报错
- 关键测试用例失败

---

### 4. TOOL_MISUSE — 工具误用

**定义**：Agent 在执行过程中错误使用了工具。

**判定规则**：
- 轨迹评分子系统中 tool-selection 的 finding
- 用 Bash cat 替代 Read
- 用 Bash grep 替代 Grep
- 使用 rm -rf 等破坏性命令
- 权限被拒后继续重试同一操作

**来源子系统**：轨迹评分（subsystem_4）

**示例**：
- 反复使用 `cat file.go` 而非 Read 工具
- 执行了 `rm -rf /path` 而未确认

---

### 5. THRASHING — 反复震荡

**定义**：Agent 在执行过程中出现无效重复或来回修改。

**判定规则**：
- 轨迹评分子系统中 step-quality 的 finding
- 相同工具+相同参数连续调用 > 2 次
- 连续失败 > 2 次未调整策略
- 修改→撤销→再修改同一文件 > 2 次

**来源子系统**：轨迹评分（subsystem_4）

**示例**：
- 连续 5 次执行相同的 go build 命令且都失败
- 对同一行代码修改了 4 次

---

### 6. UNNECESSARY_COMPLEXITY — 不必要的复杂度

**定义**：实现方案引入了不必要的复杂性。

**判定规则**：
- 来自规则检查的 finding，其 category 包含 `scope` / `over-design` / `over-engineer`
- 来自任务评测的 finding，其 category 包含 `efficiency` / `complexity`
- 代码量不合理（简单任务写了过多代码）
- 引入了不必要的抽象层
- 添加了"以防万一"的配置

**来源子系统**：规则检查（subsystem_1）+ 任务评测（subsystem_3）

**示例**：
- 为 3 行逻辑创建了 interface + factory + strategy 模式
- 简单 CRUD 添加了分布式缓存层

---

## failure-taxonomy.json Schema

```json
{
  "POLICY_VIOLATION": 2,
  "STRUCTURE_DRIFT": 1,
  "OUTCOME_FAIL": 0,
  "TOOL_MISUSE": 0,
  "THRASHING": 0,
  "UNNECESSARY_COMPLEXITY": 1,
  "details": {
    "POLICY_VIOLATION": [
      { "finding_id": "RC-001", "subsystem": "rule_checks", "title": "..." },
      { "finding_id": "RC-003", "subsystem": "rule_checks", "title": "..." }
    ],
    "STRUCTURE_DRIFT": [
      { "finding_id": "ST-002", "subsystem": "structural_tests", "title": "..." }
    ],
    "UNNECESSARY_COMPLEXITY": [
      { "finding_id": "RC-005", "subsystem": "rule_checks", "title": "..." }
    ]
  }
}
```

## 从 Finding 到 Failure Type 的映射规则

映射规则的唯一权威定义在 `checks/failure-classifier.md` 中。编排者在 Step 6 调用 failure-classifier 执行分类。

**注意**：如果 finding 已携带 `failure_type` 字段（由子系统预设），failure-classifier 将优先使用该字段。
