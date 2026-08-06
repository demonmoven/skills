---

# Craft Stage 2: E2E 集成测试生成

## 用户输入

```text
$ARGUMENTS
```

解析参数（按顺序）：
- `spec_dir`（必需）：SPEC 文档目录路径
- `e2e_test_repo`（必需）：E2E 测试仓库目录（只读参考，测试写入工作副本）
- `idl_repo_path`（必需）：IDL 仓库本地路径，包含已变更的接口定义（Thrift/Protobuf 等）
- `output_dir`（可选）：输出目录。byted-gan-fixloop 调用时会显式传入 `$OUTPUT_DIR/craft`（默认 `.costudio/craft`）；独立调用时默认为 `<spec_dir>/../craft_output/`

**前置条件**：`<output_dir>/user_journeys.md` 必须已存在（由 craft-stage1 生成）。

---

## 概述

本阶段基于用户动线文档生成 Go 集成测试代码。测试写入工作副本（`e2e_work_copy/`），**不修改源仓库**。目标是为每条用户动线生成独立的测试函数，并确保覆盖全部 SPEC 功能点。

---

## 执行流程

### 1. 准备

- 确认 `user_journeys.md` 存在（如不存在，提示先运行 craft-stage1）
- 创建工作副本目录：`<output_dir>/e2e_work_copy/`
- 将 `e2e_test_repo` 完整复制到工作副本（用 Bash 命令 `cp -r`）

### 2. 探索测试仓库

**在生成代码前必须充分了解测试仓库的约定：**

1. 阅读 `e2e_test_repo/` 下的 `README` 或 `go.mod`，了解：
   - 使用的测试框架和依赖
   - Go module 名称（用于拼接 Package 路径）
2. 浏览测试用例目录（如 `test_cases/`），阅读已有测试文件，学习：
   - package 命名规则和目录组织方式
   - import 路径（必须与已有测试文件一致）
   - 数据构造模式和 API 调用模式
   - 断言风格（`require`/`assert`）和测试函数签名
3. 浏览 API 封装目录，找到所有可用的 API 调用函数（导出函数）
4. 查找工具函数（随机字符串生成、指针转换等辅助函数）
5. 查找已有的 JSONL 测试清单文件，了解 Package 路径格式：
   ```json
   {"Package": "<go_module_name>/<test_file_relative_path>", "Test": "<TestFunctionName>"}
   ```

### 3. 阅读 IDL 定义

**在生成测试代码前必须阅读 IDL 仓库，获取准确的接口定义：**

1. 浏览 `idl_repo_path` 下的 IDL 文件（`.thrift`、`.proto` 等）
2. 提取每个接口方法的：
   - 请求结构体：字段名、类型、是否必填
   - 响应结构体：字段名、类型、嵌套结构
   - 枚举类型及其值
3. **以 IDL 为准**：当 SPEC 文档与 IDL 定义不一致时（如字段名拼写、类型、是否必填），以 IDL 为准，因为 IDL 是代码生成的实际来源

### 4. 阅读输入文档

1. 阅读 `user_journeys.md`，提取所有动线（`## 动线` 开头的段落）
2. 如果 `<output_dir>/stage1_to_stage2_coverage.md` 存在（由 coverage-analysis 生成），读取作为参考输入——其中标注了动线未覆盖的 API 端点和覆盖不足的场景，优先为这些 API 生成测试
3. 阅读 SPEC 目录下所有文档，提取 API 接口清单（N 个接口）
4. 结合 IDL 定义交叉验证接口清单的准确性
5. 计算目标测试数量：N × 2 ~ N × 3

### 5. 生成集成测试代码

**测试数量要求（必须满足）：**
- 目标测试函数数量 ≈ N × 2 ~ N × 3（N = SPEC 中 API 接口总数）
- 每条用户动线必须有至少一个独立的测试函数
- **严禁将多条动线合并到一个测试函数中**

**必须覆盖的测试类型：**

| 类型 | 要求 |
|------|------|
| Happy Path | 每条用户动线对应一个完整正向测试 |
| Negative Tests | 无效参数、缺失字段、类型错误、越界值、空 ID、不存在的资源、重复创建 |
| 字段完整性 | Create/Get/List/Update 接口的响应字段完整性，包括嵌套字段和枚举值 |
| 功能性测试 | 过滤查询、版本管理、批量操作、调试/验证功能（如 SPEC 有描述） |
| 边界测试 | 分页边界、排序参数、空列表查询 |

**资源创建模式（重要）：**
- **优先采用内联构造请求体**：直接在 Create 请求体中构造完整数据
- **禁止**依赖查询接口获取预设数据再进行创建（测试环境可能返回空数据）
- 参考测试仓库中已有测试文件的数据构造模式

