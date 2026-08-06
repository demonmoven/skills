---
name: harness-debt-scan
description: "架构级技术债扫描。不同于 lint 级别的风格检查，本技能检测需要理解代码语义才能发现的结构性问题：重复实现、模型分裂、状态散落、未抽象流程。产出结构化报告和可选 ExecPlan。当用户提到技术债扫描、架构审查、debt scan、重复实现、逻辑收敛、harness gc 时激活。"
license: Apache-2.0
metadata:
  author: guoshuai.030
  version: "1.3"
---

# Harness Debt Scan — 架构级技术债检测

## 核心理念

> **技术债就像高利贷，每天还一点小钱远比攒到还不起了再爆发好得多。**
> — OpenAI Harness Engineering

本技能检测的不是 lint 级问题（格式、命名），而是**需要理解代码语义**才能发现的架构级结构性问题。这正是 AI Agent 作为"代码垃圾回收器"的独特价值。

## 与 lint 的区别

| 层级 | 检测工具 | 示例 |
|------|---------|------|
| 风格 | eslint/biome/gofmt | 缩进、命名、未使用变量 |
| 正确性 | tsc/mypy/clippy | 类型错误、悬垂引用 |
| **结构** | **本技能** | 同一功能多处实现、模型分裂、状态散落 |

## 检测规则

### Rule 1: divergent-impl（同一功能多处独立实现）

**严重度**：高

**检测方法**：
1. 搜索语义相似的函数/方法名（如多个 `parseConfig`、`formatDate`、`buildURL`）
2. 比较实现逻辑——如果做的事情一样但代码不同，即为分歧实现
3. 检查是否有一个"应该被复用的"共享版本已经存在但没被用

**输出格式**：
```
[divergent-impl] 高优先级
  功能：用户身份解析
  实现 A：src/auth/parseUser.ts:15
  实现 B：src/api/middleware/getUser.ts:42
  差异：A 从 cookie 解析，B 从 header 解析，核心逻辑重复
  建议：收敛到 shared/auth/resolveUser.ts，通过参数区分来源
```

### Rule 2: model-divergence（数据模型分裂）

**严重度**：中高

**检测方法**：
1. 查找表示同一业务实体的类型定义（如前端 `User` vs 后端 `UserModel` vs API `UserResponse`）
2. 比较字段集——如果字段名/类型不一致但语义相同，即为模型分裂
3. 关注特别是跨层边界（前端/后端、API 契约/内部模型）的不一致

**触发信号**：
- 同一实体在不同文件中有不同字段名（`userId` vs `user_id` vs `uid`）
- 手动类型转换代码散布在多处
- API 响应结构与前端 store 结构不匹配但没有显式 adapter

### Rule 3: scattered-state（配置/状态散落）

**严重度**：中

**检测方法**：
1. 搜索同一类配置值在多个文件中被独立定义（如 API base URL、超时值、feature flags）
2. 检查是否有"唯一真相源"（一个 config 文件/环境变量）还是各处各自硬编码
3. 检查运行时状态是否散落在多个 store/全局变量中而非收敛

**触发信号**：
- 同一个 magic number 在多个文件中出现
- 环境变量在多处各自 `os.Getenv()` / `process.env.` 读取而非统一注入
- 同一个开关/标记在不同模块有各自的 boolean 字段

### Rule 4: unabstracted-flow（相似流程未抽象）

**严重度**：中

**检测方法**：
1. 查找结构相似的代码流程（如多个 CRUD handler 的 validate → execute → respond 模式）
2. 判断是否可以提取为通用模板/高阶函数/中间件
3. 注意：不是所有相似都需要抽象，只标记 3+ 处相同模式且未来大概率还会增长的

**判断标准**：
- 相似实例 ≥ 3 处
- 每处差异仅在"配置参数"层面（字段名、表名、校验规则等）
- 有新增实例的趋势（如新增业务域时会再复制一份）

### Rule 5: duplicate-persistence（重复持久化）

**严重度**：高

**检测方法**：
1. 查找同一数据在多个存储中被重复写入（如同时写 SQLite + 文件 + 内存 cache 但没有一致性机制）
2. 区分"有意的多级缓存"和"无意的重复写入"
3. 检查是否有数据不一致的风险路径

## 规则间去重

多条规则可能命中同一个问题的不同侧面（如"后端 URL 硬编码"同时触发 scattered-state 和 divergent-impl）。遇到这种情况：

1. **合并为一条发现**，归到影响最大的规则下
2. 在报告中注明"同时触发 Rule X"
3. 发现总数不重复计算

## 按技术栈的搜索模式

检测规则是语义通用的，但落地搜索时按技术栈有高效模式：

| 规则 | Go 搜索模式 | Node/TS 搜索模式 | Python 搜索模式 |
|------|-----------|-----------------|----------------|
| R1 divergent-impl | `&http.Client{`、多处 `func.*Parse`、重复 `func.*Handler` | `new axios(`、`fetch(`、重复 `export function` | 多处 `requests.get(`、重复 `def parse_` |
| R3 scattered-state | `os.Getenv("SAME_KEY")` 出现 3+ 处 | `process.env.SAME_KEY` 出现 3+ 处 | `os.environ["SAME_KEY"]` 出现 3+ 处 |
| R5 duplicate-persistence | 同一 struct 同时出现在 `db.Create` 和 `json.Marshal` 写文件路径 | 同一 key 同时 `localStorage.setItem` 和 API POST | 同一数据同时 `cursor.execute` 和 `open().write` |

## 执行流程

### Step 1: 确定扫描范围

询问用户：
- 全仓库扫描？
- 特定模块/目录？
- 只关注某类规则？

### Step 2: 建立结构认知

- 读取入口文件（AGENTS.md / ARCHITECTURE.md / README.md）
- 识别模块边界和依赖方向
- 理解技术栈和分层模式

### Step 3: 逐规则扫描

按 5 条规则逐项扫描。每条规则：
1. 使用语义搜索（函数名/类型名/模式匹配）定位候选
2. 阅读代码确认是否真的存在问题（排除 false positive）
3. 评估严重度和影响范围
4. 记录证据（文件路径 + 行号）

### Step 4: 产出报告

结构化报告包含：

```markdown
## 技术债扫描报告

### 扫描范围
- 目标：<path>
- 规则：全部 5 条 / 指定规则

### 发现汇总

| 规则 | 发现数 | 最高严重度 |
|------|--------|-----------|
| divergent-impl | N | 高/中/低 |
| model-divergence | N | ... |
| ... | ... | ... |

### 详细发现

#### [divergent-impl-001] <标题>
- **严重度**：高
- **位置**：file1:line1, file2:line2
- **描述**：...
- **收敛建议**：...

### 建议优先级
1. ...
2. ...
```

### Step 5: 可选 — 生成 ExecPlan

如果发现较多/较严重，询问用户是否生成收敛方案（ExecPlan / ADR / issue list）。

## Gotchas

- **不是所有重复都是债**。有意的代码复制（如 vendor、fork、protocol buffer 生成）不算
- **不是所有相似都该抽象**。只有 3+ 处相同模式且有增长趋势时才建议抽象
- **注意"有意的多级缓存" vs "无意的重复写入"**。前者是架构决策，后者才是债
- **false positive 比 false negative 更有害**。宁可漏报，不要误报——误报会让用户失去信任
- **报告要给出收敛路径，不只是指出问题**。每个发现都要附带"建议动作"
- **技术栈无关**。检测规则基于语义模式，不绑定具体语言或框架
