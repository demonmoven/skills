---
name: streamlog-trace-query
description: 在有logId的情况下进行日志查询的技能。
---

# 计费结算联调数据检查

## 简介

该技能用于在本地通过脚本查询 Streamlog 的 trace 日志, 并分析其中的计费结算日志是否符合预期，符合预期的情况下再通过结算单号查询结算详情。

- 打开浏览器完成登录，自动抓取 `X-Jwt-Token` / `x-custom-identity` 并缓存到本地
- 后续无需再开浏览器，直接用脚本调用 trace 查询接口并缓存结果

## 使用流程

完整流程建议按下面 6 步走：

1) 安装依赖（一次性）
2) 登录并抓取 token（需要时执行）
3) 按 logid 查询并保存到本地（查询脚本会同时缓存原始响应）
4) 按需求从本地缓存结果中过滤出需要的日志, 判断是否有报错
5) 有报错的话，根据报错信息定位问题
6) 没有报错的话，结合日志调用ftools工具，判断单据是否符合预期。


## 参数说明
- `<SKILL路径>`：本技能 `streamlog-trace-query` 目录的完整路径（技能安装在不同位置时请以实际为准）。

## 1) 安装依赖

首次一次性安装（推荐 venv；`.venv` 放在技能目录下）：

```bash
python3 -m venv <SKILL路径>/.venv
source <SKILL路径>/.venv/bin/activate
pip install playwright requests
python -m playwright install chromium
```

## 2) 登录并抓取 token

打开浏览器完成登录，并把 token 写入本地 cache：

```bash
python3 <SKILL路径>/scripts/login_and_cache_token.py
```

说明：

- 正常使用不需要关心内部抓取逻辑；如果后续查询遇到 401/403，重新运行本脚本刷新 token。
- 需要调整抓取策略时再使用参数：`--jwt-source`、`--request-wait`。

## 3) 按 logid 查询并保存到本地

查询 trace（默认 vregion：`China-Pay,China-Pay2,China-North`）：

```bash
python3 <SKILL路径>/scripts/query_trace.py --logid <LOG_ID> --scan-span-min 10 --vregion China-Pay,China-Pay2,China-North --psm-list caijing.bytepay.bytepay_charge,caijing.bytepay.bytepay_settle,caijing.bytepay.settle_center
```

输出与落盘：

- 查询脚本会把“原始响应 JSON”缓存到本地结果目录（默认 `~/.cache/streamlog-results`）。
- 终端输出默认是“归一化后的日志”`jsonl`（每条日志一行，`kv_list` 已折叠为 `kv` 字典）。
- 不需要额外参数控制输出格式或落盘文件，脚本内已固定。

常用参数：

- `--logid` 必填
- `--scan-span-min` 扫描时间范围（分钟，默认 10）
- `--vregion` 逗号拼接的 vregion 字符串，默认 `China-Pay,China-Pay2,China-North`
- `--psm-list` 逗号分隔 psm，默认 `caijing.bytepay.bytepay_charge,caijing.bytepay.bytepay_settle,caijing.bytepay.settle_center`

如果希望在 cache 缺失/过期/鉴权失败时自动拉起登录刷新 token，增加：

## 4) 从本地缓存结果中过滤日志
过滤脚本会读取本地缓存的“原始响应 JSON”，按条件筛选后输出（默认 `jsonl`，也支持 `--format text` 一行展示）：

```bash
python3 <SKILL路径>/scripts/filter_trace.py --logid <LOG_ID> --psm-list a.b.c --level-list Error
```

常用过滤参数：

- `--psm-list` 逗号分隔 psm
- `--level-list` 逗号分隔 level（Info/Warn/Error/...）
- `--keyword` 在 `_msg` 中匹配
- `--spanid` 指定 span
- `--time-from` / `--time-to` 时间范围（epoch 或 ISO）
- `--summary` 只输出统计
- `--format` 输出格式：`jsonl`（默认）或 `text`

输出结构示例（`--format jsonl`，字段以实际返回为准）：

```json
{"id":"7266554789409189362","_level":"Error","__timestamp":"1769494116434000","_location":"dolphin.go:27","_psm":"caijing.bytepay.bankcard_product","__logid":"<LOG_ID>","_spanid":"7907736690110098251","__vregion":"China-Pay","_msg":"[Judge] judge err ..."}
```

一行展示格式示例（`--format text`，字段以实际返回为准）：

```text
Error 2026-01-27 14:08:36.434 dolphin.go:27 10.171.174.200 caijing.bytepay.bankcard_product <LOG_ID> default all_dc hlzg 7907736690110098251 _podname=... __vregion=China-Pay _msg=...
```

## 5) 定位报错问题

根据报错信息定位问题，一般是根据 `_msg` 字段。

## 6) 判断结算是否符合预期

- 如果没有报错，根据日志找到 settleOrderId, 一般在对应的key为`settleOrderId`或`settle_order_id` 忽略大小写;
- 使用settleOrderId 调用ftools工具查询结算详情:

```bash
python3 <SKILL路径>/scripts/ftools.py --settle-order-id <SETTLE_ORDER_ID>
```


## 常识
1. 计收费的psm有`caijing.bytepay.bytepay_charge`
2. 结算的psm有`caijing.bytepay.bytepay_settle`、`caijing.bytepay.settle_center`
3. 追光结算的psm是`caijing.bytepay.bytepay_settle`, 结算中心的psm是`caijing.bytepay.settle_center`

## 常见问题

- 如果 API 返回 401/403，重新运行 `<SKILL路径>/scripts/login_and_cache_token.py` 获取最新 token。
- 如果脚本提示抓取失败，确认已完成登录；必要时在页面里触发一次 trace 查询，或调整 `--jwt-source` 后重试。
- 浏览器 profile 目录会保留 cookie，可加速二次登录。
- 只要缓存 token 有效，查询脚本无需打开浏览器。
