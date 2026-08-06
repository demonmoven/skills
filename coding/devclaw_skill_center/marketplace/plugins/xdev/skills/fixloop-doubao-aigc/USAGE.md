# fixloop-pytest 用法示例

## 典型调用（豆包 workspace + qa_model_effect）

最简：

```
/fixloop-pytest BRANCH=feat/creativity-new-field SPEC_DIR=~/specs/creativity-new-field BUSINESS_REPO=~/TraeProjects/doubao IDL_REPO=~/TraeProjects/alice_idl IDL_BRANCH=feat/creativity-new-field
```

解释：
- `BUSINESS_REPO=~/TraeProjects/doubao` → workspace 模式，skill 会自动嗅探 sub-repo（三路信号：git 改动 / IDL 变更 / SPEC 推断）
- `TEST_REPO` 省略：①若 CWD 或其祖先目录基名为 `qa_model_effect` 的 git 仓库 → 取该路径；②否则 clone 默认 Git 地址 `git@code.byted.org:flow/qa_model_effect.git` 到 `$OUTPUT_DIR/repos/`
- `TCE_LANE` 省略 → 自动生成 `ppe_flow_<rand4>`；同时作为 `ENV_LABEL` 注入 pytest
- 生成的测试仅落在三个 AIGC 目录：`testcases/bots/aigc_case` / `aigc_fangzhou` / `aigc_video`（详见 `references/pytest-test-discovery.md` Step A）

## 参数简写

除上述最简形式外，常用的 KEY=VALUE 参数：

```
RUNTIME_ENV=online        # 默认 online；海外场景用 i18nalisg / i18nmaliva
TEST_MARKER='l0'          # 只跑 l0 用例；生成 + 运行双阶段均生效
TEST_CONCURRENCY=10       # 降低并发（账号池紧张时）
MAX_ITERATIONS=3          # 只跑 3 轮，限制时间
SINGLE_TEST_RUN=true      # 只跑一轮"部署+测试"不进修复
GENERATE_TESTS=false      # 跳过 Stage 0，只用现有用例
SKIP_DEPLOY=true TCE_LANE=boe_flow_existing  # 跳过首轮部署，用已有泳道
```

## 只跑 Stage 0 生成测试（不跑部署/循环）

```
/fixloop-pytest BRANCH=feat/xxx SPEC_DIR=~/specs/xxx BUSINESS_REPO=~/TraeProjects/doubao IDL_REPO=~/TraeProjects/alice_idl IDL_BRANCH=feat/xxx MAX_ITERATIONS=0
```

`MAX_ITERATIONS=0` 让循环条件 `1 <= 0` 立即为假 → 跳过主循环，只产出 Stage 0 的测试文件和 user_journeys.md，Push 到 qa_model_effect 的 `feat/xxx` 分支。

## 单仓模式（显式锁定某个 sub-repo）

```
/fixloop-pytest \
  PSM=flow.alice.creativity \
  BRANCH=feat/xxx \
  BUSINESS_REPO=~/TraeProjects/doubao/creativity \
  SPEC_DIR=~/specs/xxx \
  IDL_REPO=~/TraeProjects/alice_idl \
  IDL_BRANCH=feat/xxx
```

`BUSINESS_REPO` 直指 `creativity` 子目录（含 `go.mod`） → 触发单仓模式，`PSM` 必填。

## 调试单条失败用例

如果某条 pytest 用例反复失败想隔离调试：

```
# 1. 手工编辑 pytest_selectors.txt 只留这一条
echo 'testcases/bots/aigc_video/gen_sse_video/test_sse_xxx.py::test_foo[params0-Env.Online]' > .costudio/craft/pytest_selectors.txt

# 2. 重跑（SKIP_DEPLOY + 已有 TCE_LANE 避免重新部署）
/fixloop-pytest BRANCH=feat/xxx BUSINESS_REPO=... SPEC_DIR=... IDL_REPO=... IDL_BRANCH=... \
  SKIP_DEPLOY=true TCE_LANE=boe_flow_abcd MAX_ITERATIONS=1 GENERATE_TESTS=false
```

## 常见退出码

| 退出码 | 场景 |
|---|---|
| 0 | 测试全通过 `TESTS_PASSED=true`；或 `SINGLE_TEST_RUN=true` 完成一轮 |
| 非 0 | 达到 `MAX_ITERATIONS` 仍未通过；或部署失败等 infra 错误 |

## 产出目录

```
.costudio/
  craft/                              # Stage 0
    ├── user_journeys.md
    ├── e2e_work_copy/testcases/...   # 同步到 qa_model_effect 的工作拷贝
    ├── generated_test_cases.jsonl
    ├── pytest_selectors.txt
    └── unit_tests/<sub_repo>/...
  detected_targets.jsonl              # workspace 嗅探结果
  iteration_1/
    ├── pytest.log
    ├── failed_cases.jsonl
    ├── test_stats.json
    ├── analysis_report.md
    ├── fix_summary.md
    └── iteration_summary.md
  iteration_2/
    ...
```

## 已知局限

- 测试仓库默认只支持 `qa_model_effect`。要换测试仓库需显式传 `TEST_REPO`
- 不支持 `testcases/web/`（依赖浏览器驱动）
- 账号池 `pool_user_name` 耗尽时 skill 不自动注册，只报 `[INFRA]` 建议人工处理
- workspace Git URL clone 不支持（需用户本地准备好 workspace）
