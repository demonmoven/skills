---
name: exec-plan
description: "ExecPlan 执行计划引擎 - 为复杂任务创建、审阅和执行自包含的设计文档。适用于重大重构、跨模块改动、多小时级别的任务。"
argument-hint: "[需求描述或飞书文档链接]"
---

# ExecPlan 技能

ExecPlan（执行计划）是一种自包含的设计文档，让 AI Agent 能够有条理地完成需要数小时研究、设计和实现的复杂任务。它的核心价值在于：用户可以在 Agent 开始长时间编码之前审阅设计方案，确保方向正确。

## 用户输入

```text
$ARGUMENTS
```

## 执行流程

调用本 skill 后，按以下流程自动执行。**不需要子命令**——直接传入需求描述或不传参数（续接已有 feature）。

### Step 0：FEATURE_NAME 解析 + 环境准备

#### 有参数（新建 Feature）

1. **无意义检测**：判断 `$ARGUMENTS` 是否能理解为有意义的软件需求描述或文档链接。随机字符、乱码、过于笼统的描述（如"做个东西"）视为无意义，报错反问用户。URL 格式一律视为有意义。
2. **生成 FEATURE_NAME**：
   - 从用户输入中提取需求简要描述，翻译整理为 `{req_slug}`（最多 4 个单词，kebab-case，全小写）
   - 获取当前时间戳 `{YYYYMMDDHHmmss}`
   - `FEATURE_NAME = {YYYYMMDDHHmmss}-{req_slug}`
   - 示例：`20260325143000-user-login-optimization`
3. **解析工作区**（强制流程，必须严格按 resolve_workspace.md 执行）：
   
   ⚠️ **MUST**：本步骤必须读取并按 `<skill_dir>/prompts/resolve_workspace.md` 的指令**完整执行**（包括其中的 HARD-GATE 约束），**严禁 inline 跳过、优化或自行决策任何步骤**。
   
   特别地，**严禁**以下行为：
   - ❌ 不弹 `AskUserQuestion` 而自己假定用户的确认
   - ❌ 自行决定主仓库而不让用户选择
   - ❌ 跳过 push -u origin
   - ❌ 用 inline `git rev-parse` + `git branch --show-current` 假装走完了流程
   
   读取并执行 `<skill_dir>/prompts/resolve_workspace.md`，传入：
   ```text
   SKILL_KIND     = "exec-plan"
   SPECS_DIR_NAME = "exec-plan"
   FEATURE_NAME   = {FEATURE_NAME}
   IS_NEW_FEATURE = true
   ```
   获得 `PRIMARY_REPO` / `PRIMARY_BRANCH` / `WORKSPACE_REPOS`。

   resolve_workspace 内部行为（plan/execute 两阶段模型，单仓多仓走相同的交互流程）：
   - **plan 阶段**（不动 git）：
     - 当前目录是 git 仓库 → repos 列表 = `[当前仓库]`；非 git 仓库 → 扫描一级子目录得到 repos 列表
     - 一次性脏检查所有 repos（pathspec 排除 `.claude/`、`.trae/`、`.coco/`、`.codex/`），任意脏即报错终止
     - IS_NEW_FEATURE=true 时弹出 list 让用户**单次确认**所有仓库的当前分支即"起始基准分支"
     - 用户只能选「✅ 全部正确」或「❌ 终止流程」；选终止则报错并提示用户手动 `git checkout` 后重新运行
     - 多仓库时再交互选择主仓库；单仓库时自动选定
   - **execute 阶段**（统一执行 git 写操作）：
     - 在每个 repo 上 `git checkout -b FEATURE_NAME` 并 `push -u origin FEATURE_NAME`
4. **创建目录 + 写 feature.md**：
   ```bash
   mkdir -p {PRIMARY_REPO}/docs/xdev/exec-plan/{FEATURE_NAME}
   ```
   将用户原始输入写入 `{PRIMARY_REPO}/docs/xdev/exec-plan/{FEATURE_NAME}/feature.md`

#### 无参数（续接已有 Feature）

