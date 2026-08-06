---

# Craft Stage 3: 单元测试生成

## 用户输入

```text
$ARGUMENTS
```

解析参数（按顺序）：
- `spec_dir`（必需）：SPEC 文档目录路径
- `business_repo_path`（必需）：业务代码仓库路径（after_dev 版本）
- `idl_repo_path`（必需）：IDL 仓库本地路径，包含已变更的接口定义（Thrift/Protobuf 等）
- `output_dir`（可选）：输出目录。gan-fixloop 调用时会显式传入 `$OUTPUT_DIR/craft`（默认 `.costudio/craft`）；独立调用时默认为 `<spec_dir>/../craft_output/`

**前置条件**：
- `<output_dir>/user_journeys.md` 必须已存在（由 craft-stage1 生成）
- `<output_dir>/e2e_work_copy/` 必须已存在（由 craft-stage2 生成）

---

## 概述

本阶段生成**补充 E2E 的单元测试**，聚焦 E2E 无法覆盖的代码内部逻辑。核心原则是**fitting（适配）**：先理解 E2E 已经覆盖了什么，然后只为 E2E 盲区生成单元测试，避免重复。

**4 个聚焦方向**：

| 方向 | 说明 | 典型场景 |
|------|------|---------|
| 内部分支路径 | if/switch 中 E2E 未走到的分支 | 特定条件组合、fallback 逻辑 |
| 边界值/异常处理 | 极端输入、空值、越界 | nil input、空 slice、max int |
| 计算逻辑 | 复杂转换、聚合、公式 | 价格计算、权限合并、状态机 |
| 错误返回路径 | error handling 分支 | DB 错误、外部服务超时、数据校验失败 |

---

## 执行流程

### 1. 准备

- 确认前置条件文件存在
- 创建输出目录：`<output_dir>/unit_tests/`
- 如果 `<output_dir>/stage2_to_stage3_coverage.md` 存在（由 coverage-analysis 生成），读取作为参考输入

### 2. 理解 E2E 覆盖范围

**在生成单元测试前必须先了解 E2E 已经覆盖了什么：**

1. 阅读 `<output_dir>/e2e_work_copy/` 中所有测试文件
2. 提取每个 Test 函数测试的 API 端点和验证点
3. 如果有 `stage2_to_stage3_coverage.md`，直接读取覆盖差距分析
4. 构建 E2E 覆盖清单：

   | API 端点 | E2E 覆盖的场景 | 未覆盖的内部逻辑 |
   |---------|--------------|----------------|

### 3. 阅读 IDL 定义

阅读 `idl_repo_path` 下的 IDL 文件（`.thrift`、`.proto` 等），了解接口定义中的数据结构（字段名、类型、枚举值），以便在生成单元测试时使用准确的类型和字段名。特别关注 Convertor 函数涉及的结构体映射关系。

### 4. 探索业务代码仓库

1. 阅读 `business_repo_path` 的 `go.mod`，获取 module 名称
2. 浏览主要业务逻辑目录：
   - `domain/` — 领域逻辑（Service、Convertor）
   - `infra/` — 基础设施（DAO）
   - `application/` — 应用层
3. 重点识别以下函数：
   - 包含多个 if/switch 分支的函数
   - Convertor 函数（PO2DO、DO2DTO）— 可能遗漏字段映射
   - 数据校验函数 — 多种校验规则
   - 错误处理密集的函数

### 5. 确定单元测试目标

基于 E2E 覆盖清单和业务代码探索，确定需要单元测试的函数列表：

```markdown
| 目标函数 | 所在包 | 测试理由（E2E 盲区） | 优先级 |
|---------|-------|-------------------|-------|
```

**选择标准**：
- E2E 未覆盖的内部分支 ✓
- 复杂计算/转换逻辑 ✓
- 多种错误处理路径 ✓
- 简单的 getter/setter ✗（不需要单元测试）
- E2E 已充分覆盖的逻辑 ✗（避免重复）

### 6. 生成单元测试

**package 匹配要求（关键）**：

每个测试文件的 `package` 声明**必须与被测函数所在包一致**（同包测试）：

```go
// 被测文件：domain/workflow/service.go
// package workflow

// 测试文件：domain/workflow/service_unit_test.go
// package workflow  ← 必须一致
```

