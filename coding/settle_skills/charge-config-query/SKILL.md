---
name: "charge-config-query"
description: "基于 RDS 直接查询 caijing_bytepay_charge_union 库中的计费配置信息，并以 Markdown 表格形式返回。适用于需要根据主体 ID（contract_inst）快速排查计费规则、核对线上配置的场景。"
---

# 计费配置查询 (charge-config-query)

本 Skill 用于根据指定的主体实例 ID (`contract_inst`)，直接从在线数据库（RDS）查询相关的计费配置，并将结果格式化为 Markdown 表格，以便快速、清晰地展示。

## 连接信息说明

本 Skill 设计用于连接特定的生产数据库，连接信息如下：

- **库名 (db_name)**: `caijing_bytepay_charge_union`
- **Region ID**: `bytedance`
- **VRegion**: `China-Pay`

**所有查询操作都必须通过平台提供的 RDS 能力，针对以上指定库和区域执行。**

## MCP 工具与环境依赖

- **MCP 工具依赖**:
  - 本 Skill 强依赖平台提供的 RDS 查询能力，即 `rds` 工具集下的 `mcp:rds_rds_run_sql`。
  - **严禁使用** 任何与 `aeolus-platform-analysis` 或 HSQL 相关的工具。

- **认证与授权**:
  - 调用 RDS MCP 工具需要身份认证。Agent 在执行时 **必须** 在环境中包含有效的 JWT 凭证。
  - 这通常通过在工具调用时设置 `include_secrets=true` 来实现，以确保平台将 `AIME_USER_CLOUD_JWT` 环境变量注入到运行环境中。

- **脚本与库依赖**:
  - 核心逻辑封装在 `scripts/query_charge_config.py` 脚本中。
  - **主路径 (MCP)**: 依赖 `byted_aime_sdk` 以调用平台工具。
  - **兜底路径 (直连)**: 作为备选方案，在 MCP 不可用时，脚本会尝试使用 `pymysql` 库进行数据库直连。此路径需要额外配置数据库连接环境变量（如 `CHARGE_CONFIG_DB_HOST` 等），详情见 `references/usage.md`。

## 使用说明

### 输入

- `contract_inst` (string): 唯一的合同主体实例 ID，用于精确查询。

### 执行流程

本 Skill 的执行逻辑被封装在 `scripts/query_charge_config.py` 中，其核心流程如下：

1.  **优先使用 RDS MCP 查询 (主路径)**:
    a.  脚本首先检查 `byted_aime_sdk` 是否可用以及 `AIME_USER_CLOUD_JWT` 是否存在。
    b.  基于用户提供的 `contract_inst`，使用以下 SQL 模板构造查询语句。
    c.  调用 `mcp:rds_rds_run_sql` 工具，在 `db_name='caijing_bytepay_charge_union'` 和 `region='bytedance'` 上执行 SQL。
    d.  读取 MCP 工具返回的 CSV 结果文件。

2.  **失败时回退至直连查询 (兜底路径)**:
    a.  如果 MCP 调用失败（如缺少 SDK、JWT 或其他 MCP 异常），脚本会记录错误并自动切换到兜底方案。
    b.  使用 `pymysql` 库，通过环境变量（`CHARGE_CONFIG_DB_*`）连接数据库。
    c.  执行参数化查询（`WHERE contract_inst = %s`），以防止 SQL 注入。

3.  **处理并返回结果**:
    - 无论通过哪种路径获得结果，后续处理流程一致。
    - **空结果处理**: 如果查询结果为空，向用户返回“无匹配记录”。
    - **None 值处理**: 表格中的 `None` 或空值应显示为空字符串。
    - 结果被渲染为 Markdown 表格后输出。

### SQL 查询模板

```sql
select
  a.contract_inst,
  a.scene,
  b.match_rule,
  c.charge_rule_content,
  d.collect_fee_type,
  d.collect_fee_mode
from
  (
    select
      *
    from
      bytepay_charge_contract
    where
      contract_inst = '{{contract_inst}}'
  ) a
  LEFT JOIN (
    select
      *
    from
      bytepay_charge_factor
  ) b on a.charge_contract_code = b.charge_contract_code
  LEFT JOIN (
    select
      *
    from
      charge_rule_info
  ) c on b.charge_rule_info_id = c.charge_rule_info_id
  LEFT JOIN (
    SELECT
      *
    from
      collect_fee_rule_info
  ) d on b.collect_fee_rule_info_id = d.collect_fee_rule_info_id;
```
**注意**: 在脚本中，`{{contract_inst}}` 会被安全地替换为用户输入。

### 输出格式

查询结果应以 Markdown 表格形式呈现，表头和列顺序必须如下所示：

| contract_inst | scene | match_rule | charge_rule_content | collect_fee_type | collect_fee_mode |
|---|---|---|---|---|---|
| ... | ... | ... | ... | ... | ... |

## 性能与约束

- **性能要求**: 查询为在线数据库等值查询，预期响应时间为秒级。
- **查询约束**: **必须**始终在 `WHERE` 子句中使用 `contract_inst` 进行精确过滤，严禁执行任何无此过滤条件的全表扫描查询。
- **高层封装**: 为了简化使用，核心的查询和格式化逻辑已封装在 `scripts/query_charge_config.py` 中。在标准流程下，可优先考虑直接调用此脚本。详细用法请参考 `references/usage.md`。
