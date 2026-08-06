# Metric Search — 指标搜索参考

搜索 Slardar 监控指标，支持基础指标、业务指标、事件指标三种类型。

## Contents

- metric search — 搜索指标（参数、用法）
- 指标类型判断（基础 vs 业务 vs 事件）
- 基础指标速查（通用聚合后缀、分类索引、核心性能指标、关键词映射）
- 业务指标格式（数据结构、关键字段、常见类型）
- 事件指标格式（两步获取流程、measure_name JSON 格式）
- metric fetch — 获取完整指标列表

---

## metric search — 搜索指标

使用 `slardar-web-cli metric search` 搜索指标。CLI 内置缓存（24h 自动过期）和别名映射（如"首屏"→ fcp/lcp/first_screen），能处理自然语言查询。

```bash
# 搜索基础指标
slardar-web-cli metric search --bid <BID> --type basic --keywords "<关键词>"

# 搜索业务指标（自动展示完整口径：组成、过滤条件、公式）
slardar-web-cli metric search --bid <BID> --type custom --keywords "<关键词>"

# 通过 UUID 查询单个自定义指标的完整口径
slardar-web-cli metric search --bid <BID> --type custom_get --formal-key <uuid>

# 搜索事件
slardar-web-cli metric search --bid <BID> --type event --keywords "<关键词>"

# 搜索事件下的指标
slardar-web-cli metric search --bid <BID> --type event_measure --event <event_name> --keywords "<关键词>"
```

### 参数

| 参数           | 必须               | 默认值 | 说明                                                          |
| -------------- | ------------------ | ------ | ------------------------------------------------------------- |
| `--bid`        | 是                 | -      | 业务 ID                                                       |
| `--type`       | 是                 | basic  | 搜索类型：basic / custom / custom_get / event / event_measure |
| `--keywords`   | 是\*               | -      | 搜索关键词（空格分隔）。custom_get 时可不传                   |
| `--formal-key` | custom_get 必填    | -      | 自定义指标 formal_key (UUID)                                  |
| `--event`      | event_measure 必填 | -      | 事件名                                                        |
| `--limit`      | 否                 | 20     | 返回条数上限                                                  |
| `--force`      | 否                 | false  | 强制刷新缓存                                                  |
| `--raw`        | 否                 | false  | 输出原始 JSON                                                 |

---

## 指标类型判断

| 类型         | 判断依据                     | 示例                               |
| ------------ | ---------------------------- | ---------------------------------- |
| **基础指标** | 系统内置的性能/错误/资源监控 | FCP、LCP、JS错误、API成功率、PV/UV |
| **业务指标** | 自定义复合指标、比率计算     | 慢请求比率、5s慢查、10s慢查        |
| **事件指标** | 自定义事件、埋点、行为追踪   | 自定义事件、埋点、点击事件         |

**判断优先级**：

1. 用户明确提到类型 → 直接使用
2. 包含基础指标关键词（FCP/LCP/API/JS错误等） → 基础指标
3. 包含"慢请求/慢查/比率/人均/Xs"等复合词汇 → 业务指标
4. 包含"事件/埋点/上报"等词汇 → 事件指标
5. 用户给出一个 UUID 格式的 formal_key → 使用 custom_get 直接查询
6. 无法确定 → 优先查询基础指标

---

## 基础指标速查

> 数据来源：`/api_web/web/flex/meta` 接口（bid=slardar_web）
> measure_name 均为 API 实际返回值，可直接用于查询

### 通用聚合后缀

大多数时间类指标共享以下后缀：

| 后缀            | 说明   | 推荐场景            |
| --------------- | ------ | ------------------- |
| `.avg`          | 均值   | 整体趋势            |
| `.pct50`        | 50分位 | 典型体验            |
| `.pct75`        | 75分位 | **Web Vitals 推荐** |
| `.pct90`        | 90分位 | 性能优化目标        |
| `.pct99`        | 99分位 | 长尾排查            |
| `.count`        | 上报量 | 数据量评估          |
| `.user`         | 用户数 | 影响面              |
| `.bounced_rate` | 跳出率 | 留存分析            |
| `.distribution` | 分布   | 形态分析            |