> 续接流程要求当前目录就是 git 仓库（即业务主仓库）。如果不是，报错提示用户 `cd` 进入对应仓库后再运行。

1. `git rev-parse --show-toplevel` → `PRIMARY_REPO`，失败则报错终止
2. 检查当前 git 分支名 `{branch}`：
   - 优先检查 `{PRIMARY_REPO}/docs/xdev/exec-plan/{branch}/` 目录是否存在 → `FEATURE_NAME = {branch}`
   - **兼容期 fallback**：若新目录不存在但旧的 `{PRIMARY_REPO}/exec-plan/{branch}/` 存在，提示用户运行 `mv exec-plan docs/xdev/exec-plan` 迁移后重试
3. 否则扫描 `{PRIMARY_REPO}/docs/xdev/exec-plan/` 目录（兼容期内同时也扫旧的 `{PRIMARY_REPO}/exec-plan/` 列出旧 feature 并提示用户迁移），列出已有 feature 目录名，交互式选择
4. 若 `{PRIMARY_REPO}/docs/xdev/exec-plan/` 与旧的 `{PRIMARY_REPO}/exec-plan/` 都为空 → 提示用户输入需求描述

#### 推导变量

> 新建场景下 `CWD = {PRIMARY_REPO}`（来自 resolve_workspace）；续接场景下 `CWD = {PRIMARY_REPO}`（来自 git rev-parse --show-toplevel）。

```text
CWD = {PRIMARY_REPO}
SPECS_DIR = {CWD}/docs/xdev/exec-plan/{FEATURE_NAME}
EXEC_PLAN_DOC = {SPECS_DIR}/exec-plan.md
```

#### 断点续传检测

检测 `docs/xdev/exec-plan/{FEATURE_NAME}/` 下已有产物，判断从哪一步开始：

| 检查条件 | 从哪一步开始 |
|----------|-------------|
| 新建 feature | Step 1 |
| `exec-plan.md` 不存在 | Step 1 |
| `exec-plan.md` 存在但 Progress 全为 `[ ]` | Step 2（审阅） |
| `exec-plan.md` 存在且有 `[x]` 进度 | Step 3（续接执行） |

### Step 1：读取 URL 内容 + 深度调研 + 生成 exec-plan.md

#### 1a. URL 内容读取

读取 `feature.md` 获取用户原始输入，解析其中的链接：

1. **飞书文档链接**（`https://*.feishu.cn/docx/*`、`https://*.feishu.cn/wiki/*`、`https://*.feishu.cn/docs/*`、`https://*.larkoffice.com/docx/*`）：
   - 调用飞书 MCP 工具完整读取文档内容
   - 飞书 MCP 不可用 → 提示用户配置并**终止流程**
2. **其他 HTTP/HTTPS URL**：使用 WebFetch 工具完整读取
3. **纯文本**：直接保留

所有 URL 读取到的内容**直接记忆到当前上下文中**，不再持久化为 prd.md。

#### 1b. 创建 exec-plan.md

按照下方「模式一：编写（Authoring）」的完整流程，深度调研代码库并生成 `docs/xdev/exec-plan/{FEATURE_NAME}/exec-plan.md`。

### Step 2：暂停等待用户 Review

ExecPlan 创建完毕后，**必须立即停下来**，明确告知用户"计划已创建，请 review"，等待用户确认。用户可以在此阶段讨论修改设计（按「模式三：讨论」的流程处理）。

**禁止在用户确认前擅自开始任何实现工作。**

### Step 3：按里程碑逐步执行

用户确认后，按照「模式二：执行（Implementing）」的流程逐里程碑实施。

### Step 4：报告结果

```
ExecPlan 全流程完成

规格文档：docs/xdev/exec-plan/{FEATURE_NAME}/
产出文件：
  - feature.md       — 原始需求描述
  - exec-plan.md     — 执行计划（含 Progress、Decision Log、Outcomes）
  - 代码变更         — 已提交到本地分支
```

---

## 何时使用

以下场景应使用 ExecPlan：

