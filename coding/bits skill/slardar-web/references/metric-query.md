# Metric Query — 指标查询参考

指标构建（build）、时序数据查询（query）和关联字段（related）的完整参数参考。

指标搜索（search）、指标目录和数据格式说明见 [metric-search.md](metric-search.md)。

## Contents

- metric build — 构建 measure_list（6种模式、参数说明、禁止手写 JSON）
- metric query — 查询时序数据（完整参数表、分组/筛选字段、图表类型、输出控制）
- metric related — 获取关联字段
- 补充参考: unit 速查、事件指标 filter_name 格式、polynomial 手动构建

---

## metric build — 构建 measure_list

使用 `slardar-web-cli metric build` 自动生成 measure_list JSON，供 `metric query` / `metric related` 使用。

### 各模式完整参数

#### basic — 基础指标

```bash
MEASURE=$(slardar-web-cli metric build --type basic --name "UV" --measure-key "pv_uv.user")
```

#### custom — 业务自定义指标

```bash
# formal_key 来自 Find Metrics，作为 --measure-key 传入
# --name 使用指标原名（如 Find Metrics 返回的完整名称）
MEASURE=$(slardar-web-cli metric build --type custom --name "L0 | 桌面端 | 划词 | 发消息失败率" --measure-key "<uuid>" --unit-type percent --unit %)
```

#### event — 事件指标

```bash
MEASURE=$(slardar-web-cli metric build --type event --name "上报量" --measure-key '<json>' --event-name "api_statistic")
```

#### formula — 公式模式

使用 `--formula` 和 `--members` 构建公式型指标。

#### extract — 从公式型指标中提取单个成员

```bash
# --metric-def 传入 Find Metrics 返回的单个指标完整 JSON 定义
MEASURE=$(slardar-web-cli metric build --type extract --metric-def '<指标JSON>' --member A --name "分子")
# --member A=分子, B=分母；指标 JSON 需包含 metrics.members 和 metrics.formula 字段
```

#### batch — 多指标批量构建（推荐）

> **⚠️ 多指标场景必须使用以下方式之一，不要直接拼接单指标的数组输出，否则会产生数组嵌套导致 API 报错。**

**方式 1：`--batch` 一次构建（推荐）**

传入 JSON 数组，一次性生成包含多个指标的扁平 measure_list：

```bash
MEASURE=$(slardar-web-cli metric build --batch '[
  {"type": "basic", "name": "PV", "measure_key": "pv_uv.count"},
  {"type": "basic", "name": "UV", "measure_key": "pv_uv.user"}
]')
```

batch 数组中每个对象支持的字段：

- `type`（必填）：`basic` / `custom` / `event`
- `name`：显示名称
- `measure_key`（必填）：指标标识符
- `event_name`：事件名（event）
- `unit_type`：`number` / `time` / `percent`（默认 `number`）
- `unit`：`1` / `ms` / `s` / `%`（默认 `1`）
- `filter`：筛选条件数组

**方式 2：`--unwrap` + shell 拼接**

每次输出单个对象（不带外层数组），通过 shell 手动组合：

```bash
M1=$(slardar-web-cli metric build --type basic --name "PV" --measure-key "pv_uv.count" --unwrap)
M2=$(slardar-web-cli metric build --type basic --name "UV" --measure-key "pv_uv.user" --unwrap)
MEASURE="[$M1,$M2]"
```

### build 参数说明

| 参数            | 说明                                                     |
| --------------- | -------------------------------------------------------- |
| `--type`        | 指标类型：basic / custom / event / formula / extract     |
| `--name`        | 显示名称。使用 Find Metrics 返回的指标完整原名，不要缩写 |
| `--measure-key` | 指标标识符（basic/custom/event 统一使用）                |
| `--event-name`  | 事件名（event 必填）                                     |
| `--unit-type`   | 单位类型：number / time / percent（默认 number）         |
| `--unit`        | 单位：1 / ms / s / %（默认 1）                           |
| `--filter`      | 筛选条件 JSON（默认 []）                                 |
| `--members`     | formula 模式的成员定义 JSON                              |
| `--formula`     | 公式表达式（formula 模式）                               |
| `--metric-def`  | 完整指标定义 JSON（extract 模式）                        |
| `--member`      | 提取的成员别名（extract，默认 A）                        |
| `--batch`       | 多指标批量构建 JSON 数组                                 |
| `--unwrap`      | 输出单个对象而非数组                                     |

### ⚠️ 禁止手写 JSON 规则

> **禁止手写 measure JSON**。`--measure` 参数的值 **必须** 来自 `metric build` 的输出（即上一步的 `$MEASURE` 变量），严禁自行拼接 JSON 字符串传入 `query` / `related` 等命令。手写 JSON 极易丢失必要字段（如 `filter_list`、`unit_type`），导致查询失败或结果错误。

