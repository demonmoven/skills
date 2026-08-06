# 2.1.1 Agents 文档体系

> **本节目标**：理解 Agent 文档体系的四大核心构件（"四大金刚"）、分层结构和最佳实践模板。

---

## 概念

**Agents 文档体系**是写给 AI Agent 看的项目文档，告诉 Agent "这个项目是什么、怎么组织、有什么规矩、去哪找参考资料"。

与传统 README.md 的区别：**README 写给人看，Agents 文档写给 Agent 看**。人类读文档是浏览式的，可以跳读、联想；Agent 读文档是精确消费式的，需要结构化、层次清晰、边界明确。

### 起源

**AGENTS.md** 作为一种开放格式，最初由社区提出，2025 年 8 月 GitHub Copilot 正式支持，同年 12 月捐赠给 Linux Foundation 下的 Agentic AI Foundation (AAIF)。目前已被 Cursor、Codex、Windsurf、Gemini CLI、Warp、Zed 等 60k+ 开源项目采用。

**CLAUDE.md** 是 Anthropic 为 Claude Code 设计的项目级指令文件，功能类似但始终加载到上下文中（非按需加载）。

两者的核心思想一致：**仓库即唯一真相源——Agent 看不到的信息等于不存在**。

---

## 既定事实

随着实践的深入，Agent 文档体系已从单一的 AGENTS.md 衍生出更丰富的结构。以下四类文档已成为行业既定事实：

