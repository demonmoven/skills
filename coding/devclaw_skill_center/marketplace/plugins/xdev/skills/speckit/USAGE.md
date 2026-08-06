# speckit 用户使用手册

本文档面向 speckit skill（原 spkx / devclaw-sdd-be）的使用者。

---

## 一句话理解 speckit

speckit 是一个**规格驱动的后端开发全流程引擎**：你只需给出需求描述（文本或飞书文档链接），它自动完成 功能规格 → 技术方案 → 任务拆分 → 编码开发 全链路。

---

## 快速开始

### 前置准备

1. 在你的目标代码仓库目录下启动 Claude Code：
   ```bash
   cd /path/to/your-repo
   claude
   ```

2. 确保项目根目录下存在 `constitution/constitution.md`（代码库规约文档）

### 一键全流程（最推荐）

```
/speckit run 用户登录流程优化
```

或传入飞书文档链接：
```
/speckit run https://xxx.feishu.cn/docx/xxx
```

自动完成：创建 feature → 生成规格 → 审阅 → 技术方案 → 审阅 → 任务拆分 → 编码开发。

---

## 参数说明

speckit 的参数格式为：`/speckit <action> [参数]`

| action | 参数 | 说明 |
|--------|------|------|
| `run` | `[需求描述或链接]` | 全流程（带参数=新建，不带=续接） |
| `specify` | `[需求描述或链接]` | 只生成规格（带参数=新建，不带=续接） |
| `review-spec` | `[FEATURE_NAME]` | 审阅规格 |
| `tech-guidance` | `[FEATURE_NAME]` | 补充技术指导 |
| `tech-design` | `[FEATURE_NAME]` | 生成技术方案 |
| `review-tech-design` | `[FEATURE_NAME]` | 审阅技术方案 |
| `dev` | `[FEATURE_NAME]` | 任务拆分 + 编码开发 |

**所有参数均为可选**——不带参数时自动识别：
1. 先看当前 git 分支名是否匹配 `speckit/` 下的 feature
2. 再列出已有 feature 让你选择
3. 都没有则提示输入需求描述（specify/run）或报错（其他 action）

---

## 场景化使用指南

### 场景 1：全流程（最常用）

**你有**：需求描述或飞书文档链接
**你想**：从零到代码，全链路自动化

```
/speckit run 用户登录流程优化
```

**执行流程**：
```
Step 0: 创建 feature（自动生成 FEATURE_NAME、切分支、创建目录、写 feature.md）
    ↓
Step 1: specify — 生成功能规格说明书
    ↓
Step 2: review-spec — 你来审阅规格（交互式）
    ↓
Step 3: tech-guidance — 你提供技术选型/约束（交互式）
    ↓
Step 4: tech-design — 自动生成技术方案
    ↓
Step 5: review-tech-design — 你来审阅技术方案（交互式）
    ↓
Step 6: dev — 任务拆分 + 逐 Phase 编码开发
    ↓
完成 — 代码已提交到远端分支
```

### 场景 2：只生成规格

```
/speckit specify 新增支付接口
```

后续审阅：
```
/speckit review-spec
```

### 场景 3：从中断处恢复

流程中断了？直接不带参数重新调 run：
```
/speckit run
```

自动检测已有产物并从断点继续。也可以直接不带参数调用任意步骤：
```
/speckit tech-design
/speckit dev
```

### 场景 4：重新审阅

```
/speckit review-spec
/speckit review-tech-design
```

---

## 执行流程详解

