# Phase 3: Finalize（最终摘要）

## 对应原 Stage
原 SKILL.md 行 510-554 的"循环结束 & 最终摘要"段（无 stage 编号，是循环外的收尾工作）。

## 干什么
fix-loop 整个循环结束后的收尾工作。打印最终摘要文本给用户看，包括：
- 总迭代次数
- 最终测试结果（PASSED / FAILED）
- TCE 泳道名
- Phase 1 craft 产出位置
- 各轮迭代统计
- 输出目录路径

## 执行位置
主 context（无 subagent）

## 主要 prompts

| 文件 | 谁读 | 干什么 |
|------|------|------|
| prompts/final-summary.md | 主 context | 最终摘要的输出格式与字段说明 |

## 与 stage-5-summarize 的区别
- **stage-5-summarize**：每轮迭代末写 iteration_summary.md，给下一轮做 history。**循环内**。
- **本 phase**：循环结束后执行一次，打印最终摘要给用户。**循环外**。

## 输入
- 整个循环过程中累积的 `iteration_*/test_stats.json` 文件
- `TCE_LANE`、`TESTS_PASSED` 等主 context 变量

## 产出
- 终端文本输出（最终摘要）
- 不写新文件（仅打印）

## 跨外部 skill 依赖
无

## 跳过条件
无（无论循环正常退出还是触达 MAX_ITERATIONS，最终摘要都会打印）
