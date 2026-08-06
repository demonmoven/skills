---
name: doubao-agent-scene-brainstorm
description: "豆包-创作业务线服务端 RD 在写技术方案前进行代码探索、隐性知识汇聚、决策依据准备的 brainstorm 套件。当用户带着一份 PRD/需求来分析豆包-创作 agent 链路（包含 creation_agent / creation_access / aigc_dag / aigc_management / aigc_tool / alice_idl 等仓库），希望先把屎山、Libra 切流点、SP 4 层加载、Agent→Tool→Model 链路的隐藏知识结构化梳理出来时使用此 skill。本 skill 不代写技术方案，只输出 brainstorm 核心发现，供 RD 自行决策。适用场景关键词：豆包创作 agent、文生图/图生图/AI修图/分身写真/文生视频/文生音乐、Plan-Act/ReAct 架构升级、Picasso 评测灰度、Libra 切流盘点。"
---

<!-- @format -->

# 豆包-创作 Agent 链路 Brainstorm Skill

本 skill 把豆包-创作 agent 链路里**被分散到各处的隐性知识**——尤其是 SP 来源、Libra 切流点、Tool 链路、新老架构并存——在 RD 写技术方案前一次性、结构化地呈现出来，让 3PD 的方案心智负担降到 ≤ 1.5PD，避免编码 / 自测阶段回炉。

> **scope 约束**：本 skill 仅覆盖**豆包-创作 agent 这条链路**（8 仓库见下）。社区业务、海外业务、其他 doubao 子方向另立 skill（如未来的 `doubao-community-scene-brainstorm`）。

---

## 适用痛点（基于访谈调研直接对应）

本 skill **核心目标**：解决豆包-创作/社区-服务端团队访谈中暴露的方案阶段痛点。详见 [`knowledge_repository/doubao_creation/Z. 主索引/Z.3 痛点应对地图.md`](./knowledge_repository/doubao_creation/Z.%20主索引/Z.3%20痛点应对地图.md)。

| 痛点 | 优先级 | 本 skill 应对 |
|---|---|---|
| 屎山代码 + 隐藏坑点 → 方案心智 >> 编码（3PD : 5PD 倒挂） | **P0** | 链路探索方法论 + 13 条创作动线 + 误区纠正 |
| Agent → Tool → Model 链路业务逻辑分散到各处不收口 | **P0** | tool_locator 4 级链路 + tool_chain_trace 子动作 |
| **Libra 切流逻辑散落各处**，反复手动 Libra 平台 | **P0** | libra_locator 7 处全清单 + libra_inventory **强制必跑**子动作 |
| 历史包袱 / 隐藏知识在线下口口相传 | P1 | A.团队组织 / B.架构演进史 / Z.2 Gap 表 |
| CoCo 输出价值打折（不掌握历史包袱 / 切流） | P1 | 5 个 locator 提供领域知识 + 置信度机制 |
| 业务逻辑散落、改动几行 → 人改更顺手 | P2 | brainstorm **不代写代码**，只给方向 + 决策依据 |
| 多人协作漏同步 → 自测才暴露 | P2 | brainstorm 报告强制列"涉及 owner" + 找人地图 |

⚠️ **明示范围外**（不在本 skill scope）：
- AI 单测 / CR 标准 / 字节云平台串联 / E2E 评测优化（属其他 skill 范畴）
- Libra 增/删 write 侧 skill（未来 `skill_libra_change`，本 skill 仅做 read 侧盘点）
- 局部屎山重构决策（属 refactoring 类工具范畴）

## 适用场景

✅ **应该用** 当：
- 用户带 PRD / 需求要做**豆包-创作 agent 链路**的方案分析
- 需求涉及生图 / 生视频 / AI 修图 / 分身写真 / 文生音乐 等创作能力
- 需求触达 Agent → Tool → Model 链路的任意一段
- 需求涉及 SP 调整、Libra 实验改动、Tool schema 变更、Block 协议改动
- RD 反映"代码屎山，方案心智重于编码"

❌ **不应该用** 当：
- 需求属社区 / 海外 / 分发&消费 / 商业化等其他业务线
- 需求是单纯的工程任务（如 CR 工具、单测优化、字节云平台串联）
- 用户已经有完整技术方案，只是要 review / 改动其中某一段

---

## 覆盖的 8 个仓库

| worktree | PSM | 角色 |
|---|---|---|
| `creation_access` | `flow.alice.creation_access` | 接入层（主要 for Picasso） |
| `creation_agent` | `flow.agent.creation` | C 端 Agent 主服务 |
| `creation_agent/creation_evaluation/` 子单体 | `flow.agent.creation_evaluation` | Picasso 评测专用 |
| `aigc_management` | `flow.aigc.management` | 创作能力配置中心 |
| `aigc_dag` | `flow.aigc.dag` | DAG 执行引擎 |
| `aigc_tool` | `flow.aigc.tool` | 工具调用网关 |
| `alice_idl` | — | 创作业务自有 IDL |
| `idl` (butterfly/idl) | — | aigc 三件套公共 IDL |

