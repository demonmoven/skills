---
name: argos-log
description: 查询 Argos 日志。当用户需要查询日志、通过 logid 查日志、关键字搜索日志时触发。触发词：argos、日志、logid、log、streamlog。
allowed-tools: Bash(byte-cli:*)
tags:
  - byte-skill
---

# Argos Log Query with byte-cli

查询 ByteDance Argos (Streamlog) 平台的日志数据。

## Installation (一次性设置)

1. **检查是否已安装**:
   ```bash
   byte-cli --version  # 如果输出版本号，说明已安装，跳过步骤 2
   ```

2. **安装 byte-cli（如果未安装）**:
   ```bash
   uv tool install --index https://bytedpypi.byted.org/simple --force \
     "git+https://code.byted.org/bytedance/byte-skill.git#subdirectory=byte-cli"
   ```

3. **验证安装成功**:
   ```bash
   byte-cli --version
   ```

## Prerequisites (运行依赖)

1. **Browser session**: JWT 自动注入，但需要有效的浏览器登录态。
   - CN/BOE: 在 Chrome 中登录 `cloud.bytedance.net`
   - I18N_BD: 在 Chrome 中登录 `cloud.byteintl.net`
   - I18N_TT: 在 Chrome 中登录 `cloud.tiktok-row.net`
   - EU_TTP: 在 Chrome 中登录 `cloud-eu.tiktok-row.net`
   - US_TTP: 在 Chrome 中登录 `cloud-ttp-us.bytedance.net`
   - byte-cli 会自动从浏览器 cookie 换取 JWT 并注入 `x-jwt-token` header
2. **Config path**: 当调用本 skill 时，系统会返回 "Base directory for this skill"。需要将此 base directory 与相对路径 `assets/config.json` 拼接成完整路径传给 `--config` 参数。

## Quick start

```bash
CONFIG="<base-dir>/assets/config.json"

# TraceQuery: 通过 logid 查询（CN 环境）
byte-cli --config $CONFIG Argos CN TraceQuery \
  --logid "your-logid-here" \
  --psm-list '["your.psm"]' \
  --output-filter '.response_body.data'

# TraceQuery: 不指定 PSM，跨服务搜索
byte-cli --config $CONFIG Argos CN TraceQuery \
  --logid "your-logid-here" \
  --output-filter '.response_body.data'

# KeywordQuery: 关键字搜索（CN 环境，最近 10 分钟）
START=$(($(date +%s) - 600)); END=$(date +%s)
byte-cli --config $CONFIG Argos CN KeywordQuery \
  --psm-list '["your.psm"]' \
  --start $START --end $END \
  --keyword-filter '{"include":{"case_sensitive":true,"operator":"AND","word_list":[{"word":"error","is_term":false}]},"exclude":{"words":[],"case_sensitive":true,"operator":"AND"}}' \
  --kv-filters '[{"key":"_idc","type":"STRING","operator":"in","value":["hl","lf","lq","tjdt","yg"]}]' \
  --output-filter '.response_body.data'
```

## 支持的查询类型

| Command | 说明 | 适用场景 |
|---------|------|----------|
| `TraceQuery` | 通过 logid 检索 trace | 已知 logid，快速定位问题 |
| `KeywordQuery` | 关键字检索日志 | 按关键字搜索日志内容 |

## Control planes

| Plane | 说明 | 日志服务域名 | 默认 vregion | 默认 IDC |
|-------|------|-------------|-------------|----------|
| `CN` | 国内环境（含 BOE） | logservice.byted.org | China-North | hl,lf,lq,tjdt,yg |
| `I18N_BD` | 国际化-ByteDance | logservice-mya.sinf.net | (全量国际 region) | hktobiaas,jkttobiaas,johortobiaas,sgtobiaas,jpsaas,bdsgdt,mya,myb,sgcomm1,sgsaas1larkidc1,sgsaas1larkidc2,va |
| `I18N_TT` | 国际化-TikTok | logservice-sg.tiktok-row.org | (全量国际 region) | hktobiaas,jkttobiaas,johortobiaas,sgtobiaas,jpsaas,bdsgdt,mya,myb,sgcomm1,sgsaas1larkidc1,sgsaas1larkidc2,va |
| `EU_TTP` | 欧洲 TTP | logservice-eu-ttp.tiktok-eu.org | EU-TTP | dedt,ie,ie2,iedt |
| `US_TTP` | 美国 TTP | logservice-tx.tiktok-us.org | US-TTP | tx |

### 如何选择 Control Plane

