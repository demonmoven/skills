# Stage 2 · 测试执行

执行 pytest + 可选 Go 单元测试，合并产出结构化失败清单。

## 输入

| 参数 | 来源 | 说明 |
|---|---|---|
| `TEST_REPO_PATH` | SKILL.md 解析 | qa_model_effect 本地路径 |
| `BUSINESS_REPO_PATH` + `DETECTED_SUB_REPOS` | 嗅探结果 | workspace 模式下可能多个 sub-repo；用于 Go 单元测试执行目录 |
| `TCE_LANE` | 首轮生成或用户传 | 同时注入 `ENV_LABEL` 环境变量，触发 `x-tt-env` header 染色 |
| `RUNTIME_ENV` | SKILL.md 参数 | `online` / `i18nalisg` / `i18nmaliva` |
| `TEST_CONCURRENCY` | 默认 30 | `pytest -n N` |
| `TEST_RERUNS` | 默认 1 | `pytest --reruns N` |
| `TEST_MARKER` | 可选 | `pytest -m '<expr>'` |
| `OUTPUT_DIR` | SKILL.md 参数 | 迭代数据目录 |
| `ITERATION` | 循环控制 | 当前迭代轮次 |
| Selectors | `$OUTPUT_DIR/craft/pytest_selectors.txt`（如有） | Stage 0 生成的精确选择器 |

## 执行步骤

### 1. 清理上一轮产出

```bash
cd $TEST_REPO_PATH
rm -rf allure_report/xml output/result.xml log/*
mkdir -p $OUTPUT_DIR/iteration_$ITERATION
```

### 2. 注入环境变量

```bash
export RUNTIME_ENV=$RUNTIME_ENV
export ENV_LABEL=$TCE_LANE       # 核心：泳道染色
# 可选：conftest.py 里没 hook 到 params.ENV_LABEL 的旧用例靠这条兜底
```

### 3. 决定测试范围

```bash
if [ -s $OUTPUT_DIR/craft/pytest_selectors.txt ]; then
  # Stage 0 有产出时用精确选择器
  SELECTOR_ARGS=$(cat $OUTPUT_DIR/craft/pytest_selectors.txt | xargs)
elif [ -n "$TEST_MARKER" ]; then
  SELECTOR_ARGS="testcases/ -m '$TEST_MARKER'"
else
  SELECTOR_ARGS="testcases/"
fi
```

### 4. 运行 pytest

```bash
pytest -n $TEST_CONCURRENCY \
       --reruns $TEST_RERUNS \
       --reruns-delay 0 \
       --alluredir=allure_report/xml \
       --junitxml=output/result.xml \
       -vv -s -p no:cacheprovider \
       $SELECTOR_ARGS \
       2>&1 | tee $OUTPUT_DIR/iteration_$ITERATION/pytest.log
pytest_exit=${PIPESTATUS[0]}
```

pytest 退出码：
- `0`：全部通过
- `1`：有失败（仍走解析流程）
- `2`：用户中断（停止循环）
- `3-5`：内部错误（按 infra_error 记录，停止循环）

### 5. 运行 Go 单元测试（仅 Step 0.6 产出的 sub-repo）

```bash
# 串行执行，避免 GOPATH cache 冲突
for target in $(cat $OUTPUT_DIR/detected_targets.jsonl); do
  sub_repo=$(echo "$target" | jq -r .sub_repo_path)
  if [ -s "$sub_repo/unit_test_dirs.txt" ]; then
    cd $sub_repo
    go test -json $(cat unit_test_dirs.txt | xargs) > $OUTPUT_DIR/iteration_$ITERATION/go_unit_$(basename $sub_repo).jsonl 2>&1
  fi
done
```

### 6. 解析 + 合并产出

按 [pytest-result-format.md](./pytest-result-format.md) 规则：

```bash
python3 <<'PY'
# 脚本: 解析 junit XML + allure XML + go unit jsonl → failed_cases.jsonl + test_stats.json
# 具体实现按 pytest-result-format.md 的字段清单
import os, json, xml.etree.ElementTree as ET, glob
# ... (Stage 2 的 subagent 会自己写这段脚本，按 pytest-result-format.md 规范)
PY
```

### 7. 判断循环出口

读 `$OUTPUT_DIR/iteration_$ITERATION/test_stats.json`：
- `failed == 0 && error == 0` → `TESTS_PASSED = true`，Stage 2 返回主循环，退出迭代
- `failed > 0 || error > 0` → 继续 Stage 3

**skipped > 0 且 failed+error == 0** 仍视为通过；skipped 明细保留到最终摘要。

## 容错

- **pytest 崩溃（exit 3-5）**：`failed_cases.jsonl` 写一条 `{"status": "infra_error"}`，Stage 3 按 `[INFRA]` 处理，建议人工介入
- **allure 目录空**：降级只用 junit，`allure_detail` 字段留空
- **junit XML 缺失**：pytest 可能在 collection 阶段就挂了，直接报错停止循环并输出 pytest.log 尾部

## 日志

所有原始 stdout/stderr 落到 `$OUTPUT_DIR/iteration_$ITERATION/pytest.log`，供 Stage 3 查询具体 case 的 real-time 日志。

## 与原 fixloop test-execution.md 的差异

| 维度 | 原 fixloop | 本 skill |
|---|---|---|
| 测试命令 | `go test -json $(cat test_dirs.txt)` | `pytest -n N --junitxml --alluredir $SELECTORS` |
| 结果格式 | 单一 `go test -json` jsonl | junit XML + allure XML 合并 |
| 泳道注入 | Kitex header 通过代码硬编码 | `ENV_LABEL` 环境变量 |
| 单元测试 | Stage 2 唯一任务 | Stage 2 的可选附加步骤（仅当 Step 0.6 产出） |
