# doubao-agent-scene-brainstorm 使用说明

> 豆包-创作业务线服务端 RD 在写技术方案前进行**代码探索 + 隐性知识汇聚 + 决策依据准备**的 brainstorm 套件。**不代写技术方案**，只输出 brainstorm 核心发现。

---

## 一句话理解

把豆包-创作 agent 链路里**被分散到各处的隐性知识**——尤其是 **SP 来源**、**Libra 切流点**、**Tool 链路**、**新老架构并存**——在 RD 写技术方案前一次性、结构化地呈现出来，让 3PD 的方案心智负担降到 ≤ 1.5PD，避免编码 / 自测阶段回炉。

---

## 适用痛点（基于访谈调研）

| 痛点 | 优先级 | 本 skill 应对 |
|---|---|---|
| 屎山代码 + 隐藏坑点 → 方案心智 >> 编码（3PD : 5PD 倒挂） | **P0** | 链路探索方法论 + 13 条创作动线 + 误区纠正 |
| Agent → Tool → Model 链路业务逻辑分散到各处不收口 | **P0** | tool_locator 4 级链路 + tool_chain_trace 子动作 |
| **Libra 切流逻辑散落各处**，反复手动 Libra 平台 | **P0** | libra_locator 7 处全清单 + libra_inventory **强制必跑** |
| 历史包袱 / 隐藏知识在线下口口相传 | P1 | A.团队组织 / B.架构演进史 / Z.2 Gap 表 |
| CoCo 输出价值打折（不掌握历史包袱 / 切流） | P1 | 5 个 locator 提供领域知识 + 置信度机制 |

⚠️ **明示范围外**（不在本 skill scope）：
- AI 单测 / CR 标准 / 字节云平台串联 / E2E 评测优化（属其他 skill 范畴）
- Libra 增/删 write 侧 skill（未来 `skill_libra_change`，本 skill 仅做 read 侧盘点）

---

## 快速开始

### 前置准备

1. **8 个 worktree 都已切到 `feat/design_doubao` 分支**（详见调研产物 `info.md`）
2. **在你的目标工作区启动 Claude Code**：
   ```bash
   cd /path/to/feat-design
   claude
   ```
3. **本 skill 应在 Claude Code 加载后自动出现在 skill 列表**（已注册到 xdev plugin）

### 三阶段全流程（最推荐）

**重要**：本 skill **没有 action 参数**，是 **Stage-by-Stage 强约束流程**，由顶层 `guidance.md` 协调。直接对 Claude 说：

```
我要做豆包-创作 agent 链路的需求方案分析，PRD 在 ......（飞书链接 / 文本 / workspace/downloads/xxx.md）
```

Claude 会自动：

1. 进入 **scope 自检**（确认需求落在豆包-创作 agent 链路）
2. **加载阶段一** sub-skill `skill_request_brainstorm` → 拆功能点 + 5 维度定位线索 + 复杂度定级
3. **进阶段二** sub-skill `skill_link_explore` → 6 子动作链路探索（**必跑** entry / sp / **libra** / tool / capability，按需跑 block）
4. **进阶段三** sub-skill `skill_solution_alignment` → 整合输出 brainstorm 核心发现

阶段间**强约束**：前阶段没结束严禁进下阶段；每个 sub-skill 加载后必须先做战术反思。

---

## 三阶段产出

| 阶段 | sub-skill | 输出 | 形态 |
|---|---|---|---|
| 一 | `skill_request_brainstorm` | `workspace/user/story.md` | 需求拆解 + 5 维度分析线索 |
| 二 | `skill_link_explore` | `workspace/user/link_analysis-<功能点>.md` | 链路探索 6 子动作产出（每功能点一份） |
| 三 | `skill_solution_alignment` | `workspace/artifacts/research/brainstorm_core_findings.md` | brainstorm 核心发现报告（不代写方案） |

> 阶段三的 `brainstorm_core_findings.md` 是给 RD 写技术方案前的**决策辅助**。RD 拿到后**自己写**技术方案。

