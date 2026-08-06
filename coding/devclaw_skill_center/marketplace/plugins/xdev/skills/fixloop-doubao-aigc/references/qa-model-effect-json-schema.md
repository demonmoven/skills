# qa_model_effect `.online.json` 字段规范

`@test.runtime(Env.Online, file='xxx')` 装饰的 pytest 函数会从 `test_data/xxx.online.json` 加载参数化。文件是一个 list，每个元素是一条 case 数据。

## 字段清单（⚠️ 严格遵守分层）

顶层字段：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `title` | str | 建议 | 中文描述本条 case 验证的行为 |
| `skip` | bool | 否 | `true` 表示跳过，evtest 会过滤掉；默认 `false` |
| `skip_reason` | str | 跟 `skip` 配套 | skip 时必填，说明原因 |
| `params` | dict | **是** | 测试所需所有运行时参数（含 `pool_user_name` / `tag_name_list` / `bot_id` / `send_text` / `ENV_LABEL` / …） |
| `expected` | dict / list | 建议 | 预期结果，用于断言；结构按测试业务自定义 |
| `other` / `marks` | list/dict | 否 | evtest 框架级别元数据（测试标签等） |

**`params` 内必填子字段**（测试代码通过 `params.get(...)` 取用）：

| 子字段 | 类型 | 注入为环境变量 | 说明 |
|---|---|---|---|
| `pool_user_name` | str | — | 账号池名（例：`ec_stable_biz`、`liweiyi.11`）；测试用 `params.get("pool_user_name")` 读 |
| `tag_name_list` | list[str] | — | 账号池 tag 过滤；`params.get("tag_name_list")` 读 |
| `bot_id` | str | — | 具体 bot 标识；查 `$TEST_REPO_PATH/constant/` 下的常量文件 |
| `send_text` | str | — | 用户发送文本（SSE 测试） |
| `ENV_LABEL` | str | `ENV_LABEL` | **PPE 泳道名（即 TCE_LANE）**；由 `conftest.py::pytest_runtest_call` 注入为环境变量，下游代码（`common/im/api.py` 等）读取后作为 `x-tt-env` header 染色 |
| `version_id` | str | `version_id` | 单个实验 ID |
| `version_ids` | str | `version_ids` | 多个实验 ID |
| `Minor-Mode` | int | `Minor-Mode`（=`"1"` 当值 == 1 时） | 青少年模式开关 |

### ⚠️ 致命反模式（生成 case 时曾犯过的错，勿再犯）

**反模式 1：把 `pool_user_name` / `tag_name_list` / `bot_id` 放在顶层而非 `params` 内**
- 错误结构：`{"title":"...","pool_user_name":"liweiyi.11","tag_name_list":[...],"params":{...}}`
- 后果：测试代码 `params.get("pool_user_name")` 取到 `None` → `CountPool(user_name=None)` → 发 occupy 请求时 body `{"user_name": null}` → 服务端 HTTP 400 `"Params Err, msg=user name can not by empty"` → 整轮测试全部挂在账号池阶段，表面看像"网络/JWT/账号池故障"，极难诊断
- 正确结构：`pool_user_name` / `tag_name_list` / `bot_id` 必须在 `params` 里

**反模式 2：`params.ENV_LABEL` 留空字符串并假设 Stage 2 通过 shell `export` 注入**
- 错误假设：`params.ENV_LABEL=""` + `export ENV_LABEL=$TCE_LANE` → 以为染色生效
- 真相：`conftest.py::pytest_runtest_call` 在**测试执行前**用 `os.environ["ENV_LABEL"] = str(test_params["ENV_LABEL"])` **覆盖**外部 export 的值。空字符串被设进 env，`x-tt-env` header 为空，请求落到 **prod 线上**，PPE 泳道的业务代码永远测不到
- 诊断特征：响应头 `X-Backend: *|prod|*`（本应是 PPE 泳道名）；业务断言失败但看不出明显原因
- **正确做法**：**生成 fixture 时**直接把 `params.ENV_LABEL = <TCE_LANE>`（例如 `"ppe_wutingjia_test"`）。若本轮不知道 TCE_LANE，至少留 placeholder 并在 Stage 0.4 同步前用 Python/`jq` 脚本替换成实际值

## 示例（正确结构）

```json
[
  {
    "title": "正常发消息-默认 bot-中文问候",
    "skip": false,
    "params": {
      "bot_name": "豆包",
      "bot_id": "7296571464244985907",
      "send_text": "你好",
      "pool_user_name": "ec_stable_biz",
      "tag_name_list": ["iphone"],
      "ENV_LABEL": "ppe_wutingjia_test",
      "version_id": ""
    },
    "expected": {
      "not_reject": true,
      "min_reply_len": 2
    }
  }
]
```

注意：`pool_user_name` / `tag_name_list` / `bot_id` / `ENV_LABEL` **全部在 `params` 里**。`ENV_LABEL` 写实际 TCE_LANE 值，不是空字符串。

## 文件命名约定

- 路径：`test_data/<subcategory>/<feature>.<runtime>.json`（但 qa_model_effect 不严格要求 `test_data/` 子目录；也见直接在模块目录下放 JSON 的）
- `<runtime>` 取值：`online` / `i18nalisg` / `i18nmaliva`（与 `RUNTIME_ENV` 对齐）
- 文件名 = 装饰器 `file=` 参数的值 + `.{runtime}.json`

**生成时**：遵循该模块**邻居的实际惯例**（有的模块用 `test_data/` 子目录，有的不用）。通过 `find testcases/<目标目录> -name '*.online.json' | head -3` 观察一下。

## Minor 字段发现

如果目标模块的邻居 JSON 包含本文档未列的字段（如 `version_id_cn` 等方言），优先**保留邻居字段名**，不强行改名；JSON schema 对齐 = 邻居集的子集 ∪ 邻居集的超集的保守部分。

## 账号池不自动注册

生成的 case 如果账号池 `pool_user_name` 或 tag 在邻居里找不到匹配，**不**自动发明新池名 —— 用邻居里最常见的池名 + 标注 `# NEEDS_REVIEW: 账号池可能需人工调整` 注释到 `.py` 文件里。
