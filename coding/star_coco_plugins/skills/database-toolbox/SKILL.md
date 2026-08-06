---
name: database-toolbox
description: Database Toolbox 智能助手 - 支持 ByteRDS、ByteRedis、ByteDoc 数据库连接、查询、数据分析、运维观测、工单申请。当用户提到数据库、SQL、表结构、慢查询、数据分析报告、查数据、看看有哪些表等，都应使用此 Skill。
---

# Database Workbench Toolbox

Database Workbench 工具集，支持 ByteRDS、ByteDoc、ByteRedis 等多种类型数据库的查询和分析操作。

> **⚠️ 本 Skill 仅支持字节云平台。** 若用户提到"火山引擎"/"火山"，告知暂不支持。

> **帮用户多想一步** — 不只完成任务，更提供专家洞察。

## 🔴 核心原则 (必须遵守)

1. **安全第一**: 涉及数据变更 (DML/DDL) 时，必须使用工单，**严禁**直接执行高风险 SQL。
2. **场景路由**: 收到用户请求后，**立即**根据「场景路由」判断使用哪个场景，并读取对应 reference 文件。
3. **专家视角**: 从数据分析师视角出发，提供专业分析。
4. **数据诚实**: 绝不编造数据，图表不误导。
5. **结论先行**: 先说好还是不好，再说为什么。

## 🚦 场景路由 (Scenario Router)

根据用户意图，**必须**加载并遵循相应的参考文件：

| 用户意图 | 匹配场景 | **必读文件** | 关键函数 | 产出 |
| :--- | :--- | :--- | :--- | :--- |
| "有哪些表？"<br>"表结构是什么？"<br>"查看字段信息" | **元数据探查** | ByteRDS → `references/api/byterds/metadata-query.md`<br>ByteDoc → `references/api/bytedoc.md`<br>ByteRedis → `references/api/byteredis.md` | `list_tables`, `get_table_info`, `list_instances` | 表结构信息 |
| "查下最近订单"<br>"统计销售额"<br>"分析数据趋势"<br>"多表联合分析" | **数据分析 (BI)** | `references/analysis/analysis.md`<br>→ ByteRDS → `references/api/byterds/metadata-query.md`<br>→ ByteDoc → `references/api/bytedoc.md`<br>→ ByteRedis → `references/api/byteredis.md` | `create_workflow`, `resume_workflow`, `nl2sql`, `query_sql`, `execute_sql`；多数据源联合：`MultiSourceAnalyzer` | **HTML 可视化报告 + 截图** |
| "为什么慢？"<br>"有报错吗？"<br>"排查性能问题" | **运维诊断 (Ops)** | `references/ops/index.md`（按场景路由到排查 SOP）<br>→ SOP 如 `references/ops/byterds/slow-query.md`<br>→ API 参考：ByteRDS → `references/api/byterds/ops.md`<br>　ByteDoc → `references/api/bytedoc.md`<br>　ByteRedis → `references/api/byteredis.md` | `describe_aggregate_slow_logs`, `describe_slow_logs`, `list_active_sessions` | 诊断建议 |

### 支持的数据库类型

| db_type | 说明 | 依赖 |
| :--- | :--- | :--- |
| `ByteRDS` | MySQL 数据库（默认） | — |
| `ByteDoc` | 文档数据库 | — |
| `ByteRedis` | Redis 缓存服务 | — |

**Redis 识别规则**：用户提到 Redis / 缓存 / cache / PSM 含 `.redis.` → `db_type="ByteRedis"`。

> **execute_sql 仅支持只读操作。ByteRDS：SELECT、SHOW TABLES、SHOW CREATE TABLE、EXPLAIN。ByteDoc：MongoDB 语法（`show collections`、`db.<collection>.find()`、`db.<collection>.aggregate()`、`db.runCommand()`），不支持 MySQL 语法和 shell helper。ByteRedis：Redis 只读查询命令（如 GET、HGETALL、ZRANGE），不支持写入/删除/管理命令。**
>
> ⚠️ **3000 行截断**：`execute_sql` / `query_sql` 单次最多返回 3000 行，超出部分**静默截断**（不报错）。**返回恰好 3000 行 = 数据被截断，绝不能当作真实总数。** 需要真实计数时必须用 `SELECT COUNT(*)`。

## 执行方式

必须 `cd` 到本 skill 的 `scripts/` 目录下执行，否则模块导入失败。鉴权通过 AI PaaS CLI 自动获取 JWT token，无需手动配置。