### 分类索引表

| 分类         | key            | measure_name 前缀             | 主要指标                                                 |
| ------------ | -------------- | ----------------------------- | -------------------------------------------------------- |
| 用户分析     | pageview       | `pv_uv.*`, `apdex.*`          | PV、UV、Session 数、Apdex 评分、页面停留时长             |
| 行为指标     | action         | `action.*`                    | Action 触发数、请求耗时、前端耗时、行为耗时              |
| 性能指标     | performance    | `browser_perf.*`              | FCP、LCP、CLS、FID、INP、TTFB、TTI、FMP、DOM Ready、Load |
| Js错误指标   | js_error       | `js_error.*`                  | 错误数、影响用户数、错误率                               |
| 静态资源指标 | resource       | `resource.*`                  | 总耗时、DNS/TCP/SSL/TTFB/传输 各阶段耗时                 |
| 静态资源错误 | resource_error | `resource_error.*`            | 错误数、影响用户数、错误率                               |
| 白屏指标     | blank_screen   | `blank_screen.*`              | 白屏错误数、影响用户数                                   |
| 请求指标     | http           | `http.*`                      | 成功率、成功数、失败数、请求耗时、各阶段耗时             |
| 请求错误指标 | http_error     | `http.*`                      | 错误数、错误率、影响用户数                               |
| 经营数据     | operation      | `operation.*`, `user_error.*` | 上报量、接收流量                                         |
| 图片资源     | image          | `image.*`                     | 加载耗时、CDN 命中率、尺寸优化节省量                     |
| 自定义日志   | log            | `log.count`                   | 日志数据量                                               |

### 核心性能指标 measure_name 速查

| 指标      | 全称                      | measure_name 前缀        | 单位 |
| --------- | ------------------------- | ------------------------ | ---- |
| LCP       | Largest Contentful Paint  | `browser_perf.lcp`       | ms   |
| FCP       | First Contentful Paint    | `browser_perf.fcp`       | ms   |
| CLS       | Cumulative Layout Shift   | `browser_perf.cls`       | -    |
| FID       | First Input Delay         | `browser_perf.fid`       | ms   |
| INP       | Interaction to Next Paint | `browser_perf.inp`       | ms   |
| TTFB      | Time to First Byte        | `browser_perf.ttfb`      | ms   |
| TTI       | Time to Interactive       | `browser_perf.tti`       | ms   |
| FMP       | First Meaningful Paint    | `browser_perf.fmp`       | ms   |
| DOM Ready | DOM 解析完成              | `browser_perf.dom_ready` | ms   |
| Load      | 页面完全加载              | `browser_perf.load`      | ms   |

示例：LCP 75分位 → `browser_perf.lcp.pct75`

### 关键词映射表

| 用户可能说的       | 应搜索的关键词                       |
| ------------------ | ------------------------------------ |
| 首屏、首屏时间     | fcp, lcp, first_screen               |
| 加载速度、加载时间 | load, fcp, lcp, ttfb                 |
| 白屏               | blank_screen                         |
| 卡顿、交互延迟     | inp, fid, tti                        |
| 布局抖动、布局偏移 | cls                                  |
| 接口、API、请求    | http                                 |
| 错误、异常、报错   | js_error, resource_error, http.error |
| 成功率             | success_rate                         |
| 图片、CDN          | image                                |
| 资源、静态资源     | resource                             |
| PV、UV、访问量     | pv_uv                                |

---

## 业务指标格式

> 数据来源：`/api_web/metric_management/list` 接口
> 业务指标由用户在 Slardar 平台创建，基于基础指标组合而成

### 数据结构

```json
{
  "id": 38495,
  "metric_name": "5s 慢查用户率",
  "formal_key": "b6492d68-6c50-41f7-a417-607ec1a5bbe1",
  "metrics": {
    "formula": "A/B",
    "members": {
      "A": { "metric_name": "http.success_count", "filter_conditions": [...] },
      "B": { "metric_name": "http.success_count" }
    }
  }
}
```

### 关键字段