1. **国内服务**: 使用 `CN`（默认）
2. **BOE 测试环境**: 使用 `CN`，通过 `--vregion China-BOE` 和 `--kv-filters` 中 `_idc` 设为 `["boe"]`
3. **ByteDance 国际服务**: 使用 `I18N_BD`（需在 Chrome 登录 `cloud.byteintl.net`）
4. **TikTok 国际服务**: 使用 `I18N_TT`（需在 Chrome 登录 `cloud.tiktok-row.net`）
5. **欧洲 TTP 服务**: 使用 `EU_TTP`（需在 Chrome 登录 `cloud-eu.tiktok-row.net`）
6. **美国 TTP 服务**: 使用 `US_TTP`（需在 Chrome 登录 `cloud-ttp-us.bytedance.net`）
7. **从 PSM 判断**: 国际 PSM 根据其归属选择对应 CP

## 重要限制

| 限制项 | 要求 | 说明 |
|--------|------|------|
| **scan_span** | 默认 10 分钟 | TraceQuery 的时间扫描范围 |
| **时间范围** | ≤ 3 小时（10800 秒） | KeywordQuery 的 start 到 end 时间差不能超过 3 小时 |
| **QPS** | ≤ 2 | 每次查询间隔至少 1 秒，超出会返回 429 |
| **分页** | KeywordQuery 返回 `data.context` | 非空时可用于下一次查询的 `context` 字段实现翻页 |
| **超过 3 小时** | 拆分为多次查询 | 每次不超过 3 小时 |

## TraceQuery 参数

必填参数必须通过独立 CLI flag 传入（`--body-string` 无法满足 required 校验）：

| CLI flag | 类型 | 必填 | 默认值 | 说明 |
|----------|------|------|--------|------|
| `--logid` | TEXT | 是 | - | 要查询的 logid |
| `--psm-list` | TEXT | 否 | [] | PSM 列表 JSON，如 `'["your.psm"]'`，空数组跨服务搜索 |
| `--scan-span-in-min` | INTEGER | 否 | 10 | 扫描范围（分钟） |
| `--vregion` | TEXT | 否 | 从 CP 获取 | 虚拟区域 |

## KeywordQuery 参数

必填参数必须通过独立 CLI flag 传入：

| CLI flag | 类型 | 必填 | 默认值 | 说明 |
|----------|------|------|--------|------|
| `--psm-list` | TEXT | 是 | - | PSM 列表 JSON，如 `'["your.psm"]'` |
| `--start` | INTEGER | 是 | - | 开始时间（Unix 时间戳，秒） |
| `--end` | INTEGER | 是 | - | 结束时间（Unix 时间戳，秒），end - start ≤ 10800 |
| `--keyword-filter` | TEXT | 是 | - | 关键字过滤条件 JSON（见下方） |
| `--kv-filters` | TEXT | 是 | - | KV 过滤条件 JSON，至少包含 `_idc` 过滤 |
| `--limit` | INTEGER | 否 | 100 | 每页结果数量 |
| `--context` | TEXT | 否 | [] | 分页上下文 JSON |
| `--timeout-in-ms` | INTEGER | 否 | 2000 | 查询超时（毫秒） |
| `--vregion` | TEXT | 否 | 从 CP 获取 | 虚拟区域 |

### keyword_filter 结构

```json
{
  "include": {
    "case_sensitive": true,
    "operator": "AND",
    "word_list": [{"word": "error", "is_term": false}]
  },
  "exclude": {
    "words": [],
    "case_sensitive": true,
    "operator": "AND"
  }
}
```

### kv_filters 常用配置

```json
[
  {"key": "_idc", "type": "STRING", "operator": "in", "value": ["hl", "lf", "lq", "tjdt", "yg"]},
  {"key": "_cluster", "type": "STRING", "operator": "is", "value": ["your-cluster"]}
]
```

`_idc` 根据 control plane 选择默认值：
- CN: `["hl", "lf", "lq", "tjdt", "yg"]`
- BOE: `["boe"]`
- I18N_BD / I18N_TT: `["hktobiaas", "jkttobiaas", "johortobiaas", "sgtobiaas", "jpsaas", "bdsgdt", "mya", "myb", "sgcomm1", "sgsaas1larkidc1", "sgsaas1larkidc2", "va"]`
- EU_TTP: `["dedt", "ie", "ie2", "iedt"]`
- US_TTP: `["tx"]`

## 分页查询

KeywordQuery 支持分页，流程如下：

