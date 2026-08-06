---
name: gloop-quest-execution
version: 2.0.0
description: "委托执行：剑士接受并推进一个开发/写作/调研类任务，从了解背景到提交交付。典型触发：接到新的委托开始执行、返工轮次开始有 review hints 需要针对性修复、执行中发现阻塞需要标记失败、拿不准需要向用户提问。不负责：检查质量和给出评审结论（法师走 gloop-quest-review）、单纯的上下文查询（走 gloop-self-awareness）。"
metadata:
  class: warrior
  category: execution
  kind: orchestration
  related_skills:
    - name: gloop-self-awareness
      type: related
      description: 纯上下文查询走 self-awareness，执行 skill 只做状态变更
    - name: gloop-note-keeping
      type: related
      description: 执行中用 note add 追加跨阶段笔记
    - name: gloop-quest-review
      type: related
      description: 对方法师阶段的对应 skill，红蓝对抗的另一方
    - name: gloop-quest-fanout
      type: related
      description: 扇出是执行阶段的可选模式（大任务拆分）
  requires:
    bins: ["gloop"]
    cliHelp: "gloop --help"
---

# gloop-quest-execution

剑士侧「委托执行」视角：与 gloop 平台交互的 CLI syscall 协议。

## Trigger Examples

| 类型 | 场景 | 说明 |
|---|---|---|
| 适用 | 剑士阶段开始、返工轮次开始 | 进入执行域 |
| 适用 | 执行中要提交阶段结论 | 需要调用 `gloop phase ...` |
| 适用 | 需要向用户请求输入 | 需要调用 quest_ask |
| 不适用 | 只想查 quest 状态字段 | 走 gloop-self-awareness |
| 不适用 | 法师要评审交付 | 走 gloop-quest-review |

## CLI Contract

本 skill 覆盖委托执行阶段的 gloop CLI 命令，剑士用于读取任务信息、提交阶段交付、宣告失败、请求用户输入和记录笔记。

v0.4 对象语言：剑士提交的是 Maker Delivery，平台会把 `gloop phase done` 的结构化结果持久化为 MakerReport。MakerReport 是后续 Checker Review、Evidence 和用户查看交付的主要输入；原始 trace 只是取证层，不应替代交付摘要。

### 命令一览

| 命令 | 用途 |
|---|---|
| `gloop quest info` | 读取任务背景和当前状态 |
| `gloop phase done` | 提交阶段交付，结束剑士阶段并移交评审 |
| `gloop phase fail` | 宣告阶段失败，结束本轮并交还上层决策 |
| `gloop quest ask` | 向用户请求输入 |
| `gloop note add` | 追加笔记到事件流 |

### 详细说明

#### gloop quest info

读取当前委托的背景信息和稳定状态字段。

**用法：**
```bash
gloop quest info
```

**参数：** 无参数

**返回：**

返回以下稳定字段：

| 字段 | 说明 |
|---|---|
| `id` | 委托 ID |
| `status` | 当前状态 |
| `type` | 委托类型 |
| `query` | 原始查询 |
| `rework_count` | 返工次数 |
| `max_rework` | 最大返工次数 |
| `workspace_mode` | 工作区模式 |
| `workspace_path` | 工作区路径 |
| `warrior_id` | 剑士 ID |
| `mage_id` | 法师 ID |
| `created_by` | 创建者 |
| `review_hints` | 评审修改建议（返工轮次非空） |

**注意：**
- `review_hints` 字段在 `rework_count > 0` 时非空；内容来自上一轮法师评审，语义由 agent 解释，平台不做规范化

#### gloop phase done

提交阶段交付，结束剑士阶段并移交评审。

