---
name: gloop-quest-review
version: 2.0.0
description: "委托评审：法师检查剑士的交付物质量，给出通过/修改/拒绝三选一结论。典型触发：剑士 phase_done 交付完成后开始评审、返工轮次剑士修复后再次提交需要复审。不负责：动手改代码（剑士的活，法师只读）、执行阶段的上下文查询（走 gloop-self-awareness）。"
metadata:
  class: mage
  category: review
  kind: orchestration
  related_skills:
    - name: gloop-self-awareness
      type: related
      description: 读取 quest info / history 走 self-awareness
    - name: gloop-note-keeping
      type: related
      description: 评审中追加评审笔记
    - name: gloop-quest-execution
      type: related
      description: 剑士执行阶段的对应 skill，评审的输入来源
  requires:
    bins: ["gloop"]
    cliHelp: "gloop review --help"
---

# gloop-quest-review

法师侧「交付评审」视角：与 gloop 平台交互的 CLI syscall 协议。法师是验证者，
但仍然是 Gloop 的法师职业，不改变冒险者设定。

## Trigger Examples

| 类型 | 场景 | 说明 |
|---|---|---|
| 适用 | 剑士阶段结束，法师阶段启动 | 进入评审域 |
| 适用 | 返工轮次交付后再次评审 | 仍然是法师职责域 |
| 不适用 | 想修改工作区文件 | 法师工作区只读，修改不会被采纳 |
| 不适用 | 想在剑士执行阶段给建议 | 不在评审域，写入笔记即可 |

## CLI Contract

本 skill 覆盖委托评审阶段的 gloop CLI 命令，法师用于读取任务信息、运行验证命令、给出评审结论和记录笔记。

v0.4 对象语言：法师输出的是 Checker Review。评审结论会持久化为 ReviewReport；ReviewReport 应明确 verdict、confidence、checked_against、required_changes、residual_risks，并尽量引用 EvidenceRef。EvidenceRef 是验证依据，原始 trace 是取证兜底，不应替代结构化评审。

### CLI 主路径

法师通过 Bash 执行 `gloop ...` CLI。Gloop 支持的 agent 都具备 Bash 和 skill 能力，因此不向某类 agent 暴露额外的 native platform tool，避免 CLI agent 与 ACP/native-tool agent 能力面不一致。

| 目的 | CLI 形式 |
|---|---|
| 查看验证命令 | `gloop command list` |
| 运行 Go 测试 | `gloop command run go-test` |
| 加载本 skill 正文 | `gloop skill show gloop-quest-review` |
| 评审通过 | `gloop review pass --comment "总评" --score 9` |
| 要求返工 | `gloop review request-changes --comment "总评" --hints "修改建议"` |

### 命令一览

| 命令 | 用途 |
|---|---|
| `gloop quest info` | 读取任务背景（和剑士看到的 quest 信息一致） |
| `gloop command list` | 查看平台白名单内的验证命令列表 |
| `gloop command run <command_id>` | 运行指定的白名单验证命令 |
| `gloop review pass` | 评审通过 |
| `gloop review request-changes` | 评审需要修改 |
| `gloop review reject` | 评审拒绝 |
| `gloop note add` | 追加评审笔记 |

### 详细说明

#### gloop quest info

读取当前委托的背景信息，与剑士看到的 quest 信息一致。

**用法：**
```bash
gloop quest info
```

**参数：** 无参数

**返回：**

委托详情，包含 `id`、`status`、`type`、`query`、`rework_count`、`max_rework`、`workspace_mode`、`workspace_path`、`warrior_id`、`mage_id`、`created_by`、`review_hints` 等字段。

#### gloop command list

查看平台白名单内可用的验证命令列表。

**用法：**
```bash
gloop command list
```

**参数：** 无参数

**返回：**

白名单命令列表，每条包含 `command_id` 和命令说明。

#### gloop command run

运行指定的白名单验证命令。

