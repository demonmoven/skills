# 数据分析

## 核心原则

1. **先理解，后执行**：动手前先理解用户真实需求和数据环境。
2. **专家视角**：从数据分析师角度提供专业分析，不只返回查询结果。
3. **多想一步**：主动发现异常、趋势和洞察，不仅回答字面问题。
4. **数据诚实**：绝不编造数据，图表不误导。如实呈现异常值和缺失值。
5. **结论先行**：先说好还是不好，再说为什么。

## 全局要求

- **语言适配**：跟随用户输入语言（中文→全流程中文，English→English，日本語→日本語）
- **核心产出**：每次完整分析必须包含 ① HTML 交互报告 ② PNG 静态截图 ③ 结构化文字结论
- **生成报告前必须问**：受众是谁？用途是什么？

## 工作流（7 步）

| 步骤 | 目标 | 关键 API | 输出 |
|------|------|----------|------|
| 1. 数据探查 | 理解数据环境，确定目标表和字段 | `list_instances`, `list_tables`, `get_table_info`, `query_sql`(LIMIT 10) | 表结构、字段含义、样例数据 |
| 2. 数据获取 | 获取原始数据 | `query_sql` / `execute_sql` / `nl2sql` + `MultiSourceAnalyzer` | DataFrame / CSV |
| 3. 质量检查 | 检查缺失值、重复行、异常值 | pandas: `isnull()`, `duplicated()`, `describe()` | 质量问题列表 |
| 4. EDA 分析 | 探索性分析，发现规律和异常 | pandas 聚合 + 框架分析 | 统计指标、趋势、对比 |
| 5. 结论提炼 | 输出结构化结论 | — | 核心洞察 + 行动建议 |
| 6. 报告生成 | HTML 可视化报告 + PNG 截图 | Write 工具 + Playwright | `analysis_report.html` + `.png` |
| 7. 交付 | 展示 PNG + HTML 链接给用户 | — | 最终回复 |

**阻断点**：字段含义不明、找不到目标表、列值无法对应业务含义时，**必须停下来询问用户**。

## 数据探查策略

探查顺序：数据库 → 表 → 字段 → 列值。

```python
from toolbox import create_client, list_instances, search_cached_instances, list_tables, get_table_info, query_sql

client = create_client(vregion="cn")

# 找数据库（按优先级：收藏 → 自创 → 缓存 → 全量）
favor_result = list_instances(client, favor=True)
owned_result = list_instances(client, owned=True)
cached_result = search_cached_instances(keyword="company")

# 找表
tables = list_tables(client, database="company", fetch_all=True)

# 查结构 + 样例
schema = get_table_info(client, table="orders", database="company")
sample = query_sql(client, sql="SELECT * FROM orders LIMIT 10", database="company")
```

## 数据获取

### 3000 行截断规则（必须理解）

`query_sql` / `execute_sql` 单次最多返回 **3000 行**，超出部分**静默截断**（不报错、不提示）。

> ⚠️ **返回 3000 行 = 数据被截断**，绝不能把 3000 当作真实总数。
> 需要真实计数时，必须用 `SELECT COUNT(*) FROM table WHERE ...`。

### 查询策略（必须遵守）

| 场景 | 做法 |
|------|------|
| 需要总数/聚合指标 | **必须用聚合 SQL**：`SELECT COUNT(*)`, `SUM()`, `AVG()` 等，在数据库端完成计算 |
| 小表（确认 < 3000 行） | 一条 SQL 拉原始数据，pandas 本地 groupby |
| 大表多维度分析 | SQL 端聚合：`SELECT a, b, COUNT(*), SUM(x) GROUP BY a, b`，1-2 条 SQL 覆盖所有维度 |
| ❌ 禁止 | 用返回行数当总数；每个维度单独发一条 SQL |

### 大表防护（防止超时）

聚合 SQL 也可能触发全表扫描导致超时（千万级表 GROUP BY 无索引列 → 扫描全表）。**查询前必须先评估表大小和索引**：

