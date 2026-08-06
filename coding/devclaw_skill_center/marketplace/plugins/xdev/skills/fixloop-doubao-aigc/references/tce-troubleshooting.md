# 部署/测试常见问题

## BFF 全部 4xx，但 pod 是 Running

**症状**：`tce list-instance` 看 pod `Running`，跑 pytest 所有 case 在第一跳（`flow.im.gateway` / `alice.creativity_api` 等 BFF）返回 HTTP 400。

**根因**：泳道没在环境平台注册。走 `bytedcli env create` + `env service deploy-tce` 会自动注册；走 `bytedcli tce deploy-lane` 会绕过。

**排查**：
1. 拿一个 failing logid 去 `https://op-flow.bytedance.net/trace/trace-detail` 查第一跳的 4xx 原因
2. 跑 `references/lane-route-probe.md` 的探针，确认这个 lane 是不是根本不被 BFF 识别
3. 如果是，按 `deploy-procedure.md` 用 `env create` 重新创建同名 lane（环境平台侧会把缺失的注册补上）

## 部署 ticket 超过 10 分钟还在 running

首轮 deploy-tce 约 5～7 分钟，超过 10 分钟异常。查：
1. `bytedcli --json env ticket get --ticket-id <id> --standard-env online_cn` 看 `deploy_summary` 里是否有某个 cluster 卡住
2. 工单 JSON 里找 `validate_error` / `err_code` 非空字段
3. 登环境平台 web UI 看工单详情页日志

## pod 长时间 NotReady / CrashLoopBackOff

1. `tce list-instance` 拿到 pod name
2. `bytedcli tce webshell open --psm <psm> --env <lane> --first` 建 session
3. `bytedcli tce webshell exec --session-id <sid> --command 'tail -200 /var/log/<psm>/<psm>.log'`（日志路径随服务变，不确定先 `ls /var/log`）
4. 常见原因：配置读取失败（环境变量 / TCC 配置不可用）、端口冲突、依赖 RPC 不通

## SCM 构建 `build_failed`

`bytedcli scm repo build-log <repo> <version>` 看输出。编译错误修代码重 push 后再 `scm repo build`。

## pre_check `validate_error=3`（PPE 额度不足）

`--specify-dcs` 传得太多。Doubao PPE 通常一个 IDC × 1 pod 就够测。把 `HL:1,LF:1` 减到 `LF:1`，或改 `env create --single-idc true --idc LF`。

## 工单字段名 / 状态值看起来不对

每层 API 用不同命名：
- env 工单：`data.status`（小写 `success/running/failed`）
- scm 构建：`data.items[].status`（`build_ok/building/build_failed`，小写带下划线）
- tce pod：`data.pods[].status`（大写 `Running/Pending/NotReady/Failed`）

**轮询永远是原样打完整 JSON 让 agent 语义判断**，不要 shell case / grep 匹配固定字符串，字段名或大小写换一下就会静默失败（踩过）。
