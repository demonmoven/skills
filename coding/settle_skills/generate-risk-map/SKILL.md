---
name: generate-risk-map
description: |
  从 FOP 平台自动采集追光清结算的 DDA（实时核对）和 HSQL（离线核对）任务数据，结合结算计费域风险分类标准，
  自动完成风险点分类、依赖表提取，并将全量数据灌入飞书多维表格，生成资损防控任务台账。
  适用于定期更新追光清结算核对任务清单、新增任务入台账等场景。
---

# 追光结算计费资损防控地图生成

## 概述

本 Skill 实现从 FOP（Financial Operations Platform）平台自动采集追光清结算体系下的核对任务数据，
结合风险分类规则进行智能标注，最终输出结构化的飞书多维表格台账。

## 适用场景

- 定期刷新追光清结算核对任务台账
- 新增 DDA/HSQL 任务后需要全量更新多维表格
- 需要按照风险分类标准对任务进行自动归类
- 需要将禁用/失效任务排列在表格末尾

## 输入要求

1. **FOP 平台访问权限**：需要能够访问以下页面（需登录态）
   - DDA 任务列表: `https://fop.{domain}/boss/check-core/dda/config-new?OrganizationId={org_id}`
   - HSQL 任务列表: `https://fop.{domain}/check-core/check/hsql/list?SystemIds={org_id}`

2. **飞书应用凭证**（二选一）：
   - 用户 JWT Token（通过环境变量 `AIME_USER_CLOUD_JWT` 获取，Aime 平台内置）
   - 飞书应用 App ID / App Secret（独立部署时使用，通过环境变量 `LARK_APP_ID` / `LARK_APP_SECRET` 配置）

3. **配置参数**：
   - `ORG_ID`: FOP 平台组织 ID（默认: `ORG240313190058294165643265`，即追光清结算）
   - `BITABLE_APP_TOKEN`: 目标飞书多维表格 app_token（如需写入已有表格）
   - `BITABLE_TABLE_ID`: 目标数据表 table_id

## 执行流程

### Stage 1: 数据采集

从 FOP 平台的两个页面自动提取全量任务数据。

```bash
python3 scripts/extract_fop_data.py \
  --org_id ORG240313190058294165643265 \
  --output_dir ./data
```

**输出**：
- `data/dda_data.json` — DDA 实时核对任务（含任务编码、名称、状态、创建/修改时间、修改人等）
- `data/hsql_data.json` — HSQL 离线核对任务（含任务编码、名称、优先级、状态、创建人等）

**注意**：此脚本依赖浏览器自动化能力。在 Aime 平台中使用内置 browser 工具；独立运行时需配合 Selenium/Playwright 等浏览器驱动。

### Stage 2: 数据处理与写入

将采集的数据进行风险分类、依赖表提取，并写入飞书多维表格。

```bash
python3 scripts/import_to_bitable.py \
  --dda_file ./data/dda_data.json \
  --hsql_file ./data/hsql_data.json \
  --app_token <飞书多维表格app_token> \
  --table_id <数据表table_id>
```

**处理逻辑**：
1. 读取 DDA 和 HSQL 的 JSON 数据文件
2. 根据风险分类关键词规则（见 `references/risk_classification.md`）自动标注"风险点类别"
3. 根据任务名称中的关键词提取"相关依赖表名称"
4. 为任务名称生成带超链接的 URL 格式
5. 分批（每批 450 条）写入飞书多维表格

### Stage 3: 排序优化（可选）

将"禁用"和"失效"状态的任务排列到表格末尾。

```bash
python3 scripts/reorder_bitable.py \
  --app_token <app_token> \
  --table_id <table_id> \
  --execute
```

**注意**：此操作会删除所有记录后重新插入，请确保执行前无并发写入。先不加 `--execute` 进行试运行。

## 目标表格结构

飞书多维表格字段顺序和类型如下：

| 字段名 | 类型 | 说明 |
|--------|------|------|
| 任务编码 | 文本 | 方程式ID / HSQL任务编码 |
| 任务名称 | 超链接 | 带跳转链接的任务名称 |
| 风险点类别 | 单选 | 配置类风险/时效性风险/一致性风险/业务正确性风险 |
| 归属子域 | 多选 | 人工勾选 |
| 商户类型 | 多选 | 人工勾选 |
| 业务分类 | 多选 | 人工勾选 |
| 业务子类 | 多选 | 人工勾选 |
| 运行状态 | 单选 | 生效/失效/启动/禁用 |
| 核对方式 | 单选 | 实时核对/离线核对 |
| 相关依赖表名称 | 文本 | 自动从任务名称提取 |
| 任务创建时间 | 文本 | YYYY-MM-DD HH:MM:SS |
| 任务修改时间 | 文本 | YYYY-MM-DD HH:MM:SS |
| 最后修改人 | 文本 | 人员姓名 |

## 风险分类规则

任务名称根据关键词自动匹配风险类别，优先级为：
**配置类风险 > 时效性风险 > 一致性风险 > 业务正确性风险**

详细关键词列表和分类标准参见 `references/risk_classification.md`。

## 依赖表提取规则

任务名称中包含特定关键词时，自动映射到对应的数据表名称。
例如：`settle_order` → `bytepay_settle_order`, `charge_order` → `bytepay_charge_order`

完整映射表定义在 `scripts/import_to_bitable.py` 的 `TABLE_NAME_MAPPING` 常量中。

## 环境变量

| 变量名 | 说明 | 必需 |
|--------|------|------|
| `AIME_USER_CLOUD_JWT` | Aime 平台用户 JWT（平台内自动注入） | 平台内必需 |
| `LARK_APP_ID` | 飞书应用 App ID（独立部署时） | 独立部署时必需 |
| `LARK_APP_SECRET` | 飞书应用 App Secret（独立部署时） | 独立部署时必需 |
| `LARK_USER_ACCESS_TOKEN` | 飞书用户访问令牌（独立部署时） | 独立部署时必需 |

## 注意事项

1. FOP 页面数据通过浏览器 DOM 解析获取，如果 FOP 前端改版需要更新 JS 提取表达式
2. 风险分类基于关键词匹配，存在无法分类的任务（风险点类别为空），需人工补充
3. "归属子域"、"商户类型"、"业务分类"、"业务子类" 四列为人工勾选字段
4. 重排序操作（Stage 3）会导致记录 record_id 变化，如有外部引用请注意
5. 调用飞书 API 前必须设置 `include_secrets=true` 以确保有权限访问
