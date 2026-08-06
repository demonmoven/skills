---
name: tech-solution-impact
description: 输入一份技术方案（飞书文档链接或本地 markdown），自动解析出其中所有 DB 字段/表、TCC 配置项等变更点，逐个查询支付/结算域代码依赖图谱的影响面，并聚合成一份全量影响面报告（变更点→受影响入口映射、跨仓库风险汇总、P0/P1/P2 回归优先级）。适用于技术方案评审、需求影响面评估、上线回归范围圈定、"改这些字段/配置会波及哪些接口和仓库""这个方案的影响面有多大""帮我分析这个技术方案的影响面 [飞书链接]"等场景。
---

# 技术方案影响面全量分析

给定一份技术方案，回答"这个方案改的这些东西，一共会波及哪些功能入口、哪些仓库，回归该测哪些"。

底层复用支付/结算域代码依赖图谱（`code_graph_system/output/` 下的 `impact_index.json` + `graph.json`）。
本 skill 做两件图谱查询本身不做的事：**从方案里解析出变更点**，以及**把多个变更点的影响面聚合去重成一份报告**。

## 触发方式

用户给出技术方案（飞书文档链接，或已下载的本地 `.lark.md` / `.md`）并要分析影响面时触发。
典型说法："帮我分析这个技术方案的影响面 [飞书链接]"、"这个需求改动会波及哪些接口"、"评估下这个方案的回归范围"。

## 整体流程

四步串联，每步都有确定性脚本，Agent 只在第 3 步做语义复核：

1. **拿到方案文本** → 用 `lark-doc` skill 下载飞书文档为本地 `.lark.md`（若用户已给本地文件则跳过）。
2. **解析变更点** → 运行 `extract_change_points.py`，得到结构化变更点清单。
3. **语义复核**（Agent）→ 对照方案确认/补全变更点，剔除误命中。
4. **聚合影响面并出报告** → 运行 `analyze_impact.py` 聚合，再用 `lark-doc` skill 生成飞书报告文档。

### 第 1 步：获取方案文本

- 用户给飞书链接：使用 `lark-doc` skill 下载文档（`lark_download`），得到本地 `.lark.md` 文件路径。
- 用户给本地文件：直接用该路径。

### 第 2 步：解析变更点

```bash
python3 scripts/extract_change_points.py <方案.lark.md> --out change_points.json --md
```

脚本以「图谱字典命中 + 变更信号词邻近」为主判据，输出三组：
- `change_points`：带变更信号的高置信变更点，每个已解析成可查询目标（`resolved_targets`）。
- `review_candidates`：命中图谱但方案未明确标注变更，需复核。
- `unresolved_hints`：疑似离线数据集/未建模对象（如 Hive 表），查不到在线入口。

裸列名（如 `channel_code`）会自动展开为**所有含该列的表**——方案常不写落在哪张表，逐表查才完整。

### 第 3 步：语义复核（Agent 必做）

脚本是规则匹配，会有偏差。对照方案正文快速核对，并直接编辑 `change_points.json`：
- **补漏**：方案明确要改、但脚本没抓到的对象（如只在图里或表格里出现的字段），手动加进 `change_points`，
  参照 [change-point-taxonomy.md](references/change-point-taxonomy.md) 填 `type` 和 `resolved_targets`。
- **纠偏**：把误判为变更点的通用词（英文单词恰好等于某列名）移到 `review_candidates` 或删除。
- **降级处理**：Proto/接口/枚举变更若最终落到某 DB 列，就换成对应 DB 字段目标；落不到的保留在 `unresolved_hints`。

变更点类型与「查不到入口」的处理原则，见 [change-point-taxonomy.md](references/change-point-taxonomy.md)。

### 第 4 步：聚合影响面并出报告

默认用内置引擎一步聚合（自包含，直接读图谱数据）：

```bash
python3 scripts/analyze_impact.py --change-points change_points.json \
    --out-md report_body.md --out-json impact_agg.json
```

聚合产物：变更点→入口映射、跨仓库共享风险、P0/P1/P2 回归优先级（判定规则见 taxonomy 参考文档）。

**可选：复用同目录的 `impact_query.py`**（想要调用链举证、或统一查询口径时）：
```bash
# 1) 打印去重后的查询目标
python3 scripts/analyze_impact.py --change-points change_points.json --list-targets
# 2) 用同目录 impact_query.py 批量查询这些目标并 --json 落盘（得到 raw.json）
python3 scripts/impact_query.py --json <targets...> > raw.json
# 3) 用外部结果聚合
python3 scripts/analyze_impact.py --change-points change_points.json --impact-json raw.json --out-md report_body.md --out-json impact_agg.json
```

最后用 `lark-doc` skill 把聚合结果生成为飞书报告文档。报告默认包含四个板块：概览摘要、跨仓库风险汇总、
变更点→受影响入口映射、回归建议优先级；表格用飞书 HTML `<table>` 语法（不要用 markdown 竖线表格）。
`report_body.md` 是数据草稿，`impact_agg.json` 是结构化权威数据，据此组织最终文档。

## 数据从哪来

脚本读取 `code_graph_system/output/` 下的 `impact_index.json`（权威入口倒排索引）与 `graph.json`（入口元信息、跨仓共享标记）。
定位顺序：`--graph-dir` → 环境变量 `CODE_GRAPH_OUTPUT_DIR` → `$AIME_WORKSPACE_PATH/code_graph_system/output` → 从当前目录逐级向上找。
数据不在默认位置时用 `--graph-dir code_graph_system/output` 显式指定。

图谱是某次构建的快照。若近期代码结构大改，需先由 `code_graph_system` 流水线重建再查，否则影响面结果滞后。

## 结果可信度

- **入口清单来自字段级倒排索引，是权威来源**；跨仓共享（`shared_by`）标记是漏改高危信号，务必在报告里突出。
- **查不到入口 ≠ 没风险**：可能纯内部使用、经未建模路径访问或图谱未覆盖，报告需保留 `unresolved_hints` 提示人工评估。
- 同目录 `impact_query.py` 可用于单点举证或复核调用链；`analyze_impact.py` 的内置聚合逻辑与其保持同源。