---

## metric query — 查询时序数据

查询 Slardar 监控数据，支持指标选择、维度分组、条件筛选，返回时序趋势数据。

```bash
slardar-web-cli metric query \
  --bid <BID> \                    # [必须] 业务 ID
  --measure '<measure_list JSON>' \ # [必须] measure_list JSON 数组（来自 metric build 输出）
  --start-time <ts> --end-time <ts> \ # 时间范围 Unix 时间戳（秒）
  --granularity <秒数> \           # 时间粒度（默认自动选择）
  --group '<JSON>' \               # 分组字段 JSON 数组
  --filter '<JSON>' \              # 筛选条件 JSON 数组
  --chart line \                   # 图表类型（默认 line）
  --env <value> \                  # 指定环境
  --all-env \                      # 查所有环境
  --site-type web \                # 站点类型: web/app/hybrid（默认 web）
  --region cn \                    # 区域: cn/i18n（默认 cn）
  --raw \                          # 输出原始 JSON（不格式化）
  --no-share \                     # 跳过分享链接生成
  --show-all \                     # 显示全部数据点（默认超过 30 点智能截断）
  --json-out \                     # 输出标准化 JSON
  --show-request \                 # 输出请求体 JSON（用于归因衔接）
  --summary \                      # 仅输出汇总表格（分组场景推荐）
  --top <N> \                      # 仅展示 Top N 个分组（按 sum 降序）
  --sort-by <sum|avg>              # 排序依据：sum（默认）或 avg
```

### 完整参数表

| 参数             | 必须 | 默认值   | 说明                                               |
| ---------------- | ---- | -------- | -------------------------------------------------- |
| `--bid`          | 是   | -        | 业务 ID                                            |
| `--measure`      | 是   | -        | measure_list JSON 数组（来自 `metric build` 输出） |
| `--start-time`   | 否   | 7天前    | 开始时间戳（秒）                                   |
| `--end-time`     | 否   | 当前     | 结束时间戳（秒）                                   |
| `--granularity`  | 否   | 自动     | 时间粒度秒数                                       |
| `--group`        | 否   | []       | 分组字段 JSON 数组                                 |
| `--filter`       | 否   | []       | 筛选条件 JSON 数组                                 |
| `--chart`        | 否   | line     | 图表类型（见下方图表类型说明）                     |
| `--env`          | 否   | 自动检测 | 环境                                               |
| `--all-env`      | 否   | false    | 查所有环境                                         |
| `--site-type`    | 否   | web      | 站点类型 web/app/hybrid                            |
| `--region`       | 否   | cn       | 区域 cn/i18n                                      |
| `--no-share`     | 否   | -        | 跳过分享链接生成                                   |
| `--top`          | 否   | 0        | 分组场景仅展示 Top N                               |
| `--sort-by`      | 否   | sum      | 排序方式 sum/avg                                   |
| `--summary`      | 否   | false    | 仅输出汇总表格                                     |
| `--show-all`     | 否   | false    | 显示所有数据点（默认超 30 点智能截断）             |
| `--json-out`     | 否   | false    | 输出标准化 JSON                                    |
| `--show-request` | 否   | false    | 输出请求体 JSON                                    |
| `--raw`          | 否   | false    | 输出原始 API 响应 JSON                             |

#### 时间范围参数

`--start-time <ts> --end-time <ts>`：Unix 时间戳（秒）。

当用户提供自然语言时间范围（如"最近7天"、"昨天"、"上周"等），**必须**先调用 `slardar-web-cli time parse` 将其转为 Unix 时间戳，再传入 `--start-time` / `--end-time`。**禁止**自行编造时间戳数值。

不传时间参数默认最近 7 天。

#### 粒度选择

不传 `--granularity` 时自动选择：

| 时间范围 | 自动粒度        |
| -------- | --------------- |
| ≤1h      | 60（1 分钟）    |
| ≤6h      | 300（5 分钟）   |
| ≤1d      | 1800（30 分钟） |
| ≤2d      | 3600（1 小时）  |
| ≤30d     | 86400（天级）   |
| >30d     | 604800（周级）  |

手动指定时从下方粒度表中选取。`(end-start)/granularity` 不宜超过 1000。

---

### 分组/筛选常用字段

#### 分组字段 (group_by_list)

##### 基础维度

| label      | group_by_name     | 说明         |
| ---------- | ----------------- | ------------ |
| pid        | `pid`             | 页面 ID      |
| 浏览器     | `browser_brand`   | 浏览器品牌   |
| 浏览器版本 | `browser_version` | 浏览器版本   |
| 系统       | `os`              | 操作系统     |
| 系统版本   | `os_version`      | 操作系统版本 |
| 品牌       | `device_brand`    | 设备品牌     |
| 机型       | `device_model`    | 设备机型     |

