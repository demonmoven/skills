# gstack 用户使用手册

本文档面向 gstack skill 的使用者。

> gstack skill 是 [garrytan/gstack](https://github.com/garrytan/gstack) v0.15.1.0 的 Claude Code / Trae 友好包装层。**全量迁移**了上游 33 个 skill 为统一入口 `/gstack <action>` 的 action 集合，并把切分支与产物落地对齐 speckit/exec-plan/openspec 的范式。

---

## 一句话理解 gstack

gstack 是一个**AI 工程团队角色编排引擎**：把 Claude Code 从一个通用助手变成 33 个专家角色（CEO、工程经理、设计师、Code Reviewer、QA、安全官、发布工程师、SRE、技术写手、安全官、调试专家等）的 sprint 流水线。一句需求描述即可走完 office-hours → autoplan → review → cso → qa → ship 全流程。

---

## 快速开始

### 前置条件

- Git
- 在主仓库目录或包含多个 git 子仓库的工作区目录下启动 Claude Code / Trae
- （部分 action 可选）`browse` 二进制 — 详见下方"前置条件 — browse 二进制"
- （部分 action 可选）`codex` CLI — 详见下方"前置条件 — codex CLI"

### 一键全流程（最推荐）

```
/gstack run 用户登录流程优化
```

或传入飞书文档链接：

```
/gstack run https://xxx.feishu.cn/docx/xxx
```

自动完成：切 feature 分支 → 创建 `docs/xdev/gstack/{FEATURE_NAME}/` 目录 → office-hours → autoplan → 暂停等实现 → review → cso → qa → ship。

### 单 action 使用

```
/gstack office-hours 我有一个 idea
/gstack review                       # 在当前分支跑 7 路并行 code review
/gstack qa http://localhost:3000     # 在当前分支跑浏览器 QA
/gstack ship                         # 在当前分支跑发布流程
```

每个单 action 的产物都会写到 `{PRIMARY_REPO}/gstack/{当前分支名}/{action}.md`。

---

## 33 个 Action 全集

按职能分组列出全部 33 个 action：

### 1️⃣ Sprint 全流程入口（1）

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `run` ⭐ | `[需求描述或链接]` | 全流程串联（office-hours → autoplan → review → cso → qa → ship） |

### 2️⃣ 构思与计划（5）

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `office-hours` | `[需求或 idea]` | YC Office Hours / Builder mode（产品构思） |
| `plan-ceo-review` | `[FEATURE_NAME]` | CEO / 创始人级审查（"Brian Chesky 模式"） |
| `plan-eng-review` | `[FEATURE_NAME]` | 工程经理架构审查（含 worktree 并行化策略） |
| `plan-design-review` | `[FEATURE_NAME]` | 高级设计师 7 维度审查 |
| `autoplan` | `[FEATURE_NAME]` | 一键串联 CEO + Eng + Design 三审 |

### 3️⃣ 设计与视觉（4）

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `design-consultation` | `[FEATURE_NAME]` | 设计合伙人（从零构建设计系统） |
| `design-shotgun` | `[FEATURE_NAME]` | 设计探索器（多变体并行生成） |
| `design-html` | `[FEATURE_NAME]` | 设计工程师（设计稿 → 生产级 HTML） |
| `design-review` ⚠️ | `[url]` | 80 项视觉审计 + 修复循环（**需 browse**） |

### 4️⃣ 调查与审查（4）

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `investigate` | `[bug 描述]` | 调试专家根因分析（铁律：调查先于修复） |
| `review` | `[base_branch]` | 7 路并行 code review |
| `codex` ⚠️ | `[模式]` | OpenAI Codex 跨模型审查（**需 codex CLI**） |
| `cso` | `[scope]` | 首席安全官（OWASP + STRIDE） |

### 5️⃣ QA 与浏览器（5）

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `qa` ⚠️ | `[url]` | 真实浏览器 QA + 自动修复（**需 browse**） |
| `qa-only` ⚠️ | `[url]` | QA 报告（不改代码）（**需 browse**） |
| `browse` ⚠️ | `[command]` | 持久化 Chromium 自动化（**需 browse**） |
| `connect-chrome` ⚠️ | （无） | 连接真实 Chrome（带侧边栏扩展）（**需 browse**） |
| `setup-browser-cookies` ⚠️ | （无） | 从真实浏览器导入 Cookie（**需 browse**） |

### 6️⃣ 发布与运维（6）

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `ship` | （无） | 发布全流程（test + review + push + PR） |
| `land-and-deploy` | （无） | 合并 PR + 等 CI + 部署 + 验证生产 |
| `document-release` | （无） | 自动更新所有项目文档以匹配刚发布内容 |
| `canary` | （无） | 部署后监控循环（控制台错误、性能回归） |
| `benchmark` | `[scope]` | 性能基准（页面加载、Core Web Vitals 等） |
| `setup-deploy` | （无） | 配置部署链路（首次设置） |

### 7️⃣ 复盘（1）

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `retro` | `[period]` | 工程经理周复盘 |

### 8️⃣ 安全防护（4）— ⚠️ 仅作为内容参考

> 上游通过宿主进程 hook 实现 `rm -rf` 拦截 / 文件锁定，devclaw 不提供 hook 机制。这 4 个 action 迁移过来是供 Claude 知道"这些角色存在，可以提醒用户注意"，但实际拦截能力依赖 Claude Code 主进程的安全机制。

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `careful` | （无） | 破坏性命令前警告 |
| `freeze` | `[directory]` | 锁定文件编辑范围 |
| `guard` | `[directory]` | careful + freeze 一键开启 |
| `unfreeze` | （无） | 解除 freeze |

### 9️⃣ 会话智能（3）— ⚠️ 需上游 gstack 安装

> 这 3 个 action 依赖 `~/.gstack/projects/{slug}/` 状态目录。在 devclaw 版中，本 skill 不创建该目录，需要用户独立安装上游 gstack 才能使用完整功能。

| action | 参数 | 角色 / 功能 |
|--------|------|------|
| `checkpoint` | `[name]` | 保存 / 恢复工作状态快照 |
| `health` | （无） | 代码健康度仪表板 |
| `learn` | （无） | 跨会话学习记忆管理 |

### 🔟 自管理（1）— ⚠️ devclaw 不适用

| action | 参数 | 说明 |
|--------|------|------|
| `gstack-upgrade` | （无） | 升级 gstack。**本仓库版本不适用** —— devclaw 通过 plugin marketplace 升级 gstack skill，使用 `xdev` plugin 的标准升级流程 |

---

## 场景化使用指南

### 场景 1：从零到发布的完整 sprint（最常用）

```
/gstack run 加一个团队日报 AI 摘要功能
```

执行流程：

```
Step 0: 切 feature 分支 + 创建 docs/xdev/gstack/{FEATURE_NAME}/feature.md
  ↓
Step 1: office-hours    — YC 6 强制问题 reframe 你的需求（HITL）
  ↓
Step 2: autoplan        — CEO + Eng + Design 三审（HITL）
  ↓
Step 3: 暂停，等待你完成实现
  ↓
Step 4: review          — 7 路并行 code review（自动）
  ↓
Step 5: cso             — OWASP + STRIDE 安全审计（自动）
  ↓
Step 6: qa              — 真实浏览器 QA（自动，可选）
  ↓
Step 7: ship            — test + push + PR（自动）
```

### 场景 2：在已有分支上跑单个角色

```
/gstack review                              # 7 路并行 review 当前 diff
/gstack cso                                 # 安全审计当前 diff
/gstack qa http://localhost:3000            # 浏览器 QA
/gstack ship                                # 直接走发布流程
```

每次调用产物写到 `gstack/{当前分支名}/{action}.md`。

### 场景 3：从中断处恢复

任何 action 中断了？直接不带参数重新调：

```
/gstack run                                 # 从断点恢复全流程
/gstack review                              # 续接当前分支的 review
```

gstack 会从 `docs/xdev/gstack/{FEATURE_NAME}/` 下的已有产物推断恢复点。

### 场景 4：bug 修复"调查 → 修复 → 验证"循环

```
/gstack investigate "登录后跳转 404"        # 系统化根因分析
/gstack qa http://localhost:3000            # 修复后用浏览器验证
```

### 场景 5：纯设计探索

```
/gstack design-consultation                  # 从零构建设计系统
/gstack design-shotgun                       # 多变体并行探索
/gstack design-html                          # 选定方案 → 生产级 HTML
/gstack design-review http://localhost:3000  # 80 项视觉审计 + 修复
```

### 场景 6：跨模型 review（最高置信度）

```
/gstack review            # Claude 内置 7 路 review
/gstack codex             # OpenAI Codex 独立审查
                          # 两份报告对比，重叠的 finding = 高置信度
```

### 场景 7：发布后运维

```
/gstack ship              # 创建 PR
/gstack land-and-deploy   # 合并 + 部署 + 验证生产
/gstack canary            # 部署后监控循环
/gstack document-release  # 更新文档
/gstack retro             # 一周后复盘
```

---

## 产出物说明

所有产物存放在 `{PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}/` 目录下：

```
{PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}/
├── feature.md             # 原始需求描述（仅 run 模式）
├── run.md                 # run 全流程的时间线 + 决策日志（仅 run 模式）
├── office-hours.md
├── plan-ceo-review.md
├── plan-eng-review.md
├── plan-design-review.md
├── autoplan.md
├── design-consultation.md
├── design-shotgun.md
├── design-html.md
├── design-review.md
├── investigate.md
├── review.md
├── codex.md
├── cso.md
├── qa.md
├── qa-only.md
├── ship.md
├── land-and-deploy.md
├── document-release.md
├── canary.md
├── benchmark.md
├── retro.md
└── artifacts/             # 截图、日志等附件
```

`FEATURE_NAME` 取值规则：
- `/gstack run` 模式：自动生成 `{14位时间戳}-{slug}`（如 `20260408143000-user-login-optimization`），同时切同名分支
- 单 action 模式：直接用当前分支名（main / develop / feat-xxx 等）

---

## 与 speckit / exec-plan / openspec 的关系

| 维度 | speckit | exec-plan | openspec | **gstack** |
|------|---------|-----------|----------|-----------|
| 定位 | SDD 重型多文件 spec | 自包含 ExecPlan | fluid 多 change | **角色编排 33 个 action** |
| 适合任务 | 后端新功能 / API 设计 | 1-N 小时复杂任务 | 多 change 并行 | **完整 SDLC 闭环** |
| 切分支 | 必须 | 必须 | 必须 | **run 模式必须，单 action 不切** |
| 中间产物目录 | `docs/xdev/speckit/{FEATURE_NAME}/` | `docs/xdev/exec-plan/{FEATURE_NAME}/` | `docs/xdev/openspec/changes/{name}/` | **`docs/xdev/gstack/{FEATURE_NAME}/`** |
| Action 数量 | 7 | 1 | 11 | **33** |
| 角色感 | 弱（流程驱动） | 弱（计划驱动） | 弱（artifact 驱动） | **强（专家角色驱动）** |
| 浏览器 QA | 无 | 无 | 无 | **有（依赖 browse 二进制）** |
| 跨模型 review | 无 | 无 | 无 | **有（codex action）** |

四个 skill 共用同一份「git 仓库扫描 + 主仓库选择」逻辑：`prompts/resolve_workspace.md`（4 份物理副本，gstack 是第 4 份）。

---

## 前置条件 — browse 二进制

`qa` / `qa-only` / `browse` / `connect-chrome` / `setup-browser-cookies` / `design-review` 等 6 个 action 依赖 gstack 上游的 `browse` 二进制（持久化 Chromium 自动化工具）。本 skill **不内置**该二进制，理由：跨平台编译 + ~58MB 体积不适合放进 plugin marketplace。

### 安装上游 browse 二进制

```bash
git clone --depth 1 https://github.com/garrytan/gstack ~/.claude/skills/gstack
cd ~/.claude/skills/gstack
bun install
bun run build       # 编译 browse 二进制到 ~/.claude/skills/gstack/bin/browse
cd -
```

完成后重新调用 `/gstack qa`，会自动检测到二进制并启用。

### 检测不到二进制时

依赖 browse 的 action 会通过 AskUserQuestion 让你选择：

- A) 安装上游 browse 二进制（提示安装命令，安装后重试）
- B) 切换为纯文本模式（不打开浏览器，仅基于代码 diff 与 console 日志推断）
- C) 中止本次 action