```bash
cd <本skill目录>/scripts && python3 -c "
import json
from toolbox import create_client, list_tables
client = create_client(vregion='从用户问题提取或不传')
result = list_tables(client, database='从用户问题提取', fetch_all=True)
print(json.dumps(result, indent=2, ensure_ascii=False))
" 2>&1 | grep -v DEBUG
```

## 工作流

1. 从用户问题中提取 `database`（数据库名称）、`vregion`
2. `create_client(vregion=...)` 创建客户端
3. 调用具体函数，传入 `client` + `database` + 其他参数

> **⚠️ 字节云规则**：用户提到数据库名/库名/实例名，就是 `database` 参数。
> 代码自动识别实例类型、解析内部 ID。**不得再向用户确认，必须立即执行**。
> ByteRedis：用户提到 PSM / Redis 服务名，传入 `psm` 参数。

每个函数自动校验参数、从本地缓存补全缺失值、验证实例存在。

**vregion（虚拟地域）可选值：**

| vregion | 中文别名 | SITE |
| :--- | :--- | :--- |
| `China-North` / `cn` | 中国北部 / 华北 | cn |
| `ChinaSinf-North` | 华北2（北京） | cn |
| `China-North5` | 中国北部5 | cn |
| `China-East` | 中国东部 / 华东 | cn |
| `China-Fintech` / `sdqd` | 中国财经 / 山东青岛 | cn |
| `China-North3` | 中国北部3 | cn |
| `China-North6` | 中国北部6 | cn |
| `China-Pay` | 金融支付 | cn |
| `China-Pay2` | 金融支付2 | cn |
| `China-HKPay` | 香港支付 | cn |
| `China-Aggregation` | 聚合 | cn |
| `China-Enterprise` | 企业 | cn |
| `Aliyun_NC2` | 阿里云NC2 | cn |
| `China-BOE` / `boe` | 中国BOE | boe |
| `China-BOE2` | BOE2 | boe |
| `ChinaSinf-BOE` | 信服BOE | boe |
| `China-InfBOE` | InfBOE | boe |
| `Asia-SouthEastBD` | 亚太东南（柔佛） | i18n-bd |
| `Europe-WestBD` | 欧洲西部 | i18n-bd |
| `Singapore-SaaS` | 新加坡SaaS | i18n-bd |
| `Asia-SaaS` | 亚洲SaaS | i18n-bd |
| `US-EE` | 美国EE | i18n-bd |
| `Singapore-Common` | 新加坡教育通用 | i18n-bd |
| `US-EastBD` | 美国东部 | i18n-bd |
| `US-TTP3` | US-TTP3 | i18n-bd |
| `Australia-SouthEastBD` | 澳大利亚东南 | i18n-bd |
| `Singapore-Central` | 新加坡中部 | i18n-tt |
| `EasternEuro-TT` | 欧洲东部-TT | i18n-tt |
| `I18N-Game` | 国际化游戏 | i18n-tt |
| `Europe-Central` | 欧洲中部 | i18n-tt |
| `US-East` | 美国东部 | i18n-tt |
| `US-West` | 美国西部 | i18n-tt |
| `Australia-SouthEast` | 澳大利亚东南部 | i18n-tt |

> 表中 `/` 后的短名（`cn`、`boe`、`sdqd`）可直接传入 `create_client(vregion=...)`，自动映射。

> 函数速查表的可用范围列：`全部` = 所有 VRegion 可用；其他情况直接列出支持的 VRegion。

**调用示例：**

| 用户说法 | 调用方式 |
| :--- | :--- |
| "帮我查下 **BOE** 的 **byte_test** 有哪些表" | `create_client(vregion="boe")` → `list_tables(client, database="byte_test", fetch_all=True)` |
| "查下**华东**的 **dbw_ce** 有哪些表" | `create_client(vregion="China-East")` → `list_tables(client, database="dbw_ce")` |
| "查下 **cn** 的 **byte_test** 有哪些表" | `create_client(vregion="cn")` → `list_tables(client, database="byte_test")` |
| "查下 **byte_test** 有哪些表"（未提 vregion） | `create_client()` → `list_tables(client, database="byte_test")` — vregion 自动从缓存补全 |
| "查cn的test_tool库有哪些表，实例 ByteRDS" | `create_client(vregion="cn")` → `list_tables(client, database="test_tool", db_type="ByteRDS", fetch_all=True)` — **字节云用户已给出库名+region+类型，不需要任何确认，直接执行** |
| "有哪些数据库" | `create_client()` → `list_instances(client, favor=True)` — 须传过滤项：`database`（按名称搜索）、`psm`、`favor=True`（收藏）、`owned=True`（自建）。开放性查询优先 `favor=True` |