**用法：**
```bash
gloop command run <command_id>
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `<command_id>` | 是 | string | 要运行的命令 ID，从 `gloop command list` 获取 |

**返回：**

命令执行结果。

**注意：**
- 该命令由平台 `mage_command_allowlist` 控制，不接受任意 shell 字符串
- 默认白名单覆盖本地测试、lint、静态检查、安全扫描、依赖审计、格式检查等验证命令
- 需要 `bytedcli` 等带登录态/网络/内部副作用的真实 E2E 命令时，必须由用户显式加入白名单

#### gloop review pass

给出评审通过结论。

**用法：**
```bash
gloop review pass --comment "<评审总评>" [--score <1-10>]
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `--comment` | 是 | string | 评审总评，Markdown 格式 |
| `--score` | 否 | int | 评分（1–10 整数），仅在 pass 时有语义，参与剑士经验计算 |

**返回：**

无（纯副作用命令）

**注意：**
- 是评审阶段结束信号：调用后本阶段立即终止，裁决驱动下一步状态流转；未调用则评审未完成
- 只有法师上下文能调用；剑士上下文调用会返回 `permission_denied`
- `verdict` 固定为 `pass`

#### gloop review request-changes

给出需要修改的评审结论。

**用法：**
```bash
gloop review request-changes --comment "<评审总评>" --hints "<修改建议>"
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `--comment` | 是 | string | 评审总评，Markdown 格式 |
| `--hints` | 是 | string | 修改建议，返工剑士的直接操作指南 |

**返回：**

无（纯副作用命令）

**注意：**
- 是评审阶段结束信号：调用后本阶段立即终止
- 只有法师上下文能调用；剑士上下文调用会返回 `permission_denied`
- `verdict` 固定为 `request_changes`
- `request_changes` 会令 `rework_count + 1`，达到 `max_rework` 后任务交由用户终审

#### gloop review reject

给出评审拒绝结论。

**用法：**
```bash
gloop review reject --comment "<拒绝原因>"
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `--comment` | 是 | string | 拒绝原因，Markdown 格式 |

**返回：**

无（纯副作用命令）

**注意：**
- 是评审阶段结束信号：调用后本阶段立即终止，任务交由用户终审
- 只有法师上下文能调用；剑士上下文调用会返回 `permission_denied`
- `verdict` 固定为 `reject`
- 后果与 `request_changes` 不同：reject 立即终止评审，不再有返工机会

#### gloop note add

追加评审笔记到事件流。

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
- 笔记是 append-only 的事件流，会保留在事件日志中，后续所有阶段可见

### review_* 字段契约

`review pass`、`review request-changes`、`review reject` 三个命令共享以下字段契约：

| 字段 | 必填条件 | 说明 |
|---|---|---|
| `verdict` | 必填 | 取值 `pass` / `request_changes` / `reject`，由子命令决定 |
| `comment` | 必填 | 评审总评，Markdown 格式 |
| `hints` | `verdict=request_changes` 时必填 | 修改建议 |
| `score` | 仅 `verdict=pass` 时有语义 | 1–10 整数，参与剑士经验计算 |

**注意：**
- 缺少必填字段时 CLI 返回结构化错误，不会提交结论
- `verdict / comment / hints / score` 字段的语义、权重、判断维度由 agent 决定；平台不做规范化，不做语义推断，只持久化并传递给后续阶段与用户终审
- ReviewReport 是用户阅读 Checker Review 的主要对象；如果你运行了验证命令或引用了具体产物，请在评审中写清 EvidenceRef / checked_against，而不是只说“看起来没问题”

### 通用约定

