# openspec 用户使用手册

本文档面向 openspec skill 的使用者。

> openspec skill 是一个**纯 prompt 实现** 的 fluid 规格驱动开发引擎。早期版本曾依赖 [Fission-AI/OpenSpec](https://github.com/Fission-AI/OpenSpec) CLI,当前版本已重构为零外部依赖,所有工作流由 Claude Agent 直接执行。

---

## 一句话理解 openspec

openspec 是一个 **fluid 规格驱动开发引擎**:你只需给出需求描述,它自动完成 proposal → specs → design → tasks → 实施 → 归档全链路。与 speckit 的"重型多文件 spec"不同,openspec 强调**迭代 + 灵活 + 多 change 并行**;每个 change 是一个独立的 `docs/xdev/openspec/changes/<change-name>/` 目录。

---

## 快速开始

### 前置条件

- Git
- 在主仓库目录或包含多个 git 子仓库的工作区目录下启动 Claude Code

> **零运行时依赖**:本 skill 不再需要 `npm install -g @fission-ai/openspec`。所有逻辑由 Claude Agent + 本 skill 内置的 `templates/` 模板完成。

### 第一次使用

```
/openspec propose 加一个 dark mode 切换
```

这会自动:
1. 扫描工作区的 git 仓库(单仓库直通;多仓库让你选主仓库)
2. **L1 检测**:发现 `docs/xdev/openspec/` 不存在 → 自动创建 `{specs,changes}/` 骨架
3. **L2 检测**:解析需求生成 change-name(如 `add-dark-mode`),创建 change 目录
4. **agentic context**:自动 Read 仓库根的 README/AGENTS/CLAUDE.md/package.json 作为隐式背景
5. **L3 检测**:依次生成 4 个 artifacts(proposal / specs / design / tasks)

> 旧版需要先跑 `/openspec init` 创建骨架,现在 propose 自动处理,直接提需即可。

### 一键全流程(最推荐)

```
/openspec run 加一个 dark mode 切换
```

自动:propose → 等你审阅 → apply → archive。

### 中断后续接(断点续传)

```
/openspec run
```

不带参数 = 列出所有活跃 change 让你选一个续接,然后从中断点接力跑完(propose 跳过已存在 artifact、apply 接着没勾选的 task、archive 完成归档)。

### 分步使用

```
/openspec propose 加一个 dark mode 切换    # 生成 planning 文档
# (审阅 / 修改 proposal、design、tasks)
/openspec apply add-dark-mode               # 实施代码
/openspec archive add-dark-mode             # 归档 + 合并 delta specs
```

---

## 参数说明

```
/openspec <action> [参数]
```

| action | 参数 | 说明 |
|--------|------|------|
| `propose` | `[需求描述]` | 一步生成完整 change(自动 init + L3 续接)⭐ |
| `explore` | `[topic]` | 探索性对话 |
| `apply` | `[change-name]` | 实施 tasks(自动续接未勾选项) |
| `archive` | `[change-name]` | 归档(内置 delta merge) |
| `run` | `[需求描述]` | 全流程串联;不带参数 = 断点续传 ⭐ |

**所有参数均可省略**:openspec 会从对话上下文推断或交互式让你选择。

> **旧 actions 已合并**:`init` / `new` / `continue` / `ff` / `sync` / `verify` 已并入 `propose` / `archive` / `run`。
> 详见 §「故障排查」段的「旧命令的新位置」对照表。

---

## 多仓库工作区

如果你的开发场景是「一个目录下含多个 git 子仓库」(比如同时改前端 + 后端),openspec 也能处理:

```
~/workspace/my-project/
├── frontend/        # git 仓库 1
├── backend/         # git 仓库 2
└── docs/            # 不是 git 仓库
```

在 `~/workspace/my-project/` 下运行 `/openspec propose <需求>` 时,openspec 会:

1. 扫描一级子目录,列出所有 git 仓库(frontend / backend)
2. 让你逐个确认/修改每个仓库的"基准分支"
3. 让你选择**主仓库**(`docs/xdev/openspec/` 目录将创建在主仓库根目录下)
4. L1 自动 init + 后续生成 change 一气呵成

> 单仓库场景(在 `frontend/` 下直接跑)也走相同的交互流程:repos 列表只有 1 个元素,依然会弹出确认列表让你看到/修改基准分支,主仓库自动选定。

---

## 与 speckit / exec-plan 的区别

| 维度 | speckit | exec-plan | openspec |
|------|---------|-----------|----------|
| **定位** | 重型多文件 spec | 单文件 ExecPlan | fluid 多 change |
| **粒度** | feature 级 | 1-N 小时任务 | change 级 |
| **产物** | spec / plan / data-model / contracts / ... | 单个 exec-plan.md | proposal / specs / design / tasks |
| **HITL 节点** | review-spec / tech-guidance / review-tech-design | 计划 review | proposal review |
| **实现形态** | 纯 prompt + Claude Agent | 纯 prompt + Claude Agent | 纯 prompt + Claude Agent |
| **规格目录** | `docs/xdev/speckit/{FEATURE_NAME}/` | `docs/xdev/exec-plan/{FEATURE_NAME}/` | `docs/xdev/openspec/changes/{CHANGE_NAME}/` |

三者共用同一份「git 仓库扫描 + 主仓库选择」逻辑(`prompts/resolve_workspace.md`),所以多仓库工作区行为完全一致。

---

## 工作流编排(默认 spec-driven schema)

本 skill 内置 spec-driven schema,artifact 依赖图:

```
proposal           (no deps)         → proposal.md
   ↓
   ├─→ specs       (requires: proposal) → specs/<capability>/spec.md
   ├─→ design      (requires: proposal, optional) → design.md
   ↓
tasks              (requires: specs, design) → tasks.md
   ↓
apply              (requires: tasks)         → 实施代码 + 勾选 tasks.md
```

每个 artifact 的"生成指令"集中在 `prompts/artifact_instructions.md`(被 `actions/propose.md` 在 Step 6 调用),模板文件位于 `templates/`。

> **想要自定义 schema?** 直接在对话里告诉 Claude 新的 artifact 编排逻辑(例如"propose 之后我想多加一步 risk-assessment"),Claude 会现场按你的描述调整流程,不需要改 skill 代码。

---

## 常见场景

### Q: 我的需求其实有点模糊,怎么办?

先用 `/openspec explore`,跟 AI 聊清楚后再 `/openspec propose`。

### Q: propose 出来的方案我不满意?

直接说"不对,我要的是 X"。openspec 会更新 proposal / design / tasks 后再次暂停。

### Q: 如何中途停下来手动改 design.md?

apply 阶段如果发现 design 缺陷,openspec 会自动暂停。你可以手动改完后说"继续",apply 会从断点续上(扫描 tasks.md 中未勾选的项目)。

### Q: 多个 change 并行怎么办?

每个 change 是独立目录 `docs/xdev/openspec/changes/<name>/`。可以同时 propose 多个,然后挑一个 apply:

```
/openspec propose change-a
/openspec propose change-b
/openspec apply change-a
```

### Q: 已经用 speckit / exec-plan 怎么迁移?

不用迁移,三者可以并存。已有的 `docs/xdev/speckit/{FEATURE_NAME}/` 或 `docs/xdev/exec-plan/{FEATURE_NAME}/` 目录与 `docs/xdev/openspec/changes/<name>/` 互不冲突。

### Q: 我中断了想接着做完?

两种方式:

1. **断点续传(推荐)**:直接跑 `/openspec run`(不带参数)。run 会列出所有活跃 change 让你选,然后从中断点接力跑到归档。
   - propose 会跳过已存在的 artifact,只生成缺失的
   - apply 会接着 tasks.md 中没勾选的 task 跑
   - archive 自动 sync delta specs 后归档

2. **指定 change**:`/openspec propose <change-name>`,L3 检测会自动跳过已存在的 artifact、生成缺失的;之后再手动跑 apply / archive。

### Q: 我想分阶段审阅 artifact(先看 proposal、确认后再让生成 specs)?

直接对话里说就行:

> "先只生成 proposal,让我审一下。"

我会按你的节奏走,proposal 满意了你说"接着生成 specs",我就接着生成。**不需要**专属命令(旧 `continue` 已合并到 propose 的 L3 续接,但 artifact 级停顿审阅由对话控制)。

### Q: 我想给 propose 注入项目级 context(技术栈、测试规范等)?

不需要专门的 config 文件。propose 会自动 Read 仓库根的 `README.md` / `AGENTS.md` / `CLAUDE.md` / `package.json` 作为隐式背景;还想强调某些约束,直接对话里说"项目用 Vitest,单测覆盖 80%",我会带上。

> 旧版 `docs/xdev/openspec/config.yaml` 已废弃。如果你之前手写过这个文件,propose 会输出一次 warning 提示已废弃,但**不会报错**,也不会读取它。可以手动删除。

---

## 故障排查

### `Change not found`

检查 `{PRIMARY_REPO}/docs/xdev/openspec/changes/<name>/` 是否存在。或者用 `ls docs/xdev/openspec/changes/` 看活跃 change 名称。也可以直接 `/openspec run`(不带参数),会列出所有活跃 change 让你选。

### Apply 时遇到歧义

Claude 会自动暂停并反问。直接在对话里回复你的决定即可,apply 会从断点续上。

### 想从早期"上游 CLI 包装层"版本迁移过来

如果你的仓库下已经有一个 `openspec/` 目录(早期版本生成),建议执行:

```bash
mkdir -p docs/xdev
mv openspec docs/xdev/openspec
```

之后所有 `/openspec <action>` 调用都会读写 `docs/xdev/openspec/` 下的文件。

### 旧命令的新位置(actions 精简对照表)

下面这些命令在精简后的版本里**已经移除**,但你想做的事情依然能做:

| 旧命令 | 现在的做法 |
|--------|----------|
| `/openspec init` | 直接 `/openspec propose <需求>` — propose Step 2 自动 mkdir 仓库骨架 |
| `/openspec new <name>` | 直接 `/openspec propose <需求>` — propose 自动包含 new 的 mkdir |
| `/openspec continue <name>`(中断恢复) | `/openspec run`(不带参数)→ 列出活跃 change → 选续接 |
| `/openspec continue <name>`(单步审阅) | 对话里说"先只生成 proposal,让我审一下";"OK 接着生成 specs" |
| `/openspec ff <name>` | 直接 `/openspec propose <change-name>` — propose Step 6 默认就是循环生成所有 ready artifact |
| `/openspec sync <name>` | 直接 `/openspec archive <change-name>` — Step 5b 内置 delta merge |
| `/openspec verify <name>` | 对话里说"帮我对照 spec 检查代码,看有没有不一致的地方" |

### 旧版 `docs/xdev/openspec/config.yaml` 怎么办?

新版 propose **不再读这个文件**。如果你之前手写过,propose 会输出一次 warning 提示已废弃,但不会报错。

把里面的 context / rules 信息搬到仓库根的 `README.md` / `AGENTS.md` / `CLAUDE.md` 即可,propose 会自动 Read 这些文件作为隐式背景。然后可以手动删除旧的 `config.yaml`。

---

## 设计原则

- **fluid not rigid**:actions 可任意顺序,dependency 是 enabler 而非 gate
- **iterative not waterfall**:可在任意阶段回去改 artifact
- **easy not complex**:单仓与多仓走相同的工作区解析流程,单仓时主仓库自动选定,无需选择
- **brownfield friendly**:可在已有项目上无缝接入,不要求清白的 greenfield
- **zero runtime dependency**:不依赖外部 CLI,所有工作流由 Claude Agent + 本 skill 内置模板完成
- **zero config file**:不需要 `config.yaml`,propose 自动 Read 项目根的 README/AGENTS/CLAUDE.md 作为隐式背景