### 返回格式与 context

所有函数返回 `{success, message, data, context}`。**必须先检查 `success`，再使用 `data`**。

- `success: true` → 正常使用 `data`
- `success: false` + `error.missing` → 缺参数，向用户询问后补全重试
- `success: false` + `error.fuzzy_matches` → 模糊匹配到多个候选，**必须将候选列表展示给用户，让用户选择精确名称后再继续。禁止自动选择任何一个候选**
- `success: false` + 实例不存在 → **立即将错误信息原样告知用户，请用户自行确认名称、地域和类型。禁止做任何额外操作（不要换地域重试、不要换类型重试、不要搜缓存、不要列全部实例筛选）**

**`context`** 包含 `vregion`、`database`、`db_type`。
下一次调用时直接透传 context 中的值。

```python
# 上一步输出了: {"context":{"vregion":"China-North","database":"byte_test_db","db_type":"ByteRDS"}}
# 本步直接用 context 的值：
from toolbox import create_client, get_table_info
client = create_client(vregion="China-North")
info = get_table_info(client, table="users", database="byte_test_db",
                      db_type="ByteRDS")
```

### 数据查询

**两种方式可选，Agent 自行判断**（具体限制见上方 execute_sql 说明）：
- **nl2sql**：`list_tables` → `nl2sql(query, tables=[...])` → `execute_sql`。步骤少、速度快，但生成的 SQL 可能出现字段名偏差或条件遗漏。
- **查询 schema 后自写 SQL**：`list_tables` → `get_table_info` → 根据真实字段名自行编写 SQL → `execute_sql` / `query_sql`。步骤多，但能看到真实字段名和注释，SQL 更精准。

例外：`SHOW TABLES` / `SHOW CREATE TABLE` / `EXPLAIN` 等固定语句，或用户给出了完整 SQL，直接执行无需任何转换。

## Reference 目录

执行具体函数前，应先读取对应数据源的 reference 文件获取完整参数详情、返回格式和代码示例。

| 场景 | 数据源 | 路径 | 说明 |
| :--- | :--- | :--- | :--- |
| 数据分析工作流 | 通用 | `references/analysis/analysis.md` | 分析工作流：原则、7步流程、数据获取策略、多数据源联合、AnalysisWorkflow、分析框架 |
| 报告生成 | 通用 | `references/analysis/report.md` | 报告风格、模板选型、生成流程（Write + Playwright 截图）、配色 |
| 元数据 + 数据查询 | ByteRDS | `references/api/byterds/metadata-query.md` | 实例列表、表结构、缓存搜索、工单、nl2sql、execute_sql、query_sql |
| 全部 API | ByteDoc | `references/api/bytedoc.md` | 元数据 + 数据查询 + 运维诊断（一个文件） |
| 全部 API | ByteRedis | `references/api/byteredis.md` | 元数据 + 数据查询 + 运维诊断（一个文件） |
| 运维诊断 | ByteRDS | `references/api/byterds/ops.md` | 慢查询、全量SQL、事务锁、监控、空间 |
| 运维排查 SOP | 通用 | `references/ops/index.md` | **先读此文件**：按 db_type × 场景路由到具体 SOP |
| 运维排查 SOP | ByteRDS | `references/ops/byterds/*.md` | 慢查询、死锁、锁等待、表空间、会话问题（5 个场景） |
| 运维排查 SOP | ByteDoc | `references/ops/bytedoc/slow-query.md` | 慢查询 |
| 运维排查 SOP | ByteRedis | `references/ops/byteredis/*.md` | 慢查询、大 Key（2 个场景） |

## 函数速查

下表列出所有函数和所需参数，按数据源分组。

### ByteRDS