> 详见 [`locators/psm_locator.md`](./locators/psm_locator.md)。

---

## 三阶段流程总览

按 [`guidance.md`](./guidance.md) 的强约束，必须**逐阶段**执行，**严禁一次性加载所有 sub-skill**：

```
阶段一  需求拆解         → skills/skill_request_brainstorm/
        输入：用户原始 PRD / 需求
        输出：workspace/user/story.md
              （含功能点 + 复杂度定级 + 5 维度定位线索）

阶段二  链路探索         → skills/skill_link_explore/
        输入：story.md
        输出：workspace/user/link_analysis-<功能点>.md
              （6 子动作产出，每个功能点一份）

阶段三  方案对齐         → skills/skill_solution_alignment/
        输入：story.md + link_analysis-*.md
        输出：workspace/artifacts/research/brainstorm_core_findings.md
              （brainstorm 核心发现，不代写方案）
```

> **关键边界**：本 skill 阶段三**不输出技术方案文档**，只输出"brainstorm 核心发现"。技术方案由 RD 自己写，本 skill 提供决策依据。

---

## Locator 索引（5 件套，先看这里）

每个 locator 顶部含 `last_synced_commit` + `verified_against_branch` 字段，用于追踪与代码的同步状态。

| Locator | 用途 | 场景 |
|---|---|---|
| [`locators/psm_locator.md`](./locators/psm_locator.md) | 6 PSM × 接口入口 + 调用关系 | 任何需求都要先用：定位入口 |
| [`locators/sp_locator.md`](./locators/sp_locator.md) | SP 4 层加载链 + 3 处散落代码 + 17 个 SP key | 需求触达 SP 配置时 |
| [`locators/tool_locator.md`](./locators/tool_locator.md) | 8 个 Tool × schema/网关/DAG topic/handler 四级链路 | 需求新增 / 改动 Tool 时 |
| [`locators/libra_locator.md`](./locators/libra_locator.md) | 7 处 Libra/AB 散落点 + 上下游策略差异 | **任何涉及灰度的需求都必读**（P0 痛点） |
| [`locators/block_locator.md`](./locators/block_locator.md) | 5 种 Block 协议 + WriterPacket + 上屏顺序锁 | 需求触达上屏 / Chat 交互时 |

---

## Knowledge Repository 6 类目地图

```
knowledge_repository/doubao_creation/
├── A. 组织职责与找人地图/        ← 谁负责什么 + 5 个常见误区
├── B. Agent 架构专栏/            ← 业务全景 / 架构演进 / ReAct 五层 / ContextMessage / 新老并存
├── C. 创作能力动线专栏/          ← 创作能力 × 入口矩阵（首版 4 条命中本期）
├── D. 中间件_基础设施专栏/       ← Fornax / TCC / Abase / VikingDB / Libra / ImageX / flow_ocerr
├── E. 关键术语_黑话词典/         ← 50+ 术语（含模型/工具/协议/ID/业务/工程黑话）
└── Z. 主索引/                    ← 知识地图 + PSM 索引 + PRD vs 实现 Gap 表
```

**入门读法**：先读 `Z.0 知识地图.md` 找到自己的切入点；不知道从哪开始就按 A → B.3 → 你需求命中的 C 动线 → 用到的 D 中间件。

---

## 输出物清单

| 阶段 | 路径 | 形态 |
|---|---|---|
| 一 | `workspace/user/story.md` | 需求拆解 + 5 维线索 |
| 二 | `workspace/user/link_analysis-<功能点>.md` | 链路探索（6 子动作） |
| 三 | `workspace/artifacts/research/brainstorm_core_findings.md` | brainstorm 核心发现报告 |

---

## 与 `ecom-buy-skills` 的差异

ecom-buy 三阶段定位粒度是 PSM+API 二维 + 交易模式 / 非交易模式二分；豆包是**入口（C 端 / Picasso）× 代际（Plan-Act / ReAct）× 创作能力动线**三维。本 skill **不代写技术方案**，只产出 brainstorm 核心发现，供 RD 决策。Locator 从 1 个（api_locator）扩展为 5 个（PSM / SP / Tool / Libra / Block）。Knowledge 多两类（C 创作动线、E 术语词典）。

---

## 反馈与维护

- 调研报告与 plan 沉淀于 `stone/devclaw_skills_center/feat-dev/2026/04/27/183429-doubao-agent-scene-brainstorm-skill/`
- knowledge / locator 与代码漂移时，请同时更新对应文件顶部的 `last_synced_commit`
- 本 skill 仅覆盖创作 agent 链路；其他 doubao 子方向请走对应独立 skill