##### 地理位置

| label     | group_by_name |
| --------- | ------------- |
| 国家/地区 | `country`     |
| 省份      | `province`    |
| 城市      | `city`        |

##### 网络

| label    | group_by_name  |
| -------- | -------------- |
| 网络类型 | `network_type` |
| 运营商   | `isp`          |
| IP       | `ip`           |

##### 页面

| label       | group_by_name  |
| ----------- | -------------- |
| 页面url     | `url`          |
| page domain | `pv_domain`    |
| 完整url     | `complete_url` |
| 路径        | `path`         |
| 请求参数    | `query`        |

##### 应用

| label        | group_by_name |
| ------------ | ------------- |
| env          | `env`         |
| release      | `release`     |
| loading type | `source`      |

##### 用户标识

| label      | group_by_name |
| ---------- | ------------- |
| user_id    | `user_id`     |
| device_id  | `device_id`   |
| session_id | `session_id`  |
| view_id    | `view_id`     |

##### Context 自定义字段

上报时设置的自定义 context，格式：`{"dimension":"context","map_key":"<字段名>"}`

##### 事件自定义字段

事件指标的自定义维度，格式见下方「补充参考 → 事件指标自定义字段 filter_name 格式」。

#### 筛选字段 (filter_list)

筛选字段名与分组字段相同（`filter_name` = `group_by_name`）。额外的特殊筛选：

| label            | filter_name         |
| ---------------- | ------------------- |
| 只看新增版本数据 | `is_new_release`    |
| 只看最近版本数据 | `is_latest_release` |

#### 时间粒度 (granularity)

| label  | granularity |
| ------ | ----------- |
| 1分钟  | `60`        |
| 5分钟  | `300`       |
| 10分钟 | `600`       |
| 30分钟 | `1800`      |
| 小时   | `3600`      |
| 6小时  | `21600`     |
| 12小时 | `43200`     |
| 天级   | `86400`     |
| 周级   | `604800`    |

> 自定义周级（如仅工作日 `week_weekday_only`、指定起始日 `week_316800` 等）较少使用，需要时通过 `slardar-web-cli metric related --bid <BID> --measure '<measure_list JSON>'` 确认可用粒度。

---

### 图表类型

> `--chart` 绝大多数场景无需指定，仅用户明确指定图表类型或特定分析需求时才需要。
> 分布 / distribution 类型指标仅支持使用分布图（`--chart histogram`）。

| 类型             | 说明           | 使用场景                                                                                                                             |
| ---------------- | -------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| `line`           | 折线图（默认） | 时序趋势，最常用                                                                                                                     |
| `bar`            | 柱状图         |                                                                                                                                      |
| `stacked_column` | 堆叠图         |                                                                                                                                      |
| `combo`          | 组合图         |                                                                                                                                      |
| `table`          | 数据表         |                                                                                                                                      |
| `pivot_table`    | 透视表         | 维度交叉分析（支持 `--granularity all` 查询时间段内聚合数据）                                                                        |
| `pie`            | 饼图           | 占比分布                                                                                                                             |
| `indicator_card` | 指标卡         | 单值概览（**仅支持单指标**）                                                                                                         |
| `histogram`      | 分布图         | 数值分布区间（**仅支持单指标**，**仅支持分布/distribution 指标**，可配合 `--hist-bin`/`--hist-min`/`--hist-max`/`--hist-long-tail`） |

#### 单指标限制

`indicator_card`、`histogram` 两种类型仅支持单个指标，传入多个指标会报错。

#### Histogram 指标类型限制

`histogram` 仅支持 **分布 / distribution** 类型的指标，非分布类型指标会报错。

#### Histogram 专用参数

| 参数               | 说明                             | 默认值 |
| ------------------ | -------------------------------- | ------ |
| `--hist-bin`       | 分桶数量                         | `20`   |
| `--hist-min`       | 最小值（不传则由服务端自动计算） | —      |
| `--hist-max`       | 最大值（不传则由服务端自动计算） | —      |
| `--hist-long-tail` | 显示长尾分布                     | `true` |

---

### 分组输出控制（--summary / --top / --sort-by / --json-out）

| 参数          | 说明                                             | 示例               |
| ------------- | ------------------------------------------------ | ------------------ |
| `--summary`   | 仅输出汇总表格（每个分组一行），不展开每日数据点 | 分组多时**必用**   |
| `--top N`     | 只展示 Top N 个分组（按 sum 降序）               | `--top 20`         |
| `--sort-by X` | 排序依据：`sum`（默认）或 `avg`                  | `--sort-by avg`    |
| `--json-out`  | 输出标准化 JSON（供程序二次处理）                | 需要精确计算时使用 |