```python
# 1. 查表大小（行数估算）
row_count = execute_sql(client, sql="SELECT COUNT(*) as cnt FROM my_table", database=db)

# 2. 查索引（SHOW CREATE TABLE 或 EXPLAIN）
index_info = execute_sql(client, sql="SHOW CREATE TABLE my_table", database=db)
plan = execute_sql(client, sql="EXPLAIN SELECT ...", database=db)
# 检查 type 是否为 ALL（全表扫描），rows 是否过大
```

**按表大小选策略**：

| 表大小 | 有索引覆盖 WHERE/GROUP BY | 无索引 |
|--------|--------------------------|--------|
| < 10 万行 | 直接查 | 直接查 |
| 10 万 ~ 500 万行 | 直接聚合 | 加 WHERE 缩小范围（时间、主键区间） |
| > 500 万行 | 直接聚合 | **必须**加 WHERE 缩小范围，或分段采样 |

**无索引大表的处理方式**：
- **加 WHERE 缩范围**：用时间列、主键 id 区间限制扫描范围（如 `WHERE id > max_id - 100000`）
- **分段采样**：按主键区间取多段样本，拼接后分析（牺牲精确度换可行性）
- **先 EXPLAIN**：执行前先 `EXPLAIN` 确认不会全表扫描

- 确实需要全量原始数据时，用翻页循环（最后手段）：
  ```python
  rows, offset = [], 0
  while True:
      r = execute_sql(client, sql=f"SELECT col1, col2 FROM t LIMIT 3000 OFFSET {offset}", database=db)
      batch = r["data"]["rows"]
      if not batch: break
      rows.extend(batch)
      offset += len(batch)
  ```

### 三种查询方式

1. **nl2sql**：`nl2sql(query, tables=[...])` → `execute_sql(sql=...)`。快但可能有字段偏差。
2. **查 schema 后自写 SQL**：`get_table_info` → 根据真实字段名写 SQL → `query_sql`。更精准。
3. **直接执行**：用户给了完整 SQL，或 `SHOW TABLES` / `EXPLAIN` 等固定语句。

## 多数据源联合（MultiSourceAnalyzer）

数据分散在多个数据库或需要 DB + 文件联合分析时使用。

```python
from toolbox import create_client, query_sql
from multi_source_analyzer import MultiSourceAnalyzer

analyzer = MultiSourceAnalyzer()

# 注册数据库查询结果
client = create_client(vregion="boe")
df_orders = query_sql(client, sql="SELECT * FROM orders LIMIT 1000", database="company")
analyzer.register_dataframe('orders', df_orders)

# 注册本地文件
analyzer.register_file('sales', '../data/regional_sales.csv')
analyzer.register_file('products', '../data/business_data.xlsx', sheet='Product Inventory')

# 跨源 SQL 联合查询（基于 DuckDB）
result = analyzer.query("""
    SELECT o.order_id, s.region, s.target
    FROM orders o JOIN sales s ON o.region = s.region_code
""")
```

| 方法 | 说明 |
|------|------|
| `register_dataframe(name, df)` | 注册 DataFrame |
| `register_file(name, path, sheet=None)` | 注册文件（CSV/Excel/JSON/Parquet） |
| `list_sources()` | 查看已注册数据源 |
| `preview(name, n=5)` | 预览数据 |
| `describe(name)` | 查看结构 |
| `query(sql, limit=100)` | 执行跨源 SQL |

## AnalysisWorkflow（跨执行持久化）

每次 `python3 -c "..."` 都是独立进程，变量全部丢失。AnalysisWorkflow 将中间结果保存到磁盘。

**何时用**：需要 2 步以上的分析任务（探查→查询→分析→报告）。单步查询不需要。

**按需持久化**：不是每步都要 save。核心需要持久化的：① 原始数据 ② 最终报告（HTML + PNG）。同一个 `python3 -c` 中能连续完成的步骤不需要分开持久化。

```python
from analysis_workflow import create_workflow, resume_workflow

# 创建
wf = create_workflow()  # → 记住 analysis_id
print(wf.get_analysis_id())

# 恢复
wf = resume_workflow("analysis_20260317_143022")

# 保存/加载
wf.save_step_output("02_acquisition", "raw_data.csv", df)
df = wf.load_step_output("02_acquisition", "raw_data.csv")

# 报告路径
html_path = wf.get_output_path("06_report", "analysis_report.html")
```

