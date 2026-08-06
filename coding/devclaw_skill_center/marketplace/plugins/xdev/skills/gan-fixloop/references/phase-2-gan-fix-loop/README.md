# Phase 2: Gan Fix Loop（干修复循环 - 核心）

## 干什么
fix-loop 的核心循环主体。每一轮迭代依次执行 5 个 stage：
deploy → test → analyze → fix → summarize，
直到测试全部通过或达到 MAX_ITERATIONS。

## 这是一个「容器 phase」
本目录下**没有 prompts/**，只有 README.md（你正在读的）和 5 个 stage 子目录。
phase-2 自身不直接执行任何动作，所有具体动作都在 stage 内部。

## 循环伪代码

```
ITERATION = 1
TESTS_PASSED = false

WHILE ITERATION <= MAX_ITERATIONS AND NOT TESTS_PASSED:
    stage-1-deploy        # TCE 部署 + AGW IDL 同步（首轮）
    stage-2-test          # 集成测试 + 单元测试
    判断: TESTS_PASSED?  → 退出循环
    [SINGLE_TEST_RUN=true] → 退出循环
    stage-3-analyze       # Judge 失败根因分析
    stage-4-fix           # Fixer 代码修复 + commit + push
    stage-5-summarize     # 写 iteration_summary.md
    ITERATION++
END WHILE
```

## stage 列表

| stage | 子目录 | 对应原 Stage | 干什么 |
|-------|--------|-------------|------|
| stage-1 | stage-1-deploy/ | Stage 1 + Stage 1.5 | TCE 部署 + AGW IDL 同步（首轮） |
| stage-2 | stage-2-test/ | Stage 2 | 集成测试 + 单元测试 |
| stage-3 | stage-3-analyze/ | Stage 3 | Judge 失败根因分析 |
| stage-4 | stage-4-fix/ | Stage 4 | Fixer 代码修复 |
| stage-5 | stage-5-summarize/ | Stage 5 | 迭代总结，给下一轮做 history |

## stage 之间的数据传递

跨 stage 数据格式定义见 `../_shared/iteration-data-format.md`。

主要传递通道：
- `$OUTPUT_DIR/iteration_$ITERATION/failed_cases.jsonl` — stage-2 写，stage-3 读
- `$OUTPUT_DIR/iteration_$ITERATION/test_stats.json` — stage-2 写，主 context 读判断退出
- `$OUTPUT_DIR/iteration_$ITERATION/analysis_report.md` — stage-3 写，stage-4 读
- `$OUTPUT_DIR/iteration_$ITERATION/fix_summary.md` — stage-4 写，stage-5 读
- `$OUTPUT_DIR/iteration_$ITERATION/iteration_summary.md` — stage-5 写，**下一轮的 stage-4** 读做 history

## 与 phase-3-finalize 的区别

- **stage-5-summarize**：每轮迭代末写 iteration_summary.md，给下一轮做 history。**这是循环内的步骤**。
- **phase-3-finalize**：整个循环结束后打印最终摘要文本给用户看。**这是循环外的步骤**。

两件不同的事，分别落在 stage-5-summarize 和 phase-3-finalize。

## 跳过条件
- 整个循环：`MAX_ITERATIONS=0` 时直接跳过（仅做 phase-1 的测试生成然后进 phase-3）
- 提前退出：`failed_cases==0` 或 `SINGLE_TEST_RUN=true` 触发提前退出
