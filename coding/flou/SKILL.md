---

description: 研发流程助手，旨在协同人和Agent，推动任务完成。管理研发流程、主动推动流程、持久化研发记忆
name: "flou"
version: 1.5.4
user-invocable: true
---

# 研发流程助手

**约束**：项目初始化允许读写记忆/配置文件，任务初始化/澄清/小结/归档阶段严禁操作代码，执行阶段按研发流程操作代码。

**路径说明**：本文档中提到的相对路径均相对于 flou skill 所在目录，如 `reference/interactive_flow.md`
**重要更新**：如果flou-cli --version版本早于2026-04-10 00:00:00，必须重新安装flou-cli。

## 前置条件

### 安装 CLI

```bash
bash -c "$(curl -fsSL https://tosv.byted.org/obj/ies-cs-rtf/flou/install.sh)"
flou-cli --version
```

## 目录结构

```
$FLOU_DIR = ${PROJECT_DIR}/.flou/
├── config.json          # 项目配置
├── plugins.lock.json    # 已安装插件锁文件
├── memory/              # 项目记忆
│   ├── runtime.yaml     # 运行时配置
│   └── flows/           # 自定义研发流程
├── cache/
└── tasks/               # 任务目录
    └── ${TASK-ID}/
        └── todo-flow.md # 任务进度文件
```

## 使用方式

```
/flou
/flou $ARGUMENTS
```

**⚠️ 严格约束：Flou 在工作流程阶段（任务初始化/澄清/小结/归档）禁止操作代码。只有在"项目初始化"阶段允许读取文件和写入记忆文件，在"执行"阶段加载研发流程后，才能按研发流程指导操作代码。**
Flou 所有文件统一存放在 `$FLOU_DIR` 目录下：

├── config.json          # 项目配置（技能维度、Agent列表、用户习惯）
│   ├── runtime.yaml     # 运行时配置（技术栈、依赖、PSM等）\
└── ${TASK-ID}/      # 任务工作目录
用户通过以下方式触发：

示例：

- `/flou`
- `/flou 设计一个新的缓存方案`


## Trae 项目 Hook 行为

Flou 支持通过 Trae 生命周期 Hook 自动识别并辅助 Flou 工作流。当前 Trae 仅支持项目维度 Hook，Flou 默认安装到当前项目的 `.trae/hooks.json`，不写入用户级 `~/.trae-cn/hooks.json`。

- `flou-cli hook install --scope project --agent trae --project-dir <项目目录>` 会幂等安装 `SessionStart`、`UserPromptSubmit`、`PreToolUse`、`PostToolUse`、`Stop` 全部事件。
- `project init --agent trae` 会尝试安装/更新项目 Hook；Agent 不需要、也不应执行 `flou-cli hook activate`。
- `SessionStart` 不识别用户输入，也不判断是否已调用 Flou skill；它只负责初始化 Hook 环境，并会在 stderr 以及 `.flou/trae-hooks/hook.log` 打印 `$TRAE_ENV_FILE` 实际路径和写入完成日志。随后通过 `$TRAE_ENV_FILE` 追加 `export FLOU_HOOK_INSTALLED=...`、`export FLOU_HOOK_SESSION_ID=...`、`export FLOU_HOOK_STATE_DIR=...`、`export FLOU_PROJECT_DIR=...`、`export FLOU_HOOK_DEFAULT_ENABLED=...` 等语句，确保后续 Hook 命令和 RunCommand 子进程可继承环境变量。若 Trae UI 未展示 stderr，以项目内 `.flou/trae-hooks/hook.log` 为排查入口；Hook 配置本身保持为 `flou-cli hook run`，不内联 `printf`/`echo` 调试逻辑。
- 后续事件由 `flou-cli hook run` 内部 dispatcher 读取环境变量与 session state，并按插件注册范围决定是否执行；`PreToolUse` 会对非 flou 且命中 `install` 内置 skill 列表的 Skill 调用按 skill 名称召回相关记忆，但不额外激活 Flou session；未进入 Flou 工作流时 `PostToolUse`、`Stop` 默认 no-op/allow stop。
- `UserPromptSubmit` 在识别 `/flou` 或明确 Flou 工作流请求后激活当前 session；`PostToolUse` 可记录工具/命令活动，`Stop` 会保守检查任务闭环。

## 能力范围

