---
name: boe-sql-ticket
description: BOE SQL 变更工单原子,封装 data-rds-boe Mira market skill。功能:把 SQL 文件提交为 BOE 工单 → 输出rds工单链接提示用户审批 → 强制询问发起人 BOE 是否已执行。仅服务商活动费率配置子流程使用。
---

# boe-sql-ticket

## 何时调用

仅 **`service_merchant_activity_rate`** 子流程的第 5 步调用本原子(代替通用 `coding-facade`)。其它子流程严禁调用 — 通用编码 / DDL 请走 BITS / SCM 流水线节点。

## 输入

| 参数 | 必填 | 默认值 | 说明 |
|---|---|---|---|
| `sql_file_path` | 是 | — | SQL 文件绝对路径(由 `online-service-activity-sql` 产出) |
| `db` | 是 | `caijing_bytepay_charge` | **目标数据库,本原子硬锁定为该值,传其它值直接报错** |
| `ticket_title` | 是 | — | 工单标题,推荐格式 `[服务商活动费率配置] <PRD 标题>` |
| `ticket_desc` | 是 | — | 工单描述,推荐含 PRD 链接 + 需求摘要 |
| `approver_mode` | 否 | `self` | 当前仅支持 `self`(申请人自审批);其它值报错 |
| `env` | 否 | `boe` | 当前仅支持 `boe` |

## 输出

```json
{
  "ticket_id": "<工单 ID>",
  "ticket_url": "<工单页面 URL>",
  "db": "caijing_bytepay_charge",
  "env": "boe",
  "approver": "<申请人 username>",
  "approved_at": "<ISO8601>",
  "sql_excerpt": "<前 10 行>"
}
```

## 执行步骤

### 前置条件：鉴权
复用 bytedcli SSO → Codebase JWT 链路;若 data-rds-boe 接口需要独立 token,按 data-rds-boe openapi 文档执行 `bytedcli --site boe auth get-bytecloud-jwt-token`。

### Step 0:参数校验(硬锁定)

```python
assert db == "caijing_bytepay_charge", f"BOE 工单库锁定为 caijing_bytepay_charge,收到 {db}"
assert env == "boe", f"本原子只支持 boe,收到 {env}"
assert Path(sql_file_path).is_file() and Path(sql_file_path).stat().st_size > 0
```

任一项失败 → 抛错。

### Step 1:调 data-rds-boe market skill 提交工单

通过 Mira `Skill` 工具调用 `prod:data-rds-boe`(或当前 skill_key
`skills:skills.byted.org/douyin/space/data-rds-boe:0.1.0`),
透传:

- `action`: `create_sql_ticket`
- `db`: `caijing_bytepay_charge`
- `env`: `boe`
- `sql`: SQL 文件内容(读 `sql_file_path` 全文)
- `title`: `ticket_title`
- `description`: `ticket_desc`
- `applicant`: 当前会话用户(`MIRA_CURRENT_USERID` / `lipingyu.0608`)

工单创建成功 → 拿到 `ticket_id` / `ticket_url` 提示用户完成工单审批。失败 → 抛 data-rds-boe 真实错误并终止。

> 实际参数名以 `data-rds-boe` skill 的契约为准;本原子在第一次实跑时若发现字段名不一致,
> 按 skill 报错信息调整后再 commit 修正,**禁止猜参数名**。

## 强约束(违反即视为原子实现错误)
1**永不**把工单库改成 `caijing_bytepay_charge` 以外的值
2**永不**在本原子内尝试「直接执行 SQL」/「调用 BOE 数据库执行接口」— 执行必须由人工在 BOE 平台触发,
   本子流程的第 6 步会强制询问用户确认执行状态
3**永不**在生产环境(`env=online`)调用本原子,任何 prod 请求一律拒绝
4**不**在日志中打印 SQL 全文(可能含敏感字段),只打前 10 行 excerpt + 工单链接
5**必须**走修复后的标准 DBW OpenAPI 提交流程，确保请求中显式带上 `Action` / `Version` / `Service`
6**鉴权**优先 AK/SK（或 STS）直连 OpenAPI；仅存在 JWT 时自动回退 DBW JWT 网关

## 异常分支

| 情况 | 处理 |
|---|---|
| `data-rds-boe` skill 未启用 | 抛错并提示路由层用 `mira_skill__skill_write enable` 启用 |
| `data-rds-boe` 鉴权失败 | 抛 `AUTH_REQUIRED`,提示用户在主机完成 data-rds-boe 登录 |
| 工单创建 400(字段缺失) | 把 skill 返回错误抛给上游 |
| 自审批失败(权限不够) | 抛错,保留 ticket_id,提示用户在 BOE 平台手动审批 |
| 工单状态轮询超时(>2min) | warn + 返回最后状态,由子流程层决定是否继续 |
