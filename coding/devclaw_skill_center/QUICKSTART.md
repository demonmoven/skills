# xdev QUICKSTART

> 拿来就用，不废话。

---

## Step 1. 安装 xdev CLI

```bash
npm install -g @byted/xdex --registry https://bnpm.byted.org
```

验证：

```bash
xdev --version
```

> 后续 xdev 启动时会自动检查升级，不用管。

---

## Step 1.X. Bonus

### 1. 安装最新版本 Lark CLI

装了这个，Agent 里可以直接操飞书文档、日历、消息等，无需每周手动 refresh token。

```bash
xdev lark-mcp mac setup
```

按提示走完 5 步即可：

```
ℹ️  xdev lark-mcp mac setup

Step 1/5  检查 lark-cli
  ✅ 已安装 v1.0.6 (/opt/homebrew/bin/lark-cli)

Step 2/5  飞书应用配置
  ⚠️  未配置应用
  👉 请运行：lark-cli config init --new
     该命令会输出授权链接，在飞书中打开链接完成应用创建
     完成后重新运行 xdev lark-mcp mac setup

Step 3/5  用户授权
  ⚠️  未授权
  👉 请运行：lark-cli auth login
     该命令会输出 OAuth 链接，在飞书中确认即可

Step 4/5  部署 MCP Server
  ✅ ~/.xdev/lark-mcp-server/index.mjs

Step 5/5  注册到 Claude Desktop 配置
  ✅ 写入 ~/Library/Application Support/Claude/claude_desktop_config.json

✅ lark-cli MCP Server 配置完成！
👉 请完全退出 Claude Desktop App（Cmd+Q），然后重新打开
```

> 中间步骤中断了没关系，按提示跑完对应命令后重新 `xdev lark-mcp mac setup`，已完成的步骤会自动跳过。

---

### 2. 配置 Claude Settings

把作者维护的 `~/.claude/settings.json` 模板**增量合并**到本机（已有字段绝不覆盖；只把模板里有、本地没有的字段补进去）。模板主要包含：

- **AI 代码统计上报**（PostToolUse / Stop hooks → TEA `caribou`）
- 默认 **effort=max** + **`model: opus[1m]`**（深度推理 + 1M context）
- 常用 **Bash + MCP 工具白名单**（go test/build/vet/doc/get、mcp__fetch、brave-search、ide diagnostics 等）
- **中文输出** + `alwaysThinkingEnabled` + `autoUpdatesChannel: latest`
- `skipDangerousModePermissionPrompt: true`

```bash
xdev claude-settings mac setup
```

也可以先 `--dry-run` 预览到底要新增哪些字段，确认无误再真跑：

```bash
xdev claude-settings mac setup --dry-run
```

输出示例：

```
[xdev] claude-settings mac setup
  模板来源: /opt/homebrew/lib/node_modules/@byted/xdex/marketplace/plugins/xdev/claude-settings/settings.template.json
  目标文件: /Users/you/.claude/settings.json (已存在)
  合并差异 (10 项):
     + env.CLAUDE_CODE_ATTRIBUTION_HEADER = "0"
     + env.CLAUDE_CODE_EFFORT_LEVEL = "max"
     + permissions.allow: 9 项新增
     + permissions.defaultMode = "default"
     + alwaysThinkingEnabled = true
     + autoUpdatesChannel = "latest"
     + skipDangerousModePermissionPrompt = true
     + hooks.PostToolUse: 1 个 hook 新增
     + hooks.Stop: 1 个 hook 新增
     + enabledPlugins.gopls-lsp@claude-plugins-official = true
  ✅ 备份: /Users/you/.claude/settings.json.xdev-bak.2026-04-24T06-39-21-181Z
  ✅ 已写入: /Users/you/.claude/settings.json
```

> 写入前会自动备份原文件到 `~/.claude/settings.json.xdev-bak.<timestamp>`，且仅在内容真有变化时才生成备份。二次跑（无变化场景）会输出「已是合并完成状态，无变化」并 no-op。

---

### 3. 安装最新版本 TMUX

一键 `brew install tmux`（如未装）+ 写入推荐 `~/.tmux.conf` 配置。配置内容主要是：

- **鼠标支持**（`set -g mouse on`）
- **鼠标拖选**自动同步复制到 **Mac 系统剪贴板**（`pbcopy`）
- **键盘复制**（`M-w` / `y` / `Enter`）也同步到 Mac 剪贴板