| 能力     | 文档                                                          |
| ------ | ----------------------------------------------------------- |
| 项目记忆管理 | [project-memory.md](references/project-memory.md)           |
| 流程管理   | [workflow-management.md](references/workflow-management.md) |
| 任务进度跟踪 | [todo-flow.md](references/todo-flow.md)                     |
| 技能检索   | [find-skills.md](references/find-skills.md)                 |
| 插件管理   | `flou-cli memory plugin add/add-preset/list`                |

## 工作流程

**必须首先读取** **[interactive\_flow.md](statics/interactive_flow.md)** **作为工作流程详细指引，严格按照该文件定义的阶段和门禁条件执行。**

## 记忆检索去重约定

为避免重复检索记忆关键词，执行阶段遵循以下约定；Trae Hook 激活 Flou session 后也会复用该缓存文件记录自动召回关键词：

- 关键词缓存文件：`$FLOU_DIR/tasks/${TASK-ID}/memory_fetch.json`
- 执行 `flou-cli memory recall` **前**先读取 `memory_fetch.json`，仅对**新增关键词**执行搜索
- 搜索结束后由 Agent 或 Flou Hook 更新 `memory_fetch.json` 中的 `keywords` 列表与 `updated_at`

**文件格式（仅关键词列表）**：

```json
{
  "task_id": "TASK-YYYYMMDD-xxx",
  "keywords": ["流程", "部署", "测试"],
  "updated_at": "YYYY-MM-DDTHH:MM:SSZ"
}
```

## 内置研发流程

| 流程         | 文件                                                | 适用场景                     |
| ---------- | ------------------------------------------------- | ------------------------ |
| SDD        | [sdd\_flow\_spec.md](statics/sdd_flow_spec.md)    | 标准开发流程                   |
| TDD        | [tdd\_flow.md](statics/tdd_flow.md)               | 测试驱动开发                   |
| SDD-Plus  | [sdd-plus-flow.md](statics/sdd-plus-flow.md)  | 复杂需求开发 |
| Hotfix     | [hotfix\_flow.md](statics/hotfix_flow.md)         | 线上紧急修复                   |
| Spike      | [spike\_flow.md](statics/spike_flow.md)           | 技术预研                     |
| Oncall     | [oncall\_flow.md](statics/oncall_flow.md)         | Oncall 问题处理              |
| Onboarding | [onboarding\_flow.md](statics/onboarding_flow.md) | 新人项目上手                   |
| TCC-Config | [tcc\_config\_flow.md](statics/tcc_config_flow.md)         | TCC配置类任务自助接入   |

**优先级**：插件/项目流程（`flou-cli memory recall "流程" --stage flow`）> 项目历史偏好（`flou-cli memory recall "流程"`）> 内置流程（`statics/`）

### 依赖技能

| 技能包      | 包含内容                                                                                                                                                                                                                                                                                       |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| bytedcli | bytedance-env, bytedance-log, bytedance-bits, bytedance-mcp, bytedance-overpass, bytedance-tcc, bytedance-bam                                                                                                                                                                              |
| gdpa-cli | bam-query                                                                                                                                                                                                                                                                                  |
| lark-cli | lark-im, lark-doc, lark-calendar, lark-mail, lark-base, lark-sheets, lark-task, lark-vc, lark-minutes, lark-contact, lark-wiki, lark-drive, lark-event, lark-openapi-explorer, lark-shared, lark-skill-maker, lark-whiteboard, lark-workflow-meeting-summary, lark-workflow-standup-report |
| 其他       | todo-flow, auto-test-case                                                                                                                                                                                                                                                                  |

## 流程扩展

### 创建自定义流程

1. 创建流程文件：`$FLOU_DIR/memory/flows/FLOW-<流程名>.md`
2. 归档到记忆：`flou-cli memory add '流程概述：<>, 文件地址：<文件相对路径>' --tags "项目流程"`

流程文件规范详见 [workflow\_specification\_rules.md](statics/workflow_specification_rules.md)

### 技能检索

```bash
flou-cli install find-skills "需要的能力描述"
```

详见 [find-skills.md](references/find-skills.md)

### 插件装载（MVP）

Flou CLI 现支持统一插件模型，首版兼容三种内容来源：

- `ttadk`：TTADK `Preset -> Plugin` 模式
- `flou_memory_git`：仓库根目录包含 `.flou/`，支持 `flow` 和 `memory`
- `skills_memory`：skills 风格知识仓，默认提供 `memory`

常用命令：

```bash
flou-cli memory plugin add <ref> --format auto
flou-cli memory plugin add-preset <preset>
flou-cli memory plugin scan-ttadk <preset> --output <dir>
flou-cli memory plugin list
flou-cli memory recall [query] --stage research
```

