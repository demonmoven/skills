---
name: sp_locator
description: SP 4 层加载链 + 3 处散落代码 + 17 个 SP key 清单 + 完整 trace 示例。本期 PRD 计划"SP 收敛"未落地，反多增一处
last_synced_commits:
  creation_agent: "651a643215e1bd2f713aee504ec46dce40672a72"
  alice_idl:      "4f3160750ddde220e02075f579f8356a74d750de"
verified_against_branch:
  creation_agent: "feature/agent_26_lyw"
  alice_idl:      "feature/agent_creation_26"
last_synced_date: "2026-04-27"
---

# SP Locator — System Prompt 加载链路

> ⚠️ **重要警告**：本期 PRD § Agent 逻辑层 § SP 管理 设计了"统一 SP 加载"，但 diff 显示**未实现收敛，反而新增一处**（agent_phase_26/agent/sp/），导致 SP 加载逻辑现在散落在 **3 处**。**任何触达 SP 的需求都必须三处都验证**，不能假设统一。

---

## 一、SP 4 层加载优先级（PRD 设计）

来源：`feat-design/background/prd/prd.md` § Agent 逻辑层 § SP管理；`feat-design/background/prd/resources/whiteboard_02_整体流程.json`

```
┌─────────────────────────────────────────────────────────┐
│ ① Picasso 评测注入                                       │ 最高
│   字段：AgentPayload.doubao_agent_config_v2_data          │
│   场景：picasso 评测平台直接下发 SP 内容（用于评测调试）   │
├─────────────────────────────────────────────────────────┤
│ ② Libra / AB 实验配置                                    │
│   命中实验则用实验下发的 SP 版本                           │
├─────────────────────────────────────────────────────────┤
│ ③ TCC 固化配置                                           │
│   实验未命中时，从 TCC 兜底取 SP 内容                      │
├─────────────────────────────────────────────────────────┤
│ ④ Fornax 默认版本（线上）                                 │ 最低
│   兜底从 Fornax 配置中心取 SP_NAME 的线上版本              │
└─────────────────────────────────────────────────────────┘
```

来源标记（白板 02 整体流程节点）：
- `[来源于Picasso评测]` → 优先级 ①
- `[来源于实验]` → 优先级 ②
- `[来源于固化实验]` → 优先级 ③
- `[Fornax 线上]` → 优先级 ④
- `[未命中AB实验]` → ② 失败转 ③
- `[未命中AB实验且未命中TCC配置]` → ③ 失败转 ④

---

## 二、3 处散落代码（**核心痛点的代码证据**）

| # | 位置 | 角色 | 是否本期新增 | 触达检查 |
|---|---|---|---|---|
| **A** 旧 | `creation_agent/internal/rpc/agent_util/sp.go::LoadSPWithDefault()` | 老主入口；Fornax 优先（`util.FornaxCli().GetPrompt(ctx)`），失败降级 TCC（`tcc.GetDowngradePlanConf(ctx).SpBackup`） | 否 | C 端老 Plan-Act 链路用 |
| **B** phase3 | `creation_agent/internal/rpc/phase3/agent_runtime/sp_manager.go::SystemPromptManager` | phase3 中间态实现，定义 17 个 SP key 常量 | 否 | phase3 链路用 |
| **C** 新 | `creation_agent/agent_phase_26/agent/sp/agent_sp.go` | 新 ReAct 架构的 SP 加载 | ★ 是 | agent_phase_26 链路用 |

**三处都用 Fornax + TCC，但实现独立**。

### 配套 TCC 客户端

- `creation_agent/internal/util/tcc/base.go` ★ tccclient 客户端初始化 40+ 个 Getter（Fornax 配置、AB 参数、模型降级、资源模板等）
- `creation_agent/agent_phase_26/tcc/` ★ 新架构的 TCC 配置（独立！）

### 配套 Fornax 客户端

- 老链路：`util.FornaxCli().GetPrompt(ctx)`
- 新链路：`creation_agent/agent_phase_26/internal/rpc/fornax/`

---

## 三、17 个 SP key 常量清单

