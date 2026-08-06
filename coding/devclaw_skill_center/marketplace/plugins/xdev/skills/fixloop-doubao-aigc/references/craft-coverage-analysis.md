---

# Craft Coverage Analysis: 覆盖度分析

## 用户输入

```text
$ARGUMENTS
```

解析参数（按顺序）：
- `spec_dir`（必需）：SPEC 文档目录路径
- `idl_repo_path`（必需）：IDL 仓库本地路径，包含已变更的接口定义（Thrift/Protobuf 等）
- `output_dir`（可选）：输出目录，默认为 `<spec_dir>/../craft_output/`
- `mode`（可选）：分析模式，`stage1_to_stage2` 或 `stage2_to_stage3`，默认两者都执行
- `business_repo_path`（模式 B 必需）：业务代码仓库路径，用于追踪调用链分析 E2E 盲区

---

## 概述

覆盖度分析是各生成阶段之间的**桥接步骤**，分析上游产出的覆盖范围，识别下游需要补充的覆盖差距。产出的文档是**参考输入**，帮助下游阶段更精准地生成测试。

```
Stage 1 (user_journeys.md)
    ↓ coverage-analysis (stage1→stage2)
    ↓   产出: stage1_to_stage2_coverage.md
    ↓   内容: 哪些动线/API 需要 E2E 测试重点覆盖
Stage 2 (e2e_work_copy/)
    ↓ coverage-analysis (stage2→stage3)
    ↓   产出: stage2_to_stage3_coverage.md
    ↓   内容: 哪些内部逻辑是 E2E 盲区，需要单元测试
Stage 3 (unit_tests/)
```

---

## 模式 A：Stage 1 → Stage 2（动线到 E2E 覆盖分析）

**前置条件**：`<output_dir>/user_journeys.md` 已存在

### 执行步骤

#### 1. 提取动线清单

阅读 `user_journeys.md`，提取每条动线的：
- 动线编号和名称
- 涉及的 API 端点列表
- 关键验证点（断言）
- 涉及的业务场景类型（CRUD、异步、批量等）

#### 2. 提取 SPEC & IDL API 清单

阅读 SPEC 目录下所有文档和 `idl_repo_path` 下的 IDL 文件，提取完整的 API 接口列表：
- API 路径 + HTTP 方法
- 功能描述
- 请求/响应关键字段（以 IDL 定义为准）

#### 3. 分析覆盖差距

构建覆盖矩阵：

| API 端点 | 动线覆盖 | 建议的 E2E 测试场景 |
|---------|---------|-------------------|
| POST /api/v1/xxx | 动线 1, 3 | Happy path, 必填字段缺失, 重复创建 |
| GET /api/v1/xxx | 动线 2 | 分页边界, 过滤条件, 空列表 |
| ... | 未覆盖 | **需要新增动线或直接补充 E2E 测试** |

**重点识别**：
- 动线未覆盖的 API 端点
- 只有 Happy Path 覆盖但缺少 Negative 场景的端点
- 异步操作缺少状态轮询场景
- 批量操作缺少边界测试（空列表、超大批次）

#### 4. 输出报告

写入 `<output_dir>/stage1_to_stage2_coverage.md`：

```markdown
# Stage 1 → Stage 2 覆盖度分析

## 覆盖矩阵

| API 端点 | HTTP 方法 | 动线覆盖 | E2E 测试建议 |
|---------|---------|---------|------------|

## 未覆盖的 API 端点
- [列出动线完全未覆盖的 API]

## 覆盖不足的场景
- [列出只有 Happy Path 但缺少 Negative/边界测试的场景]

## E2E 测试重点建议
1. [按优先级列出建议]

ANALYSIS_COMPLETE
```

---

## 模式 B：Stage 2 → Stage 3（E2E 到单元测试覆盖分析）

**前置条件**：`<output_dir>/e2e_work_copy/` 已存在，`business_repo_path` 已提供

### 执行步骤

#### 1. 分析 E2E 测试覆盖范围

阅读 `e2e_work_copy/` 中所有测试文件，提取：
- 每个 Test 函数调用的 API 端点
- 验证的响应字段
- 覆盖的业务场景

#### 2. 追踪业务代码调用链

从 E2E 测试的 API 端点出发，沿调用链追踪到业务代码内部：

```
API 端点 → Handler → Service → DAO → Convertor
```

对于每个 Service 函数，识别：
- 条件分支数量（if/switch）
- E2E 覆盖了哪些分支
- 哪些分支是 E2E 盲区

#### 3. 识别单元测试目标

**E2E 盲区类型**：

| 类型 | 说明 | 示例 |
|------|------|------|
| 未走到的分支 | E2E 只走了 if 的 true 分支 | 参数为空时的 fallback 逻辑 |
| 错误处理路径 | E2E 只测试成功路径 | DB 查询失败、外部服务超时 |
| Convertor 边界 | E2E 不验证内部转换细节 | nil 输入转换、空 slice 转换、字段遗漏 |
| 计算逻辑 | E2E 只验证最终结果 | 中间步骤的数值精度、聚合逻辑 |
| 校验函数 | E2E 只传有效参数 | 各种无效输入的校验结果 |

#### 4. 输出报告

写入 `<output_dir>/stage2_to_stage3_coverage.md`：

```markdown
# Stage 2 → Stage 3 覆盖度分析

## E2E 覆盖状态

| 业务函数 | 所在包 | E2E 覆盖状态 | 单元测试建议 |
|---------|-------|------------|------------|

## E2E 盲区（需要单元测试）

### 优先级 1：Convertor 函数
- [列出 PO2DO/DO2DTO 函数及其边界场景]

### 优先级 2：多分支 Service 函数
- [列出条件分支 >= 3 的函数]

### 优先级 3：错误处理路径
- [列出 E2E 未覆盖的 error handling]

### 优先级 4：计算/校验逻辑
- [列出复杂计算或数据校验函数]

## 不建议单元测试的函数
- [列出简单 getter/setter 或 E2E 已充分覆盖的函数]

ANALYSIS_COMPLETE
```

---

## 关键规则

- **分析必须基于实际代码**：追踪调用链时必须读取实际文件，不能猜测
- **产出是参考输入**：下游阶段可以参考但不必机械遵循
- **使用绝对路径**：报告中引用源文件时使用绝对路径
- **不修改任何文件**：覆盖度分析是只读操作

---

## 产出物

| 产出文件 | 说明 |
|---------|------|
| `stage1_to_stage2_coverage.md` | 动线到 E2E 的覆盖差距报告 |
| `stage2_to_stage3_coverage.md` | E2E 到单元测试的覆盖差距报告 |


## pytest 场景下的已有测试基线读取

Stage 0 Step 0.2 / Step 0.5 的覆盖度分析需要读"已有测试"作为对比基线。pytest 场景下：

1. 对每个目标模块目录（`$TEST_REPO_PATH/testcases/<module>/`）：
   - 列出所有 `test_*.py`
   - 对每个 .py：
     - 提取 `@test.runtime(...)` 装饰器里的 `file=` → 关联 `test_data/<file>.{runtime}.json`
     - 读 JSON 里每条 case 的 `title`（或 `expected` 字段如果 title 缺失）
     - 得到"已覆盖行为"清单
2. 合并所有模块的已覆盖清单，对比 `user_journeys.md`：
   - 若 journey.目的 已被某条 case 的 title 或 expected 匹配 → 标为"已覆盖"
   - 否则 → 新增候选
3. 同类失败反复出现 → 可能是覆盖度假象（测的是 happy path，缺边界）
