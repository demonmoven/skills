<!-- @format -->

# Brainstorm 核心发现报告 — 模板

> 本模板用于阶段三 `skill_solution_alignment` 的输出 `workspace/artifacts/research/brainstorm_core_findings.md`。
>
> **核心边界**：本报告**不是技术方案**，而是 RD 写技术方案前的**决策辅助报告**。读者拿到后应该是「我现在更知道这个需求会触达哪些坑、Gap、散落点；可以下笔写方案了」，而**不是**「这个报告就是技术方案，我抄过去就行」。

---

## 写作约束（必读）

### ❌ 必须避免

- ❌ "我建议把 `foo.go::bar()` 第 N 行改成 ..." 类的代码改写指令
- ❌ 直接给 diff / 伪代码改动
- ❌ 罗列文件路径 + 行号当作"分析"（属于调试信息，应整合到流程图 + 清单）
- ❌ "已证实 / 待确认 / 调研索引" 等研究笔记结构（直接给结论，不给过程）
- ❌ 用字母 A/B/C 编号改动点（用功能点命名）

### ✅ 必须包含

- ✅ 需求功能点对应到链路上的"穿越点"（用 mermaid 高亮）
- ✅ 显式列出本需求触达的 SP / Libra / Tool / Block 散落点
- ✅ 显式对照 `Z.2 PRD vs 实现 Gap 表`，提醒 RD 可能踩坑的项
- ✅ "方案方向建议"只给方向（如"用扩展点而非修改核心链路"），不给具体代码
- ✅ 必出 ≥ 3 张 mermaid（时序图 / 调用拓扑图 / 4 层 SP 树状图，按需求场景挑）

---

## 报告骨架（7 章节）

