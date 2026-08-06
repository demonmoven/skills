# Phase 2 / Stage 2: Test（集成 + 单元测试）

## 对应原 Stage
原 Stage 2。

## 干什么
执行集成测试（E2E）+ 单元测试，解析 go test -json 输出，生成失败用例文件和统计文件。

## 执行位置
subagent

## 主要 prompts

| 文件 | 谁读 | 干什么 |
|------|------|------|
| prompts/test-execution.md | subagent | 注入 x-tt-env header → 跑 go test -json → 解析失败 → 写 jsonl + stats |

## lib
（无 — go-test-json-format.md 放在 `_shared/`，因为 stage-3-analyze 也需要）

## 输入
- `TEST_REPO_PATH`、`BUSINESS_REPO_PATH`
- `TEST_SCOPE`：来自 `phase-1/craft/test_dirs.txt`，或 fallback 为 `./...`
- `UNIT_TEST_SCOPE`：来自 `phase-1/craft/unit_test_dirs.txt`，可空
- `TCE_LANE`：来自 stage-1 的产出
- `TEST_ENV`：默认 `fornax_boe`，遇 404 自动回退 `boe`
- `ITERATION`：当前迭代轮次

## 产出
- `$OUTPUT_DIR/iteration_$ITERATION/failed_cases.jsonl`（每行一个失败/跳过的测试）
- `$OUTPUT_DIR/iteration_$ITERATION/test_stats.json`（汇总统计）

## 关键动作
1. 读取 `test_env/${TEST_ENV}.config.yaml`，注入 `x-tt-env: $TCE_LANE` header（让请求路由到泳道）
2. 对每条测试构造 `go test -json -v -count=1 -p=1 -run "^Test$" ./dir` 命令
3. 用 Python 管道脚本流式解析输出，遇 fail/skip 立即追加到 failed_cases.jsonl
4. 提取 LogID（X-Tt-Logid 等正则匹配）写入 log_ids 字段
5. **404 回退**：若失败用例含 404 且 TEST_ENV != "boe"，切到 boe 重跑一次
6. `git checkout --` 恢复配置文件（无论成功失败）
7. 单元测试在 BUSINESS_REPO_PATH 下执行（不需要 x-tt-env），失败追加到同一个 failed_cases.jsonl

## 跨外部 skill 依赖
无

## 跳过条件
无（每轮必跑）

## 后续判断
主 context 读 `test_stats.json`：
- `failed_cases == 0` → 设置 `TESTS_PASSED=true`，退出循环
- `SINGLE_TEST_RUN=true` → 输出结果直接退出（不进入修复阶段）
- 否则 → 进入 stage-3-analyze
