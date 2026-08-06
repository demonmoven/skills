# Stage 2 · E2E 测试生成（pytest 版）

目标：把 `user_journeys.md` 里的每条动线转化为 `qa_model_effect` 风格的 pytest + `.online.json` 用例，推送到 `$TEST_REPO_PATH`。

> **业务范围**：本 skill 面向 **Doubao/Flow 创作（creativity）业务**。生成的测试用例必须落在 `qa_model_effect` 仓库的以下三个 AIGC 目录之一：
> - `testcases/bots/aigc_case` — 通用 AIGC（文生图入口、社区创作、音乐、Bot 主页交互等）
> - `testcases/bots/aigc_fangzhou` — 方舟（Seed 视觉大模型）后端：text2image / image2image
> - `testcases/bots/aigc_video` — 视频生成与进度查询
>
> 具体二级目录由 Step A 按 SPEC / user_journey 关键词 + 现有邻居结构决定；若归不上去则停下来让人工决策，不要落到其他模块。

## 输入

- `$SPEC_DIR`：需求文档目录
- `$OUTPUT_DIR/craft/user_journeys.md`：Step 0.1 产出的用户动线清单
- `$TEST_REPO_PATH`：qa_model_effect 仓库本地路径
- 可选 `$IDL_REPO_PATH`：解释接口变更
- 可选 `$OUTPUT_DIR/craft/stage1_to_stage2_coverage.md`：覆盖度基线

## 产出

- `$OUTPUT_DIR/craft/e2e_work_copy/` — 待同步到 `$TEST_REPO_PATH` 的工作拷贝
- `$OUTPUT_DIR/craft/generated_test_cases.jsonl` — 用例清单
- `$OUTPUT_DIR/craft/pytest_selectors.txt` — pytest node-id 列表（供 Stage 2 运行）

## 执行流程

本阶段**必须严格遵循** [邻居驱动的测试发现协议](./pytest-test-discovery.md)。不要重复该协议的内容，直接引用。

### 0. 读取上游产出

- `user_journeys.md` → 每条 UJ 含：目的、触发条件、期望路径、异常分支
- 如有 `stage1_to_stage2_coverage.md`，跳过已覆盖的 UJ

### 1. 执行 4-step 协议

按 `pytest-test-discovery.md` 的 Step A ~ Step D 执行：
- **Step A**：SPEC + user_journey 驱动模块定位（禁止硬编码）
- **Step B**：邻居探索学风格
- **Step C**：生成 pytest + JSON
- **Step D**：登记 JSONL + selectors.txt

### 2. 产出校验

完成后运行：

```bash
# 校验 1：生成的 .py 是否语法正确
cd $OUTPUT_DIR/craft/e2e_work_copy
python3 -m py_compile $(find . -name '*.py')

# 校验 2：生成的 .online.json 是否是有效 JSON
for f in $(find . -name '*.online.json'); do
  python3 -c "import json; json.load(open('$f'))" || echo "INVALID: $f"
done

# 校验 3：pytest collection 能否发现所有生成的 selectors
cd $TEST_REPO_PATH
cp -r $OUTPUT_DIR/craft/e2e_work_copy/* .
pytest --collect-only $(cat $OUTPUT_DIR/craft/pytest_selectors.txt | xargs) > /tmp/collect.txt 2>&1
collected=$(grep -c '<Function' /tmp/collect.txt)
expected=$(wc -l < $OUTPUT_DIR/craft/pytest_selectors.txt)
[ "$collected" -eq "$expected" ] || echo "WARN: collected=$collected expected=$expected"
```

任一校验失败：修复生成物再重试；3 次仍失败则降级为部分产出 + `needs_review` 标记。

## 账号池与 bot_id 查找约定

- 账号池：从邻居 `.online.json` 高频取值；若全部无邻居，用最通用的 `ec_stable_biz`（豆包场景常见默认）
- bot_id：
  1. 先在 `$TEST_REPO_PATH/constant/` 下 grep SPEC 里提到的 bot 名
  2. 找不到就在邻居 JSON 里找同类测试用的 bot_id，复用
  3. 都找不到：写"TODO"字符串 + `skip: true` + `skip_reason: "bot_id 待人工填"`

## 与原 fixloop Stage 2 (Go E2E) 的差异

| 维度 | 原 fixloop（Go） | 本 skill（pytest） |
|---|---|---|
| 测试文件 | `*_test.go` | `test_*.py` + `*.online.json` |
| 框架 | `testing` + 自定义 harness | `pytest` + `@test.runtime` 装饰器 + 外部 JSON 参数化 |
| 泳道注入 | 代码里直接带 Kitex header | `ENV_LABEL` 环境变量经 `conftest.py` hook 注入 `x-tt-env` header |
| 参数化 | `subtests` | `.online.json` 文件内 list，每条一个 case |
| Selectors 格式 | `TestName ./dir` | `testcases/path/to/test_x.py::test_y[params0-Env.Online]` |
