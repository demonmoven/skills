# 独立代码审计

## Table of Contents

- [为什么需要独立审计](#为什么需要独立审计)
- [何时触发](#何时触发)
- [执行流程](#执行流程)
- [同模型交叉审查技巧](#同模型交叉审查技巧)
- [大规模变更的审计策略](#大规模变更的审计策略)
- [注意事项](#注意事项)

---

## 为什么需要独立审计

实现者在长时间编码后会产生认知盲区——"我写的代码我觉得没问题"。独立审计由**不同于实现者的审查者**（另一个 AI 模型、另一个 Agent 实例、或人类 Reviewer）对变更进行盲审，弥补实现者的认知盲区。

> OpenAI Harness Engineering 的实践：几乎全部 Review 都由 Agent 之间完成。
> Codex 先自己 Review 一遍、再请其他 Agent Review，来回迭代直到所有审查者满意。

## 何时触发

在 ExecPlan 生命周期中，独立审计发生在 **Phase 4（开发与验证）完成后、Phase 5（结果汇报）之前**：

```
Phase 4: 开发与验证
    ↓ 所有里程碑完成
独立代码审计（本文件描述的流程）
    ↓ 审计发现修复后
Phase 5: 结果汇报
```

**触发条件**（满足任一即应触发）：
- 变更超过 500 行代码
- 涉及安全相关逻辑（认证、授权、密钥处理）
- 涉及并发/状态管理
- 跨模块改动
- 用户明确要求

**可跳过的场景**：
- 纯文档更新
- 配置文件调整
- 小型 bug 修复（<50 行且有测试覆盖）

## 执行流程

### Step 1: 确定审计范围

```bash
# 查看当前变更
git diff --stat HEAD~N   # N = 本次 ExecPlan 的 commit 数
git diff --stat          # 如果有未提交变更
```

审计提示词应明确范围：
- 未提交变更：`Review the current uncommitted changes`
- 最近 N 次提交：`Review the changes introduced by the last N commits`
- 相对于某分支：`Review changes in current branch relative to <base>`

### Step 2: 选择审计工具

**默认使用 OpenAI Codex CLI**。Codex 基于不同于实现者的模型运行只读沙箱审查，天然提供认知多样性。

```bash
command -v codex
```

- 如果 `codex` 已安装：直接使用（见 Step 3）
- 如果 `codex` 未安装：告知用户按照内部文档配置 Codex CLI（https://bytedance.larkoffice.com/wiki/HaBuwpIFYi1nx9kyHiXckCxGn3g），安装完成后重新触发审计
- 如果用户明确拒绝安装 Codex：回落到下方备选方案

**备选方案**（Codex 不可用时）：

| 工具 | 适用场景 | 优点 | 缺点 |
|------|---------|------|------|
| **同模型交叉审查** | 没有额外工具时 | 零成本、即时可用 | 同一模型的盲区可能重复 |
| **人工 Review** | 关键变更 | 最高可靠性 | 最慢、最贵 |
| **CI 集成审查** | 有 PR 流程时 | 自动化、可追溯 | 反馈延迟 |

回落到"同模型交叉审查"时，关键是让审查实例**不继承实现上下文**（全新会话、只看代码变更）。

### Step 3: 执行审计

#### 使用 Codex CLI（默认）

根据变更状态确定审计范围：

| 场景 | 提示词中的范围描述 |
|------|-----------------|
| 有未提交变更 | `Review the current uncommitted changes in this git repository.` |
| 工作区干净，审查最近提交 | `Review the changes introduced by the last N commits.` |
| 审查相对于某分支 | `Review changes in current branch relative to <base>.` |

执行命令：

```bash
RESULT_FILE="/tmp/codex_audit_result_$$.txt"

codex exec \
  --cd "$(pwd)" \
  --sandbox read-only \
  -o "$RESULT_FILE" \
  "<范围描述> Focus on bugs, regressions, missing tests, and behavioral risks. \
  Return a concise markdown report with findings ordered by severity; \
  if no issues are found, say so explicitly."
```

如果在 tmux 环境中，可在新 pane 中执行让用户实时观察：

```bash
# 检测 tmux
[ -n "$TMUX" ] && echo "tmux" || echo "no-tmux"

# tmux 模式：在垂直 pane 中运行，主流程等待信号
tmux split-window -h "cd '$(pwd)' && codex exec --cd '$(pwd)' --sandbox read-only \
  -o '$RESULT_FILE' '<prompt>' 2>&1 | tee /tmp/codex_audit_stdout_$$.txt; \
  tmux wait-for -S codex-audit-done"
tmux wait-for codex-audit-done
```

执行完毕后从 `$RESULT_FILE` 读取审计结果，进入 Step 4 验证。

#### 构建审计 prompt（通用，Codex 和备选方案均适用）

通用审计 prompt 模板：

```
Review the following code changes. Focus on:
1. Bugs and regressions — logic errors, edge cases, off-by-one
2. Security issues — injection, auth bypass, secret exposure
3. Concurrency safety — race conditions, deadlocks, shared state
4. Missing tests — untested paths, especially error paths
5. API contract violations — does the implementation match documented behavior?

Do NOT flag:
- Pure style preferences (formatting, naming conventions)
- Issues that are clearly documented as intentional design decisions

Return a concise markdown report with findings ordered by severity.
If no issues are found, say so explicitly.
```

根据项目特点追加重点关注项：
- Go 项目：`Also check for goroutine leaks, context cancellation, and error wrapping`
- TypeScript 项目：`Also check for type safety, null handling, and async error propagation`
- 安全相关：`Pay special attention to authentication, authorization, and input validation`

### Step 4: 验证审计发现

> **核心原则**：审计工具以有限上下文运行，经常会编造不存在的问题或误解项目设计。
> 验证是独立审计的核心价值。不经验证直接转发审计发现是不可接受的。

对每一条发现逐项验证：

1. **检查实际源码**：确认审计指出的问题代码是否真实存在于该文件的该行号
2. **检查项目文档**：查阅 AGENTS.md、ARCHITECTURE.md 等，确认是否为有意的设计选择
3. **判断真伪**：
   - 这是真实的 bug，还是审计工具因上下文不足产生的误解？
   - 这个问题有实际影响，还是纯粹的风格偏好？

4. **分类标注**：

| 分类 | 含义 | 处理方式 |
|------|------|---------|
| **真实 bug** | 源码验证了问题确实存在 | 在 Phase 5 前修复 |
| **潜在改进** | 不是 bug 但值得优化 | 记录到 ExecPlan 的"建议后续" |
| **误判** | 审计工具上下文不足导致 | 标注原因，不修复 |
| **风格建议** | 纯风格层面 | 忽略 |

### Step 5: 修复真实问题

如果验证后存在真实 bug：

1. 在 ExecPlan 的 意外发现 中记录审计发现
2. 修复 bug
3. 重新运行受影响的验收检查
4. 更新 进度追踪

### Step 6: 产出审计报告

```markdown
## 独立审计报告

- **审计范围**：<未提交变更 / 最近 N 次提交 / 相对于 base 分支>
- **审计工具**：<工具名称>
- **发现总数**：N 条
- **验证通过**：X 条 | **被否决**：Y 条

### 验证通过的发现

#### 1. <问题标题>
- **文件**：`path/to/file:42`
- **严重级别**：高 / 中 / 低
- **描述**：<问题描述>
- **验证依据**：<为何确认此发现合理>
- **修复状态**：已修复 / 记录为后续改进

### 被否决的发现

#### 1. <审计工具的原始发现>
- **否决原因**：<如"该设计是项目约定，见 ARCHITECTURE.md">

### 总结

<审计结论。如果所有发现都被否决或无发现，明确说明"审计通过，未发现有效问题">
```

审计报告嵌入 ExecPlan 的 产物与备注 章节。

## 同模型交叉审查技巧

如果只有当前模型可用（无法调用其他 AI），可以用以下方式模拟"独立"：

1. **创建全新上下文**：使用 Task 工具启动子 Agent，只提供 diff 和必要的项目文档，不继承当前对话
2. **角色切换 prompt**：明确指示"你是一个独立的代码审查者，你没有参与这段代码的实现"
3. **关注反模式**：提供项目的 AGENTS.md 约束列表，让审查者检查变更是否违反

这不如真正的不同模型有效，但比不做审计好很多。

## 大规模变更的审计策略

当 diff 超过 5000 行时：

1. **按模块分批**：每个模块/包的变更独立审计
2. **按风险分级**：先审计安全相关、并发相关的变更，再审计普通业务逻辑
3. **集中审计接口层**：优先审计模块间的接口变更（API、类型定义、协议）

## 注意事项

- 审计工具不了解项目的战略目标和架构决策，会经常高估问题严重性——这就是验证步骤存在的原因
- 验证步骤是本流程的核心价值——跳过验证直接展示审计输出是错误的做法
- 审计发现的修复也需要更新 ExecPlan 的进度追踪，不能只改代码
- 如果审计工具执行超时或报错，记录到 意外发现 并继续（审计失败不阻塞整个流程，但应向用户报告）