**代码质量要求：**
- 每个测试函数独立可运行（包含 setup 和 cleanup）
- 使用 defer 或测试末尾调用 Delete 进行清理
- 断言要具体，验证关键字段而非仅检查 `err == nil`
- 使用工具函数生成唯一名称避免冲突（参考已有测试）
- 测试函数命名：`TestJourney{N}_{动线名称}` 或 `TestJourney{N}_{功能点}_{场景}`

**文件组织：**
- 测试文件放在工作副本的 `test_cases/` 目录下，按功能模块组织（参考已有结构）
- 相关功能的测试可在同一 `_test.go` 文件中，但每条动线是独立 Test 函数

### 6. 生成测试清单

将所有测试用例写入 `<output_dir>/generated_test_cases.jsonl`，每行一条：

```json
{"Package": "<go_module_name>/<test_file_relative_path>", "Test": "<TestFunctionName>"}
```

**清单条目数必须等于生成的 Test 函数总数。**

### 7. Review 与迭代（最多 5 轮）

**第 1 轮**：生成初始测试代码和清单。

**第 2+ 轮（Review）**：

| 检查项 | 优先级 | 要求 |
|-------|--------|------|
| **测试数量** | 最高 | Test 函数数量 ≥ N × 2；清单条目数 = Test 函数数；有遗漏必须补充 |
| **测试有效性** | 高 | 无占位符 ID、无软断言（t.Logf 替代 t.Fatalf）、关键字段必须断言、无 TODO 注释跳过验证 |
| **SPEC 覆盖度** | 高 | 逐一对照 SPEC API 列表，每个接口至少有一个测试；Negative test 覆盖 |
| **动线对齐** | 中 | 每条用户动线有独立 Test 函数，无合并 |
| **IDL 一致性** | 高 | 请求/响应字段名、类型、枚举值与 IDL 定义严格一致（IDL 优先于 SPEC） |
| **SPEC 正确性** | 中 | 接口路径、请求参数名/类型、响应字段与 SPEC 一致 |
| **资源创建模式** | 中 | 禁止依赖查询接口获取数据创建资源 |
| **代码质量** | 中 | import 路径、函数签名、数据类型正确 |
| **兼容性** | 低 | Test 函数名符合 Go 规范；清单 Package/Test 与代码匹配 |

**无效测试特征（发现即修复）：**

1. **占位符 ID** — 传入空 ID 列表、零值 ID 或注明「需要填入真实 ID」的占位符
   - **修复**：通过 Create API 在测试内创建真实资源，使用返回的真实 ID

2. **软断言** — 使用 `t.Logf`/`t.Log` 代替 `t.Fatalf`/`t.Fatal` 进行关键字段断言
   - **修复**：将所有关键断言的 `t.Logf` 改为 `t.Fatalf` 或 `require.Equal`

3. **条件断言** — 关键响应字段断言被注释掉或使用 `if xxx != nil { t.Logf(...) }` 模式
   - **修复**：去掉条件包裹，直接用 `require.NotNil` + `require.Equal` 断言

4. **跳过注释** — 函数体内有「跳过验证」、「暂不验证」、「TODO」等注释
   - **修复**：实现实际的验证逻辑，删除 TODO 注释

5. **错误静默** — 错误发生时只 `t.Logf` 而非 `t.Fatalf`
   - **修复**：改为 `require.NoError(t, err)` 或 `t.Fatalf("unexpected error: %v", err)`

- 发现问题 → 直接修改工作副本中的测试文件，继续下一轮
- 所有检查通过 → 在输出中写入 `CONVERGED`

---

## 关键规则

- **不修改源仓库**：只在 `e2e_work_copy/` 中写入测试代码，`e2e_test_repo` 保持只读
- **每条动线独立函数**：严禁将多条动线合并为一个 Test 函数
- **最小测试数量**：N × 2（N = API 接口数），Review 时测试数量不足不得 CONVERGED
- **清单与代码同步**：`generated_test_cases.jsonl` 条目数必须与 Test 函数数严格一致

---

## 产出物

| 产出文件 | 说明 |
|---------|------|
| `<output_dir>/e2e_work_copy/` | 生成的集成测试代码（工作副本） |
| `<output_dir>/generated_test_cases.jsonl` | 测试清单（JSONL 格式） |

---

## 注意事项

- 从已有测试文件中推断正确的 import 路径和 API 调用方式，不要猜测
- 请求/响应的字段名和类型必须与 IDL 定义一致，不要从 SPEC 文档中猜测拼写
- 对于查询预设数据的接口（如模板列表），只测试接口是否正确响应，不依赖返回结果作为其他测试的前置数据
- 前置条件中的所有资源通过 Create API 在测试内创建，测试结束时通过 Delete API 清理