- 任务涉及多个文件、多个模块的协调修改
- 预估实现时间超过 1 小时
- 有显著的技术不确定性需要先研究验证
- 用户明确要求"先规划再动手"
- 需要原型验证（spike/POC）后再全面实现
- 从已有的 ExecPlan 文件恢复并继续工作

## Agent 能力边界

执行 ExecPlan 的 Agent 具有以下能力：列文件、读文件、搜索代码、运行项目、跑测试。

但 Agent 存在三个根本限制，ExecPlan 的编写必须考虑这些限制：

- **不知道任何先前上下文**：每次启动都是从零开始，之前的对话、推理过程全都不存在。因此 ExecPlan 必须把所有前提、假设、决策理由全部写进去。
- **无法从之前的 milestone 推断意图**：不能说"如前所述"或"按之前的方式处理"，必须在每个需要的地方重复声明假设和约定。
- **看不到外部文档**：不要指向外部博客或文档链接。如果某个知识是必需的，用自己的话嵌入到计划里。

## 四种操作模式

### 模式一：编写（Authoring）

当需要**创建新的 ExecPlan** 时使用。

流程：
1. **读取仓库文档体系**：从项目根目录的 `AGENTS.md` 开始，理解仓库的文档层级结构（L0–L4）、知识导航表、各模块入口文档。然后按需读取相关代码文件，理解仓库结构
2. **记录当前 Git 分支和 commit SHA**：运行 `git rev-parse --abbrev-ref HEAD` 和 `git rev-parse HEAD` 获取当前分支名和完整 commit SHA，填入 ExecPlan 的元信息区
3. **确认用户本地时区并统一时间戳格式**：优先使用用户明确指定的时区；若用户未指定，则使用仓库/会话约定的本地时区。后续 Progress 与日志时间戳必须统一使用该时区，禁止混用 UTC 与本地时区。
4. 从下方的骨架模板开始，逐步填充
5. 深入研究源码，确保每个细节准确
6. 将完成的 ExecPlan 保存为 `docs/xdev/exec-plan/{FEATURE_NAME}/exec-plan.md`
7. **评估文档更新需求**：基于对仓库文档体系的理解，评估本次任务是否涉及新增模块、新增 API、架构变更、新增操作流程等需要更新仓库文档的内容。如有必要，在 Plan of Work 中将文档更新规划为**最后一个里程碑**（详见"文档更新里程碑"章节）
8. **⚠️ 主动暂停，等待用户 Review**：保存 ExecPlan 后，必须立即停下来，明确告知用户"计划已创建，请 review"，并等待用户明确确认无误后才可进入执行模式。**禁止在用户确认前擅自开始任何实现工作。**

编写原则：
- 完全自包含：一个对仓库毫无了解的新手，仅凭 ExecPlan 就能完成全部实现
- 以用户可观察的行为描述验收标准，例如"启动服务后访问 /health 返回 HTTP 200"，而非"添加了 HealthCheck 结构体"
- 命名所有涉及的文件时使用仓库根目录的相对路径
- 对任何非日常英语的术语，在首次出现时立即用通俗语言解释
- 不引用外部博客或文档，如果需要某个知识点，用自己的话嵌入到计划中
- 用叙述性散文为主，仅在 Progress 章节使用 checkbox 列表
- 规划文档更新：评估任务完成后是否需要更新仓库文档（如 `AGENTS.md`、`ARCHITECTURE.md`、`docs/reference/`、`docs/guidance/` 或模块级 `AGENTS.md`）。如有必要，将文档更新安排为 Plan of Work 的最后一个里程碑，确保文档与代码改动保持同步

### 模式二：执行（Implementing）

当需要**按照已有的 ExecPlan 实施**时使用。

流程：
1. 读取 ExecPlan 文件，理解全部上下文
2. 按照里程碑顺序逐步推进，不要询问用户"下一步做什么"
3. **每完成一个里程碑，立即更新 ExecPlan 中的 Progress 章节**（与代码变更在同一轮 tool call 中完成）
4. 遇到歧义时自主决策，将决策记录到 Decision Log 章节
5. 发现意外行为时记录到 Surprises & Discoveries 章节
6. 频繁提交代码
7. 完成全部工作后撰写 Outcomes & Retrospective