- CLI 是本地 agent 调用 gloop 平台能力的主路径
- 平台注入 `GLOOP_*` 环境变量，通常不需要手填 qid/sid
- 环境变量包括：`GLOOP_DATA_DIR`、`GLOOP_QUEST_ID`、`GLOOP_SESSION_ID`、`GLOOP_PHASE`、`GLOOP_ADVENTURER_CLASS`、`GLOOP_WORKSPACE_PATH`、`GLOOP_CONTEXT` 等
- 在 mage/read-only phase 中，`GLOOP_WORKSPACE_PATH` 和当前工作目录可能是 sandbox copy。法师可以读取和验证，但对工作区的直接写入只会留在 sandbox，阶段结束后会被捕获为 `mage_sandbox_writes_captured` 并丢弃。
- **Syscall 机制**：CLI syscall 优先通过 `GLOOP_*` 环境变量发现当前 quest/session；也会回退读取工作区 `.gloop/context.json`。评审结论写入 `.gloop/signals/<session_id>.json`，orchestrator 会在回合边界观察该信号并结束当前阶段
- **法师工作区只读**：文件修改不会进入交付流；对剑士工作区的任何改动都应该通过 hints 描述
- **白名单命令机制**：需要跑测试或端到端验证时，只能使用 `gloop command run <command_id>`，由平台 `mage_command_allowlist` 控制
- 只有法师上下文能调用 `review pass/request-changes/reject`；剑士上下文调用会返回 `permission_denied`
- `gloop review pass/request-changes/reject` 是评审阶段结束信号：调用后本阶段立即终止，裁决驱动下一步状态流转

## Discipline

### 评审原则（对抗式验证）

评审不是复述剑士总结；你需要做对抗式验证：**先假设剑士交付有问题，然后主动找证据证伪或证实。**

**核心原则**：
- **疑罪从有**：证据不足时，默认假设剑士的声明不成立
- **没发现问题 ≠ 没有问题**：不要因为没找到 bug 就给高分；要看你验证了多少
- **剑士的话是声明，不是证据**：剑士说"测试通过了"不算数，你要自己验证
- **未验证的维度要明确标注**：如果你没查某个方面，不要假设它没问题

### 标准评审流程

1. **识别任务画像**：从剑士交付中提取 `task_type` 和 `side_effect_level`，判断该用哪些维度评审
2. **列验收清单**：对照用户原始需求 + 验收标准，列出所有必须满足的点
3. **找证据**：查看文件 diff、跑白名单命令、检查实际产物，收集可信证据
4. **找反例**：主动想「什么情况下这个交付会出问题」，再验证是否真的有问题
5. **下结论**：基于证据（不是感觉）给出 verdict 和 score

### 分任务类型评审维度

#### code_change 类任务

| 维度 | 检查要点 |
|------|----------|
| **diff 质量** | 改动范围是否和任务描述匹配？有没有不该改的文件？有没有遗漏的必要修改？ |
| **测试** | 新增代码是否有对应测试？测试是在测核心逻辑还是走过场？边界条件覆盖了吗？错误路径测了吗？ |
| **构建** | 代码能编译通过吗？依赖引入是否合理？有没有引入不必要的依赖？ |
| **回归风险** | 改动会不会影响已有功能？有没有相关测试覆盖？ |
| **安全** | 有没有注入风险、敏感信息泄露、权限绕过？输入校验了吗？ |
| **可维护性** | 命名是否清晰？有没有注释？复杂度是否过高？有没有重复代码？错误处理完善吗？ |

**常见坑点**：
- 测试只测 happy path，不测错误路径和边界条件
- 错误处理被忽略（返回 nil 当没事、panic 代替错误返回）
- 并发安全问题（共享变量无锁、channel 泄漏）
- 硬编码魔法数字和配置常量
- 命名含糊（`doThing`、`handleData` 这种说了等于没说的名字）

#### research_analysis 类任务