| 字段              | 位置     | 说明                            |
| ----------------- | -------- | ------------------------------- |
| `metric_name`     | 顶层     | 指标名称（用于搜索匹配）        |
| **`formal_key`**  | **顶层** | 查询时作为 `--measure-key` 使用 |
| `metrics.formula` | 嵌套     | 计算公式（如 A/B）              |
| `metrics.members` | 嵌套     | 公式中各变量的定义              |

**⚠️ `formal_key` 是顶层字段，不在 `metrics` 子对象中。查询时使用 `formal_key` 的值作为 `--measure-key` 传入 `metric build`。**

### 常见业务指标类型

| 类型   | 公式模式          | 典型场景                   |
| ------ | ----------------- | -------------------------- |
| 比率型 | `A / B`           | 慢请求比率、错误率、超时率 |
| 人均型 | `A / UV`          | 人均错误数、人均请求数     |
| 筛选型 | 单变量 + 过滤条件 | 特定条件下的请求数         |
| 组合型 | 多变量组合        | 复杂业务逻辑               |

### 搜索建议

业务指标名称通常包含以下关键词模式：

- 数值+单位：`5s`、`10s`、`1000ms`
- 行为描述：`慢查`、`慢请求`、`超时`
- 统计类型：`比率`、`率`、`人均`、`用户数`
- 前后端：`前端`、`后端`、`全链路`

---

## 事件指标格式

> 数据来源：
>
> - 事件列表：`/api_web/web/flex/meta_event_list`
> - 事件指标：`/api_web/web/flex/meta_event_measure`

### 两步获取流程

1. 调用 `meta_event_list` 获取事件列表 → 字段 `data.event_meta_list`
2. 根据事件名调用 `meta_event_measure` 获取具体指标 → 字段 `data.measure_meta_list`

### 事件列表数据结构

```json
{
  "data": {
    "event_meta_list": [
      { "event_name": "page_load_cost", "label": "页面加载耗时" }
    ]
  }
}
```

**⚠️ 字段名是 `event_meta_list`，不是 `event_list`**

### 事件指标数据结构

```json
{
  "data": {
    "measure_meta_list": [
      {
        "label": "上报量",
        "measure_name": "{\"metric\":\"custom.count\",\"event_dimension\":\"event_name\",\"event_name\":\"page_load_cost\"}",
        "unit": { "unit_type": "number", "unit": "1" }
      }
    ]
  }
}
```

**⚠️ 字段名是 `measure_meta_list`，不是 `measure_list`**

### measure_name JSON 格式

事件指标的 `measure_name` 是 **JSON 字符串**：

```json
{
  "metric": "custom.metrics.avg",
  "event_dimension": "event_name",
  "event_name": "page_load_cost",
  "map_key": "duration"
}
```

### 指标类型

#### 通用指标（所有事件都有）

| metric 值      | 说明       |
| -------------- | ---------- |
| `custom.count` | 事件上报量 |
| `custom.user`  | 触发用户数 |

#### metrics 类型（数值型字段的聚合）

| metric 值                        | 说明        |
| -------------------------------- | ----------- |
| `custom.metrics.avg`             | 均值        |
| `custom.metrics.pct50` ~ `pct99` | 分位数      |
| `custom.metrics.min` / `max`     | 最小/最大值 |
| `custom.metrics.sum`             | 总和        |
| `custom.metrics.distribution`    | 分布        |

#### categories 类型（分类字段）

| metric 值                 | 说明       |
| ------------------------- | ---------- |
| `custom.categories_count` | 分类上报量 |
| `custom.categories_user`  | 分类用户数 |

---

## metric fetch — 拉取缓存

仅拉取已缓存的指标数据（不执行搜索匹配），用于确认缓存状态或获取完整指标列表。

```bash
slardar-web-cli metric fetch --bid <BID> --type basic
slardar-web-cli metric fetch --bid <BID> --type custom
slardar-web-cli metric fetch --bid <BID> --type event
slardar-web-cli metric fetch --bid <BID> --type event_measure --event <event_name>
slardar-web-cli metric fetch --bid <BID> --type custom_get --formal-key <uuid>
```

参数与 `metric search` 相同（`--bid`、`--type`、`--event`、`--formal-key`、`--force`）。