关键要求：
- 不要停下来等用户指示，自主推进到下一个里程碑
- **⚠️ Progress 实时更新是硬性要求，不可延迟**：每完成一个 milestone 的代码变更或验证后，必须在同一轮 tool call 中更新 ExecPlan 的 Progress 章节（标记 `[x]` 并更新时间戳）。禁止将所有 milestone 的 Progress 更新攒到最后一次性完成。原因：ExecPlan 是恢复上下文的唯一真实来源（single source of truth），如果中途会话中断或上下文丢失，未实时更新的 Progress 会导致无法准确识别恢复点。
- 每个停顿点都必须更新 Progress，即使需要把一个部分完成的任务拆成"已完成"和"剩余"两项
- 所有章节保持同步更新，确保 ExecPlan 始终反映当前真实状态
- **⚠️ 完成全部里程碑后触发独立代码审计**：当所有功能实现和验证里程碑均完成后、撰写 Outcomes & Retrospective 之前，必须触发仓库中可用的独立代码审计技能（如基于其他 AI 模型的交叉审查 Skill），由不同模型对变更进行盲审，弥补实现者的认知盲区。如果审计发现经验证的真实问题，应在撰写 Retrospective 前修复。

### 模式三：讨论（Discussing）

当需要**修改已有 ExecPlan 的设计**时使用。

流程：
1. 读取现有 ExecPlan
2. 与用户讨论变更点
3. 将每个决策记录到 Decision Log，包含决策内容、理由和日期
4. 更新受影响的所有章节，确保自包含性不被破坏
5. 在文件末尾添加变更说明

### 模式四：调研（Researching）

当任务面临**重大技术未知数或方案不确定性**，需要先探路再动工时使用。调研模式的目标不是产出生产代码，而是产出**可运行的验证结果**和**明确的可行性结论**。

流程：
1. 读取 ExecPlan（如果已有）或创建一份以调研为主的 ExecPlan
2. **了解仓库文档体系**：从根目录的 `AGENTS.md` 开始，按知识导航表定位与调研主题相关的已有文档（如 `docs/reference/` 下的参考文档、`docs/guidance/` 下的操作手册、各模块的 `AGENTS.md` 入口）。已有文档是重要的调研输入，可以避免重复调研或遗漏已知信息
3. 用里程碑组织 proof-of-concept 和"玩具实现"（toy implementations），在正式动工前验证方案可行性
4. **读库的源码而非文档**：不要只看 README 或 API 文档，要真正读目标库的实现代码，理解其内部行为
5. **深度调研，不浅尝辄止**：深入到足以判断可行性的程度，不能草草得出结论
6. 写原型来引导正式实现——原型的目的不是交付，而是为后续完整实现提供经验和依据
7. 将调研结论（行/不行/有条件可行）记录到 Decision Log，附上验证证据
8. 如果调研结论支持继续推进，更新 ExecPlan 的 Plan of Work 和后续里程碑，融入调研中获得的知识

关键要求：
- 调研里程碑必须清楚标注范围为"调研/原型验证"，与正式实现里程碑区分
- 每个调研里程碑都要说明如何运行和观察结果，以及判定"行"还是"不行"的标准
- 调研产出的原型代码如果不打算合入正式实现，要在 ExecPlan 中明确标注为"仅供验证，后续清理"

## ExecPlan 必需章节

每个 ExecPlan 必须包含以下章节，缺一不可：

**Purpose / Big Picture** — 用几句话解释这个改动的意义：改完之后用户能做什么之前做不了的事，如何看到它在工作。

**Progress** — 用 checkbox 列表追踪颗粒度进度。每个条目都必须带**精确到秒的时间戳**，并且使用**用户本地时区**（格式：`YYYY-MM-DD HH:MM:SS±HH:MM`，例如 `2026-03-14 13:34:59+08:00`）。包括初始创建的待办项（使用创建 ExecPlan 时的时间戳）。这是唯一允许使用 checkbox 的章节。每个停顿点必须更新。

