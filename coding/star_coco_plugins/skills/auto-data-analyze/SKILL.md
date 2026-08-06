---
name: auto-data-analyze
description: 基于 TQS 的数据分析专家。用于自然语言数据需求→自动探索表结构→生成并预校验 HSQL→异步执行与轮询取数→导出/解析 CSV→本地数据分析与可视化。适用于数据查询、指标统计、业务数据探索、报表生成、TopN/趋势/分组聚合分析等场景。
---

# 工作流（从需求到报告）

按以下顺序执行；如某步信息不足，先向用户补齐关键参数（时间范围、维度、指标、过滤条件、口径）。

## 1. 知识准备（业务文档→口径/模型）
- 读取用户提供的飞书业务文档（或本仓库内的业务说明），抽取：
  - 业务术语 ↔ 表/字段 的映射，初步理解每张表的用途
- 如果缺少业务文档：当且仅当客户提到星图相关的问题时，才去读取该飞书文档，https://bytedance.larkoffice.com/wiki/wikcnsIAuWCdekaVxUBjo8th3Ed，理解星图相关知识。否则提示用户提供文档链接或关键段落。

## 2. 意图识别（自然语言→结构化需求）
将用户输入解析为结构化查询意图：
- 查询目标（要回答的问题）
- 时间范围（最近 N 天/自然周/月/自定义区间）
- 维度（分组字段，如作者/行业/地区/渠道）
- 指标（聚合指标，如 `sum(gmv)`、`count(distinct xxx)`）
- 过滤条件（业务条件、状态、类型等）
- TopN/排序/是否需要明细或汇总

## 3. 表结构探索（DESC→字段/分区）
；对候选表执行 `DESC <table>` （和执行SQL是相同的，都需要执行SQL，获取job_id, 调用 `TqsGetResult` 取数）获取字段/类型/注释/分区信息，用于：
- 校验字段是否存在
- 确定 join key 与时间/分区过滤写法
- 指导 SQL 生成
- 可以使用 select max_pt('table_name'); 来验证分区的格式, 大部分表的分区字段都是 date/p_date, 如果有多个分区, 你需要自行挖掘。

## 4. SQL 生成（HSQL）
- 基于意图 + 表结构生成 HSQL。
- 优先保证：
  - 分区过滤正确（避免全表扫描）
  - 维度/指标口径一致
  - join 条件明确、字段来源清晰

## 5. SQL 预校验（必须）
使用 MCP 工具 `TqsAnalyzeSql` 做执行前校验：
- 语法错误、表/字段不存在
- 分区字段使用不当
- 将错误信息转为可操作的修复建议，并回到“SQL 生成”步骤迭代

## 6. SQL 执行（异步）
使用 MCP 工具 `TqsExecuteSql` 提交查询并获取 `job_id`。

推荐默认参数：
- `cluster`: `cn`
- `yarn_queue`: `root.tuchong.stats`

## 7. 任务轮询（状态→结果）
使用 `TqsGetJobStatus` 轮询：
- `pending/running`：持续等待并向用户反馈进度
- `failed`：透出错误并给出重试/修复建议
- `success`：进入取数

## 8. 结果获取与导出
使用 `TqsGetResult` 获取：
- 预览数据（通常仅少量行）用于快速验证口径
- 全量结果的 CSV 下载信息（如工具返回包含下载链接/文件信息）

注意：
- `max_rows` 最大 1,000,000；超过时建议拆分日期分批拉取或先聚合再取数。

最终以 Markdown 形式输出一份可复制的分析报告：
- 结论（先给答案）
- 关键指标与口径说明
- 结果表（TopN/明细/汇总）
- 可视化（如趋势图/柱状图）
- SQL（便于复用）

## 9. 本地分析与可视化【如有必要】
在本地对 CSV/结果集进行分析（优先用 pandas）：
- 常见：分组聚合、TopN、环比/同比、趋势、异常点
- 输出：统计摘要 + 表格 +（可选）图表（matplotlib/plotly）

分析信息追加到分析报告的飞书文档中。

# MCP 工具速查

| 工具 | 唯一标识 | 用途 |
| --- | --- | --- |
| `TqsAnalyzeSql` | `star_auto_data_tqs_analyze_sql` | SQL 语法与结构校验 |
| `TqsExecuteSql` | `star_auto_data_tqs_execute_sql` | 异步执行 HSQL |
| `TqsGetJobStatus` | `star_auto_data_tqs_get_job_status` | 查询任务状态 |
| `TqsGetResult` | `star_auto_data_tqs_get_result` | 获取查询结果与 CSV 信息 |

# 安全与体验约束

- 执行前必须做 `TqsAnalyzeSql` 预校验。
- 执行前必须展示 SQL 并让用户确认（或用户已明确同意自动执行）。
- 控制结果规模：优先聚合后取数；必要时限制 `max_rows`，避免资源耗尽。
- 错误信息要可执行：指出错误点（表/字段/语法/分区）并给出下一步修改建议。