来源：`creation_agent/internal/rpc/phase3/agent_runtime/sp_manager.go`（phase3 定义最全，新链路 agent_phase_26 复用）

| SP key | 用途领域 |
|---|---|
| `PlannerSpKey` | Planner 模块（老 Plan-Act 主 SP） |
| `T2IDivergent` | 文生图发散描述 SP |
| ... | 还有约 15 个 SP key（具体数量与命名以代码现状为准；**写需求时务必从 sp_manager.go 当场列出，不要凭记忆**） |

> 新链路 `agent_phase_26` 在 ReAct 改造下可能会**减少 SP 数量**（因为 ReAct 的 think + tool_call 由模型自己规划，不需要单独 Planner SP）。**这是新老架构的关键差异点**。

---

## 四、完整 Trace 示例：c 端文生图请求的 SP 加载

> 以下是一个**老 Plan-Act 链路**的 SP 加载完整 trace，用于让 RD 理解 SP 触达点。

```
1. 用户请求 → creation_agent/handler.go::AgentServiceImpl.AgentStream
2. handler 顶部判断 SupportAgent26(ctx) = false
   → 进入老 Plan-Act 链路
3. dispatch → handler/phase3handle/* 选择 PlannerSpKey
4. SystemPromptManager.GetValue(ctx, PlannerSpKey)
5. 内部按 4 层优先级加载：
   a. 看 AgentPayload.doubao_agent_config_v2_data 有没有 SP 注入  
      → 没有（c 端不会注入）
   b. 看 ab_params.go 是否命中实验  
      → 命中 libra 实验 X，返回 SP 版本 v2.3
   c. （未走到）TCC.GetDowngradePlanConf().SpBackup
   d. （未走到）Fornax 线上版本
6. 返回 SP 内容 + 来源标记 = "实验 v2.3"
7. SP 内容被注入到 LLM 输入构造
```

> **新链路 trace** 走 `agent_phase_26/agent/sp/agent_sp.go`，逻辑相似但代码独立——这就是"3 处分散"的实际症状。

---

## 五、SP 改动的 brainstorm 建议

### 5.1 改动决策树

```
你想做什么？
├── 新增 SP key
│   → ① 在 Fornax 平台配置 SP_NAME
│   → ② 在 sp_manager.go 添加常量（agent_phase_26 也要加）
│   → ③ 在使用点接入
├── 调整现有 SP 内容
│   → 如果是临时实验：走 Libra（不改代码）
│   → 如果是固化：改 TCC 兜底配置（不改代码）
│   → 如果是默认版本：改 Fornax 平台（不改代码）
│   → 如果是结构变化：必须改三处！sp.go + sp_manager.go + agent_sp.go
└── 下线 SP key
    → 三处都要清，不能漏一处
```

### 5.2 必读 Gap

`Z.2 PRD vs 实现 Gap 表.md` 中"SP 管理 3 处分散"项的 status 是 **regression**（劣化）：本期反而比改造前**多一处**。

### 5.3 多租户 SP 不可用

`agent_phase_26/agent/sp/sp.md` 自述：「当前**有意不考虑** bizID/tenantID，仅实现单租户 SP 选择」。如果你的需求要做"按租户分发不同 SP"，**必须先加多租户切面**。

---

## 六、SP 与 Libra 切流的关系

SP 加载链的 ② Libra 实验是 **`libra_locator.md` 7 处散落点中的一处**。当你查 SP，必然要联动查 Libra。

参考 [`libra_locator.md`](./libra_locator.md) §3 "上游 Libra 散落点"。

---

## 七、写需求时的 SP 检查清单

- [ ] 确认本需求**是否触达 SP**（看 PRD 关键词：prompt / sys_prompt / 提示词 / SP）
- [ ] 如果触达：在 [3 处散落代码] 都验证
- [ ] 决策：走 Fornax 改配置 vs 走 Libra 实验 vs 改 TCC 兜底 vs 改代码
- [ ] 如果改代码：确认 4 层优先级是否被破坏
- [ ] 如果涉及租户：警告 RD 需自行加多租户切面（PRD 设计未落地）
- [ ] 触发的链路是否新老都生效？（agent_phase_26 + phase3 + 老 Plan-Act）