**Surprises & Discoveries** — 记录实施中发现的意外行为、bug、性能权衡或洞察。格式为"观察 + 证据"。

**Decision Log** — 记录每个设计决策，格式为"决策 + 理由 + 日期/作者"。

**Outcomes & Retrospective** — 在主要里程碑或完成时总结成果、差距和经验教训。

**Context and Orientation** — 描述与任务相关的仓库当前状态，假设读者什么都不知道。列出关键文件的完整路径。

**Plan of Work** — 用散文描述编辑和新增的顺序。对每个编辑，指明文件、位置（函数/模块）以及要插入或修改的内容。**如果任务涉及新增模块、新增 API、架构变更、新增操作流程等，必须将文档更新规划为最后一个里程碑**（详见"文档更新里程碑"章节）。

**Concrete Steps** — 列出精确的命令及其工作目录。当命令产生输出时，展示预期的简短输出片段。随工作推进更新。

**Validation and Acceptance** — 描述如何启动或验证系统。**⚠️ 验收标准必须包含刚性、可量化的指标，禁止仅用模糊的定性描述**。每个验收条件都必须是可自动化验证、有明确通过/失败判定的。典型的刚性验收指标包括但不限于：
- 单元测试通过率（例如：`go test ./... 全部 PASS，0 failures，0 skipped`）
- 代码覆盖率门禁（例如：`新增代码覆盖率 = 100%`，或 `backend/script/test_coverage.sh 通过`）
- 编译成功（例如：`go build ./... 零错误`、`tsc --noEmit 零错误`）
- Lint 检查通过（例如：`golangci-lint run 零 warning`、`biome check . 零错误`）
- API 行为断言（例如：`curl /api/v1/xxx 返回 HTTP 200 且 body 包含 {"status":"ok"}`）
- 性能基准（例如：`P99 延迟 < 200ms`、`QPS ≥ 1000`）

禁止出现"功能正常工作"、"页面能正确显示"等无法自动判定的验收描述。如果某项验收暂时无法量化（如 UI 视觉效果），必须明确标注为"人工验收"并说明具体的检查步骤和判断标准。

**⚠️ 被 Skip 的测试视同失败**：任何测试输出中出现 `SKIP`（含 `--- SKIP`、`t.Skip`）都必须视为异常信号，不能按"测试通过"处理。默认不要创建可以被 skip 的单测。如果 skip 的原因无法在当前环境解决（如缺少 API Key、外部服务不可达、测试数据不存在等），Agent 必须立即停止并向用户反馈以下信息：被跳过的测试名称、跳过原因、建议的修复方案（如提供环境变量、配置测试数据、改用 mock 等），由用户决定是否接受 skip 或要求修复。

**⚠️ 前端验证必须使用浏览器工具实际操作**：涉及前端 UI 的验收项，必须使用浏览器工具（如 Playwright MCP）打开页面进行实际操作验证，不能仅凭"编译通过"或"代码逻辑正确"就声称验收通过。验证流程：
1. 使用浏览器工具导航到目标页面
2. 按照验收标准中描述的操作步骤进行实际交互
3. 对关键验证结果截图，截图文件保存在 `docs/xdev/exec-plan/{FEATURE_NAME}/artifacts/` 目录下
4. 将截图路径以 Markdown 图片语法嵌入到 ExecPlan 的 Artifacts and Notes 章节中，标注对应的验收项

**⚠️ 截图落档细则（强制）**：
1. Artifacts and Notes 中必须写成”编号步骤 + 对应截图”的形式，步骤描述要具体到操作与结果（例如”点击设为默认 -> 卡片出现默认标识且开关不可关闭”），不能只贴图不写过程。
2. 每个关键步骤应对应**不同截图**，禁止同一张图反复复用冒充多步骤；若确需复用，必须在步骤中明确说明”复用同图，原因是仅验证同一状态”。
3. 图片链接必须使用**相对路径**，相对于 ExecPlan 文件位置编写，推荐使用 `artifacts/<file>.png`。
4. 截图文件名应可读且带顺序前缀，推荐 `xxx-flow-01-...png` 或 `xxx-step-01-...png`，便于按步骤追溯。
5. 写入 ExecPlan 前必须确认图片已真实落盘到 `docs/xdev/exec-plan/{FEATURE_NAME}/artifacts/` 且文件名与链接一致，避免”文档有链接但文件不存在”。
6. “充分体验改动”至少覆盖：主成功路径、关键状态变更、刷新后持久化（如适用）、关键保护/失败提示路径（如适用）。