```bash
xdev tmux mac setup
```

输出示例：

```
[xdev] tmux mac setup — 4 steps
  [1/4] 检查 Homebrew...
        ✅ brew 已安装
  [2/4] 检查 tmux...
        ✅ tmux 3.6a
  [3/4] 写入 ~/.tmux.conf 的 XDEV-TMUX 段（幂等）...
        ✅ /Users/you/.tmux.conf（已新增 XDEV-TMUX 段）
  [4/4] 提示
        👉 已运行的 tmux session 请在其内执行 `tmux source ~/.tmux.conf` 让新配置生效
        👉 新开的 tmux session 自动生效
[xdev] tmux mac setup done.
```

> 配置用 `# XDEV-TMUX:START` / `# XDEV-TMUX:END` 段标记包裹，二次跑只替换段内，**不动你 `.tmux.conf` 段外的自定义内容**。

#### 常用快捷键（tmux 默认 prefix = `Ctrl+b`）

| 快捷键 | 作用 |
|--------|------|
| `tmux` | 启动一个新的 tmux session |
| `Ctrl+b` 然后 `"` | 水平分屏（上下分屏） |
| `Ctrl+b` 然后 `%` | 垂直分屏（左右分屏） |
| `Ctrl+b` 然后 `E` | 均匀分布当前所有 panes（常用于横向对齐） |

> 用法：先按下 `Ctrl+b` 松开，再按后续键。例如想左右分屏就是「`Ctrl+b` → 松开 → 按 `%`」。

---

## Step 2. 启动 xdev (会自动更新到最新版本)

```bash
cd /path/to/your-project    # 先 cd 到你的项目工程目录
xdev
```

终端输出：

```
[xdev] current: 0.0.12 · remote: 0.0.13  ↑ 发现新版本，正在自动升级…

Select coding agent:
  1) Trae CN (trae-cn)
  2) Trae (trae)
  3) Coco / TRAE CLI (coco)
  4) Claude Code (cc)
  5) Codex (cdx)
Choice [1/2/3/4/5]:
```

选完 agent 后 xdev 自动安装 plugin 并启动对应 agent，开干。

---

## Step 3. 使用 xdev 研发工具

### 3.1 需求开发