配置约定：

- 项目只保留 `memory.presets`
- 不再使用 `plugins.include` / `plugins.exclude`
- 旧项目若仍有 `plugins.presets`，CLI 会迁移到 `memory.presets`

装载后会生成：

- `.flou/plugins.lock.json`：归一化后的插件锁文件

TTADK 静态仓推荐先扫描成 Flou 记忆仓：

- `flou-cli memory plugin scan-ttadk ttadk/backend --output <dir>`
- 产物目录下会生成 `.flou/memory/archive.json`
- 每条归档记忆会记录 `source_plugin/source_path`
- 根目录 `plugin.json` 会记录 `source_repo/source_commit/source_preset`
- 可直接作为 `flou_memory_git` 插件再次安装或上传到 git 分发

运行时知识召回统一走：

- `flou-cli memory recall [query]`
- 支持 `--layers` / `--plugin` / `--stage` / `--limit`，其中 `--layers` 用于先限定插件范围
- 该命令只串联项目里已安装的插件记忆；对 TTADK 静态仓，推荐先 `scan-ttadk` 生成 `.flou` 记忆仓，再安装为 `flou_memory_git`

### 插件知识 Spark 调研编排

当执行阶段进入一个研发节点并通过 `flou-cli memory recall` 命中插件知识、flow 或 prompt 时，主 agent 必须按以下顺序使用这些结果：

1. **生成 Spark Prompt**：先把当前阶段、用户目标、召回到的 `flow/memory/prompt`、候选 `tags/layers/objects`、约束与期望输出组装成一段 Spark Prompt。
2. **子 agent 产出 JSON 计划**：如果运行环境提供 `spawn_agent` / `spawnAgent` 等子 agent 能力，必须先调用一个子 agent 执行 Spark Prompt，要求其只返回合法 JSON 调研计划；JSON 至少包含 `tasks[]`，每个 task 包含 `layer`、`objective`、`prompt`、`candidate_entries`、`constraints`。主 agent 负责校验 JSON，必要时只允许重试一次修正格式。
3. **按分层派发子 agent**：拿到 JSON 后，按 `tasks[].layer` 拆分；有子 agent 能力时，每个 layer/task 分别启动子 agent 执行对应 `prompt`，并等待所有必需子 agent 完成后再汇总。每个子 agent 消息必须把可执行任务放在开头，并明确说明仓库指令文件只是约束，不能作为答复本身。
4. **无子 agent 兜底**：如果当前环境没有子 agent 能力，主 agent 必须使用同一份 JSON 计划按 `tasks[]` 顺序串行执行，不允许跳过召回结果或静默省略分层。
5. **汇总与追溯**：最终输出必须汇总每层结论、引用的 plugin/entry、未确认事项；若可写事件日志，应记录 `research.plan.created`、`research.subagent.started/completed/failed` 或主 agent 串行兜底原因。

## 文档管理规范

### 文档输出要求

研发流程的**每个关键步骤**都需要输出文档供用户审阅，并在 todo-flow 文件中引用关联。

### 文档目录结构

```
$FLOU_DIR/tasks/${TASK_ID}/
├── todo-flow.md              # 任务进度文件（必须）
└── docs/                     # 文档目录
    ├── requirement.md        # 需求调研文档
    ├── requirement-confirm.md # 需求确认记录
    ├── design.md             # 技术方案文档
    ├── design-review.md      # 技术评审记录
    ├── development-tasks.md  # 开发任务文档
    ├── test-plan.md          # 测试计划文档
    ├── test-report.md        # 测试报告
    └── release-notes.md      # 发布说明
```

### 各阶段文档要求

| 阶段   | 必输文档                         | 可选文档 | 审阅要求   |
| ---- | ---------------------------- | ---- | ------ |
| 需求调研 | requirement.md               | -    | 必须用户确认 |
| 需求确认 | requirement-confirm.md       | -    | 记录确认历史 |
| 技术方案 | design.md                    | -    | 必须用户确认 |
| 方案评审 | design-review\.md            | -    | 记录评审意见 |
| 任务拆分 | development-tasks.md         | -    | 建议用户确认 |
| 测试阶段 | test-plan.md, test-report.md | -    | 记录测试结果 |
| 发布阶段 | release-notes.md             | -    | 建议用户确认 |

### 文档引用格式

在 todo-flow 文件的对应阶段中，必须引用关联文档：