```
┌─────────────────────────────────────────────────────────────┐
│ Step 0: FEATURE_NAME 解析 + 环境准备                        │
│  · 带参数 → 新建 feature（切分支、创建目录）                │
│  · 不带参数 → 识别已有 feature（断点续传）                  │
│  · 校验 constitution/constitution.md 存在                   │
└─────────────┬───────────────────────────────────────────────┘
              ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 1: specify（自动）                                     │
│  · 读取需求描述（feature.md）+ 展开引用文档 → prd.md        │
│  · 基于 prd.md 生成功能规格说明书 → spec.md                  │
└─────────────┬───────────────────────────────────────────────┘
              ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 2: review-spec（交互）                                 │
│  · 审阅 spec.md，提出修改意见                                │
│  · 确认完毕后继续                                           │
└─────────────┬───────────────────────────────────────────────┘
              ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 3: tech-guidance（交互）                               │
│  · 提供技术选型、性能要求、架构约束等                        │
│  · 无额外输入可直接跳过                                     │
└─────────────┬───────────────────────────────────────────────┘
              ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 4: tech-design（自动）                                 │
│  · 并发组 A: analyze + research + mining                    │
│  · mining-diff: 提取独特发现                                │
│  · 串行 plan: data-model → contracts → config → integ → plan│
└─────────────┬───────────────────────────────────────────────┘
              ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 5: review-tech-design（交互）                          │
│  · 审阅 5 个技术方案文件                                    │
│  · 确认完毕后继续                                           │
└─────────────┬───────────────────────────────────────────────┘
              ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 6: dev（自动 + 阻塞时交互）                            │
│  · task-split → subtasks_*.md                               │
│  · phase-dev → 逐 Phase 编码 → commit + push               │
│  · 遇到阻塞 → 向你转问 → 续接开发                          │
└─────────────┬───────────────────────────────────────────────┘
              ▼
┌─────────────────────────────────────────────────────────────┐
│ 完成：输出执行摘要                                           │
└─────────────────────────────────────────────────────────────┘
```

---

## 产出物说明

所有文档存放在 `docs/xdev/speckit/{FEATURE_NAME}/` 目录下：

```
docs/xdev/speckit/{FEATURE_NAME}/
├── feature.md              # 原始需求描述留底
├── prd.md                  # 完整需求文档（用户输入 + 引用文档展开）
├── spec.md                 # 功能规格说明书
├── tech-guidance.md        # 技术指导文档
├── analyze.md              # 代码库现状分析
├── research.md             # 技术方案调研
├── mining-raw.md           # 隐性需求挖掘原始版
├── mining.md               # 隐性需求挖掘差异化版
├── data-model.md           # 数据模型设计
├── contracts.md            # API 接口契约
├── configuration.md        # 配置设计
├── integration.md          # 系统集成设计
├── plan.md                 # 技术实现方案
└── subtasks_*.md           # Phase 任务拆分
```

代码库规约文档位于项目根目录：`constitution/constitution.md`

---

## 前置条件

1. **在代码仓库目录中**启动 Claude Code
2. **Constitution 文档**已准备：`constitution/constitution.md`
3. **Git 可用**：dev 阶段会自动 commit + push
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
直接不带参数执行 `/speckit run`，自动从断点续接。

### Q: 想重做某个步骤？
删除对应产出文件后重新执行。例如重做技术方案：
```bash
rm docs/xdev/speckit/{FEATURE_NAME}/plan.md docs/xdev/speckit/{FEATURE_NAME}/data-model.md docs/xdev/speckit/{FEATURE_NAME}/contracts.md docs/xdev/speckit/{FEATURE_NAME}/configuration.md docs/xdev/speckit/{FEATURE_NAME}/integration.md
```
然后 `/speckit tech-design`。

### Q: constitution.md 是什么？
代码库的规约文档，定义架构约定、编码规范等。放在项目根目录 `constitution/constitution.md`。

### Q: tech-guidance 可以跳过吗？
可以，直接说"继续推进"。

### Q: 需求描述可以是飞书文档链接吗？
可以。需先配置飞书 MCP（见前置条件）。未配置时会提示你手动贴入文档内容。

### Q: 不带参数时怎么知道用哪个 feature？
自动检测：先看当前 git 分支名，再扫描 speckit/ 目录列出已有 feature 供选择。

---

## Action 速查表

| 我想... | 命令 |
|---------|------|
| 全流程（新建） | `/speckit run 需求描述` |
| 全流程（续接） | `/speckit run` |
| 只生成规格 | `/speckit specify 需求描述` |
| 审阅规格 | `/speckit review-spec` |
| 补充技术指导 | `/speckit tech-guidance` |
| 只做技术方案 | `/speckit tech-design` |
| 审阅技术方案 | `/speckit review-tech-design` |
| 只做开发 | `/speckit dev` |