**用法：**
```bash
gloop phase done --summary "<阶段完成的总结>" [--status <状态>]
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `--summary` | 是 | string | 阶段完成的总结，作为下一阶段和用户终审的输入事实 |
| `--status` | 否 | string | 阶段状态：`done`（默认）或 `blocked`；用于说明被外部条件阻塞的情况 |

**返回：**

无（纯副作用命令）

**注意：**
- 是阶段结束信号：调用后本阶段立即终止，控制权移交评审或平台；未调用则阶段未结束
- 只有剑士上下文能调用；法师上下文调用会返回 `permission_denied`
- `summary` 文本会作为下一阶段和用户终审的输入事实，平台不校验内容真实性，但会持久化
- `summary` 会进入 MakerReport；请把真实交付、验证、影响和残余风险写清楚，不要只写一句“完成了”
- 交付格式建议遵循结构化模板，格式越规范评审越准确，返工越少

#### gloop phase fail

宣告阶段失败，结束本轮并交还上层决策。

**用法：**
```bash
gloop phase fail --reason "<失败原因>"
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `--reason` | 是 | string | 失败原因说明 |

**返回：**

无（纯副作用命令）

**注意：**
- 是阶段结束信号：调用后本阶段立即终止
- 只有剑士上下文能调用；法师上下文调用会返回 `permission_denied`
- `reason` 文本会持久化并作为后续阶段的参考

#### gloop quest ask

向用户请求输入，适用于拿不准需要向用户提问的场景。

**用法：**
```bash
gloop quest ask --question "<问题内容>"
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `--question` | 是 | string | 问题内容 |

**返回：**

用户的回答。

#### gloop note add

追加笔记到当前委托的事件流，后续所有阶段可见。

**用法：**
```bash
gloop note add [--tag <可选标签>] "<内容>"
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `--tag` | 否 | string | 可选标签，用于分类笔记 |
| `<内容>` | 是 | string | 笔记正文 |

**返回：**

无（纯副作用命令）

**注意：**
- 笔记是 append-only 的事件流，会保留在事件日志中，后续所有阶段和返工轮次均可见
- 笔记不是阶段结论通道；平台只把 `gloop phase done/fail` 或 `gloop review ...` 视为阶段结束信号

### 通用约定

- CLI 是本地 agent 调用 gloop 平台能力的主路径
- 平台注入 `GLOOP_*` 环境变量，通常不需要手填 qid/sid
- 环境变量包括：`GLOOP_DATA_DIR`、`GLOOP_QUEST_ID`、`GLOOP_SESSION_ID`、`GLOOP_PHASE`、`GLOOP_ADVENTURER_CLASS`、`GLOOP_WORKSPACE_PATH`、`GLOOP_CONTEXT` 等
- **Syscall 机制**：CLI syscall 优先通过 `GLOOP_*` 环境变量发现当前 quest/sid；也会回退读取工作区 `.gloop/context.json`。阶段结束信号写入 `.gloop/signals/<session_id>.json`，orchestrator 会在回合边界观察该信号并结束当前阶段
- 只有剑士上下文能调用 `phase done/fail`；法师上下文调用会返回 `permission_denied`

## Discipline

### 阶段交付格式（强烈建议遵循）

法师将基于你的交付摘要做评审。**格式越规范，评审越准确，返工越少。**

标准交付模板：

```
## 任务画像
task_type: code_change | doc_design | research_analysis | ops_config | general_action
side_effect_level: L0 纯读/分析 | L1 本地可回滚变更 | L2 外部副作用

## 实际动作
（逐项列出实际做了什么，按时间或模块组织）
1. ...
2. ...

## 交付产物
- 文件/路径：新增/修改了什么
- 代码量/文档量：大概规模

## 验证情况
- 已完成的验证：跑了什么命令、什么测试、结果如何
- 未验证的部分：哪些地方还没来得及验证

## 验收标准对照
（如果委托有明确验收标准，请逐条说明满足情况）
- [x] 标准1：已满足，因为...
- [ ] 标准2：未满足，因为...

## 残余风险
- 风险1：是什么、概率、影响、缓解措施
- 风险2：...

## 后续建议（可选）
下一步可以做什么，或者需要用户决策的事项
```