---

## 7 维度定位矩阵（skill 的核心模型）

每个需求落到豆包-创作 agent 链路上时，按 7 维度逐一探索：

| 维度 | 内容 | 来源 locator |
|---|---|---|
| **D1 入口** | C 端（AgentStream）/ Picasso（ExecuteAgent）/ 双侧 | psm_locator |
| **D2 代际** | 新 ReAct（agent_phase_26）/ 老 Plan-Act / **双轨** | psm_locator + B.5 新老并存 |
| **D3 能力** | 文生图 / 图生图 / AI 修图 / 分身写真 / 文/图生视频 / 文生音乐 + 多入口 | C. 创作能力动线专栏 |
| **D4 SP** | Picasso > Libra > TCC > Fornax 4 层加载 + 3 处散落代码 | sp_locator |
| **D5 切流** | 上游 Libra + 下游 TCC + 一致性 hash 共 7 处散落 | libra_locator ★ P0 |
| **D6 工具/DAG** | schema → 网关 → DAG topic → handler 4 级链路 | tool_locator |
| **D7 上屏** | CreationBlock / LoadingBlock / ThinkingBlock / TextBlock / ButtonBlock | block_locator |

---

## 5 个 Locator（先看这里）

每个 locator 顶部有 `last_synced_commit` + `verified_against_branch` 字段，追踪与代码同步状态。

| Locator | 用途 | 何时必读 |
|---|---|---|
| `locators/psm_locator.md` | 6 PSM × 接口入口 + 调用关系拓扑 | 任何需求都先看 |
| `locators/sp_locator.md` | SP 4 层加载链 + 3 处散落代码 + 17 SP key | 需求触达 SP 配置 |
| `locators/tool_locator.md` | 8+ Tool 4 级链路 + ToolInfo 协议变迁 | 需求新增 / 改动 Tool |
| `locators/libra_locator.md` | **7 处 Libra/AB 散落点全清单** | **任何涉及灰度的需求都必读**（P0 痛点） |
| `locators/block_locator.md` | 5 种 Block + WriterPacket + 上屏顺序锁 | 需求触达上屏 / Chat 交互 |

---

## Knowledge Repository 6 类目速查

```
knowledge_repository/doubao_creation/
├── A. 组织职责与找人地图/        ← 谁负责什么 + 8 条常见误区
├── B. Agent 架构专栏/            ← 业务全景 / 演进史 / ReAct 五层 / ContextMessage / 新老并存
├── C. 创作能力动线专栏/          ← 创作能力 × 入口 矩阵（首版完整 4 条 + 9 骨架）
├── D. 中间件_基础设施专栏/       ← Fornax / TCC / Abase / VikingDB / Libra / ImageX / flow_ocerr
├── E. 关键术语_黑话词典/         ← 91 词条（模型 / 工具 / 协议 / ID / 业务 / 工程黑话）
└── Z. 主索引/                    ← 知识地图 + PSM 索引 + Gap 表 + ★痛点应对地图
```

**入门读法**：先读 `Z.0 知识地图.md` 找切入点；不知道从哪开始就按 A → B.3 → 你需求命中的 C 动线 → 用到的 D 中间件。

**反向追溯**：如果你想确认本 skill 能解决你的具体痛点，直接看 `Z.3 痛点应对地图.md`——访谈每条痛点 → skill 应对位置。

---

## 三个关键产出物（来自 Z.主索引）

| 文件 | 用途 |
|---|---|
| `Z.1 PSM 索引.md` | 6 PSM × 关键代码路径 / build / IDL 速查 |
| `Z.2 PRD vs 实现 Gap 表.md` | PRD 设计 vs 当前实现的 gap / regression / silent breaking 显式标注（**写方案必看**） |
| `Z.3 痛点应对地图.md` | 访谈痛点 → skill 应对位置的反向追溯映射 |

---

## 常见 Q&A

### Q1：本 skill 跟 CoCo 通用 LLM 输出的区别？

