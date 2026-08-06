# Phase 1 / Bug Craft / Step 2 — 改动点抽取

> **使用 Agent 工具启动 subagent 执行**。主 context 把参数传给 subagent，subagent Read 本文件作为执行指南。

## 角色

你是一位 Go 代码分析师。任务是**从 git diff 中提取每一个"改动点"**，每个改动点对应一个需要被测试覆盖的代码逻辑单元。

## 输入

- `BUSINESS_REPO_PATH`：业务代码仓库路径
- `OUTPUT_DIR`：输出基目录
- `COMMIT_RANGE`：用户指定的 diff 范围（仅供参考，diff 内容已写入文件）

可以读取的文件：
- `$OUTPUT_DIR/craft/full_diff.txt`：完整 diff
- `$OUTPUT_DIR/craft/go_changed_files.txt`：Go 改动文件列表
- `$OUTPUT_DIR/craft/diff_stat.txt`：改动统计

## 流程

### 步骤 1: 读取 full_diff.txt 并按文件拆分

每个文件的 diff 以 `diff --git a/<path> b/<path>` 开头。按文件拆开，对每个文件单独分析。

### 步骤 2: 对每个 Go 文件，提取 hunk 和函数

对每个 diff hunk（以 `@@` 开头）：

1. **从 hunk header 提取函数名**：
   ```
   @@ -100,15 +100,20 @@ func (s *Service) HandleRequest(ctx context.Context, req *Request) (*Response, error) {
   ```
   上面的 `func (s *Service) HandleRequest(ctx context.Context, req *Request) (*Response, error)` 是 hunk 所在的函数签名。

2. **如果 hunk header 没有函数信息**（比如改动在文件头部），用 Read 工具读取 `BUSINESS_REPO_PATH/<file>` 文件的对应行号前后几十行，手动找函数边界。

3. **记录改动范围**：
   - 新增行（`+` 开头）
   - 删除行（`-` 开头）
   - 上下文行（空格开头）

### 步骤 3: 判断改动性质

对每个改动点分类：
- **bug-fix**：关键词如"fix / bug / 修复 / 异常 / nil / panic"
- **新逻辑分支**：新增了 if/switch/case 分支
- **新参数**：函数签名变更（新增或改类型）
- **新函数**：完全新增的函数
- **重构**：函数行为等价但实现改变
- **性能优化**：算法或数据结构变更

> 分类依赖 diff 内容和上下文代码的综合判断，不严格要求准确，只用于后续测试需求生成时的参考。

### 步骤 4: 抽取每个改动点的关键信息

对每个改动点记录：
- `file`：文件路径（相对 BUSINESS_REPO_PATH）
- `package`：Go 包名（从 file 顶部 `package xxx` 语句）
- `function`：函数名（含 receiver 类型）
- `signature`：完整函数签名
- `kind`：bug-fix / new-branch / new-param / new-func / refactor / perf
- `changed_lines`：改动涉及的行号范围
- `summary`：一句话描述改动的意图
- `dependencies`：函数直接调用的其它函数 / 依赖的结构体（通过读代码分析）

### 步骤 5: 输出 change_points.md

写入 `$OUTPUT_DIR/craft/change_points.md`，格式：

```markdown
# Change Points（改动点清单）

> 来源: COMMIT_RANGE={COMMIT_RANGE}
> 文件数: {N}
> 改动点数: {M}

## 改动点 1: {package}.{function}

- **文件**: `<relative path>`
- **签名**: `func (s *Service) HandleRequest(ctx context.Context, req *Request) (*Response, error)`
- **类别**: bug-fix
- **改动行**: 100-115
- **意图**: 修复了 req 为 nil 时的 panic
- **依赖**: `validateRequest`, `s.store.Query`
- **关键改动**:
  ```go
  // 新增 nil 检查
  if req == nil {
      return nil, ErrNilRequest
  }
  ```

## 改动点 2: {package}.{function}

...

## 总览矩阵

| # | 文件 | 函数 | 类别 |
|---|-----|------|------|
| 1 | handler/request.go | Service.HandleRequest | bug-fix |
| 2 | ... | ... | ... |
```

### 步骤 6: 自检

- [ ] 每个改动点都能对应到 diff 中的具体 hunk
- [ ] 每个函数的 signature 都通过 Read 业务代码确认过（不是只从 diff 推测）
- [ ] package 名正确（读 `package xxx` 语句）
- [ ] kind 分类合理

## 产出

- `$OUTPUT_DIR/craft/change_points.md`

## 失败处理

- 无法从 hunk header 提取函数名 → Read 源文件手动定位
- 改动点跨多个函数 → 拆成多个独立改动点
- 函数签名变更 → 记录新旧两个签名
- diff 内容异常（格式错误等） → 报错,让用户检查 COMMIT_RANGE