| 函数 | 分类 | 可用范围 | 说明 |
| :--- | :--- | :--- | :--- |
| `list_instances` | 元数据 | 全部 | 查询实例列表（须传过滤项：`database`/`psm`/`favor`/`owned`） |
| `list_tables` | 元数据 | 全部 | 列出表（`fetch_all=True` 获取全部） |
| `get_table_info` | 元数据 | 全部 | 获取表结构 |
| `search_cached_instances` | 元数据 | 全部 | 搜索本地缓存（不需要 client） |
| `set_global_config` | 元数据 | 全部 | 设置全局配置（不需要 client） |
| `get_ticket_url` | 元数据 | 全部 | 获取工单链接。写操作/加索引/改表/清表/调参/升降配时主动调用。**涉及 SQL 时需先生成可用 SQL** |
| `nl2sql` | 查询分析 | China-North, China-East, China-BOE, China-North5, China-Fintech | 自然语言转 SQL |
| `execute_sql` | 查询分析 | 全部 | 执行只读 SQL，返回 dict |
| `query_sql` | 查询分析 | 全部 | 执行查询，返回 DataFrame |
| `describe_aggregate_slow_logs` | 运维诊断 | 全部 | **推荐首选**：慢查询聚合统计 |
| `describe_slow_logs` | 运维诊断 | China-North, China-East, China-BOE, China-North5, China-Fintech | 慢查询明细（建议传 `sql_template_id` 过滤） |
| `slow_query_trend` | 运维诊断 | China-North, China-East, China-BOE, China-North5, China-Fintech | 慢查询时间序列趋势 |
| `slow_query_advice_task_history` | 运维诊断 | China-North, China-East, China-North5, China-Fintech | 诊断任务历史 |
| `list_slow_query_advice` | 运维诊断 | China-North, China-East, China-BOE, China-North5, China-Fintech | 获取优化建议 |
| `describe_full_sql_detail` | 运维诊断 | China-North, China-BOE, China-North5, China-Fintech | 全量 SQL 历史 |
| `list_active_sessions` | 运维诊断 | 全部 | 实时连接/进程（类似 SHOW PROCESSLIST） |
| `describe_deadlock` | 运维诊断 | 全部 | 查询死锁信息 |
| `list_transactions` | 运维诊断 | 全部 | 事务和锁列表 |
| `transaction_snapshots` | 运维诊断 | China-North, China-East, China-BOE, China-North5, China-Fintech | 事务快照 |
| `export_transactions` | 运维诊断 | China-North, China-East, China-BOE, China-North5, China-Fintech | 创建事务导出任务 |
| `table_write_analysis` | 监控空间 | China-North, China-BOE, China-North5, China-Fintech | 表级写入分析（按表聚合 DML/DDL 统计，定位写入最频繁的表） |
| `describe_table_metric` | 监控空间 | China-North, China-BOE, China-North5, China-Fintech | 表级监控 DML/DDL |
| `get_metric_data_predict` | 监控空间 | China-BOE | 监控数据预测（跨度 ≤ 7 天） |
| `describe_instance_nodes` | 监控空间 | 全部 | 实例节点列表（IP:Port、角色） |
| `describe_health_summary` | 监控空间 | 全部 | 实例健康概览（CPU、内存等） |
| `describe_table_space` | 监控空间 | 全部 | 表空间详情 |

### ByteDoc

> 用户只需传实例名称作为 `database`，传 `db_type="ByteDoc"` 即可，代码自动解析。

| 函数 | 分类 | 可用范围 | 说明 |
| :--- | :--- | :--- | :--- |
| `list_instances` | 元数据 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | 查询文档数据库实例列表（须传 `database` 或 `favor`） |
| `list_tables` | 元数据 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | 列出集合 |
| `get_table_info` | 元数据 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | 获取集合结构 |
| `execute_sql` | 查询分析 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | 执行只读查询（MongoDB 语法：`db.<col>.find()`、`db.<col>.aggregate()`） |
| `nl2sql` | 查询分析 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | 自然语言转查询语句（**必须指定 tables**） |
| `describe_slow_logs` | 运维诊断 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | 慢查询日志 |
| `list_active_sessions` | 运维诊断 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | 实时连接/进程 |
| `describe_instance_nodes` | 监控 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | 实例节点列表 |
| `describe_table_space` | 监控空间 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | collection 磁盘详情 |
| `get_ticket_url` | 元数据 | China-North, China-East, China-BOE, China-North5, China-Fintech, ChinaSinf-North | 获取工单链接 |

### ByteRedis

| 函数 | 分类 | 说明 |
| :--- | :--- | :--- |
| `list_instances` | 元数据 | 搜索 Redis 服务（`psm` 传 PSM） |
| `execute_sql` | 数据查询 | Redis 只读查询命令（GET、HGETALL、ZRANGE 等） |
| `describe_slow_logs` | 运维诊断 | Redis 慢日志 |
| `redis_list_big_keys` | 运维诊断 | 大 Key 分析（⚠️ China-BOE 不支持） |
| `get_ticket_url` | 元数据 | Redis 工单链接（扩容、缩容、执行命令、删Key） |

