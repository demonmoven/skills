# DevClaw ExecPlan 用户使用手册

本文档面向 devclaw-exec-plan skill 的使用者。

---

## 一句话理解 ExecPlan

ExecPlan 是一个**执行计划驱动的复杂任务引擎**：你给出需求描述，它自动完成 深度调研代码库 → 创建自包含的执行计划 → 审阅确认 → 逐里程碑编码实施 全链路。

---

## 快速开始

### 前置准备

1. 在你的目标代码仓库目录下启动 Claude Code：
   ```bash
   cd /path/to/your-repo
   claude
   ```

2. 仓库中建议存在以下文件（非必需但推荐）：
   - `AGENTS.md` — 仓库文档入口
   - `constitution/constitution.md` — 代码库规约

### 一键全流程（最推荐）

```
/devclaw-exec-plan 重构认证中间件
```

或传入飞书文档链接：
```
/devclaw-exec-plan https://xxx.feishu.cn/docx/xxx
```

自动完成：创建 feature → 深度调研代码库 → 生成执行计划 → 审阅 → 逐里程碑编码实施。

---

## 参数说明

ExecPlan **不需要子命令**，直接传入需求描述或不传参数：

```
/devclaw-exec-plan [需求描述或飞书文档链接]
```

| 用法 | 说明 |
|------|------|
| `/devclaw-exec-plan 需求描述` | 新建 feature，从头跑全流程 |
| `/devclaw-exec-plan https://xxx.feishu.cn/docx/xxx` | 从飞书文档读取需求，新建 feature |
| `/devclaw-exec-plan` | 续接已有 feature，从断点继续 |

**不带参数时的自动识别**：
1. 先看当前 git 分支名是否匹配 `exec-plan/` 下的 feature
2. 再列出已有 feature 让你选择
3. 都没有则提示输入需求描述

---

## 场景化使用指南

### 场景 1：全流程（最常用）

**你有**：需求描述或飞书文档链接
**你想**：从零到代码，全链路自动化

```
/devclaw-exec-plan 重构认证中间件
```

### 场景 2：从中断处恢复

流程中断了？直接不带参数重新调：
```
/devclaw-exec-plan
```

自动检测已有产物并从断点继续。

### 场景 3：审阅阶段讨论修改

在 Step 2（审阅）阶段，你可以直接与 Agent 对话修改执行计划的设计，所有变更会同步更新到 exec-plan.md 的各章节。

### 场景 4：想重做执行计划

删除 exec-plan.md 后重新执行：
```bash
rm docs/xdev/exec-plan/{FEATURE_NAME}/exec-plan.md
```
然后 `/devclaw-exec-plan`。

---

## 执行流程详解

```
┌─────────────────────────────────────────────────────────────┐
│ Step 0: FEATURE_NAME 解析 + 环境准备                        │
│  · 带参数 → 新建 feature（切分支、创建目录）                │
│  · 不带参数 → 识别已有 feature（断点续传）                  │
└─────────────┬───────────────────────────────────────────────┘
              ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 1: 读取 URL + 深度调研 + 生成 exec-plan.md（自动）      │
│  · 读取用户输入中的飞书/HTTP 链接内容                       │
│  · 读取仓库文档体系（AGENTS.md 等）                         │
│  · 深入研究相关源码                                         │
│  · 生成执行计划 → exec-plan.md（含 13 个必需章节）          │
└─────────────┬───────────────────────────────────────────────┘
              ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 2: 审阅执行计划（交互）                                 │
│  · 审阅 exec-plan.md，提出修改意见                          │
│  · 重点关注：目标、里程碑、验收标准                         │
│  · 确认完毕后继续                                           │
└─────────────┬───────────────────────────────────────────────┘
              ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 3: 按里程碑逐步执行（自动 + 阻塞时交互）               │
│  · 按里程碑顺序逐步编码实施                                 │
│  · 每完成一个里程碑实时更新 Progress                        │
│  · 遇到阻塞 → 向你转问 → 续接执行                         │
│  · 频繁提交代码                                             │
└─────────────┬───────────────────────────────────────────────┘
              ▼
┌─────────────────────────────────────────────────────────────┐
│ 完成：输出执行摘要 + Outcomes & Retrospective                │
└─────────────────────────────────────────────────────────────┘
```