1. 首次查询：`context` 传空数组 `[]`
2. 检查返回的 `data.context`：如果非空，用它作为下一次查询的 `context` 值
3. 每次查询间隔至少 1 秒（`sleep 1`）
4. 当 `data.context` 为空或 `data.content` 为空时停止

## 错误处理

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 101401 | Unauthorized | JWT 注入失败，在 Chrome 重新登录对应域名 |
| 101403 | AccessForbidden | JWT 有效但无权限，检查账号权限 |
| 401 | JWT 过期 | 在 Chrome 重新登录对应域名 |
| 429 | 超过 QPS 限制 | 降低请求频率，间隔至少 1 秒 |
| 空数据 | content 为空 | 1. 检查 control plane 2. 检查时间范围 3. 检查 PSM 是否正确 |

## 示例场景

### 场景 1：通过 logid 查询日志（CN）

```bash
byte-cli --config $CONFIG Argos CN TraceQuery \
  --logid "abc123xyz" \
  --psm-list '["code_forge.pipeline.api"]' \
  --output-filter '.response_body.data'
```

### 场景 2：通过 logid 跨服务搜索（不指定 PSM）

```bash
byte-cli --config $CONFIG Argos CN TraceQuery \
  --logid "abc123xyz" \
  --output-filter '.response_body.data'
```

### 场景 3：关键字搜索最近 10 分钟的 error 日志

```bash
START=$(($(date +%s) - 600)); END=$(date +%s)
byte-cli --config $CONFIG Argos CN KeywordQuery \
  --psm-list '["code_forge.pipeline.api"]' \
  --start $START --end $END \
  --keyword-filter '{"include":{"case_sensitive":true,"operator":"AND","word_list":[{"word":"error","is_term":false}]},"exclude":{"words":[],"case_sensitive":true,"operator":"AND"}}' \
  --kv-filters '[{"key":"_idc","type":"STRING","operator":"in","value":["hl","lf","lq","tjdt","yg"]}]' \
  --output-filter '.response_body.data'
```

### 场景 4：查询超过 3 小时的日志（拆分查询）

6 小时需要拆分为 2 次查询（每次 3 小时）：

```bash
NOW=$(date +%s)
# 第一次：最近 3 小时
byte-cli --config $CONFIG Argos CN KeywordQuery \
  --psm-list '["your.psm"]' \
  --start $((NOW - 10800)) --end $NOW \
  --keyword-filter '{"include":{"case_sensitive":true,"operator":"AND","word_list":[{"word":"error","is_term":false}]},"exclude":{"words":[],"case_sensitive":true,"operator":"AND"}}' \
  --kv-filters '[{"key":"_idc","type":"STRING","operator":"in","value":["hl","lf","lq","tjdt","yg"]}]' \
  --output-filter '.response_body.data'

sleep 1

# 第二次：3-6 小时前
byte-cli --config $CONFIG Argos CN KeywordQuery \
  --psm-list '["your.psm"]' \
  --start $((NOW - 21600)) --end $((NOW - 10800)) \
  --keyword-filter '{"include":{"case_sensitive":true,"operator":"AND","word_list":[{"word":"error","is_term":false}]},"exclude":{"words":[],"case_sensitive":true,"operator":"AND"}}' \
  --kv-filters '[{"key":"_idc","type":"STRING","operator":"in","value":["hl","lf","lq","tjdt","yg"]}]' \
  --output-filter '.response_body.data'
```

### 场景 5：查询 I18N_BD 区域日志

```bash
byte-cli --config $CONFIG Argos I18N_BD TraceQuery \
  --logid "abc123xyz" \
  --psm-list '["your.intl.psm"]' \
  --output-filter '.response_body.data'
```

### 场景 6：BOE 环境查询

BOE 使用 CN control plane，通过 `--vregion` 和 `--kv-filters` 指定 BOE 环境：

```bash
byte-cli --config $CONFIG Argos CN TraceQuery \
  --logid "abc123xyz" \
  --vregion "China-BOE" \
  --output-filter '.response_body.data'
```

BOE KeywordQuery 示例：
```bash
START=$(($(date +%s) - 600)); END=$(date +%s)
byte-cli --config $CONFIG Argos CN KeywordQuery \
  --psm-list '["your.psm"]' \
  --start $START --end $END \
  --keyword-filter '{"include":{"case_sensitive":true,"operator":"AND","word_list":[{"word":"error","is_term":false}]},"exclude":{"words":[],"case_sensitive":true,"operator":"AND"}}' \
  --kv-filters '[{"key":"_idc","type":"STRING","operator":"in","value":["boe"]}]' \
  --vregion "China-BOE" \
  --output-filter '.response_body.data'
```