```markdown
#### 需求调研
- [x] 理解需求目标
- [x] 明确验收标准
- [x] 记录需求文档
- [x] **文档引用**: [requirement.md](./docs/requirement.md)

**当前节点**: 需求调研 - 用户确认
**下一步**: 等待用户审阅需求文档
```

### 开发方案变更同步要求

当开发执行、排障（Troubleshooting）或测试反馈导致实现方案发生变化时，必须执行“三同步”：

1. **同步技术方案文档**：更新 `docs/design.md` 中的概要设计、详细设计、异常处理、实现计划、外部依赖等相关段落。
2. **同步需求调研文档**：更新 `docs/requirement.md` 中的需求范围、验收标准、验证结果、风险识别等受影响段落，说明方案变化对需求边界和验收口径的影响。
3. **同步 todo-flow 关键决策**：在 `todo-flow.md` 的“关键决策列表”记录变更原因、决策内容、影响范围、关联文档和确认状态。

若方案变更来自排障过程，记录应突出问题现象、根因摘要、被否决方案和最终选择，保证后续小结与归档可追溯。

### 文档模板规范

### SDD 调研 sub-agent 生成逻辑

需求调研阶段不再通过 `flou-cli sub-agent exec` 让 CLI 内部启动 agent。`gen` 也不再提供单个 `research` 输入包装；当前只公开两种 prompt 生成方式：`research-spark` 和 `review`。

- `flou-cli sub-agent gen research-spark "<query>"`：生成 research-spark 输入文本，保存为 `docs/research-spark.prompt.txt`，优先由当前运行时子 agent 执行后产出 `docs/research-spark-tasks.json`
- `flou-cli sub-agent gen review "<query>"`：生成 review 输入文本，用于审查已有调研/方案/代码风险
- `research-spark` 返回的 JSON 中，每个维度任务已携带可直接执行的 `prompt`；主 agent 应直接把这些 `prompt` 交给对应运行时子 agent，不再调用 `flou-cli sub-agent gen research` 二次包装
- 如果当前运行时没有子 agent 能力，回退到 `flou-cli sub-agent chat --provider opencode --input <prompt-file>` 执行 `research-spark.prompt.txt`；对 JSON 中各维度 `prompt`，主 agent 可直接写入临时 prompt 文件后调用 chat 兜底，并把输出保存为 `docs/research-*.raw.txt`
- `gen` 只生成输入文本，不代表任务已经执行；运行、并发、结果解析和 `requirement.md` 汇总都由主 agent 负责

#### 需求调研文档模板

```markdown
# 需求调研文档

## 1. 调研输入
- runtime.yaml: <path/to/.flou/memory/runtime.yaml>
- 原始需求描述
- 对话关键决策上下文

## 2. research-spark 拆解
- research-spark gen 命令
- research-spark 输入文本: ./research-spark.prompt.txt
- 子任务 JSON 文件: ./research-spark-tasks.json
- 子任务原始输出: ./research-spark-tasks.raw.txt（使用 chat 兜底时必填）
- 拆解维度: arch / business / project_memory

## 3. 多 sub-agent 调研汇总
| 维度 | summary | key_files | 状态 |
| --- | --- | --- | --- |
| arch |  |  |  |
| business |  |  |  |
| project_memory |  |  |  |

## 3.1 research 输入文本
- arch: ./research-arch.prompt.txt
- business: ./research-business.prompt.txt
- project_memory: ./research-project-memory.prompt.txt
- 原始输出: ./research-*.raw.txt

## 4. 需求背景
- 业务价值

## 5. 需求目标
- 核心目标
- 成功指标

## 6. 需求范围
- 涉及模块
- 影响范围
- 排除范围

## 7. 验收标准
- 功能验收标准
- 性能验收标准
- 安全验收标准

## 8. 验证结果
- 现象验证
- 数据验证
- 范围验证

## 9. 风险识别
- 技术风险
- 业务风险

## 10. 方案变更影响
- 变更来源（开发排障/测试反馈/用户调整）
- 对需求范围和验收标准的影响
- 需重新确认的事项

## 11. 待确认问题
- 暂无
```

#### 技术方案文档模板

```markdown
# 技术方案文档

## 1. 背景
## 2. 概要设计
## 3. 详细设计
### 3.1 接口设计
### 3.2 数据模型
### 3.3 核心流程
### 3.4 异常处理
## 4. 技术选型
## 5. 实现计划
## 6. 外部依赖
## 7. 方案变更记录
- 变更原因
- 调整后的设计
- 影响范围
- 关联需求调研段落
```

