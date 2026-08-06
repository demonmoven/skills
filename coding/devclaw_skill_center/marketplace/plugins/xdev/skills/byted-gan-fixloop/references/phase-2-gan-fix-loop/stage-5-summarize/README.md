# Phase 2 / Stage 5: Summarize（迭代总结）

## 对应原 Stage
原 Stage 5。

## 干什么
**每轮迭代末**写一份 iteration_summary.md，给**下一轮**的 stage-4-fix 作为 history 上下文。

## 与 phase-3-finalize 的区别
- **本 stage**：每轮都写一次，是循环内的步骤，目的是给下一轮做 history。
- **phase-3-finalize**：循环结束后才执行一次，输出最终摘要文本给用户看。

两者完全不同，不要混淆。

## 执行位置
主 context（不启 subagent，简单文件读写）

## 主要 prompts

| 文件 | 谁读 | 干什么 |
|------|------|------|
| prompts/iteration-summary.md | 主 context | iteration_summary.md 合成流程 |

## 输入
- 当前轮的 `iteration_$ITERATION/fix_summary.md`
- 历史所有轮的 `iteration_*/fix_summary.md` 和 `test_stats.json`

## 产出
- `$OUTPUT_DIR/iteration_$ITERATION/iteration_summary.md`

## 关键动作
- **首轮**：直接 `cp iteration_1/fix_summary.md → iteration_1/iteration_summary.md`
- **后续轮**：Read 所有历史 fix_summary.md + test_stats.json，合成迭代经验总结

## 后续传递
本 stage 写出的 `iteration_summary.md` 会在下一轮的 stage-4-fix 中作为 `HISTORY_SUMMARY_FILE` 参数传入。

## 跨外部 skill 依赖
无

## 跳过条件
- 本 stage 在循环最后一步，每轮都执行