> **ByteRedis 参数说明**：`psm` 传 PSM（如 `toutiao.redis.explorer`），`sql` 为 Redis 只读查询命令。
> **⚠️ ByteRedis 不支持以下函数**：`list_tables`、`get_table_info`、`nl2sql`。Redis 是 Key-Value 存储，没有表和列的概念。查看 Redis 数据直接用 `execute_sql(client, sql="GET mykey", psm="...", db_type="ByteRedis")`。

### 数据分析工作流 & 多数据源联合

**AnalysisWorkflow**（`from analysis_workflow import create_workflow, resume_workflow`）：多步分析任务的跨执行持久化。需要 2 步以上的分析任务必须用 workflow。核心方法：`create_workflow()` 创建、`resume_workflow(analysis_id)` 恢复、`wf.save_step_output()` / `wf.load_step_output()` 持久化。详见 `references/analysis/analysis.md`。

**MultiSourceAnalyzer**（`from multi_source_analyzer import MultiSourceAnalyzer`）：基于 DuckDB 的跨数据源 SQL 联合查询。用 `register_dataframe()` / `register_file()` 注册数据源，`query(sql)` 执行联合查询。详见 `references/analysis/analysis.md`。

## 参数说明

> **参数补全规则**：`vregion` 在 `create_client(vregion=...)` 时传入，不传则从全局配置读取。
> `database` 不传则从 `psm` 推导（`psm.split(".")[-1]`）或从全局配置读取。
> `db_type` 不传则从缓存 → 全局配置 → 默认 "ByteRDS"。
> `vdc` 可选，同名数据库在多个机房（VDC）中存在时用于消歧。正常情况不传；当函数返回 `fuzzy_matches` 错误（含 `vdc` 字段列表）时，将用户选择的 vdc 传入后续所有调用。返回值 `context.vdc` 可直接透传。
> **大数据量截断**：返回列表较多时，`data` 中会包含 `total`（原始总数）、`returned_count`（本次返回数量）、`truncated: true` 和 `artifact_path`（完整数据的临时文件路径，JSON 格式）。这些元数据字段在 JSON 中位于列表字段之前。当 `truncated=true` 时，根据任务判断是否需要完整数据：定位 Top 问题（"有没有慢查询"、"最慢的 SQL"）inline 数据已足够；全量统计、分布分析、遍历所有条目时，基于 `artifact_path` 文件进行读取或分析（如用 python/jq 做聚合统计）。
> Agent 不需要知道 VeDBMySQL、Mongo 等内部类型。

## 🚨 错误处理

| 错误情况 | 处理方式 |
| :--- | :--- |
| `nl2sql` 生成的 SQL 有误 | 用 `get_table_info` 获取真实字段名后自行编写 SQL |
| 缺少 `instance_id` | **必须**先调用 `list_instances()` 探查，**不可**瞎编 |
| 执行 SQL 被安全规则拦截 / 用户要求写操作（INSERT/UPDATE/DELETE/DDL） | **先生成可用 SQL**（用 `nl2sql` 或 `get_table_info` 了解表结构后自行编写），再调用 `get_ticket_url(client, database=...)` 获取工单链接，**同时提供 SQL 和工单链接** |

## ⚠️ 必须询问用户的情况

- 字段含义不明（无法从字段名/注释判断业务含义）
- 多个表都相关（不确定该查哪个表）
- 列值取值不明（英文值无法对应业务含义）
- 术语不熟悉（成功率指的是什么？）
- 缺少必要参数（无法推断 instance_id、database 等）

## 注意事项

- **本地缓存 ≠ 收藏 ≠ 实际数据**：
  - "收藏的数据库" → `list_instances(client, favor=True)` — 远程 API 查平台收藏夹
  - "之前用过的 / 本地缓存" → `search_cached_instances()` — 本地历史记录
  - 用户提到具体库名 → `list_instances(client, database="xxx")` 按名称搜索
  - 开放性查询（"有哪些数据库"） → `list_instances(client, favor=True)` 查收藏；无结果时 `owned=True` 查自建
- **HTTP 401**：提示用户重新登录 AI PaaS CLI，不要反复重试。