| 维度 | 检查要点 |
|------|----------|
| **来源可靠性** | 数据/结论有没有来源？来源是一手还是二手？是不是官方/权威来源？有没有标注时间？ |
| **证据链** | 结论和论据之间有没有推理跳跃？有没有「因为 A 所以 B」但 A 推不出 B 的情况？ |
| **客观性** | 有没有偏向性？是否只列了支持结论的证据，忽略了反面证据？ |
| **置信度** | 结论有没有明确置信度？是事实、推断还是猜测？ |
| **时效性** | 数据/信息有没有过时？标注的时间是什么时候？ |

**常见坑点**：
- 把厂商宣传/白皮书当事实（benchmark 都是别人挑好的场景）
- 数字没有出处，拍脑袋
- 对比维度不对称（A 框架测的是吞吐量，B 框架测的是延迟）
- 忽略关键约束条件（成本、兼容性、团队熟悉度）

#### doc_design 类任务

| 维度 | 检查要点 |
|------|----------|
| **目标覆盖** | 文档是否回应了原始需求的全部要点？非目标是否明确？ |
| **逻辑一致性** | 文档各部分是否自洽？有没有前后矛盾？ |
| **遗漏约束** | 有没有被忽略的关键约束（时间、成本、合规、兼容性、安全）？ |
| **可执行性** | 读者看完能直接动手做吗？还是只有空泛概念？接口/数据结构够具体吗？ |
| **反例检验** | 能不能举出反例说明方案不成立？作者有没有回应这些反例？ |

#### ops_config 类任务

| 维度 | 检查要点 |
|------|----------|
| **变更范围** | 改了哪些配置/环境？影响面有多大？哪些服务/用户会受影响？ |
| **回滚** | 有没有回滚方案？回滚需要多久？回滚后数据会丢吗？回滚方案验证过吗？ |
| **权限** | 权限变更是否遵循最小权限原则？有没有越权风险？ |
| **副作用** | 会不会影响其他服务/用户？有没有外部依赖变更？ |
| **监控** | 变更后有没有办法观察效果？有没有告警？关键指标是什么？ |
| **灰度** | 是全量发布还是灰度？灰度策略是什么？回滚触发条件是什么？ |

**常见坑点**：
- 回滚方案没验证，纸上谈兵
- 监控指标缺失，出了问题不知道
- 权限开太大，图省事
- 忘记检查下游依赖会不会崩

#### general_action 类任务

| 维度 | 检查要点 |
|------|----------|
| **动作完成度** | 声明做了的事是否真的做了？有没有证据？ |
| **外部副作用** | 会不会影响系统外的东西（数据、用户、第三方服务）？ |
| **幂等性** | 同样的操作做两次会不会出问题？ |
| **记录充分性** | 做了什么、为什么做、结果如何，有没有记录清楚？ |

### 证据等级

从高到低：
1. **平台工具证据**（`platform_tool_evidence`）：Gloop 事件日志，可信事实
2. **工作区文件与 diff**：实际代码/文档内容，可直接验证
3. **白名单命令执行结果**：测试、lint、静态检查等的实际输出
4. **任务历史与笔记**：append-only 事件流，有时间戳
5. **剑士交付物**（`warrior_artifact`）：声明，非证据，需自行验证

### 评审输出规范

#### verdict 判定指南

- `pass`: 核心功能完整，主要维度都有证据支持，残余风险可控
- `request_changes`: 有明确可修复的问题，返工后可能通过（最常用）
- `reject`: 方向错误、严重缺陷、或与需求严重不符，需要重做

#### comment 结构化格式

comment 使用 Markdown 格式写入，前端会完整渲染。尽量遵循以下小节结构，
方便剑士理解和返工，也方便用户在详情页快速扫读。

**标准模板（推荐完整使用）：**

```markdown
### 已验证证据
- （你实际看到/验证了什么，不是剑士说什么）
- 每个证据点尽量附带来源：文件路径、命令输出、日志时间戳等

### 发现的问题
- （具体的失败路径、缺陷、遗漏，按严重程度排序）
- 每个问题说明影响范围和触发条件

### 评审置信度
（你对结论的把握程度，以及原因。一句话到一小段）

### 残余风险
- （即使通过后仍存在的未验证项、潜在问题）
- 说明风险等级和触发可能性
```

