# 邻居驱动的测试发现协议

Stage 0 Step 0.3 (E2E 生成) 子流程。面向 qa_model_effect 风格：pytest + 外部 JSON 数据 + `@test.runtime` 装饰器 + 账号池。

## 核心原则

1. **SPEC 驱动定位**：从 SPEC 内容推断目标测试所属模块目录；禁止在本 skill 里硬编码任何具体模块名（如 `im/chat`）
2. **邻居驱动模仿**：定位模块后，读该目录下的**现有活跃测试**作为风格样例，而不是凭空生成
3. **JSON schema 对齐**：新生成的 `.online.json` 字段集合必须是邻居 JSON 的子集（或严格超集——不发明新字段）
4. **账号池复用**：`pool_user_name` / `tag_name_list` 只从邻居 JSON 抽取高频值，不尝试自动注册新池

## 四步协议

### Step A · 模块定位（给每条 user_journey 找归属目录）

> **本 skill 面向 Doubao/Flow 创作（creativity）业务，候选目录收窄为以下三个 AIGC 子目录**，不再扫描全量 `testcases/`：
>
> | 目录 | 职责边界 |
> |---|---|
> | `testcases/bots/aigc_case` | 通用 AIGC 场景：文生图（`gen_image` / `gen_pic` / `pocket_gen_image`）、社区创作（`aigc_communitycreation`）、音乐生成（`gen_sse_music`）、个人收藏（`test_personal_collection`）、Bot 主页交互（`test_main_bot_actionbar` / `test_tab_plus_page`） |
> | `testcases/bots/aigc_fangzhou` | 方舟（豆包视觉大模型 Seed）后端相关：`text2image`、`image2image` |
> | `testcases/bots/aigc_video` | 视频生成场景：`gen_sse_video`、创作进度查询（`creation_progress_query`） |
>
> 如果某条 user_journey 无法归入上述三个目录，应在日志里提示 **"超出 AIGC 范围，请确认是否要扩展 fixloop-pytest 的覆盖边界"**，并标 `needs_review: true`；**不要**自行落到其他目录（如 `im/`、`safety/`）。

```
1. ls $TEST_REPO_PATH/testcases/bots/{aigc_case,aigc_fangzhou,aigc_video} → 三个候选根目录及其二级子目录清单

2. 对每个候选根目录 d（最多三个）:
   - 枚举 d/ 下二级目录（见上表职责说明）
   - 取 2-3 个近 30 天修改的 .py 文件名和 docstring
   - LLM 结合表中职责边界归纳子模块的具体职责（一句话）

3. 对每条 user_journey：
   - LLM 按 SPEC + user_journey 关键词（如"视频生成/进度查询"→ aigc_video；"文生图/图生图"→ aigc_fangzhou 或 aigc_case，前者偏后端，后者偏入口场景）推断归属根目录
   - 尽量细到二层（如 aigc_case/gen_pic、aigc_fangzhou/text2image），不要只停在根
   - 输出：{journey_id: [目标目录1, 目标目录2, ...]}
   - 归不上去的：标 needs_review，写日志等人工确认
```

**禁止的反模式**：
- ❌ 跳过三个 aigc 目录的边界，落到 `testcases/im/`、`testcases/safety/` 等不相关模块
- ❌ 在 SKILL.md 或本 reference 里把"某个 UJ 关键词 → 某个具体二级目录"的映射写死到字段级（二级目录选择仍由 LLM 从当前邻居归纳，避免目录结构演进后失效）
- ❌ 为每种模块单独写一份 reference（与"SPEC 驱动定位"矛盾）

### Step B · 邻居探索（对每个目标目录学风格）

对每个目标目录，选**参考样例**：

```
cd $TEST_REPO_PATH/testcases/<目标目录>
# 按 commit 时间排序，取近 30 天且非全 skip 的测试文件
git log --since=30.days --name-only --pretty=format: testcases/<目标目录>/*.py \
  | sort -u | head -10 → 候选集

# 过滤：每个文件的 .online.json 里至少有一条 skip != true
# 取 2-3 个作为"邻居样例"
```