```markdown
# Brainstorm 核心发现：<需求名>

> 业务线：豆包-创作 / 子方向：<生图/视频/...>
> 复杂度：CL<1/2/3>
> 报告生成时间：YYYY-MM-DD
> 关联中间产物（不引用，仅追溯）：story.md @ commit、link_analysis-*.md @ commit

---

## 一、需求复述

简述需求目标、范围、约束。**只复述用户给的，不夹杂分析**。

包含：
- 业务背景
- 功能点清单（按 "模块 ｜ 动作 ｜ 标题" 格式）
- 验收标准（如果 PRD 里有）

---

## 二、入口与代际定位

| 维度 | 落点 |
|---|---|
| 流量入口 | C 端（AgentStream）/ Picasso（ExecuteAgent）/ 双侧 |
| 架构代际 | 新 ReAct（agent_phase_26）/ 老 Plan-Act / 双轨 |
| 触发条件 | 例如：`option.SupportAgent26(ctx) && tenant_id == X` |

**入口决策树**（mermaid 必出）：

\`\`\`mermaid
flowchart TD
    A[用户请求] --> B{C端 or Picasso?}
    B -- C端 --> C[flow.agent.creation::AgentStream]
    B -- Picasso --> D[flow.alice.creation_access::ExecuteAgent]
    D --> E[flow.agent.creation_evaluation 子单体]
    C --> F{SupportAgent26?}
    E --> F
    F -- 是 --> G[agent_phase_26/agent/runtime ReAct]
    F -- 否 --> H[handler/phase3handle 老 Plan-Act]
\`\`\`

---

## 三、命中的创作动线

| 命中动线 | 来自 knowledge | 该动线本期是否被 PRD 改造 |
|---|---|---|
| 文生图/主bot 闲聊 | C/文生图/主bot 闲聊入口.md | ✅/⚠️/❌ |

**用户动线时序图**（mermaid，从 knowledge 复用）：

\`\`\`mermaid
sequenceDiagram
    participant U as 用户
    participant CA as creation_agent
    participant AT as aigc_tool
    participant AD as aigc_dag
    U->>CA: 输入"画一只猫"
    CA->>AT: SyncInvokeTool(image_gen)
    AT->>AD: Exec(DAG topic)
    AD-->>AT: 图片 URL
    AT-->>CA: 工具结果
    CA-->>U: CreationBlock 上屏
\`\`\`

---

## 四、链路探索结论（按功能点）

### 4.1 功能点 1：<标题>

#### 4.1.1 链路穿越点（mermaid 标红）

时序图：在第三章基础上**高亮本功能点改动的位置**。

#### 4.1.2 关键文件 / 模块清单

按"清单 + 流程图"表达，**不要堆砌行号**：

| 角色 | 仓库 | 模块 / 文件 | 职责 |
|---|---|---|---|
| 入口 | creation_agent | handler/handler.go::AgentServiceImpl | 流式接收 |
| Tool 管理 | aigc_management | service/domain/ability/schema/<x>.json | schema |
| ... | ... | ... | ... |

#### 4.1.3 配置 / 数据模型

- 关键配置 Key：TCC / Fornax / Abase / 各自的命名空间
- 关键数据模型：AgentPayload / ContextMessage / RuntimeMemory / Block
- **重点**：评估"改配置 vs 改代码"的可能性

#### 4.1.4 上下游强弱依赖

| 上 / 下游 PSM | 接口 | 强弱依赖 | 报错影响 |
|---|---|---|---|
| ... | ... | ... | ... |

### 4.2 功能点 2：<标题>

（结构同 4.1）

---

## 五、SP / Libra / Tool 三处分散点盘点

### 5.1 SP 触达检查（强制）

| 维度 | 命中 | 验证位置 |
|---|---|---|
| ① Picasso 注入 | ✅/❌ | AgentPayload.doubao_agent_config_v2_data |
| ② Libra / AB 实验 | ✅/❌ | creation_evaluation/constant/ab_params.go |
| ③ TCC 固化配置 | ✅/❌ | internal/util/tcc/base.go |
| ④ Fornax 默认 | ✅/❌ | internal/rpc/fornax/ |

**3 处散落代码**（如果命中 SP 必看）：
- 旧：`creation_agent/internal/rpc/agent_util/sp.go::LoadSPWithDefault`
- phase3：`creation_agent/internal/rpc/phase3/agent_runtime/sp_manager.go::SystemPromptManager`
- 新：`creation_agent/agent_phase_26/agent/sp/agent_sp.go`

### 5.2 Libra 切流盘点（P0 痛点）

| 散落点 | 命中本需求 | 当前切流配置 |
|---|---|---|
| `creation_agent/handler/handler.go::SupportAgent26` | ✅/❌ | 实验 key = ?, 切流比例 = ? |
| `creation_access/constant/libra.go` | ✅/❌ | ... |
| `creation_evaluation/constant/ab_params.go` | ✅/❌ | ... |
| `creation_agent/internal/rpc/phase3/agent_runtime/sp_manager.go` | ✅/❌ | ... |
| `aigc_dag/biz/common/ab.go` | ✅/❌ | ... |
| `aigc_tool/dal/dag_switch.go::SwitchToNewTopic` | ✅/❌ | ... |

### 5.3 Tool 链路盘点

| Tool | schema 是否需改 | 网关入口 | DAG topic | handler |
|---|---|---|---|---|
| ... | ... | ... | ... | ... |

---

## 六、Gap 与风险

### 6.1 命中的 PRD vs 实现 Gap

来自 `Z.2 PRD vs 实现 Gap 表.md`，本次需求可能踩到的：

| Gap 项 | 状态 | 对本需求的影响 |
|---|---|---|
| 多租户隔离未落地 | gap | 如果你想加租户级开关，需自行补隔离逻辑（PRD 设计已说要做但 sp.md 自述未做） |
| 错误码 flow/ocerr 未集成 | gap | 错误透传只能通过 biz_err_status_code/msg |
| SP 管理 3 处分散 | regression | 改 SP 必须三处验证 |
| ... | ... | ... |

### 6.2 业务风险

- **隐藏依赖**：列出可能漏掉但访谈痛点提示要查的（线下口口相传知识）
- **新老并存窗口**：本次改动是否影响新老链路？
- **数据兼容**：是否触达 Memory / 上屏协议的存量数据？
- **强弱依赖变更**：是否引入新的强依赖下游？

### 6.3 测试风险

- 单测覆盖盲区
- E2E 测试触发链路（Picasso 评测 vs C 端真实流量）

---

## 七、给 RD 的方案方向建议（仅方向，不代写）

### 7.1 改动定位建议

**建议**：本次改动**应该**主要落在 `<某模块>`，而**不要**散到 `<某另一模块>`。

理由：
1. 该模块是本次链路的<入口/调度/收敛点>
2. 改其他模块会触达 <X 个 Libra / 现有 SP / ...>，扩大影响面

### 7.2 扩展机制建议

| 改动类型 | 建议机制 | 备选 |
|---|---|---|
| 新增 Tool | 走 aigc_management schema 注册 + aigc_dag handler | — |
| 新增 SP | 走 Fornax 配置（不要硬编码） | TCC 兜底 |
| 灰度策略 | 走 Libra 实验 + ① 优先级（Picasso 注入） | TCC 切流 |

### 7.3 必须验证的事项（写方案时务必想清楚）

- [ ] 你的改动在新老两条链路里都生效 / 都不生效？
- [ ] 你的改动是否要兼容存量 Memory 数据？
- [ ] 你的改动是否影响异步上屏？
- [ ] 你的改动有 Gap 表里的项目要绕过吗？

### 7.4 待用户决策的问题（如果有）

- 问题 1：xxx
- 问题 2：xxx

---

## 附录：关联资源

| 类型 | 路径 |
|---|---|
| 本 skill | `marketplace/plugins/xdev/skills/doubao-agent-scene-brainstorm/` |
| 知识地图 | `knowledge_repository/doubao_creation/Z. 主索引/Z.0 知识地图.md` |
| Gap 表 | `knowledge_repository/doubao_creation/Z. 主索引/Z.2 PRD vs 实现 Gap 表.md` |
| 调研报告 | `stone/devclaw_skills_center/feat-dev/2026/04/27/183429-doubao-agent-scene-brainstorm-skill/research.md` |

```

---

## 与 ecom-buy `report_template.md` 的差异

ecom-buy 模板是**技术方案文档模板**（含坐标 / 改动点 / diff 代码 / 接口表 / 风险评估），目的是 RD 拿走可直接交付。

本模板是**brainstorm 决策辅助模板**：

| 维度 | ecom-buy | 本模板 |
|---|---|---|
| 是否含 diff 代码 | ✅ 强制 | ❌ 严禁 |
| 是否含改动点坐标（File:Line） | ✅ 强制 | ❌ 改为流程图高亮 |
| 是否含接口 QPS / 超时表 | ✅ 强制 | ⚠️ 仅"上下游强弱依赖" |
| 是否含 Gap 对照 | — | ✅ 强制 |
| 是否含 SP/Libra/Tool 散落盘点 | — | ✅ 强制 |
| 章节命名 | 中文一二三四 | 中文一二三四（沿用） |
| 最终读者动作 | 拿走交付 | 拿走→ RD 自己写技术方案 |