**Idempotence and Recovery** — 说明步骤是否可以安全重复。如果有风险性操作，提供重试或回滚路径。

**Artifacts and Notes** — 包含最重要的终端输出、diff 或代码片段作为缩进示例。保持简洁。

其中前端任务的 Artifacts and Notes 必须包含“实操步骤清单 + 每步图片”，而不是仅罗列图片文件路径。

**Documentation Update** — 列出需要更新的仓库文档及更新内容。Agent 应在编写阶段基于对仓库文档体系（从根目录 `AGENTS.md` 的知识导航表出发）的理解，评估本次任务完成后哪些文档需要同步更新。典型需要更新的文档包括但不限于：`ARCHITECTURE.md`（架构变更时）、模块级 `AGENTS.md`（新增模块或变更模块职责时）、`docs/reference/`（新增 API 或参考资料时）、`docs/guidance/`（新增操作流程时）、根目录 `AGENTS.md` 的知识导航表（新增文档入口时）。如经评估确认无需更新文档，写明"已评估，无需文档更新"及理由。

**Interfaces and Dependencies** — 列出使用的库、模块和服务。指定必须存在的类型、接口和函数签名。

## 里程碑的撰写方式

当 Plan of Work 中的工作被拆分为多个阶段或阶段时，必须将每个阶段格式化为正式的里程碑章节，使用 `### Milestone N: <描述>` 作为标题。不要只是在散文中提到"第一阶段"、"第二阶段"——每个里程碑需要自己的小节，包含以下要素：

- 范围：这个里程碑做什么
- 成果：完成后会存在什么之前不存在的东西
- 命令：运行什么来验证
- 验收：**必须列出刚性量化指标**（如测试通过数、覆盖率百分比、编译零错误、lint 零告警等），而非"功能正常"等模糊描述

每个里程碑必须可以独立验证，并且增量地实现整体目标。把它写成一个故事：目标 → 工作 → 结果 → 证明。

### 原型里程碑

当任务有重大不确定性时，鼓励添加专门的原型里程碑来降低风险。例如：
- 验证第三方库是否满足需求的 spike
- 评比两种实现方案的对比实验
- 验证性能是否达标的基准测试

原型里程碑要明确标注范围为"原型验证"，描述运行和观察方法，并说明推进或放弃原型的判断标准。

### 文档更新里程碑

当任务涉及以下任一情形时，**必须**在所有功能实现和验证里程碑之后，添加一个专门的文档更新里程碑：

- 新增了模块、子目录或独立组件
- 新增或变更了 API 接口（含内部接口和外部接口）
- 变更了架构设计、模块边界或数据流
- 新增了操作流程、部署步骤或配置项
- 引入了新的第三方依赖或工具链
- 变更了现有文档中描述的行为

文档更新里程碑应包含以下内容：

- 范围：列出需要更新的具体文档文件（如 `ARCHITECTURE.md`、`docs/reference/xxx.md`、模块级 `AGENTS.md` 等）
- 更新内容：对每个文档，说明需要新增、修改或删除的具体内容
- 验收：更新后的文档与代码实际行为一致，无过时描述，新增内容在相应的知识导航表（`AGENTS.md` 的"知识导航"章节）中可达

如果经评估确认任务不涉及上述任何情形（如纯 bug 修复、内部重构且不改变外部接口），可以不添加文档更新里程碑，但必须在 Decision Log 中记录"已评估，无需文档更新"及理由。

### 并行过渡

在大规模迁移或重构中，允许新旧两条路径并存一段时间，降低"一刀切"式迁移的风险。具体做法：