| 文档 | 定位 | 写什么 | 不写什么 |
|------|------|--------|----------|
| **AGENTS.md** | 仓库总入口 / 模块入口 | 项目简介、知识导航表、核心约束、常用命令 | 不写详细 API 文档、不写操作 SOP、不重复其他文件已有的内容 |
| **ARCHITECTURE.md** | 架构鸟瞰 | Bird's Eye View、Code Map、Architecture Invariants、模块依赖方向 | 不写 API 接口细节、不写操作流程、不堆业务逻辑说明 |
| **references/** | 参考资料（事实说明） | 稳定的技术事实：API 契约、数据模型、集成点、运行时行为、外部依赖接入指南与踩坑记录 | 不写操作步骤（那是 guidances 的事） |
| **guidances/** | 操作手册（流程 SOP） | 分步操作流程：调试手册、发布指南、测试 SOP、联调流程 | 不写技术事实定义（那是 references 的事） |

---

## 文档分层体系

以 DevClaw 项目为例，展示一个成熟的分层结构。注意：**四大金刚不仅在根目录存在，每个模块/子模块也有自己的一套**：

```text
项目根/
├── AGENTS.md                                  ← L0 总入口（~100行）
├── ARCHITECTURE.md                            ← L0 架构鸟瞰（150-350行）
├── README.md                                  ← L0 人类入口
│
├── docs/                                      ← L1 中层知识库
│   ├── AGENTS.md                              ← L1 中层索引
│   ├── references/                            ← L1 参考资料
│   │   ├── sso-login-flow.md                  ← e.g. SSO 登录技术细节
│   │   ├── gateway-agent-api.md               ← e.g. 网关 Agent API 契约
│   │   ├── frontend-module-boundaries.md      ← e.g. 前端模块边界定义
│   │   ├── local-observability-playbook.md    ← e.g. 本地观测栈技术说明
│   │   └── ...（16 篇）
│   └── guidances/                             ← L1 操作手册
│       ├── release-guide.md                   ← e.g. 发布操作 SOP
│       ├── e2e-testing.md                     ← e.g. E2E 测试操作流程
│       ├── frontend-automation-playbook.md    ← e.g. 前端自动化验收 SOP
│       └── ...（7 篇）
│
├── frontend/                                  ← L2 前端模块
│   ├── AGENTS.md                              ← L2 前端模块入口
│   └── src/
│       └── modules/
│           ├── chat/                          ← L3 业务域
│           │   ├── AGENTS.md                  ← L3 域级约束（可选）
│           │   ├── docs/
│           │   │   ├── references/            ← 域级参考（如消息协议说明）
│           │   │   └── guidances/             ← 域级操作（如联调手册）
│           │   ├── components/
│           │   ├── hooks/
│           │   └── api/
│           ├── skills/
│           ├── agents/
│           └── cron/
│
└── backend/                                   ← L2 后端模块
    ├── AGENTS.md                              ← L2 后端模块入口
    └── modules/
        ├── auth/
        ├── skill/
        └── agent/                             ← L3 业务域
            ├── AGENTS.md                      ← L3 域级约束（可选）
            ├── ARCHITECTURE.md                ← L3 域级架构（可选）
            └── docs/
                ├── references/                ← 域级参考
                │   └── opencode-integration-guide.md  ← e.g. OpenCode 接入指南
                └── guidances/                 ← 域级操作
```

### 分层原则

| 层级 | 文件 | 职责 | 行数/规模 |
|------|------|------|----------|
| L0 | `AGENTS.md` | 总入口：知识导航表 + 核心约束 + 常用命令 | ~100 行 |
| L0 | `ARCHITECTURE.md` | 架构鸟瞰：Code Map + Invariants + 依赖方向图 | 150-350 行 |
| L1 | `docs/AGENTS.md` | 中层枢纽：按任务导航到 references/ 或 guidances/ | ~50 行 |
| L1 | `docs/references/` | 事实说明：API 契约、数据模型、集成点、接入指南 | 一主题一文件 |
| L1 | `docs/guidances/` | 操作手册：发布、测试、联调 SOP | 一流程一文件 |
| L2 | `module/AGENTS.md` | 模块入口：本模块结构、本地约束、常用命令 | ~80 行 |
| L3 | `module/.../AGENTS.md` | 域级约束：仅适用于当前子模块的规则 | ≤600 tokens |
| L3 | `module/.../docs/` | 域级 references + guidances（可选，复杂域才需要） | 按需 |

---

## 最佳实践：四大金刚模板

### 模板一：AGENTS.md

````markdown
# AGENTS.md — {项目名} 仓库文档总入口

<!-- 规范:
  - 控制在 ~100 行，是地图不是百科全书
  - 根目录 AGENTS.md 在 Agent 启动时立即加载，大小直接影响上下文预算
  - 模块级 AGENTS.md 在 Agent 分析到对应模块时按需加载，单个不超过 ~600 tokens
-->

<!-- 思想:
  - 最小作用范围：当前层级只写适用于当下模块的规则
  - 最少指令：保证够用的前提下尽量少——多了反而稀释重点
  - 指令清晰：无歧义，避免模棱两可
-->

<!-- 禁区:
  - 不写 Agent 可自主获取的内容（目录结构、依赖关系等）——Agent 需要时会自己 ls/cat
  - 不写容易过期的内容（功能细节描述、版本号等）——过期指令比没有指令更危险
  - 不复述全局规则中已有的内容——重复不会增强执行意愿
-->

> **本文件是 {项目名} 仓库的总路由，不是百科全书。**

## 项目简介

{一段话概括项目是什么、用什么技术栈}

[e.g. DevClaw]

DevClaw 是一个 AI 研发工作台项目，采用 Wails 2 桌面应用 + Go 后端服务的架构：
- 桌面端（app/）：Wails 2 + React + TypeScript
- 后端（backend/）：Go + Hertz + GORM + Thrift/hz 代码生成

## 文档体系

{表格列出各层级文档的入口和作用}

[e.g. DevClaw]

| 层级 | 入口 | 作用 |
|---|---|---|
| L0 总入口 | AGENTS.md | 仓库任务路由、协作约束 |
| L0 架构入口 | ARCHITECTURE.md | 代码地图、边界、不变量 |
| L1 中层索引 | docs/AGENTS.md | 专题文档、知识库入口汇总 |
| L2 模块入口 | frontend/AGENTS.md、backend/AGENTS.md | 各模块的本地入口 |
| L3 专题知识库 | docs/references/、docs/guidances/ | 事实说明、操作手册 |

## 知识导航

{最重要的部分——让 Agent 按任务快速定位应该读哪个文档}

[e.g. DevClaw]

| 我要做什么 | 去哪里看 |
|---|---|
| 理解整体架构与代码边界 | ARCHITECTURE.md |
| 改前端页面或状态流 | frontend/AGENTS.md |
| 了解后端目录与命令 | backend/AGENTS.md |
| 查看 SSO 登录流程细节 | docs/references/sso-login-flow.md |
| 查看桌面端发布流程 | docs/guidances/release-guide.md |
| 维护前端 E2E 测试 | docs/guidances/e2e-testing.md |

## 核心约束（所有 Agent 必须遵守）

{硬规则，违反会被 hook 拦截或导致架构腐化}

[e.g. DevClaw]

- 文件行数上限：600 行
- Go 函数长度上限：120 行
- 禁止提交占位符代码（TODO implement / STUB / HACK / FIXME）
- API 契约变更先改 IDL，再走生成流程；禁止手改生成文件
- 仅根目录允许 README.md，其他模块入口统一用 AGENTS.md

## 常用命令

{构建/测试/lint 的 one-liner}

[e.g. DevClaw]

```bash
cd backend && air              # 后端热重载
cd backend && go test ./...    # 后端测试
cd frontend && npm run dev     # 前端开发
golangci-lint run ./...        # Go 静态检查
```
````

---

### 模板二：ARCHITECTURE.md

````markdown
# {项目名} Architecture

<!-- 规范:
  - 控制在 150-350 行
  - 只负责解释代码架构、模块边界和不变量
  - 不承担文档导航职责（那是 AGENTS.md 的事）
-->

<!-- 思想:
  - Architecture Invariant 是最有价值的部分——写出"刻意不存在的东西"
  - "某事物的缺席"光看代码很难发现，一旦被 Agent 打破就是架构腐化
-->

> 仓库总入口见 AGENTS.md，人类阅读入口见 README.md

## Bird's Eye View

{一段话概括系统是什么、做什么、输入输出是什么}

[e.g. DevClaw]

DevClaw 是一个面向 AI 研发协作的工作台，采用 Wails 2 桌面应用 + Go 后端服务的双层架构：
- 桌面端（app/）：Wails 2 应用，Go 后端 + React 前端
- 后端服务（backend/）：独立的 Hertz HTTP 服务器
- Windy Runtime（windy/）：本地与远端共用的统一运行时

{ASCII 架构图}

[e.g. DevClaw]

```text
┌──────────────────────────────────┐
│  Desktop App (Wails 2)           │
│  ┌──────────┐  ┌──────────────┐  │
│  │ Go Layer │←─│ React Front  │  │
│  └────┬─────┘  └──────────────┘  │
│       │ HTTP                     │
└───────┼──────────────────────────┘
        v
  Hertz Server (backend/)
     ├── Generated Router → Handlers → pkg/storage, pkg/db
     └── Custom Router → Auth / Workspace / DevBox Handlers
```

## Code Map

{每个顶级目录的职责，附 Architecture Invariant}

### `frontend/` — 前端

{目录职责说明}

**Architecture Invariant:** {这个模块"刻意不做什么"}

[e.g. DevClaw]

**Architecture Invariant:** 前端通过 wailsjs/ binding 调用 Go 方法获取数据，不直接访问后端 HTTP API。`shared/` 不能反向依赖 `modules/*`。禁止跨域"横向抄逻辑"。

### `backend/` — 后端

{目录职责说明}

**Architecture Invariant:** {这个模块"刻意不做什么"}

[e.g. DevClaw]

**Architecture Invariant:** 启动流程是"先基础设施初始化（DB/observability），再注册路由并启动服务"；handler 不负责初始化全局资源。生成代码标注 DO NOT EDIT，业务增量下沉到 pkg/。API 契约变更先改 IDL，禁止手改生成文件"反向修契约"。

## Cross-Cutting Concerns

{横切关注点：认证、日志、测试策略等}

[e.g. DevClaw]

- **认证**：桌面端采用 ByteSSO device code flow，后端落地为本地 Session
- **可观测性**：VictoriaLogs/Traces 本地观测栈，结构化日志含 action + result
- **测试**：覆盖率采用"非生成代码"口径，由 test_coverage.sh 执行
````

---

### 模板三：references/{topic}.md

````markdown
# {主题名}

<!-- 规范:
  - 文档类型：参考资料（reference）——回答"是什么"
  - 一个主题一个文件，文件名即主题（sso-login-flow.md 比 doc-003.md 好）
  - 不写操作步骤（那是 guidances/ 的事）
  - 内容应是稳定的技术事实，不频繁变动
-->

<!-- 思想:
  - references 是 Agent 的"字典"——需要查某个技术细节时按需读取
  - 写清楚"是什么"和"为什么这么设计"，不写"怎么操作"
  - 特别适合记录外部依赖的接入指南和踩坑记录：项目在迭代过程中接入的
    各种外部系统（如 DevClaw 先后接入 OpenClaw 和 OpenCode 两套 Agent
    引擎），其技术细节、版本矩阵、已知坑、兼容性策略，都应沉淀在
    references 中——这些知识是 Agent 无法从代码中自主推断的
-->

> 文档类型：知识库事实说明（reference）。
> 上游入口：../../AGENTS.md、../AGENTS.md。

## 架构概览

{技术架构、数据流向、系统边界}

[e.g. DevClaw sso-login-flow.md]

```
桌面端 React → Wails Binding → Desktop Go → HTTP → Backend (Hertz) → ByteSSO
```

三层架构：
- **前端 UI**：SplashScreen.tsx
- **桌面端 Go**：app.go（代理 HTTP 请求 + 本地 session 持久化）
- **后端服务**：auth_service.go（SSO 交互 + 用户/会话管理）

## 核心流程 / 数据模型 / API 契约

{具体技术细节：时序图、字段定义、状态机等}

[e.g. DevClaw tos-usage.md]

核心链路：应用代码 → `pkg/tcc.GetTosConfig()` → `pkg/storage.TOSStore` → TOS

代码落点：
- TCC 配置结构：`backend/pkg/tcc/config.go`
- 存储抽象：`backend/pkg/storage/storage.go`
- Key 规范：`backend/pkg/storage/keys.go`

## 设计决策

{为什么这么设计，有什么取舍}

[e.g. DevClaw tos-usage.md]

对象 key 必须通过统一工具函数 `BuildSkillFileKey` 生成，禁止手写拼接路径。所有 key 必须做路径清洗并拒绝路径穿越（如 `../`）。

## 已知限制 / 踩坑记录

{当前实现的已知问题、版本矩阵、兼容性坑}

[e.g. DevClaw opencode-integration-guide.md]

- OpenCode managed mode 下 Windy 自动检测/安装 opencode 二进制
- External mode 下需显式配置 base_url
- session_store_path 维护 instance_key → session_id 映射
````

---

### 模板四：guidances/{process}.md

````markdown
# {流程名}

<!-- 规范:
  - 文档类型：操作手册（guidance）——回答"怎么做"
  - 一个流程一个文件
  - 不写技术事实定义（那是 references/ 的事）
  - 步骤必须可机械执行——Agent 或新人按步骤走就能完成
-->

<!-- 思想:
  - guidances 是 Agent 的"SOP 手册"——需要执行某个流程时按需读取
  - 每一步给出具体命令和预期输出，不留模糊空间
  - 常见问题和排障信息尤其重要——Agent 遇到报错时会回来查
-->

> 文档类型：知识库操作手册（guidance）。
> 上游入口：../../AGENTS.md、../AGENTS.md。

## 概览

{一段话说明这个流程的目的和适用场景}

[e.g. DevClaw release-guide.md]

当前发布流程采用"候选版本 → 构建上传 → 手动设为线上版本"的中心化模式：
1. 发布中心创建候选版本（写入 backend 数据库）
2. 构建脚本通过 ldflags 把版本注入二进制
3. 上传产物到 TOS
4. 手动 Promote 为线上目标版本

## 前置条件

{环境、权限、依赖等}

[e.g. DevClaw release-guide.md]

- Go >= 1.23
- Node.js >= 18 与 npm
- Wails CLI v2
- TOS 写入权限

## 操作步骤

### Step 1: {动作}

```bash
{具体命令}
```

{预期输出 / 成功标志}

[e.g. DevClaw e2e-testing.md]

### Step 1: 启动浏览器模式

```bash
cd app/frontend && ./scripts/run-frontend-e2e.sh
```

脚本会自动启动 `wails dev -browser`，等待 Vite ready 后执行 Playwright 测试。

### Step 2: 查看测试结果

测试完成后脚本自动清理进程。失败用例的截图保存在 `test-results/` 目录下。

## 常见问题 / 排障

{踩过的坑、错误处理}

[e.g. DevClaw e2e-testing.md]

- 若端口被占用，先 `lsof -i :34115` 检查并 kill 残留进程
- fake runtime 注入失败时检查 `fixtures.ts` 中的 `window.runtime` mock 是否覆盖了新增的 binding

## 回滚方案

{如果出了问题怎么恢复}
````

---

## Good Case vs Bad Case

### Good Case：DevClaw 的做法

- 根 AGENTS.md ~100 行，包含知识导航表和 8 条核心约束
- ARCHITECTURE.md ~250 行，每个模块都有 Architecture Invariant
- `docs/references/` 16 篇参考资料（含 TOS 踩坑、SSO 流程、OpenClaw 兼容策略等外部依赖接入记录）
- `docs/guidances/` 7 篇操作手册（发布、E2E 测试、联调 SOP）
- 前端、后端各有自己的模块级 AGENTS.md，只管本模块的事
- references 文件头标注"文档类型：知识库事实说明"，guidances 标注"文档类型：知识库操作手册"

### Bad Case：常见错误

| 错误 | 后果 |
|------|------|
| 把所有文档内容堆进一个大 AGENTS.md | 上下文溢出，Agent 遗忘关键约束 |
| 用 LLM 生成 AGENTS.md | GitHub 研究：8 个场景中 5 个降低任务成功率 |
| 没有 Architecture Invariant | Agent 无法判断什么是"故意不做的"，容易破坏架构 |
| references 和 guidances 混在一起 | Agent 分不清"是什么"和"怎么做"，答非所问 |
| 外部依赖的接入踩坑不记录到 references | Agent 重复踩同样的坑，每次从头排查 |
| 写 Agent 可自主获取的内容（目录树、依赖） | 浪费 token 预算，Agent 需要时会自己 `ls` |
| 写容易过期的内容（版本号、功能细节） | 过期指令比没有指令更危险 |
| 模块级 AGENTS.md 复述全局规则 | 重复浪费 token，不会增强执行意愿 |

---

[下一节：MCP vs CLI →](021b-mcp-vs-cli.md) | [返回上级：上下文控制](021-context-control.md)
