# 审查协议（Review Protocol）

本文件定义所有 LLM 审查 skill 的通用执行协议。各 skill 通过引用本文件获取标准化的审查流程。

---

## 1. 上下文模板

所有 LLM 审查 skill 在开始审查前，必须呈现以下上下文信息：

```
这是第 **{{iteration}}** 轮迭代，共 **{{maxIterations}}** 轮 E2E 迭代修复。
{{#if isMultiReview}}
这是第 **{{reviewRound}}/{{totalReviewRounds}}** 轮 Review（Multi-Review 模式）。
{{/if}}

### 环境信息
- **任务输出目录**: {{taskOutputDir}}
- **后端代码路径**: {{backendPath}}
- **E2E 测试代码路径**: {{e2eTestPath}} (只读参考，不审查)
{{#if specDir}}
- **规格文档目录**: `./specs/` (符号链接到 {{specDir}})
{{/if}}
```

---

## 2. 自动生成代码排除规则

以下目录包含**自动生成的代码**，代码量很大，**不需要查看也不需要审查**：
- `loop_gen/` — 由 IDL 工具自动生成的代码
- `kitex_gen/` — 由 Kitex 框架自动生成的 RPC 代码
- `**/*.gen.go` — 任何以 .gen.go 结尾的文件（如 GORM Gen 生成代码）

这些目录中的代码是通过工具从 IDL 定义自动生成的，不能直接修改，只能通过修改 IDL 后重新生成。
**审查时请跳过这些目录，不要在报告中提及这些目录中的问题。**

---

## 3. Multi-Review 步骤 0 协议

当 `hasPreviousRound` 为 true 时，在正式审查前必须执行步骤 0：

```
### 步骤 0: 读取上一轮 Review 结果

**重要**: 这是 Multi-Review 模式的第 {{reviewRound}} 轮。你必须首先读取上一轮的审查报告，以便：
1. 避免重复报告已发现的问题
2. 发现上一轮可能遗漏的问题
3. 将本轮发现的**新问题**与上一轮的问题合并汇总

读取上一轮报告：
cat {{previousRoundDir}}/{报告文件名}

**本轮审查要求**：
- 参考上一轮的审查结果
- 重点发现上一轮可能遗漏的问题
- 在报告中明确标注哪些是"新发现"，哪些是"确认上轮问题"
- 最终问题列表应包含所有轮次发现的问题（累加）
```

---

## 4. 步骤 1: 读取编码规范

```bash
cat {{backendPath}}/CLAUDE.md 2>/dev/null || cat {{backendPath}}/.claude/CLAUDE.md 2>/dev/null || echo "未找到 CLAUDE.md"
```

---

## 5. 步骤 2: 获取代码变更

**Review-All 模式**（reviewAll 为 true）:
```bash
# 获取仓库第一个 commit 的 hash
FIRST_COMMIT=$(cd {{backendPath}} && git rev-list --max-parents=0 HEAD | head -1)
cd {{backendPath}} && git diff --name-only $FIRST_COMMIT..HEAD 2>/dev/null || git diff --name-only
cd {{backendPath}} && git diff $FIRST_COMMIT..HEAD 2>/dev/null || git diff
```

**当前迭代模式**（reviewCurrentOnly 为 true）:
```bash
cd {{backendPath}} && git diff --name-only HEAD~1..HEAD 2>/dev/null || git diff --name-only
cd {{backendPath}} && git diff HEAD~1..HEAD 2>/dev/null || git diff
```

---

## 6. TodoWrite 进度跟踪模式

所有逐文件审查步骤必须使用 TodoWrite 跟踪进度：

1. 使用 TodoWrite 为每个需要审查的变更文件创建 todo
2. 审查每个文件时，将其标记为 in_progress
3. 完成审查后，将其标记为 completed
4. 这确保每个变更文件都被系统性审查
