# Phase 1 深度分析执行指南

本文档是 harness-bootstrap Phase 1 "仓库深度理解" 的详细执行指南。
Agent 在执行 Phase 1 时参考本文档。

## 第一轮：自动深度分析

在运行 `<skill_dir>/actions/init/phase-1-analyze/scripts/detect_stack.sh` 获取基础技术栈信息后，Agent 必须执行以下 5 个分析步骤。
每个步骤产出一段分析报告，最终汇总为 `harness-init-analysis.md`（临时文件，不入库）。

### 1. 代码模式采样

**目标**：识别仓库中反复出现的代码结构模式和惯例。

**执行方法**：
1. 从每个主要模块选取 2-3 个典型文件：
   - 入口文件（main、app、index、router）
   - 核心业务逻辑文件（service、handler、controller、store）
   - 测试文件（对应核心文件的测试）
2. 逐文件分析并记录：
   - **命名惯例**：文件名（kebab-case? PascalCase?）、函数名、变量名、类型名
   - **错误处理模式**：return error? throw? Result<T>? try-catch 位置?
   - **依赖注入方式**：构造函数注入? 装饰器? 全局容器? Wire?
   - **状态管理方式**：Zustand? Redux? 全局变量? Context?
   - **异步模式**：async/await? goroutine? channel? Promise.all?
3. 寻找重复出现的结构模式，用一句话总结：
   - 例："所有 HTTP handler 遵循 validate → process → respond 三段式"
   - 例："所有 React 组件使用 FC + props interface + CSS module 的固定结构"

**产出格式**：

```
## 代码模式采样报告

### 命名惯例
- 文件：kebab-case（如 chat-input.tsx, message-store.ts）
- 组件：PascalCase（如 ChatInput, MessageStore）
- 函数：camelCase
- 常量：UPPER_SNAKE_CASE

### 错误处理
- [描述观察到的模式]

### 依赖注入
- [描述观察到的模式]

### 结构模式
- [pattern 1]：[一句话描述] — 出现在 [文件列表]
- [pattern 2]：...
```

### 2. 模块依赖图

**目标**：理解模块间的依赖方向和边界。

**执行方法**：
1. 对于 JS/TS 项目：分析 package.json 的 dependencies、tsconfig paths
2. 对于 Go 项目：分析 go.mod 和 import 语句
3. 对于 Python 项目：分析 requirements.txt/pyproject.toml 和 import 语句
4. 对于 Rust 项目：分析 Cargo.toml 的 dependencies 和 use 语句
5. 绘制依赖方向图（ASCII 或 Mermaid），标注每个模块的角色

**产出格式**：

```
## 模块依赖图

### 角色标注
- 入口层：[列出]
- 核心业务层：[列出]
- 基础设施层：[列出]
- 工具/配置：[列出]

### 依赖方向（A → B 表示 A 依赖 B）
[ASCII 图或 Mermaid 图]

### 关键边界
- [边界 1]：[哪些模块不应跨越此边界]
```

### 3. Git 考古

**目标**：从 git 历史中推断开发模式、痛点和隐性知识。

**执行命令**：

```bash
# 近期开发方向
git log --oneline -200

# 高频修改文件 top 10
git log --pretty=format: --name-only -200 | sort | uniq -c | sort -rn | head -20

# revert 记录
git log --oneline --grep='revert' --grep='Revert' --grep='REVERT' -i -20

# 大型 merge commit
git log --oneline --merges -20

# 贡献者分布
git shortlog -sn --no-merges -200
```

**产出格式**：

```
## Git 考古报告

### 近期开发方向
[从 git log 总结最近的开发主题]

### 高频修改文件 Top 10
| 排名 | 文件 | 修改次数 | 推测原因 |
|------|------|----------|----------|

### Revert 记录
[列出 revert 记录，每条标注可能的原因]

### 风险信号
[从以上数据推断的潜在问题区域]
```

### 4. 外部集成点

**目标**：识别项目与外部系统/服务的所有集成点。