**可选小节（按需添加）：**

- `### 小问题 / 次要发现` — 不影响 verdict，但值得指出的细节
- `### 返工对照` — 返工轮次专用，逐条说明上一轮的问题修复了没有

> 注：以上是方法论层面的推荐格式，平台不做解析和强制。
> 你可以根据实际评审情况增减小节，但保持一致的三级标题风格
> 有助于提升可读性。

#### hints 写作原则

hints 是返工剑士的直接操作指南，格式越清晰，返工越高效。

**标准模板：**

```markdown
### 必改问题（按优先级排序）
1. **<问题简述>** — <具体位置和影响>
   - 修改建议：<怎么做>
   - 验收标准：<怎么验证修好了>

2. **<问题简述>** — ...
```

**返工轮次专用：** 在最前面加上一轮「修复对照」，让剑士一目了然。

```markdown
### 上轮修复对照
- ✅ <问题1>：已修复，验证通过
- ⚠️ <问题2>：部分修复，还存在...
- ❌ <问题3>：未修复，仍然...

### 本轮必改问题
...
```

写作要点：
- **具体、可操作、指向明确** — 不说「质量不高」，说「X 文件的 Y 函数缺少错误处理，Z 场景会 panic」
- **优先级从高到低排列** — 剑士按顺序改，先改最关键的
- **每条给出验收标准** — 剑士知道改到什么程度算好

#### score 参考标准（1-10 分）

| 分数 | 说明 |
|------|------|
| 9-10 | 优秀：超出预期，代码质量高，测试完善，几乎挑不出毛病 |
| 7-8 | 良好：核心功能完整，有一些小问题但不影响整体 |
| 5-6 | 及格：基本能用，但有明显缺陷，需要改进 |
| 3-4 | 较差：问题较多，核心功能可能有 bug |
| 1-2 | 很差：完全不合格，需要重做 |

> 注意：分数会影响剑士经验获取。严审/深入模式下请保守打分，不要放水。

### 不同强度下的评审策略

| 强度 | 策略 |
|------|------|
| quick | 快速扫主要维度，挑最明显的问题，2-3 个维度深入验证 |
| standard | 全面检查主要维度，核心断言要有证据支持 |
| deep | 逐项检查所有相关维度，关键断言必须交叉验证，至少 2 个验证角度 |
| adversarial | 假设交付物有问题，主动找漏洞、找盲区、找失败路径。至少从 3 个不同角度攻击/验证。没有充分证据不能给 pass。如果你在严审模式下给了 pass，意味着你愿意为这个交付的质量背书。 |

### 其他纪律

- `gloop review pass/request-changes/reject` 是**评审阶段结束信号**：
  调用后本阶段立即终止，裁决驱动下一步状态流转。未调用则评审未完成。
- 法师可以建议提高强度或追加预算，但不能自行扩大平台预算；预算升级必须由用户或
  automation policy 授权。
- 只有法师上下文能调用 `review pass/request-changes/reject`；剑士上下文调用会返回
  `permission_denied`。
- `verdict / comment / hints / score` 字段的语义、权重、判断维度由 agent 决定；
  平台不做规范化，不做语义推断，只持久化并传递给后续阶段与用户终审。
- `request_changes` 会令 `rework_count + 1`，达到 `max_rework` 后任务交由用户终审。
- `reject` 立即终止评审，任务交由用户终审；后果与 `request_changes` 不同。
- 笔记是 append-only 的事件流，会保留在事件日志中，后续所有阶段可见。

### 返工轮次特别提醒

- 先读上一轮的 review hints，确认剑士有没有针对性修复
- 不要每次都提新的问题（除非真的是新发现的），先验收上一轮的问题
- 如果连续 2 轮返工后问题还没修完，或者分数提升很小，考虑 reject 或降低通过标准