| #                           | Skill               | 适用场景                | 定位                                                 | 文档位置                                          | 输出文档数 |
| --------------------------- | ------------------- | ------------------- | -------------------------------------------------- | --------------------------------------------- | ----- |
| [选项1](#选项1-xdevexec-plan)   | `/xdev:exec-plan`   | 新手 / 小需求 / 敏捷       | 一个文档搞定需求对齐 + 方案决策 + 任务跟踪                           | `docs/xdev/exec-plan/{FEATURE}/`              | 1     |
| [选项2](#选项2-xdevopenspec)    | `/xdev:openspec`    | 新手 / 中小需求           | 轻量 SDD，断点续传 + 归档沉淀，archive 自动合并回主 specs            | `docs/xdev/openspec/changes/{name}/`          | 4     |
| [选项3](#选项3-xdevspeckit)     | `/xdev:speckit`     | 入门 / 中大需求 / 长流程     | 重量级 SDD 后端引擎（specify → design → dev）               | `docs/xdev/speckit/{FEATURE}/`                | 5+    |
| [选项4](#选项4-xdevce)          | `/xdev:ce`          | 入门 / 知识复利导向         | Compound Engineering，每次产出让下一次更容易，80% 规划复盘 20% 执行   | `docs/xdev/ce/{solutions,plans,brainstorms}/` | 3+    |
| [选项5](#选项5-xdevsuperpowers) | `/xdev:superpowers` | 熟练（新手只用 brainstorm） | 通用 Agent 工作流（brainstorm → plan → execute → review） | `docs/xdev/superpowers/specs/`                | 2     |
| [选项6](#选项6-xdevgstack)      | `/xdev:gstack`      | 高手 / OPC 全角色        | AI 扮演 CEO/RD/Designer/QA/CSO，研发工具的最高级形式            | `docs/xdev/gstack/{FEATURE}/`                 | 6+    |


### 3.2 端到端测试 & UI 还原


| #                             | Skill                 | 定位                          | 文档位置                |
| ----------------------------- | --------------------- | --------------------------- | ------------------- |
| [选项7](#选项7-xdevfixloop)       | `/xdev:fixloop`       | 自动测试修复循环，部署→测试→分析→修代码，循环到全过 | `.costudio/`        |
| [选项8](#选项8-xdevx-ui-workflow) | `/xdev:x-ui-workflow` | Figma 设计稿→代码实现→还原度校验        | 工作目录下按 page_name 组织 |


---

#### 选项1. /xdev:exec-plan

敏捷开发。只生成一个 `exec-plan.md`，同时做需求对齐、方案决策、任务跟踪。


| with action                    | 说明                           | 前置条件              |
| ------------------------------ | ---------------------------- | ----------------- |
| `/xdev:exec-plan [需求描述(可含链接)]` | 调研代码库 → 生成执行计划 → 审阅 → 逐里程碑实施 | 无                 |
| `/xdev:exec-plan`              | 续接当前分支已有 feature             | 已有 `exec-plan.md` |


---

#### 选项2. /xdev:openspec

轻量级 spec-driven 流程。支持断点续传，**完成后 `archive` 自动将变更 delta merge 回主 specs**，沉淀为仓库持久知识。


| with action                            | 说明                                             | 前置条件          |
| -------------------------------------- | ---------------------------------------------- | ------------- |
| `/xdev:openspec run [需求描述]`            | 全流程自动推进：propose → review → apply → archive     | 无             |
| `/xdev:openspec run`                   | 断点续传，列出活跃 change 续接                            | 已有进行中的 change |
| `/xdev:openspec propose [需求描述]`        | 生成完整 change（proposal + specs + design + tasks） | 无             |
| `/xdev:openspec explore [topic]`       | 探索性对话，不创建产物                                    | 无             |
| `/xdev:openspec apply [change-name]`   | 按 tasks.md 实施代码                                | propose 完成    |
| `/xdev:openspec archive [change-name]` | 归档：delta merge 回主 specs，沉淀为仓库知识                | apply 完成      |


```
/xdev:openspec run 加一个 dark mode 切换
```

---

#### 选项3. /xdev:speckit

SDD 后端引擎。重量级规格驱动全流程。


| with action                                       | 说明                                                                                     | 前置条件                  |
| ------------------------------------------------- | -------------------------------------------------------------------------------------- | --------------------- |
| `/xdev:speckit run [需求描述(可含链接)]`                  | 全流程自动推进：specify → review-spec → tech-guidance → tech-design → review-tech-design → dev | 无                     |
| `/xdev:speckit specify [需求描述(可含链接)]`              | 生成规格说明书                                                                                | 无                     |
| `/xdev:speckit review-spec [FEATURE_NAME]`        | 审阅规格（交互式）                                                                              | specify 完成            |
| `/xdev:speckit tech-guidance [FEATURE_NAME]`      | 补充技术指导                                                                                 | review-spec 完成        |
| `/xdev:speckit tech-design [FEATURE_NAME]`        | 生成技术方案                                                                                 | tech-guidance 完成      |
| `/xdev:speckit review-tech-design [FEATURE_NAME]` | 审阅技术方案（交互式）                                                                            | tech-design 完成        |
| `/xdev:speckit dev [FEATURE_NAME]`                | 任务拆分 + 逐 Phase 开发                                                                      | review-tech-design 完成 |


```
/xdev:speckit run 实现用户积分兑换功能，支持积分查询、兑换下单、兑换记录查询
```

---

#### 选项4. /xdev:ce

Compound Engineering（复利工程）。上游：EveryInc/compound-engineering-plugin。核心理念：**每次工程产出都让下一次更容易**，80% 规划复盘、20% 执行，通过 `compound` 步骤把解决方案沉淀为可检索的团队知识。


| with action                  | 说明                                   | 前置条件          |
| ---------------------------- | ------------------------------------ | ------------- |
| `/xdev:ce brainstorm [需求描述]` | 交互式 Q&A 探索需求和方案                      | 无             |
| `/xdev:ce plan`              | 将需求转化为结构化实施计划                        | brainstorm 完成 |
| `/xdev:ce work`              | 按计划执行（worktree 并行开发 + 任务跟踪）          | plan 完成       |
| `/xdev:ce review`            | 多 Agent 分层 Code review（安全/性能/架构/正确性） | work 完成       |
| `/xdev:ce compound`          | 沉淀本次解决方案为可检索知识，更新 AGENTS.md          | review 完成     |
| `/xdev:ce ideate`            | 主动发现代码库中的高价值改进点                      | 无             |


```
/xdev:ce brainstorm 重构支付模块的错误处理
```

---

#### 选项5. /xdev:superpowers

通用 Agent 工作流。上游：obra/superpowers。


| with action                           | 说明                           | 前置条件          |
| ------------------------------------- | ---------------------------- | ------------- |
| `/xdev:superpowers brainstorm [需求描述]` | 需求澄清 + spec 生成               | 无             |
| `/xdev:superpowers plan`              | 拆成 task 级 plan（TDD checkbox） | brainstorm 完成 |
| `/xdev:superpowers execute`           | subagent 派发执行（推荐）            | plan 完成       |
| `/xdev:superpowers execute-batch`     | 批量执行（无 subagent 备选）          | plan 完成       |
| `/xdev:superpowers tdd`               | RED-GREEN-REFACTOR 强约束       | plan 完成       |
| `/xdev:superpowers debug [问题描述]`      | 4 阶段根因调试                     | 无             |
| `/xdev:superpowers verify`            | 完成前必经验证                      | execute 完成    |
| `/xdev:superpowers review`            | Code review                  | 有代码变更         |
| `/xdev:superpowers parallel`          | 并行 subagent 编排               | plan 完成       |
| `/xdev:superpowers finish`            | 结束 feature 分支                | verify 完成     |


```
/xdev:superpowers brainstorm 用户登录流程优化
```

---

#### 选项6. /xdev:gstack

OPC（One Person Company）一人公司架构。AI 扮演 CEO / RD / Designer / QA / CSO 全角色，Sprint 全链路覆盖，**研发工具的最高级形式**。移植自 garrytan/gstack，33 个 action。


| with action                                       | 说明                                                         | 前置条件            |
| ------------------------------------------------- | ---------------------------------------------------------- | --------------- |
| `/xdev:gstack run [需求描述]`                         | 全流程自动推进：office-hours → autoplan → review → cso → qa → ship | 无               |
| `/xdev:gstack office-hours [需求]`                  | CEO 角色：产品构思（YC Office Hours 模式）                            | 无               |
| `/xdev:gstack autoplan [FEATURE_NAME]`            | 一键串联 CEO + 工程经理 + 设计师三审                                    | office-hours 完成 |
| `/xdev:gstack investigate [bug 描述]`               | RD 角色：调试专家根因分析                                             | 无               |
| `/xdev:gstack review [base_branch]`               | RD 角色：7 路并行 code review                                    | 有代码变更           |
| `/xdev:gstack cso [scope]`                        | CSO 角色：安全审计（OWASP + STRIDE）                                | 有代码变更           |
| `/xdev:gstack qa [url]`                           | QA 角色：真实浏览器 QA + 自动修复                                      | 服务已部署           |
| `/xdev:gstack ship`                               | RD 角色：发布全流程（test + review + push + PR）                     | qa 完成           |
| `/xdev:gstack retro [period]`                     | 工程经理角色：周复盘                                                 | 无               |
| `/xdev:gstack design-consultation [FEATURE_NAME]` | Designer 角色：从零构建设计系统                                       | 无               |
| `/xdev:gstack design-html [FEATURE_NAME]`         | Designer 角色：设计稿 → 生产级 HTML                                 | 有设计稿            |
| `/xdev:gstack land-and-deploy`                    | DevOps 角色：合并 PR + 等 CI + 部署 + 验证生产                         | ship 完成         |
| `/xdev:gstack canary`                             | DevOps 角色：部署后监控循环                                          | 已部署             |


```
/xdev:gstack run 实现多租户权限隔离
```

---

#### 选项7. /xdev:fixloop

自动测试修复循环。生成测试 → 部署 → 跑测试 → 分析失败 → 修代码，循环到全过。

**直接 `/xdev:fixloop` 不带参数即可，进入交互式引导逐个填写：**

```
> /xdev:fixloop

检测到以下上下文信息：
  BUSINESS_REPO = /home/user/backend（当前工作目录）
  BRANCH = feat/user-points（当前 git 分支）

请确认或修改以上推断，并补充以下缺失参数：

PSM（TCE 服务标识）？
> stone.cozeloop.prompt

TEST_REPO（测试代码仓库路径）？
> /home/user/api_test

SPEC_DIR（需求文档目录）？
> /home/user/specs

IDL_REPO（IDL 仓库路径）？
> /home/user/idl

IDL_BRANCH（IDL 变更分支）？
> feat/user-points

参数确认完毕，开始执行 Fix-Loop...
=== Stage 0: 测试生成 ===
...
```

**完整参数表：**


| 参数                | 必填  | 默认值     | 说明       |
| ----------------- | --- | ------- | -------- |
| `PSM`             | 是   | —       | TCE 服务标识 |
| `BRANCH`          | 是   | —       | 业务仓库分支   |
| `BUSINESS_REPO`   | 是   | —       | 业务代码仓库路径 |
| `TEST_REPO`       | 是   | —       | 测试代码仓库路径 |
| `SPEC_DIR`        | 是   | —       | 需求文档目录   |
| `IDL_REPO`        | 是   | —       | IDL 仓库路径 |
| `IDL_BRANCH`      | 是   | —       | IDL 变更分支 |
| `MAX_ITERATIONS`  | 否   | `10`    | 最大迭代轮次   |
| `GENERATE_TESTS`  | 否   | `true`  | 是否自动生成测试 |
| `SINGLE_TEST_RUN` | 否   | `false` | 只跑一次不修复  |
| `TCE_LANE`        | 否   | 自动创建    | 复用已有泳道   |
| `SKIP_DEPLOY`     | 否   | `false` | 跳过部署     |


> 必填参数会自动从对话上下文推断（如之前用过 speckit），推断不到的才交互式询问。

---

#### 选项8. /xdev:x-ui-workflow

Figma 设计稿到代码实现、组件文档、UI 修复的完整链路。前置：需设置 `FIGMA_ACCESS_TOKEN` 环境变量。


| with action                                                                          | 说明                     | 前置条件             |
| ------------------------------------------------------------------------------------ | ---------------------- | ---------------- |
| `/xdev:x-ui-workflow /design-from-figma <page_name> <figma_url>`                     | 从 Figma 串联全流程 + 生成组件文档 | 无                |
| `/xdev:x-ui-workflow /ui-analyze <page_name> <figma_url>`                            | 获取设计稿截图 + 高保真 JSX      | 无                |
| `/xdev:x-ui-workflow /ui-structure-and-coding <page_name> <figma_url> <image> <jsx>` | 结构分析 + 代码生成（高效模式）      | /ui-analyze 完成   |
| `/xdev:x-ui-workflow /ui-structure <page_name> <figma_url> <image> <jsx>`            | 仅 UI 结构分析              | /ui-analyze 完成   |
| `/xdev:x-ui-workflow /ui-coding <page_name>`                                         | 仅生成 UI 代码              | /ui-structure 完成 |
| `/xdev:x-ui-workflow /ui-fix "<需求描述>" <figma_url> <target_dir>`                      | 按设计稿修复现有代码             | 无                |
| `/xdev:x-ui-workflow /ui-fix-batch <page_name>`                                      | 批量并行校验所有组件还原度          | /ui-coding 完成    |
| `/xdev:x-ui-workflow /design-from-directory <reference_dir>`                         | 从已有目录补齐组件文档            | 无                |


```
/xdev:x-ui-workflow /design-from-figma my-page https://www.figma.com/design/xxxxx
```

---

## Step 4. Harness Engineering

给仓库装上工程实践基础设施（文档体系、hooks、lint、架构约束）。


| with action                  | 说明                                                            | 前置条件    |
| ---------------------------- | ------------------------------------------------------------- | ------- |
| `/xdev:harness init`         | 新仓库初始化（6 步：分析 → hooks → docs → AGENTS.md → ARCHITECTURE → 验证） | 无       |
| `/xdev:harness debt-fix`     | 技术债扫描修复（每次最多 3 项，逐 commit 验证）                                 | init 完成 |
| `/xdev:harness doc-fix`      | 以代码为准修复文档（文档/代码对齐）                                            | init 完成 |
| `/xdev:harness evolve`       | 知识演进健康检查（出报告 + 给建议，不改代码）                                      | init 完成 |
| `/xdev:harness lint-promote` | 把反复违反的文档规范升级成 lint 规则                                         | init 完成 |


```
/xdev:harness init
```