对每个邻居样例，LLM 提取：
- **装饰器 pattern**：`@test.runtime(Env.Online, Env.I18nAlisg, file='xxx')` 的参数组合
- **测试函数骨架**：入参（通常 `app, params, request`）、fixture 使用、返回值断言风格
- **JSON schema**：字段集合（`pool_user_name`, `tag_name_list`, `bot_id`, `params.ENV_LABEL`, `params.version_id`, `expected`, `skip`, `title`, ...）
- **导入 pattern**：`from common.im.api import AliceApi` 这类
- **Marker 使用**：文件级 `pytestmark = pytest.mark.l0` 或函数级 `@pytest.mark.l0`

合成**邻居模板**：一份 pytest 函数骨架 + 一份 JSON schema 清单。

### Step C · 测试生成

基于邻居模板合成新测试：

1. **文件命名**：`test_<feature_name>.py`；放入推断的目标目录（Step A 产出）
2. **装饰器**：`@test.runtime(Env.Online, file='<feature_name>')`；环境集从 user_journey 的适用环境决定
3. **Marker**：优先 `l0`（核心链路）或 user_journey 显式指定的 marker；不确定时用 `bot_engine`
4. **JSON 数据**：
   - `pool_user_name` / `tag_name_list`：从邻居 JSON 的高频取值采
   - `bot_id`：从 SPEC 或 user_journey 推（SPEC 里一般会提对应 bot 名，查 `$TEST_REPO_PATH/constant/` 下是否有映射）；找不到就写邻居里最常见的
   - `params.ENV_LABEL`：留空字符串（Stage 2 运行时由 `TCE_LANE` 注入）
   - `title`：描述该条 case 要验证什么（中文 OK）
   - `skip`: `false`（生成的 case 默认不 skip）
5. **测试函数**：
   - 参考邻居的"取号 → 清上下文 → 发请求 → 校验 → 释放"标准流
   - 断言使用 `common/im/asserts.py` 等现有断言库（不自建）
   - LogID 提取使用 `common/im/sse.py` 里的工具
6. **参数化条数**：每个 user_journey 生成 2-3 条参数化（正常路径 + 1-2 个边界）

### Step D · 登记

生成：

```
$OUTPUT_DIR/craft/e2e_work_copy/
  └── testcases/
      └── <推断的目标目录>/
          ├── test_<feature>.py
          └── test_data/
              └── <feature>.online.json

$OUTPUT_DIR/craft/generated_test_cases.jsonl
  # 每条 user_journey 的每条参数化一行：
  {"module": "im/chat", "test_file": "testcases/im/chat/test_xxx.py",
   "test_func": "test_foo", "data_file": "test_data/xxx.online.json",
   "param_index": 0, "marker": "l0", "source_journey": "UJ-001"}

$OUTPUT_DIR/craft/pytest_selectors.txt
  # 每行一个 pytest node-id（供 Stage 2 精确运行）：
  testcases/im/chat/test_xxx.py::test_foo[params0-Env.Online]
  testcases/im/chat/test_xxx.py::test_foo[params1-Env.Online]
```

## 同步规则（Step 0.4）

```bash
cp -r $OUTPUT_DIR/craft/e2e_work_copy/* $TEST_REPO_PATH/
cd $TEST_REPO_PATH
# 基于远程默认分支创建 $BRANCH
DEFAULT_BRANCH=$(git remote show origin | grep 'HEAD branch' | awk '{print $NF}')
git checkout "$DEFAULT_BRANCH" && git pull
git checkout -b "$BRANCH" 2>/dev/null || git checkout "$BRANCH"
git add -A
git commit -m "feat: auto-generated E2E tests for $FEATURE_NAME"
git push origin "$BRANCH"
```

## 失败兜底

- **Step A 定位不到目标目录**：LLM 返回空数组时，直接报错停止 Stage 0，提示"SPEC 与现有 testcases/ 模块映射失败，请补充 SPEC 或指定目标目录"
- **Step B 邻居样例为空**（新模块 / 目录空）：降级为"读 SPEC + testcases/ 整体 README 生成最基础骨架，标注 NEEDS_REVIEW 注释"，并在 generated_test_cases.jsonl 里 `needs_review: true`
- **Step C bot_id 等业务字段找不到**：留 TODO 注释，生成 case 但标 `skip: true` + `skip_reason: "bot_id 待人工填"`
