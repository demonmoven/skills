# 7 维度检查清单 — 豆包-创作 agent 链路

> 用法：阶段二 `skill_link_explore` 完成后，对照本清单逐项检查 link_analysis 是否覆盖了应该覆盖的维度。

---

## D1：入口轴（流量来源）

- [ ] 已判定 C 端（AgentStream）/ Picasso（ExecuteAgent）/ 双侧
- [ ] 第一落点函数已定位
- [ ] 已说明流量入口与本需求的关联（如"本需求只 Picasso 评测灰度，所以仅触达 ExecuteAgent"）

## D2：代际轴（架构代际）

- [ ] 已判定走新 ReAct（agent_phase_26）/ 老 Plan-Act / 双轨
- [ ] **如果是双轨**：双份位置均已列（agent_phase_26 + phase3 / 老 Plan-Act）
- [ ] `option.SupportAgent26(ctx)` 当前灰度比例已查（去 Libra / TCC 平台）
- [ ] 已评估"仅改一轨"是否会带来漏改风险

## D3：能力轴（创作能力动线）

- [ ] 已命中 13 条创作动线之一
- [ ] 已链向 C 类目对应文件
- [ ] 该动线本期是否被 PRD 改造（✅/⚪/❌）已标注
- [ ] 该动线的"7 章节"已被参考（特别是用户动线 + 本期影响点）

## D4：SP 轴（System Prompt）

- [ ] 4 层加载链触达情况已列（① Picasso ② Libra ③ TCC ④ Fornax）
- [ ] **3 处 SP 散落代码**已全数检查
  - [ ] 旧：`internal/rpc/agent_util/sp.go`
  - [ ] phase3：`internal/rpc/phase3/agent_runtime/sp_manager.go`
  - [ ] 新：`agent_phase_26/agent/sp/agent_sp.go`
- [ ] 涉及的 SP key 候选已列
- [ ] **多租户 gap** 已显式提醒（PRD 设计未落地）
- [ ] **regression 警告** 已显式给 RD（"本期 SP 反而多增一处"）

## D5：切流轴（Libra / AB）★ P0 痛点

- [ ] **7 处散落点全数检查**（即使无关也明示"不触达"）
  - [ ] # 1 `creation_agent/handler/handler.go::SupportAgent26`
  - [ ] # 2 `SupportAgent26Ark`
  - [ ] # 3 `creation_access/constant/libra.go`
  - [ ] # 4 `creation_evaluation/constant/ab_params.go`
  - [ ] # 5 `phase3/agent_runtime/sp_manager.go`（SP 来源选择）
  - [ ] # 6 `aigc_dag/biz/common/ab.go`
  - [ ] # 7 `aigc_tool/dal/dag_switch.go`
- [ ] 上下游切流策略差异（Libra vs TCC + 一致性 hash）已显式说明
- [ ] 实验命名约定 `creation_<能力>_<场景>_v<版本>` 已遵循
- [ ] **下线计划**（防止 dead code 屎山源头）已写入

## D6：工具/DAG 轴（Tool 链路）

- [ ] 命中的 Tool 已列（image_gen / image_edit / image_to_video / host / Hold / Quota / Safety / saliency_segmentation 等）
- [ ] 4 级链路至少 1 个 Tool 完整：
  - [ ] schema：`aigc_management/service/domain/ability/schema/<x>.json`
  - [ ] 网关：`aigc_tool/method/<x>.go`
  - [ ] DAG topic：`aigc_management::GetOnlineDAGTopic` 路径
  - [ ] handler：`aigc_dag/biz/handler_{sync,async}/<x>.go`
- [ ] Sync vs Async 决策给出（视频必须 Async）
- [ ] **ResourceID 跨层一致性**已检查（image_gen_X 在模型 / 工具 / Memory / Block 各层一致）
- [ ] 是否触达 `ToolInfo` 协议简化（silent breaking，详见 Z.2 Gap）
- [ ] 是否需要新增 Tool（如需，schema JSON 路径已规划）

## D7：上屏轴（Block 协议）

- [ ] 命中的 Block 类型已列（CreationBlock 2074 / LoadingBlock 10101 / TextBlock / ThinkingBlock 10040 / ButtonBlock 10103）
- [ ] CreditBlock / LoginBlock 不在本期范围 → 不要假设可用
- [ ] WriterPacket 协议（PacketType incr / full）影响已评估
- [ ] **同步 vs 异步上屏**决策给出（耗时长场景必须 async）
- [ ] **上屏顺序锁**（GetPacketSeq）影响已评估（涉及多 Tool 并发时）
- [ ] **父子 Block 关系**（如 ThinkingBlock + TextBlock）已评估（涉及思考态时）
- [ ] **Memory 兼容**已评估（改 Block 协议是否影响 MsgCreationAgentMemoryInfo 反序列化）

---

## 综合检查

- [ ] 6 子动作产出**全数**包含 mermaid + 清单 + 风险点三件套
- [ ] 跨子动作的共性发现已显式归纳（如"本需求触达 SP regression + Libra 散落 4 处 + Tool 跨 4 仓库改动"）
- [ ] PRD vs 实现 Gap（Z.2 表）的命中项已显式标注
- [ ] **A.误区纠正**的 8 条误区是否被踩？已逐条 review
- [ ] 写作风格符合"流程图 + 表格清单"，**未堆砌文件路径 + 行号**
- [ ] 没有"我建议怎么写代码"的代写倾向（这是阶段三的工作）

---

## 子动作"必跑"原则

虽然 6 子动作不是每个都"强制必出产出"，但下面是**最低可接受标准**：

| 子动作 | 何时可省 | 何时必出 |
|---|---|---|
| `entry_locate` | **永远不可省** | 任何需求都要 |
| `capability_match` | **永远不可省** | 任何需求都要 |
| `sp_inventory` | 完全不触达 SP（极少） | 含"提示词 / 规则 / 意图"的需求都要 |
| `libra_inventory` | **永远不可省**（即使无关也要明示"不触达"，这是 P0 痛点） | 含"灰度 / 实验 / AB" 都要 |
| `tool_chain_trace` | 不触达任何 Tool（极少） | 含"工具 / DAG / 调度"都要 |
| `block_protocol_check` | 不触达上屏（如纯后端任务） | 含"展示 / 上屏 / Chat 交互"都要 |

---

## 反思建议（执行完用）

- 我跑的 6 子动作中**最弱的是哪个**？是因为 locator / knowledge 不够用，还是我跳过了？
- 我有没有**找到 PRD 没明说但代码里有的复杂度**（如 silent breaking / undocumented change）？
- 我的 link_analysis 给阶段三 `skill_solution_alignment` 提供了**足够的决策依据**吗？