**执行方法**：
1. 搜索 HTTP client 初始化：`fetch`, `axios`, `http.NewRequest`, `requests.get`, `reqwest`
2. 搜索环境变量引用：`.env*` 文件、`process.env.`、`os.Getenv`、`os.environ`
3. 搜索配置文件中的外部 URL/host
4. 搜索 SDK 初始化代码（数据库、消息队列、缓存、监控等）

**产出格式**：

```
## 外部集成点清单

| 服务/系统 | 用途 | 接入方式 | 配置位置 |
|-----------|------|----------|----------|
```

### 5. 测试模式

**目标**：理解项目的测试策略和常用 mock 模式。

**执行方法**：
1. 分析测试目录结构和命名规范
2. 识别测试框架（Jest/Vitest/pytest/go test/cargo test）
3. 识别 mock 策略（mock library、fixture 文件、in-memory 替代、testcontainer）
4. 检查覆盖率配置
5. 检查 E2E 测试（Playwright/Cypress/Selenium）

**产出格式**：

```
## 测试模式报告

### 测试框架
[列出各模块使用的测试框架]

### Mock 策略
[描述观察到的 mock 模式]

### 覆盖率
[覆盖率要求和配置]

### 测试组织
[测试文件位置、命名规范]
```

---

## 第二轮：结构化用户访谈

基于第一轮自动分析结果，向用户提出 5-8 个关键问题。

### 必问问题（3 个）

这三个问题无论分析结果如何都必须问：

1. **核心业务场景**：
   > "这个项目的核心业务场景是什么？请用 1-2 段话描述，帮助 AI 理解业务域。"

2. **常见痛点**：
   > "团队开发中最常遇到的 bug 类型或问题是什么？"
   > 选项示例：构建配置问题 / 状态管理 bug / API 兼容性 / 性能问题 / 环境差异

3. **关键设计决策**：
   > "有哪些关键的设计决策是你希望 AI 一定要知道的？比如为什么选择了某个框架或架构模式。"

### 条件触发问题（根据分析结果选择 2-5 个）

| 触发条件 | 问题 |
|----------|------|
| 检测到 DI 容器 / IoC 模式（如 `@Injectable`、`wire.Build`、`container.register`） | "注册新服务/新 Port 的标准流程是什么？" |
| 检测到多区域/多环境部署（多个 `.env.*` 文件或 region/env 配置） | "各区域/环境之间有哪些行为差异？" |
| 检测到 SSE/WebSocket/长连接（如 `EventSource`、`WebSocket`、`grpc.Stream`） | "实时通信的重连策略和错误恢复机制是什么？" |
| Git 考古发现某文件被高频 revert（≥2 次 revert） | "文件 [X] 为什么容易出问题？" |
| 检测到代码生成目录（`generated/`、`DO NOT EDIT`、`.proto`、`.thrift`） | "代码生成的触发方式和注意事项？" |
| 检测到复杂状态管理（多个 store 文件、Zustand/Redux/MobX） | "核心数据流是怎样的？（从用户操作到 UI 更新）" |
| 检测到 monorepo（多个 package.json / go.mod） | "子包之间的发布依赖和版本协调方式？" |
| 检测到外部服务集成（≥3 个外部服务） | "各外部服务的职责和故障时的降级策略？" |

### 问题提出方式

- **一问一答**：不要一次抛出所有问题
- **每个问题提供选项**：尽可能给出多选项 + 自由输入
- **根据回答调整后续问题**：如果用户在某个问题上提供了丰富信息，可以跳过相关的后续问题

---

## 第三轮：验证与补问

1. 基于前两轮信息，生成"仓库理解摘要"（2-3 页），包含：
   - 项目定位与核心业务场景
   - 技术架构概述（引用依赖图）
   - 关键设计决策清单
   - 已识别的代码模式清单
   - 已识别的风险区域和痛点
   - 已识别的知识缺口（Phase 1 无法确定的内容）

2. 向用户展示摘要，逐节确认：
   > "以下是我对仓库的理解摘要，请检查是否有错误或遗漏。"

3. 用户纠正后，形成最终分析报告，用于后续 Phase 参考。
