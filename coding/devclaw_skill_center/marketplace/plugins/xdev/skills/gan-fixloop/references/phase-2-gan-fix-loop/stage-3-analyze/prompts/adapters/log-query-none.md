# Stage 3 / Log Query / Adapter: none

> PROFILE=none 的日志查询 adapter。**直接跳过日志查询**，Judge 仅基于 stdout / failed_cases.jsonl 内容做分析。
>
> 由 `failure-analysis.md` 的「步骤 2.5: 通过 LogID 查询服务日志」段调用。

## 适用场景

PROFILE=none 时使用。用户的服务部署在 BASE_URL，可能没有 PSM 概念，可能没有 bytedance-log 系统，无法用 LogID 拉服务端日志。

## 流程

什么都不做，直接返回。

## Judge 的降级

Judge 在分析每个失败用例时：
- 不再尝试查询 `/tmp/logid_filtered_<LOGID>.log`
- 仅基于 failed_cases.jsonl 中的 `message` 字段（go test stdout 内容）做分析
- 仍然检查 5 种无效测试模式 + 系统性遗漏清单
- 仍然检查测试代码 vs 业务代码

## 与 bytedance adapter 的差异

| 维度 | bytedance | none |
|------|--------|------|
| 拉取服务端日志 | 是（bytedcli log + LogID + PSM） | 否 |
| 分析数据源 | go test stdout + 服务端日志 | 仅 go test stdout |
| 分析准确率 | 高（含完整调用链上下文） | 中（仅有测试侧的视角） |

## 跳过条件

无（PROFILE=none 时本 adapter 必须 Read 但实际是 noop）。

## 用户提示

如果用户希望在 PROFILE=none 时也能查日志，可以：
1. 手动把服务日志输出到本地文件
2. 在 phase-0 wizard 中提供日志文件路径作为额外参数
3. 或者改用其它 PROFILE（如未来的 docker / k8s）

但这些都是未来增强,本次 MVP 不实现。