**如果不确定包名**，先 grep 确认：
```bash
head -5 <business_repo_path>/<target_file>.go | grep "^package"
```

**测试规范**：

- **文件命名**：`<source_file>_unit_test.go`，放在同一目录
- **函数命名**：`Test<FunctionName>_<Scenario>`
- **Table-driven tests**：使用 `[]struct` 定义测试用例表
- **并行执行**：每个 subtest 调用 `t.Parallel()`
- **Mock 框架**：使用 `testify/mock` 隔离外部依赖（DB、HTTP client 等）
- **断言**：使用 `testify/assert` 或 `testify/require`

```go
func TestConvertWorkflowPO2DO_NilInput(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name     string
        input    *gormmodel.Workflow
        expected *entity.WorkflowDetail
    }{
        {
            name:     "nil workflow returns nil",
            input:    nil,
            expected: nil,
        },
        {
            name:     "empty nodes returns workflow with empty node list",
            input:    &gormmodel.Workflow{ID: 1},
            expected: &entity.WorkflowDetail{ID: 1, Nodes: []entity.NodeDetail{}},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            result := WorkflowPO2DO(tt.input, nil)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

**Mock 使用原则**：
- Mock 外部依赖（数据库、HTTP 客户端、RPC 调用）
- 不要 Mock 被测函数内部调用的同包函数
- Mock interface 必须与实际 interface 签名一致

**Go 命名规范**：
- 代码生成目录（`kitex_gen/`、`gorm_gen/` 等）使用 PascalCase
- 不确定时 `grep` 确认实际命名

### 7. 生成测试清单

将所有单元测试写入 `<output_dir>/generated_unit_test_cases.jsonl`：

```json
{"Package": "<module_name>/<package_path>", "Test": "<TestFunctionName>"}
```

### 8. Review 与迭代（最多 5 轮）

**第 1 轮**：生成初始测试代码和清单。

**第 2+ 轮（Review）**：

| 检查项 | 优先级 | 要求 |
|-------|--------|------|
| **Fitting 检查** | 最高 | 每个测试覆盖的是 E2E 盲区（内部分支/边界值/错误路径），不与 E2E 重复 |
| **Alignment 检查** | 高 | 每个测试与 after_dev 实际代码行为一致（不是测试臆想的行为） |
| **Depth 检查** | 高 | 测试验证的是真实业务逻辑（分支/计算/错误处理），不是无意义的赋值检查 |
| **编译检查** | 中 | import 路径正确、类型匹配、mock interface 签名正确 |

**Fitting 检查详细标准**：
- 是否覆盖了 E2E 未覆盖的分支？（查看 if/switch 的另一个分支）
- 是否覆盖了边界值？（空值、极大值、零值）
- 是否与某个 E2E 测试重复？（如果重复，删除单元测试）
- 被测函数是否值得单元测试？（简单 getter 不需要）

- 发现问题 → 修改测试代码，继续下一轮
- 所有检查通过 → 在输出中写入 `CONVERGED`

---

## 关键规则

- **fitting 优先**：不重复 E2E 已覆盖的场景，只覆盖 E2E 盲区
- **package 必须匹配**：测试文件的 package 声明与被测文件一致
- **不修改业务代码**：只生成测试文件，不修改 `business_repo_path` 中的源代码
- **Mock 外部依赖**：DB、HTTP、RPC 等外部依赖使用 mock，不启动真实服务
- **最小测试数量**：目标函数数 × 2（每个函数至少 2 个测试场景）

---

## 产出物

| 产出文件 | 说明 |
|---------|------|
| `<output_dir>/unit_tests/` | 生成的单元测试文件 |
| `<output_dir>/generated_unit_test_cases.jsonl` | 单元测试清单（JSONL 格式） |

---

## 注意事项

- 代码生成目录（`kitex_gen/`、`gorm_gen/`）中的代码不需要写单元测试
- Convertor 函数（PO2DO、DO2DTO）是单元测试的重点目标，容易遗漏字段映射
- 如果被测函数使用了 `context.Context`，测试中传入 `context.Background()` 即可
- 测试文件最终需要复制到 `business_repo_path` 对应目录才能运行（由调用方负责）
