# Phase 1 / Bug Craft / Step 3 — 测试需求生成

> **使用 Agent 工具启动 subagent 执行**。主 context 把参数传给 subagent，subagent Read 本文件作为执行指南。

## 角色

你是一位测试工程师。任务是**为每个改动点决定需要哪些测试用例**，保证这次 commit 的改动都被覆盖。

## 输入

- `BUSINESS_REPO_PATH`：业务代码仓库路径
- `OUTPUT_DIR`：输出基目录

可以读取的文件：
- `$OUTPUT_DIR/craft/change_points.md`（来自 step 2）
- `$OUTPUT_DIR/craft/full_diff.txt`

## 流程

### 步骤 1: 读取 change_points.md

理解每个改动点的性质、影响范围、依赖关系。

### 步骤 2: 读取改动函数的源代码

对每个改动点：
- Read `BUSINESS_REPO_PATH/<file>`
- 理解函数完整实现（不只是 diff 里的改动行）
- 了解函数的上下游：被谁调用 / 调用了谁 / 依赖什么 mock

### 步骤 3: 为每个改动点决定测试需求

按改动类别决定测试重点：

#### bug-fix 类

- **1 个 happy path 测试**：证明 bug 修复后正常场景仍然 work
- **1 个 regression 测试**：模拟触发 bug 的输入,证明修复生效
- 例: nil 检查修复 → 测试用例 `input=nil` 应该返回 err 而不是 panic

#### new-branch 类

- **覆盖所有新分支**：每个新的 if/switch case 一个测试
- **覆盖已有分支**：保证新改动没破坏旧的路径

#### new-param 类

- **参数组合矩阵**：每个参数的典型值（正常 / 边界 / 无效）
- 例: 新增 `limit int` 参数 → 测试 `limit=0/1/100/-1/MAXINT`

#### new-func 类

- **完整测试**：happy path + 至少 2-3 个边界或异常
- 探索函数的所有行为路径

#### refactor 类

- **行为等价性测试**：与重构前预期行为一致的测试
- 优先测试函数的公共契约（输入输出关系），不测试内部实现

#### perf 类

- **正确性测试**：优化前后的结果应该一致
- 不需要性能测试（bug-fixloop MVP 不做 benchmark）

### 步骤 4: 决定 mock 策略

对于每个测试需求:
- 依赖的外部资源（数据库 / HTTP client / 其它 service）需要 mock
- 优先用 `github.com/stretchr/testify/mock` 或手写 interface + fake 实现
- 如果业务代码已经用 DI（依赖注入），直接注入 fake
- 如果业务代码硬编码了外部依赖,在测试里用 monkey patch 或 dependency override

### 步骤 5: 输出 test_requirements.md

写入 `$OUTPUT_DIR/craft/test_requirements.md`，格式：

```markdown
# 测试需求清单

> 对应改动点: {change_points.md 中的 N 个点}

## 改动点 1: {package}.{function}

**类别**: bug-fix

### 测试需求 1.1: Happy path

- **测试名**: `TestHandleRequest_HappyPath`
- **场景**: 正常请求正常响应
- **输入**: `req = &Request{ID: "valid"}`
- **预期**: 返回非 nil response, err == nil
- **Mock**: `s.store.Query` 返回固定数据

### 测试需求 1.2: Nil 请求（回归测试）

- **测试名**: `TestHandleRequest_NilRequest`
- **场景**: req 为 nil 时不应 panic
- **输入**: `req = nil`
- **预期**: 返回 nil, err == ErrNilRequest
- **Mock**: 不需要

## 改动点 2: ...

...

## 总览

| 改动点 | 测试需求数 | 预计生成测试数 |
|-------|---------|------|
| 1 | 2 | 2 |
| 2 | 3 | 3 |
| ... | ... | ... |

**总计**: {N} 个测试用例
```

### 步骤 6: 自检

- [ ] 每个改动点都至少有 1 个测试需求
- [ ] 每个测试需求都有清晰的名字 / 输入 / 预期
- [ ] Mock 策略明确（哪些依赖需要 mock,怎么 mock）
- [ ] 避免过度覆盖（每个改动点最多 5 个测试）

## 产出

- `$OUTPUT_DIR/craft/test_requirements.md`

## 约束

- 每个改动点**最多 5 个**测试需求（避免测试爆炸）
- 总测试数**最多 30 个**（MVP 限制）
- 不生成性能测试
- 不生成 E2E 测试（bug-fixloop MVP 仅单元测试）