CoCo 给的是"通用 LLM 知识"，不掌握豆包链路的历史包袱、Libra 切流、SP 散落、新老架构并存等领域细节，输出价值容易打折。

本 skill 用 5 个 locator + 6 类目 knowledge 提供**领域权威知识**，并用置信度机制让 RD 知道哪些可信、哪些待确认 —— 等于**用领域 knowledge 校准 CoCo 通用输出**。

### Q2：为什么本 skill 不代写技术方案？

业务方访谈反馈：**业务逻辑分散，每处改动量很小（几行）→ 人改更顺手；CoCo 找错就多了一次判断对错的成本**（来自王瑶访谈）。

所以本 skill 设计哲学是：让 AI 提供"决策辅助"（你应该改哪个模块、避开什么坑、查哪些 Gap），把"几行改动"留给人。RD 拿到 brainstorm 核心发现后，用更短的时间下笔写方案。

### Q3：本 skill 的 scope 边界是什么？

**仅覆盖豆包-创作 agent 链路 8 个仓库**（creation_agent / creation_access / aigc_dag / aigc_management / aigc_tool / alice_idl / idl(butterfly)）。

社区、海外、分发&消费、商业化等其他业务线**不在 scope**。未来如有需要，建议另立 skill（如 `doubao-community-scene-brainstorm`）。

### Q4：knowledge 内容会不会跟代码漂移？

会。本 skill 通过以下机制控制：
- 每个 locator 顶部的 `last_synced_commit` 字段标注同步代码版本
- `Z.2 Gap 表` 显式标注 PRD 设计 vs 当前实现的 gap
- `Z.0 知识地图` 顶部标注上次同步日期
- 首版采用**静态快照**，未来 v2 接 hook 自动检测漂移

### Q5：新老架构并存期间，方案如何处理？

**强制双轨分析**。skill_link_explore 的 entry_locate / sp_inventory / tool_chain_trace 三个子动作**默认双轨**（同时分析 `agent_phase_26/` 新链路 + `phase3/` 老链路 + 老 Plan-Act）。

详见 `knowledge_repository/doubao_creation/B. Agent 架构专栏/5. 新老架构并存策略.md`。

### Q6：Libra 增/删的标准化操作什么时候支持？

王瑶访谈明确希望"Libra 增/删的标准化 skill"，但本 skill 仅做 **read 侧盘点**（库存 / 导览 / 切流策略差异）。

**write 侧** 是后续 `skill_libra_change` 的工作（不在本 skill scope）。详见 `locators/libra_locator.md` § 七 未来扩展点。

### Q7：调研报告 / SOP 在哪？

本 skill 的设计调研、跨服务链路分析、实施计划、SOP（含飞书可读版本）都沉淀在主仓 `stone/devclaw_skills_center` 的：

```
feat-dev/2026/04/27/183429-doubao-agent-scene-brainstorm-skill/
├── research.md                  调研报告（446 行，含 §〇 8 子章节需求复述）
├── cross-services-analysis.md   跨服务调用链路分析（418 行）
├── plan.md                      实施计划（446 行，6 Phase × 27 任务）
├── sop.md                       SOP 完整版（711 行）
├── sop-slim.md                  SOP 60% 精简版（421 行）
└── sop-slim-2.md                SOP 极简版（265 行，已转飞书文档）
```

---

## 反馈与维护

- **反馈渠道**：豆包-创作业务线 RD 用 skill 时踩到的坑，回填到 `Z.3 痛点应对地图.md`
- **维护频率**：每 2-4 周拉飞书白板 JSON 同步 A.团队组织 + 检查 locator `last_synced_commit` 是否仍指向当前分支
- **维护责任人**：AI Coding 专家团队 + 业务团队联合维护
- **scope 提醒**：本 skill 仅覆盖豆包-创作 agent 链路；其他子方向请走对应独立 skill

---

> 这是一个**活文档**。任何 RD 用 skill 跑出新洞见 → 反馈 → skill 持续打磨。