---

## 前置条件 — codex CLI

`/gstack codex` 依赖 OpenAI Codex CLI：

```bash
npm install -g @openai/codex
codex auth login    # 授权一次
```

未安装时 `/gstack codex` 会报错并提示安装命令。

---

## 与 worktree 的关系

gstack 上游**没有专门的 worktree skill**。`worktree` 仅在 `plan-eng-review` 内嵌一段"并行化策略"分析，告诉用户如何用 git worktree 把工作拆分到多个分支并行执行。这段已随 `plan-eng-review` action 一并迁移。

如果你需要真正的并行 worktree 编排，参考上游 gstack 推荐的 [Conductor](https://conductor.build) 工具（独立于 gstack 之外）。

---

## 常见问题

### Q: gstack 与 speckit / exec-plan / openspec 应该用哪个？

- **新建后端 API / 数据模型** → speckit（重型多文件 spec）
- **从 idea 到 ship 的完整 sprint，含 QA 与 review** → gstack
- **复杂重构 / 跨模块改动 / 1-N 小时任务** → exec-plan
- **多 change 并行 / brownfield 迭代** → openspec

四者可以并存，互不冲突。

### Q: 我不想用 office-hours / plan-* 怎么办？

直接跳到单 action：

```
/gstack review
/gstack qa http://localhost:3000
/gstack ship
```

### Q: 全流程跑到一半发现方向不对怎么办？

在 office-hours / autoplan / Step 3（暂停等实现）阶段直接说"不对，我要的是 X"，gstack 会重新生成对应文档。

### Q: ship 失败了怎么办？

`docs/xdev/gstack/{FEATURE_NAME}/ship.md` 会记录失败原因。修复后重新 `/gstack ship` 即可（断点续传）。

### Q: 在 main 分支跑 single action 会怎样？

产物会落到 `gstack/main/{action}.md`。下次再跑同样的 action 会**追加**到该文件（用 `## 第 N 次执行` 分隔）。如果要避免追加，可以先创建一个 feature 分支再跑。

### Q: gstack/ 目录要不要 commit 进 git？

由你决定。gstack 不会主动改你的 `.gitignore`：

- **commit 进 git**：PR 评审者能看到 AI sprint 的所有中间产物（推荐用于多人协作）
- **加入 .gitignore**：保持仓库干净（推荐用于个人项目）

### Q: careful / freeze / checkpoint / learn / health 这些 action 真的能用吗？

- **careful / freeze / guard / unfreeze**：上游通过宿主 hook 实现，devclaw 没有 hook，本仓库版本仅起到"提醒模式"作用，不能真正拦截命令
- **checkpoint / health / learn**：依赖 `~/.gstack/projects/` 状态目录，需要用户独立 `git clone garrytan/gstack ~/.claude/skills/gstack && ./setup` 才能完全工作

如果你确实需要这些功能，建议同时安装上游 gstack；否则把这些 action 当作"角色提示"使用。

### Q: gstack-upgrade 不能用？

对，本仓库的 gstack skill 由 devclaw 的 plugin marketplace 管理升级，使用 `xdev` plugin 的标准升级流程即可。

---

## 速查表

| 我想... | 命令 |
|---------|------|
| 全流程（新建 feature） | `/gstack run 需求描述` |
| 全流程（飞书文档） | `/gstack run https://xxx.feishu.cn/docx/xxx` |
| 全流程（续接） | `/gstack run` |
| 产品构思 | `/gstack office-hours 我有 idea` |
| CEO 审查 | `/gstack plan-ceo-review` |
| 工程审查 | `/gstack plan-eng-review` |
| 设计审查 | `/gstack plan-design-review` |
| 一键三审 | `/gstack autoplan` |
| 设计系统 | `/gstack design-consultation` |
| 多变体探索 | `/gstack design-shotgun` |
| 设计稿到 HTML | `/gstack design-html` |
| 视觉审计 | `/gstack design-review http://localhost:3000` |
| Code review | `/gstack review` |
| 跨模型 review | `/gstack codex` |
| 安全审计 | `/gstack cso` |
| 调查 bug | `/gstack investigate "bug 描述"` |
| 浏览器 QA | `/gstack qa http://localhost:3000` |
| QA 报告（不改代码） | `/gstack qa-only http://localhost:3000` |
| 浏览器命令 | `/gstack browse <command>` |
| 连真实 Chrome | `/gstack connect-chrome` |
| 导入 Cookie | `/gstack setup-browser-cookies` |
| 发布 | `/gstack ship` |
| 部署到生产 | `/gstack land-and-deploy` |
| 部署后监控 | `/gstack canary` |
| 性能基准 | `/gstack benchmark` |
| 更新文档 | `/gstack document-release` |
| 配置部署 | `/gstack setup-deploy` |
| 周复盘 | `/gstack retro` |
| 保存进度 | `/gstack checkpoint` ⚠️ |
| 健康检查 | `/gstack health` ⚠️ |
| 学习记忆 | `/gstack learn` ⚠️ |
| 安全模式 | `/gstack careful` / `/gstack guard` ⚠️ |

⚠️ = 需上游 gstack 安装或仅作内容参考
