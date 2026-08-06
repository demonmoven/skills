# Phase 1 / Bug Craft / Step 4 — 测试代码生成

> **使用 Agent 工具启动 subagent 执行**。主 context 把参数传给 subagent，subagent Read 本文件作为执行指南。

## 角色

你是一位 Go 测试代码编写者。任务是**根据测试需求生成可编译可运行的 Go 单元测试代码**。

## 输入

- `BUSINESS_REPO_PATH`：业务代码仓库路径
- `OUTPUT_DIR`：输出基目录

可以读取的文件：
- `$OUTPUT_DIR/craft/change_points.md`
- `$OUTPUT_DIR/craft/test_requirements.md`
- `$OUTPUT_DIR/craft/go_changed_files.txt`

## 流程

### 步骤 1: 准备工作目录

```bash
mkdir -p $OUTPUT_DIR/craft/unit_tests
```

生成的测试代码**不直接写入 BUSINESS_REPO_PATH**，而是先写到 `$OUTPUT_DIR/craft/unit_tests/`（保持与业务代码相同的目录结构）。step 5 会负责 cp 到 BUSINESS_REPO。

### 步骤 2: 探索业务代码结构

对每个改动点：

1. **Read 业务代码文件**：理解函数完整实现
2. **Read 同 package 内的其它文件**：了解 interface 定义 / struct / 包级常量
3. **Read 同 package 下已有的 `_test.go` 文件**：模仿现有的测试风格
4. **找 mock 工具**：
   - `go.mod` 里有没有 `testify/mock`、`gomock`、`mockery` 等？
   - 如果有，用对应工具生成 mock
   - 如果没有，手写 fake 实现
5. **确定 import 清单**

### 步骤 3: 为每个测试需求生成测试代码

对 `test_requirements.md` 里的每个测试需求：

1. **创建测试文件**（如果对应的 `_test.go` 不存在）：
   ```
   $OUTPUT_DIR/craft/unit_tests/<same_relative_path>/<file_stem>_test.go
   ```
   
   例：如果改动点在 `internal/handler/request.go`，生成 `$OUTPUT_DIR/craft/unit_tests/internal/handler/request_test.go`

2. **测试函数命名**：按 Go 约定
   ```go
   func TestHandleRequest_HappyPath(t *testing.T) { ... }
   func TestHandleRequest_NilRequest(t *testing.T) { ... }
   ```

3. **测试函数模板**：
   ```go
   func Test<FuncName>_<Scenario>(t *testing.T) {
       // Arrange
       ctx := context.Background()
       mockStore := &MockStore{...}
       s := &Service{store: mockStore}
       
       // Act
       result, err := s.HandleRequest(ctx, &Request{ID: "valid"})
       
       // Assert
       require.NoError(t, err)
       require.NotNil(t, result)
       require.Equal(t, "expected", result.Value)
   }
   ```

4. **使用 testify/require 做强断言**（失败立即停止）：
   ```go
   import "github.com/stretchr/testify/require"
   
   require.NoError(t, err)
   require.Equal(t, expected, actual)
   require.NotNil(t, result)
   ```

5. **Mock 依赖**：
   - 优先通过 interface 注入 fake 实现
   - 避免在测试里用 monkey patch（脆弱）

### 步骤 4: 生成 jsonl 清单

每个生成的测试写一行 jsonl 到 `$OUTPUT_DIR/craft/generated_unit_test_cases.jsonl`：

```json
{"Package":"github.com/org/repo/internal/handler","Test":"TestHandleRequest_HappyPath","File":"internal/handler/request_test.go","ChangePoint":"Service.HandleRequest","Scenario":"happy path"}
{"Package":"github.com/org/repo/internal/handler","Test":"TestHandleRequest_NilRequest","File":"internal/handler/request_test.go","ChangePoint":"Service.HandleRequest","Scenario":"nil 请求回归"}
```

每行对应一个测试函数。

### 步骤 5: 本地编译校验

```bash
# 把 unit_tests/ 下的文件临时 cp 到 BUSINESS_REPO 对应位置
cp -r $OUTPUT_DIR/craft/unit_tests/* $BUSINESS_REPO_PATH/

# 校验编译
cd $BUSINESS_REPO_PATH
go build ./... 2>&1

# 编译失败 → 分析错误 → 修复测试代码 → 再次 go build
# 最多重试 5 次

# 编译成功后,把测试文件 mv 回 unit_tests/（step 5 会正式 cp + push）
# ... 或者保留在 BUSINESS_REPO 里让 step 5 做后续的 git add/commit
```

**注意**：为简化 step 5 的逻辑，这里直接让生成的测试文件留在 BUSINESS_REPO_PATH，由 step 5 负责 git add + commit + push。同时在 `$OUTPUT_DIR/craft/unit_tests/` 保留副本供审计。

### 步骤 6: 自检

- [ ] 每个测试需求都有对应的测试函数
- [ ] 所有测试文件的 `go build ./...` 通过
- [ ] 测试函数名符合 Go 约定（`TestXxx`）
- [ ] 使用 `testify/require` 做断言
- [ ] 没有修改业务代码（只添加测试文件）
- [ ] `generated_unit_test_cases.jsonl` 每行格式正确

## 产出

- `$OUTPUT_DIR/craft/unit_tests/` 目录下的测试代码
- `BUSINESS_REPO_PATH/` 下对应位置的测试代码（未 commit）
- `$OUTPUT_DIR/craft/generated_unit_test_cases.jsonl`

## 失败处理

- 编译失败且 5 次重试无法修复 → 写入 `craft/build_errors.md`,让下一轮 phase-2 的 Fixer 处理
- 某个改动点无法生成合理测试（如依赖太复杂） → 跳过并记录在 `craft/skipped.md`
- 生成的测试与现有测试文件冲突 → 不覆盖现有测试,追加到文件末尾

## 约束

- **只生成单元测试**（bug-fixloop MVP 限制）
- 不生成 E2E 测试
- 不修改业务代码
- 不修改现有测试代码（除非追加新函数）
- 每次最多 30 个测试（按 step-3 的约束）
