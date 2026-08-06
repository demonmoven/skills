# Phase 3 / Finalize — 最终摘要执行指南

> 主 context 直接 Read 本文件，按以下格式打印最终摘要给用户。
> 不启动 subagent。

## 触发时机

整个 phase-2-gan-fix-loop 循环结束之后（无论是因为 `failed_cases==0` 提前退出，还是触达 `MAX_ITERATIONS`，还是 `SINGLE_TEST_RUN=true` 直接退出）。

## 数据来源

- 主 context 持有的 `ITERATION` 变量（最终轮次）
- 主 context 持有的 `TESTS_PASSED` 标志
- 主 context 持有的 `TCE_LANE` 泳道名
- 累积的 `$OUTPUT_DIR/iteration_*/test_stats.json` 文件
- `$OUTPUT_DIR/craft/` 目录下的 phase-1 产出（如果 `GENERATE_TESTS=true`）

## 输出格式

```
=== Fix-Loop 最终摘要 ===
总迭代次数: N
最终测试结果: PASSED / FAILED (X 个失败)
TCE 泳道: <lane_name>

Stage 0 产出:
  用户动线: $OUTPUT_DIR/craft/user_journeys.md
  E2E 测试: $OUTPUT_DIR/craft/e2e_work_copy/
  单元测试: $OUTPUT_DIR/craft/unit_tests/

各轮迭代统计:
  迭代 1: pass=X, fail=Y, skip=Z
  ...

输出目录: $OUTPUT_DIR/
```

## 字段说明

| 字段 | 取值来源 |
|------|---------|
| 总迭代次数 | `$ITERATION` 主 context 变量 |
| 最终测试结果 | `$TESTS_PASSED ? "PASSED" : "FAILED"`，FAILED 时附带最终轮失败用例数 |
| TCE 泳道 | `$TCE_LANE` |
| Stage 0 产出 | 仅当 `GENERATE_TESTS=true` 时打印这一段，否则跳过 |
| 各轮迭代统计 | 遍历 `$OUTPUT_DIR/iteration_*/test_stats.json`，按轮次顺序打印 pass/fail/skip 计数 |

## 注意

本 phase 不写新文件，只是 stdout 打印。所有数据都已经在前序 phase / stage 写好了。
