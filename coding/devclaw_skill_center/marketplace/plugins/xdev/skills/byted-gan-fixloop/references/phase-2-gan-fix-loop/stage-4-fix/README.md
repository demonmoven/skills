# Phase 2 / Stage 4: Fix（代码修复）

## 对应原 Stage
原 Stage 4。

## 干什么
扮演 **Fixer** 角色，根据 stage-3 产出的 Judge 报告修复代码（业务代码或测试代码），完成后 commit + push。

## 执行位置
subagent (Fixer 角色)

## 主要 prompts

| 文件 | 谁读 | 干什么 |
|------|------|------|
| prompts/fix-procedure.md | subagent | 完整的 Fixer 修复流程（多层修复策略 + 编译重试 + 防回退） |

## 输入
- `JUDGE_REPORT_PATH`：来自 stage-3 的 `iteration_$ITERATION/analysis_report.md`
- `FAILED_CASES_FILE`
- `BUSINESS_REPO_PATH`、`TEST_REPO_PATH`
- `BRANCH`：修复后 push 到的目标分支
- `HISTORY_SUMMARY_FILE`：上一轮 stage-5 写的 iteration_summary.md（仅 ITERATION>1）

## 产出
- 实际代码修改（业务代码 + 测试代码）
- `$OUTPUT_DIR/iteration_$ITERATION/fix_summary.md`
- 自动 commit + push（业务仓库 + 测试仓库各一次）

## 关键动作
1. Read JUDGE_REPORT_PATH，提取 `JUDGE_STRUCTURED_OUTPUT` JSON 块
2. 按 `fix_priority` 顺序逐个修复（**最多 5 个/轮**）
3. **多层修复策略**（按 ITERATION 递进）：
   - ITERATION=1: 标准修复
   - ITERATION=2: 策略切换
   - ITERATION>=3: 深层追踪 + 系统性遗漏检查 + 6 方向轮换
4. **Diff 感知**：避免重复修同一文件的同一区域
5. **防回退检查**：读上一轮 test_stats.json，不破坏已通过测试
6. **编译验证**：`go build ./...`，编译失败最多重试 10 次
7. 写 fix_summary.md
8. commit + push 业务仓库和测试仓库

## 跨外部 skill 依赖（条件性）
当 root_cause 涉及运行时错误（500 / panic / nil pointer）时：
- `$BYTEDCLI_SKILLS_DIR/bytedance-tce/SKILL.md`
- `$BYTEDCLI_SKILLS_DIR/bytedance-log/SKILL.md`

## 跳过条件
- 仅在 stage-3 完成后执行