- 新的实现（如新 adapter、新模块）与旧实现同时存在于代码库中
- 测试同时覆盖新旧两条路径，确保旧路径不退化、新路径行为正确
- 在新路径完全验证通过后，再安全地下掉旧路径（在单独的里程碑中完成）
- ExecPlan 中要明确标注哪些里程碑是"并行共存"阶段，哪个里程碑负责"退役旧路径"

## 格式要求

- 当 ExecPlan 是一个独立的 `.md` 文件时，不需要外层的三反引号包裹
- 在 ExecPlan 内部展示命令、代码或输出时，使用缩进块而不要使用三反引号代码围栏（避免破坏文档结构）
- 每个标题后空两行
- 使用 `#`、`##` 等标准 Markdown 标题语法
- 以散文为主，避免大量列表和表格（Progress 章节除外）

## 修订纪律

**任何对 ExecPlan 的修改都必须遵守以下纪律，不仅限于"讨论模式"：**

1. **全面同步所有段落**：改了 Plan of Work，就必须检查 Progress 是否还准确、Decision Log 是否记录了变更原因、Concrete Steps 里的命令是否还适用、Validation and Acceptance 的验收标准是否需要更新。不能只改一个地方就完事。
2. **底部追加修改说明**：在 ExecPlan 文件末尾写一条修改记录，描述改了什么以及为什么改。格式建议：`[YYYY-MM-DD HH:MM:SS±HH:MM] 修改说明：<描述>`。
3. **保持自包含性**：修订后的 ExecPlan 仍然必须满足"自包含、新手友好"的标准。修订不能引入新的隐含依赖或未解释的术语。
4. **记录"做什么"更要记录"为什么"**：ExecPlan 不仅要描述 the what，还必须描述 the why——几乎所有内容都是如此。每个修订都要附带理由。

## 骨架模板

创建新 ExecPlan 时，从以下模板开始填充：

    # <简短的行动导向描述>

    本 ExecPlan 是一份活文档。Progress、Surprises & Discoveries、Decision Log 和 Outcomes & Retrospective 章节必须随工作推进持续更新。

    本文档遵循 ExecPlan 规范维护（路径：<在此填入 PLANS.md 或 SKILL.md 的仓库相对路径>）。

    **创建时代码基线：**
    - 分支：<填入 git rev-parse --abbrev-ref HEAD 的输出>
    - Commit SHA：<填入 git rev-parse HEAD 的完整 SHA>

    ## Purpose / Big Picture

    用几句话解释改动完成后用户能获得什么，以及如何看到它在工作。

    ## Progress

    - [x] (2025-01-01 10:00:00+08:00) 示例已完成步骤
    - [ ] (2025-01-01 10:00:00+08:00) 示例待完成步骤
    - [ ] (2025-01-01 10:00:00+08:00) 示例部分完成步骤（已完成：X；剩余：Y）

    ## Surprises & Discoveries

    - 观察：...
      证据：...

    ## Decision Log

    - 决策：...
      理由：...
      日期/作者：...

    ## Outcomes & Retrospective

    在主要里程碑或全部完成时填写。

    ## Context and Orientation

    描述与任务相关的仓库当前状态。列出关键文件完整路径。定义所有非显而易见的术语。

    ## Plan of Work

    用散文描述编辑和新增的顺序。

    ## Concrete Steps

    列出精确命令、工作目录和预期输出。

    ## Validation and Acceptance

    描述如何验证和观察系统行为。用具体的输入输出描述验收标准。

    ## Documentation Update

    列出需要更新的文档文件及更新内容。如经评估无需更新，写明"已评估，无需文档更新"及理由。

    ## Idempotence and Recovery

    说明步骤的可重复性和失败恢复路径。

    ## Artifacts and Notes

    关键的终端输出、diff 或代码片段。

    ## Interfaces and Dependencies

    使用的库、接口签名、类型定义。

## 文件存放与产出物

所有文档统一存放在 `docs/xdev/exec-plan/{FEATURE_NAME}/` 目录下，不做目录间移动：