| 方法 | 说明 |
|------|------|
| `create_workflow()` | 创建工作区 |
| `resume_workflow(analysis_id)` | 恢复工作区 |
| `wf.get_analysis_id()` | 获取 analysis_id |
| `wf.get_workspace_path()` | 获取工作区路径 |
| `wf.save_step_output(step, filename, data)` | 保存（JSON/CSV） |
| `wf.load_step_output(step, filename)` | 加载 |
| `wf.get_output_path(step, filename)` | 获取输出文件路径 |
| `wf.save_data_sources(sources)` | 保存多数据源配置 |
| `wf.get_workspace_url()` | 获取 file:// URL |

### 工作区目录

| 步骤 | 目录 | 典型输出 |
|------|------|----------|
| 数据探查 | 01_exploration/ | tables.json |
| 数据获取 | 02_acquisition/ | raw_data.csv |
| 质量检查 | 03_quality/ | quality_report.json |
| EDA 分析 | 04_eda/ | metrics.json |
| 结论提炼 | 05_conclusion/ | conclusion.md |
| 报告生成 | 06_report/ | analysis_report.html, .png |

## 麦肯锡分析框架

根据问题类型选择框架，分析时必须明确引用所用框架名称：

| 问题类型 | 推荐框架 | 核心动作 |
|----------|----------|----------|
| 利润/增长问题 | 逻辑树 (Logic Tree) | MECE 拆解：利润 = 收入 - 成本；收入 = 量 × 价 |
| 行业/战略问题 | 波特五力 / PESTEL | 分析外部环境、竞争对手、替代品 |
| 市场/客户问题 | 3C 模型 / STP | 分析 Customer, Competitor, Company |
| 复杂归因问题 | 假设驱动 (Hypothesis-Driven) | 提出假设 → 数据验证 → 修正结论 |

**分析方法论要点**：
- **80/20 法则**：找出贡献 80% 结果的 20% 因子（头部客户、爆款产品）
- **对比分析**：必须包含同比(YoY)或环比(MoM)
- **金字塔原理**：结论先行（总→分→总），核心结论一句话直接回答用户问题
- **MECE 拆解**：分析维度不重叠、不遗漏
- **假设验证**：复杂归因时，提出 ≥3 个假设并用数据逐一验证

## EDA 分析类型

根据数据特征和用户问题，选择合适的分析类型组合：

| 分析类型 | 适用场景 | 核心操作 |
|----------|----------|----------|
| 分组统计 | 按维度聚合（按区域、按产品） | `groupby().agg()` |
| 时间趋势 | 含日期字段，看走势 | `resample()`, `pct_change()` |
| 分布分析 | 了解数据集中/离散程度 | `describe(percentiles=...)`, `value_counts()` |
| 相关性分析 | 探索变量间关系 | `corr()`, 散点图 |
| 排名分析 | Top-N、头部贡献者 | `nlargest()`, 累计占比 |
| 漏斗分析 | 转化率、流失率 | 按步骤 `nunique()` |
| 对比分析 | YoY/MoM、A/B 对比 | 同环比计算 |

## 深度分析（复杂问题时）

当分析涉及"为什么"、归因、预测等复杂问题时，需完成以下步骤（简单查询/统计可跳过）：

- **假设验证**：提出 ≥3 个假设（假设 → 数据证据 → 是否验证 → 影响程度）
- **归因分析**：回答"为什么"，量化各因素贡献占比
- **多维度拆解**：从与问题相关的维度拆解（产品、区域、时间、客单价等），识别头部贡献者和异常点

## 结论输出结构

结论必须包含以下要素（不需要严格模板，但要素不可缺）：

1. **数据来源声明**：实例、数据库、表名、查询 SQL、数据量
2. **框架应用说明**：问题类型 + 选用框架 + 拆解逻辑
3. **核心洞察**：每条必须引用具体数据（数值、百分比、趋势方向）
4. **行动建议**：具体动作 + 预期效果，可落地

## 分析底线

1. **不编造**：绝不编造数据或字段含义。不知道就问用户。
2. **有证据**：每条结论都有具体数据支撑。禁止"可能"、"大概"等模糊词。
3. **可落地**：行动建议必须具体可执行，包含预期效果。