#### 确认记录文档模板

```markdown
# 确认记录

## 确认历史

### 确认轮次 1
- **时间**: YYYY-MM-DD HH:MM
- **反馈**: 
- **修改**: 
- **结果**: 待确认 / 已确认

## 最终确认
- **时间**: 
- **结果**: ✅ 已确认 / ❌ 需修改
```

***

## 行为准则

### 核心原则

1. **严格遵守阶段约束**：
   - 项目初始化阶段：**允许读取文件和写入记忆文件、配置文件**
   - 任务初始化/澄清/小结/归档阶段：**严禁操作代码**
   - 执行阶段：**按研发流程指导操作代码**
2. **先观察再执行**：获取运行事实 → 分析问题 → 确认分析 → 执行修改
3. **多能力组合**：根据流程节点自由组合测试、日志、接口等能力
4. **事实为依据**：每个阶段都要有运行事实支撑
5. **主动帮助用户**：提供上下文、引导操作、及时反馈
6. **文档驱动**：每个关键步骤输出文档，供用户审阅确认
7. **方案变更三同步**：开发或排障中一旦调整实现方案，必须同步更新 `docs/design.md`、`docs/requirement.md` 相关段落，并维护 `todo-flow.md` 的关键决策列表

### 主动推动机制

- **任务目标**：推动需求研发流程从开始到完成，达到用户满意、测试通过、可上线规划
- **推动策略**：定期检查进度、识别阻塞点、提醒推进、自动流转
- **用户介入**：无法获取的信息、无法自主操作的流程、需要用户决策的选项
- **行为规范**：主动推进、减少询问、持续执行，直到需要用户介入
- **意图沟通**：必看[output\_examples](statics/output_examples.md)

### 任务ID规则

格式：`TASK-YYYYMMDD-<需求名>`

### 进度文件

每个任务生成进度文件：`$FLOU_DIR/tasks/${TASK-ID}/todo-flow.md`

每次执行完一个节点后，必须更新此文件记录当前进度。

## Troubleshooting

### Skill/工具调用失败

当 skill 或工具调用失败时，按以下步骤排查：

1. **查询项目记忆**：使用 `flou-cli memory recall` 搜索历史解决方案
   ```bash
   flou-cli memory recall "当前任务关键词" --tags troubleshooting
   ```
2. **检索相关技能**：使用 `flou-cli install find-skills` 查找可用的技能
   ```bash
   flou-cli install find-skills "当前任务关键词"
   ```

### 开发方案排障与变更

开发任务中如因测试失败、日志定位、接口验证或依赖限制需要调整原方案，按以下顺序处理：

1. **先排障后改方案**：记录运行事实（失败用例、接口响应、logid、错误栈）并定位根因，避免仅凭猜测改动。
2. **评估方案影响**：判断是否影响需求范围、验收标准、接口契约、数据模型、测试策略或外部依赖。
3. **执行文档三同步**：
   - 更新 `docs/design.md` 的受影响设计段落和方案变更记录。
   - 更新 `docs/requirement.md` 的受影响需求范围、验收标准、验证结果或风险识别。
   - 更新 `todo-flow.md` 的“关键决策列表”，写明变更原因、最终决策、影响范围、关联文档和确认状态。
4. **再继续开发验证**：完成文档同步后再按新方案修改代码、补充测试并更新 todo-flow 进度。

### 没有可用 Skill

当遇到没有可用 skill 的场景：

1. **检索技能市场**：
   ```bash
   flou-cli install find-skills "需要的能力描述"
   ```
2. **安装缺失技能**：
   ```bash
   flou-cli install --skills <skill-name> --source <source-name>
   ```
3. **安装缺失 CLI 依赖**：
   ```bash
   flou-cli install --global-tools <tool-name>
   ```

## 违规检查清单

在执行过程中，必须遵守以下约束：

- [ ] **项目初始化阶段**：仅读取文件和写入记忆文件、配置文件，未修改业务代码
- [ ] **任务初始化阶段**：没有使用 Glob/Grep/Read 等工具搜索/读取代码
- [ ] **澄清阶段**：没有使用 Glob/Grep/Read 等工具搜索/读取代码
- [ ] **执行阶段**：仅在研发流程节点中操作代码，非节点期间不操作
- [ ] **小结阶段**：没有使用 Glob/Grep/Read 等工具搜索/读取代码
- [ ] **归档阶段**：未读取无关代码；如为多仓 worktree 模式，仅对用户确认的仓库执行必要的归档动作