```
docs/xdev/exec-plan/{FEATURE_NAME}/
├── feature.md              # 原始需求描述（用户输入原文）
├── exec-plan.md            # 执行计划（核心文档，含 13 个必需章节）
└── artifacts/              # 截图、产物等（如有）
```

ExecPlan 文件直接命名为 `exec-plan.md`，无需日期前缀。

## 跨计划引用规则

当新的 ExecPlan 需要引用之前的 ExecPlan 时，遵循以下规则：

- **已签入仓库的历史计划**（如 `exec-plan/` 下其他 feature 的 exec-plan.md）：可以直接引用其仓库相对路径，例如"参见 `exec-plan/20260310-xxx/exec-plan.md` 的 Context and Orientation 章节"。但引用的目的仅为"追溯上下文来源"，不能替代在当前计划中嵌入必要的上下文——当前计划仍然必须自包含。
- **未签入仓库的历史计划**（如存在于对话记录或临时文件中的计划）：必须将旧计划中所有相关的上下文**完整复制**到新计划中。不能说"参见之前的讨论"或"按上次的方案"。
- **一般原则**：引用历史计划是为了提供溯源和审计线索，而非减少新计划的内容量。新计划的自包含性永远高于引用的便利性。

## 常见陷阱

避免以下问题：

- 使用未定义的术语而不解释
- 描述功能时过于狭窄，导致代码编译通过但实际什么有用的事都没做
- 把关键决策推给读者，而不是在计划中自行做出决策并解释原因
- 引用"之前定义的"或"架构文档中的"内容而不直接嵌入
- 只写代码变更而忽略可观察的验证步骤
- 把里程碑写得过于简略，遗漏将来实现时可能至关重要的细节
- **攒批更新 Progress**：不要因为追求"效率"而把多个 milestone 的 Progress 更新推迟到最后一次性完成。每完成一个 milestone 就立即更新 Progress，这是 ExecPlan 作为活文档和容错恢复点的核心价值。如果中途会话中断，未更新的 Progress 会导致后续 Agent 无法准确识别恢复点，可能重复已完成的工作或遗漏未完成的工作。
- **时区混用**：Progress、Decision Log、里程碑进展中禁止混用 `Z`（UTC）和本地时区时间戳。默认按用户本地时区统一记录，除非用户明确要求 UTC。
- **写不可行的验证命令**：Validation 章节中的命令必须实际可独立运行。编写时要确认测试包中是否有纯单元测试——如果只有集成测试（依赖外部服务如数据库、Agent Hub），那 `go test` 命令不能标为"可独立运行"，必须明确标注前置依赖或使用其他验证方式（如仅 go build）。执行时也要在运行测试前检查前置依赖是否就绪。
- **创建带 Skip 的单测**：禁止默认创建包含 `t.Skip()` 的单测。如果测试依赖外部输入（API Key、测试数据、外部服务等），应优先使用 mock/stub 替代；如果确实无法 mock，必须向用户明确说明并获得许可后才可使用 `t.Skip()`。任何被 skip 的测试都不计入"测试通过"。
- **截图路径写错**：不要在 Markdown 图片中使用绝对文件系统路径，也不要混用临时目录路径；ExecPlan 中只保留相对路径（推荐 `artifacts/...`）。
- **步骤与截图脱节**：不要只写“见截图”。每张图必须有对应步骤说明，说明“做了什么、看到了什么、为何证明验收通过”。
- **截图重复凑数**：不要用同一张截图覆盖多个不同验收步骤，除非明确声明复用原因并且该步骤确实验证同一状态。
- **忽略文档更新**：不要认为"代码写完就算完事"。每个 ExecPlan 都必须评估是否需要更新仓库文档。新增模块没有对应的 `AGENTS.md` 入口、新增 API 没有参考文档、架构变更没有更新 `ARCHITECTURE.md`——这些都是常见的文档腐化源头。文档更新应作为最后一个里程碑，在所有功能实现和验证之后执行，确保文档内容反映代码的最终状态而非中间状态。