---

## 产出物说明

所有文档存放在 `docs/xdev/exec-plan/{FEATURE_NAME}/` 目录下：

```
docs/xdev/exec-plan/{FEATURE_NAME}/
├── feature.md              # 原始需求描述留底
├── exec-plan.md            # 执行计划（核心文档，含 13 个必需章节）
└── artifacts/              # 截图、产物等（如有）
```

### exec-plan.md 包含的章节

| 章节 | 用途 |
|------|------|
| Purpose / Big Picture | 改动意义和用户可感知价值 |
| Progress | 里程碑完成度追踪（带时间戳的 checkbox） |
| Surprises & Discoveries | 意外发现记录 |
| Decision Log | 设计决策（决策 + 理由 + 日期） |
| Outcomes & Retrospective | 成果和经验总结 |
| Context and Orientation | 仓库状态、关键文件、术语定义 |
| Plan of Work | 里程碑式的工作计划 |
| Concrete Steps | 精确命令和工作目录 |
| Validation and Acceptance | 刚性量化验收标准 |
| Idempotence and Recovery | 可重复性和恢复路径 |
| Artifacts and Notes | 关键输出物 |
| Documentation Update | 需要更新的文档 |
| Interfaces and Dependencies | 使用的库和接口 |

---

## 与 SDD-BE 的区别

| 维度 | SDD-BE | ExecPlan |
|------|--------|----------|
| 适用场景 | 后端功能开发（有明确的 API/数据模型） | 通用复杂任务（重构、跨模块改动、基础设施等） |
| 核心文档 | 多文件拆分（spec + 5 个技术方案文件） | 单一自包含文档（exec-plan.md） |
| 设计粒度 | 细粒度（数据模型、API 契约、配置等分开） | 里程碑粒度（按可独立验证的阶段组织） |
| 调用方式 | 子命令模式（specify / tech-design / dev 等） | 无子命令，直接传需求描述 |

**选择建议**：
- 新增后端 API/功能 → 用 SDD-BE
- 重构、性能优化、跨模块改动 → 用 ExecPlan
- 不确定 → 用 ExecPlan（更通用）

---

## 前置条件

1. **在代码仓库目录中**启动 Claude Code
2. **Git 可用**：implement 阶段会自动 commit
3. **（推荐）仓库文档**：存在 `AGENTS.md` 或 `constitution/constitution.md`
4. **（可选）飞书 MCP**：如需传入飞书文档链接，需配置飞书 MCP：
   ```json
   {
     "mcpServers": {
       "lark": {
         "command": "npx",
         "args": ["-y", "@larksuiteoapi/lark-mcp", "mcp", "-a", "<APP_ID>", "-s", "<APP_SECRET>"]
       }
     }
   }
   ```

---

## 常见问题

### Q: 流程中断了怎么办？
直接不带参数执行 `/devclaw-exec-plan`，自动从断点续接。

### Q: 想重做执行计划？
删除 exec-plan.md 后重新执行：
```bash
rm docs/xdev/exec-plan/{FEATURE_NAME}/exec-plan.md
```
然后 `/devclaw-exec-plan`。

### Q: 执行中遇到阻塞问题？
Agent 会自动向你转问，回答后自动续接执行。

### Q: 不带参数时怎么知道用哪个 feature？
自动检测：先看当前 git 分支名，再扫描 exec-plan/ 目录列出已有 feature 供选择。

### Q: ExecPlan 和普通 plan.md 有什么区别？
ExecPlan 是**自包含**的——新手仅凭这一个文档就能完成全部实现，不需要额外上下文。同时它是**活文档**，Progress 和 Decision Log 随工作推进实时更新。

### Q: 需求描述可以是飞书文档链接吗？
可以。需先配置飞书 MCP（见前置条件）。未配置时会提示你手动贴入文档内容。

---

## 速查表

| 我想... | 命令 |
|---------|------|
| 全流程（新建） | `/devclaw-exec-plan 需求描述` |
| 全流程（飞书链接） | `/devclaw-exec-plan https://xxx.feishu.cn/docx/xxx` |
| 全流程（续接） | `/devclaw-exec-plan` |