**分组查询推荐用法**：

```bash
# 分组数量多时（>10 个），用 --summary --top 显示汇总排名
slardar-web-cli metric query ... --group '[...]' --summary --top 20

# 需要看具体趋势时，用 --top 限制数量（多指标会按分组值聚合展示）
slardar-web-cli metric query ... --group '[...]' --top 5

# 需要程序化处理时，用 --json-out 获取标准化结构
slardar-web-cli metric query ... --group '[...]' --json-out
```

---

### 结果呈现说明

> **分享链接已自动生成**：query 命令查询成功后会自动调用 save_query_config 生成 Slardar 分享链接，无需额外操作。如需跳过，使用 `--no-share` 参数。

向用户展示：

1. **数据摘要**：指标名、时间范围、平均值/总和（已自动带单位）
2. **趋势描述**：如"FCP 从 1,523.45 ms 下降至 1,389.12 ms"
3. **分组对比**（如有）：按分组值聚合展示（同一分组的多指标紧邻排列）
4. **分享链接**：Slardar 平台详细图表

> **分组数量多时**：必须加 `--summary --top N` 只看 Top N 汇总，避免输出过长。默认按 sum 降序排列。

> **单位自动推断**：CLI 按优先级推断：显式 unit > 公式模式 > 名称关键词 > 值域范围 > 默认 number。时间指标显示 ms、比率指标自动转百分比、计数指标千分位分隔。无需手动处理。

> **--json-out 输出结构**：标准化 JSON 包含 `series`（平铺列表，每项含 `name`/`metric`/`group`/`avg`/`sum`/`data`/`max`/`min`）和 `grouped`（按分组值聚合的字典视图）。便于直接解析排序，无需猜测 API 响应格式。

---

## metric related — 获取关联字段

用于获取指标可用的分组/筛选/粒度字段（对应 API: `meta_measure_related`）：

```bash
slardar-web-cli metric related --bid <BID> --measure '<measure_list JSON>'
```

| 参数          | 必须 | 默认     | 说明              |
| ------------- | ---- | -------- | ----------------- |
| `--bid`       | 是   | -        | 业务 ID           |
| `--measure`   | 是   | -        | measure_list JSON |
| `--region`    | 否   | cn       | 区域              |
| `--env`       | 否   | 自动检测 | 环境              |
| `--all-env`   | 否   | false    | 查所有环境        |
| `--site-type` | 否   | web      | 站点类型          |

> `--measure` 为必传参数：不同指标（尤其自定义指标）可用的分组/筛选字段不同。输出中 `->` 右侧即为 `group_by_name`（分组）或 `filter_name`（筛选）的值，直接用于 query 的 `--group` / `--filter` 参数。结果自动缓存 24 小时。

---

## 补充参考

### unit 字段速查

| unit_type | unit   | 适用场景               |
| --------- | ------ | ---------------------- |
| `number`  | `1`    | 计数类：UV、PV、错误数 |
| `time`    | `ms`   | 时间类：FCP、LCP、TTI  |
| `percent` | `%`    | 比率类：慢查率、错误率 |
| `size`    | `byte` | 大小类：资源大小       |

### 事件指标自定义字段 filter_name 格式

事件指标的自定义字段使用 JSON 格式的 `filter_name`：

| 维度类型 | filter_name 格式                                         | 示例字段                  |
| -------- | -------------------------------------------------------- | ------------------------- |
| 分类维度 | `{"dimension":"custom.categories","map_key":"<字段名>"}` | status_code, request_path |
| 数值维度 | `{"dimension":"custom.metrics","map_key":"<字段名>"}`    | duration, size            |

用于 `--filter` 参数或 `raw_measure_list[].filter_list` 中：

```json
{
  "op": "gte",
  "filter_name": "{\"dimension\":\"custom.metrics\",\"map_key\":\"duration\"}",
  "values": ["3000"]
}
```

### polynomial 手动构建（罕见）

> 仅当 `slardar-web-cli metric build --type formula` 无法满足时手动构建（如需要特殊的 `filter_list` 或 `unit` 组合）。优先使用 `metric build --type formula`。

polynomial 类型的 measure 结构：

- `type`: `polynomial`
- `customId`: UUID
- `raw_measure_list`: 每个成员带 `alias`（A/B/C...）
- `formula`: 表达式如 `A/B`、`(A-B)/B`

```json
[
  {
    "type": "polynomial",
    "customId": "<uuid>",
    "raw_measure_list": [
      { "measure_name": "<分子>", "filter_list": [], "alias": "A" },
      { "measure_name": "<分母>", "filter_list": [], "alias": "B" }
    ],
    "formula": "A/B",
    "name": "自定义比率",
    "unit": { "unit_type": "percent", "unit": "%" }
  }
]
```