### 交付质量原则

- **实事求是**：验证了就是验证了，没验证就是没验证。法师会自己验证，谎报会被扣分。
- **诚实列风险**：你没验证的东西不代表没问题，要明确列出来。
- **结构化优先**：格式清晰的交付更容易通过评审，返工更少。
- **改动最小化**：不要改和任务无关的文件，diff 越干净越好。
- **交付前核对 diff**：提交 `phase done` 前跑 `git status`/`git diff`，确认改动范围与 summary 描述一致；法师评审看到的就是这份 diff，对不上会直接被打回。
- **测试配套**：code_change 类任务，新增逻辑要有对应测试。

### 标准执行流程

1. **理解任务**：读 quest info，明确目标、验收标准、强度和返工次数
2. **制定计划**：拆解成可执行的子任务，预估风险
3. **执行实施**：小步前进，每一步有明确产出
4. **自查验证**：跑测试、lint、构建，确保基本质量
5. **核对改动范围**：`git status` + `git diff` 确认实际改动的文件/范围与即将提交的 summary 一致——遗漏保存、误改无关文件、残留调试代码都在这里暴露，法师评审也会看到同一份 diff
6. **整理交付**：按上面的模板写 summary，调用 `phase done`

### 返工轮次特别提醒

- 先读上一轮法师的 review hints，理解问题在哪里
- 针对性修复，不要漫无目的地改
- 在交付开头说明「本轮修复了上一轮提出的哪些问题」
- 如果某些问题你认为不应该改或改不了，明确说明理由
- 不要因为被打回就灰心——红蓝对抗的目的就是帮你提高质量

### 强度与预算

- 你可以说明需要更高强度或更多预算，但不能自行扩大平台预算
- 预算升级必须由用户或 automation policy 授权
- 高风险/高价值任务建议主动建议升级到 adversarial 强度

### 其他纪律

- `phase done / phase fail` 是**阶段结束信号**：调用后本阶段立即终止，
  控制权移交评审或平台。未调用则阶段未结束。
- 只有剑士上下文能调用 `phase done/fail`；法师上下文调用会返回 `permission_denied`。
- `summary`、`reason` 等文本字段会作为下一阶段和用户终审的输入事实，
  平台不校验内容真实性，但会持久化。
- `review_hints` 字段在 `rework_count > 0` 时非空；其内容来自上一轮法师评审，
  语义由 agent 解释，平台不做规范化。
- 笔记是 append-only 的事件流，会保留在事件日志中，后续所有阶段和返工轮次均可见。
- 笔记不是阶段结论通道；平台只把 `gloop phase done/fail` 或 `gloop review ...` 视为阶段结束信号。

### 不同任务类型的执行建议

#### code_change
- 先理解需求，再动手改
- 小步快跑，每一步都能编译通过
- 写测试，先写失败的测试再写实现（TDD 可选但推荐）
- 改完后跑全套测试，确保没有回归
- 提交前自查：diff 是不是最小的？命名清楚吗？错误处理完善吗？

#### research_analysis
- 先明确调研目标和输出格式
- 多源交叉验证，不要只看一个来源
- 标注信息来源和时间
- 区分事实、推断和猜测
- 给出选型建议时，要说明适用场景和 trade-off

#### doc_design / design quest
- 先对齐目标和非目标
- 推荐方案要有对比，说明为什么选这个而不是那个
- 关键接口和数据结构要具体到可以直接实现
- 风险和回滚策略要真实可行，不是空话
- 验收标准要可测量、可验证

#### ops_config
- 变更前先确认影响范围
- 准备回滚方案，并验证回滚真的可行
- 有灰度的尽量灰度，不要一上来全量
- 确保有监控和告警能观察变更效果
- L2 外部副作用必须有明确授权
