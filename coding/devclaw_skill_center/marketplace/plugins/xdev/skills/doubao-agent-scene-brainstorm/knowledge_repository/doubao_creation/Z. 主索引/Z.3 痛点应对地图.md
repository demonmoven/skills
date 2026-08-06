# Z.3 痛点应对地图

> 本文件把访谈痛点（来源：`feat-design/background/interview/problems.md`，飞书原文档 `https://bytedance.larkoffice.com/docx/MwRqd8gChorQHaxVAHOc5b8VnUh`）中**逐条**映射到本 skill 的具体应对位置。
>
> **作用**：让 skill 维护者 / RD 能反向追溯——"我有 XX 痛点，本 skill 怎么帮我"。
>
> **审查日期**：2026-04-27

---

## 一、痛点优先级与应对总览

| # | 痛点 | 来自 | 优先级 | 应对状态 |
|---|---|---|---|---|
| P-A2 | 方案心智负担 >> 编码（屎山 + 隐藏坑点，3PD : 5PD） | 张新健 / 唐晔晨 / 郭斌浩 / 林宜葳 | **P0** | ✅ 多处覆盖 |
| P-B1 | Agent → Tool → Model 链路业务逻辑**分散到各处不收口** | 唐晔晨 / 郭斌浩 / 林宜葳 / 王英豪 | **P0** | ✅ 多处覆盖 |
| P-B2 | **Libra 切流逻辑散落各处**，反复手动 Libra 平台 | 唐晔晨 / 杨守亮 / 王瑶 | **P0** | ✅ 强覆盖 |
| P-B3 | 历史存量逻辑不熟，需求/方案环节容易遗漏 → 编码阶段返工 | 杨守亮 / 张新健 | P1 | ✅ 覆盖 |
| P-D1 | PRD 普遍潦草，大量隐藏信息在线下/电话 | 王瑶 | P1 | ✅ 部分覆盖 |
| P-D2 | 新人链路不熟，CoCo 输出价值打折（不了解历史包袱 / 切流） | 王瑶 | P1 | ✅ 覆盖 |
| P-D3 | 业务逻辑散落、每处改动量小 → 人改更顺手；CoCo 找错增加判断成本 | 王瑶 | P2 | ✅ 部分覆盖 |
| P-D4 | Libra 增/删的标准化 skill | 王瑶 | P2 | ⚠️ **范围外**（已明示） |
| P-E1 | 多人协作易遗漏同步信息 → 自测才暴露冲突 | 王英豪 | P2 | ✅ 部分覆盖 |
| P-A1 | 代码库交接，局部细节不熟，评估不足回炉 | 张新健 | P2 | ✅ 部分覆盖 |
| P-A3 | 局部屎山纠结重不重构 | 张新健 | P2 | ⚠️ 不直接覆盖 |
| P-林宜葳 | 文档库维护 & 协同迭代方式 | 林宜葳 | P2 | ✅ 隐式覆盖（locator 静态版本机制） |
| **共性 C1** | AI 单测覆盖率为导向，价值不大 | 共性 | — | ❌ **scope 外**（不属代码探索范畴） |
| **共性 C2** | 无 CR 标准规范 | 共性 | — | ❌ **scope 外** |
| **共性 C3** | 字节云多平台串联重复劳动 | 共性 | — | ❌ **scope 外** |
| **共性 C4** | 效果评测全流程繁琐（E2E） | 共性 | — | ❌ **scope 外** |

---

## 二、逐条应对详解

### P-A2 / B1：方案心智负担 >> 编码 + 链路分散不收口（P0）

**痛点描述**：
- 屎山代码 + 隐藏坑点 → 方案阶段花费 3PD，编码阶段反而 5PD（典型倒挂）
- Agent → Tool → Model 链路上大量业务逻辑分散到各处，漏评即 bug

**本 skill 应对**：

| 应对维度 | 具体位置 |
|---|---|
| 链路探索方法论 | [skill_link_explore](../../../skills/skill_link_explore/SKILL.md) 的 6 子动作 |
| 6 子动作中的 D6（tool_chain_trace） | **专门解决** Agent → Tool → Model 链路 |
| 4 级链路追踪（schema → 网关 → DAG topic → handler） | [tool_locator §二](../../../locators/tool_locator.md) |
| 创作动线 13 条矩阵 | [`C. 创作能力动线专栏/0. 动线总览.md`](../C.%20创作能力动线专栏/0.%20动线总览.md) |
| 4 条命中本期动线的完整 7 章节 | [文生图](../C.%20创作能力动线专栏/文生图/主bot%20闲聊入口.md)、[图生图](../C.%20创作能力动线专栏/图生图/主bot%20闲聊入口.md)、[视频](../C.%20创作能力动线专栏/文生视频_图生视频/主bot%20指令集入口.md)、[分身写真](../C.%20创作能力动线专栏/分身写真/推理链路%20主bot%20闲聊.md) |
| 误区纠正 8 条（防止把屎山误归类） | [`A. 组织职责与找人地图/误区纠正.md`](../A.%20组织职责与找人地图/误区纠正.md) |
| 强制 ResourceID 跨层一致性检查 | [tool_locator §六](../../../locators/tool_locator.md) |

**自检**：✅ 强覆盖 + 多处冗余

---

### P-B2：Libra 切流散落（P0 痛点核心）

**痛点描述**：
- 代码逻辑中散落较多实验切流逻辑
- 每次方案过程中需要反复手动 Libra 平台上观察切流进度，影响技术决策
- 上线前多端切流操作非常繁琐

**本 skill 应对**：

| 应对维度 | 具体位置 |
|---|---|
| **专门 locator** | [`libra_locator.md`](../../../locators/libra_locator.md) — **整个文件就是为这条痛点设计的** |
| 7 处散落点全清单 | libra_locator §三 |
| 上下游切流策略差异 | libra_locator §四（Libra vs TCC + 一致性 hash） |
| "看到 Libra 改动应查啥" 导览 | libra_locator §五 |
| Libra 标准动作模板（增 / 删） | libra_locator §六 |
| **强制必跑** libra_inventory 子动作 | [skill_link_explore](../../../skills/skill_link_explore/SKILL.md) §6 子动作 + [`7_dimension_checklist.md`](../../../skills/skill_link_explore/resources/7_dimension_checklist.md) D5 章节明确 "永远不可省" |
| brainstorm 报告强制 7 行 Libra 表 | [`report_template.md`](../../../report_template.md) §5.2 |

**自检**：✅✅ **强覆盖**（这是本 skill 在所有维度最重视的痛点）

---

### P-A2 子项：SP 加载混乱（同 P-B1，但单独突出）

**痛点描述**（隐含于 P-A2 / B1）：
- 4 层 SP 加载（Picasso / Libra / TCC / Fornax）混乱
- 改 SP 不知道在哪改

**本 skill 应对**：

| 应对维度 | 具体位置 |
|---|---|
| 专门 locator | [`sp_locator.md`](../../../locators/sp_locator.md) |
| 4 层加载链清晰化 | sp_locator §一 |
| **3 处散落代码**（regression 警告） | sp_locator §二 |
| 完整 trace 示例 | sp_locator §四 |
| **强制 sp_inventory 子动作**（含 3 处分散点） | [`7_dimension_checklist.md`](../../../skills/skill_link_explore/resources/7_dimension_checklist.md) D4 |
| Z.2 Gap 表 G3 SP 收敛 regression | [`Z.2 PRD vs 实现 Gap 表.md`](./Z.2%20PRD%20vs%20实现%20Gap%20表.md) |

**自检**：✅ 强覆盖

---

### P-B3：历史存量逻辑不熟 → 编码阶段返工

**痛点描述**：
- 产品 / 新研发对历史存量逻辑不熟
- 需求/方案环节容易遗漏
- 导致编码阶段返工回需求/方案阶段

**本 skill 应对**：

| 应对维度 | 具体位置 |
|---|---|
| 工程架构演进史（5 代） | [`B. Agent 架构专栏/2. 工程架构演进史.md`](../B.%20Agent%20架构专栏/2.%20工程架构演进史.md) |
| 新老架构并存策略（双轨分析必读） | [`B. Agent 架构专栏/5. 新老架构并存策略.md`](../B.%20Agent%20架构专栏/5.%20新老架构并存策略.md) |
| 各模块新老对照表 | B.5 §三 |
| 5 个 locator 顶部 last_synced_commit 字段 | 防漂移 |
| **强制双轨分析**：阶段二的 entry_locate 子动作 | [`7_dimension_checklist.md`](../../../skills/skill_link_explore/resources/7_dimension_checklist.md) D2 |
| Z.2 Gap 表 + 误区纠正 | 防止把历史包袱当 PRD 已落地 |

**自检**：✅ 覆盖

---

### P-D1：PRD 潦草 + 大量隐藏信息在线下

**痛点描述**：
- PRD 普遍潦草
- 大量隐藏信息在线下 / 电话等即时交流中

**本 skill 应对**：

| 应对维度 | 具体位置 |
|---|---|
| skill_request_brainstorm **第三步：额外信息检索** | [`skill_request_brainstorm/SKILL.md`](../../../skills/skill_request_brainstorm/SKILL.md) §第三步 |
| 飞书消息 / 文档检索（沿用 ecom-buy 思路） | 同上 |
| 找人地图（A.团队组织） | 引导主动找 owner 沟通 | [`A. 组织职责与找人地图/团队组织.md`](../A.%20组织职责与找人地图/团队组织.md) |

**自检**：✅ 部分覆盖（**飞书检索工具**本身依赖外部能力，本 skill 仅"约定调用"，不实现工具）

---

### P-D2：CoCo 价值打折（不了解历史包袱 / 切流 / 隐藏知识）

**痛点描述**：
- 王瑶："CoCo 确实能给出较多信息，但由于不了解历史包袱、切流信息、隐藏知识等，导致 CoCo 的信息无法高效利用"

**本 skill 应对**：

| 应对维度 | 具体位置 |
|---|---|
| **置信度机制** | [`skill_request_brainstorm/SKILL.md`](../../../skills/skill_request_brainstorm/SKILL.md) §第四步：每条线索都标"高/中/低"+ 多候选 + 待确认 |
| **代码证据要求** | [`alignment_writing_guide.md`](../../../skills/skill_solution_alignment/resources/alignment_writing_guide.md) — 输出必有 file/path 引用 |
| 知识库（A/B/C/D/E）补充历史包袱 + 切流知识 | 整个 knowledge_repository |
| **5 个 locator 提供"权威知识" vs CoCo 的"通用 LLM 知识"** | locators/ |

**自检**：✅ 覆盖（核心思路：用本 skill 的领域知识**替代**或**校准** CoCo 通用输出）

---

### P-D3：业务逻辑散落、每处改动量小（人改更顺手）

**痛点描述**：
- 王瑶："每处改动量很小（几行）"
- "更倾向于人直接分析和找，改代码就是顺手的事"
- "CoCo 找错就意味着多了一次判断对错的成本"

**本 skill 应对**：

| 应对维度 | 具体位置 |
|---|---|
| **置信度机制** | 高/中/低标注 → RD 一眼知道哪些可信哪些不可信 |
| **brainstorm 不代写代码** | 给 RD 决策依据，不让 AI 直接写"几行改动" |
| 流程图 + 表格清单（不堆路径行号） | RD 自己快速定位，不被堆砌信息分心 |

**自检**：✅ 部分覆盖（设计哲学**就是为这条痛点考虑**：让 AI 提供"决策辅助"而非"代写"，把"几行改动"留给人）

---

### P-D4：Libra 增/删标准化 skill（write 侧）

**痛点描述**：
- 王瑶："Libra 相关代码变更逻辑，希望能有一个标准化的 skill—— 增加一个 Libra 或下线一个 Libra，代码变更模式能统一"

**本 skill 应对**：

| 应对维度 | 具体位置 |
|---|---|
| **明示 scope 外**：本 skill 只做 read 侧 | [`SKILL.md`](../../../SKILL.md) §适用场景 |
| Libra 增/删标准动作模板（**为未来 skill_libra_change 提供雏形**） | [`libra_locator.md`](../../../locators/libra_locator.md) §六 |

**自检**：⚠️ **范围外**（明示 + 提供雏形，未来扩展）

---

### P-E1：多人协作漏同步 → 自测才暴露

**痛点描述**：
- 王英豪："尤其在多人协作的需求里，经常相互忘记同步信息，到编码、自测阶段才暴露冲突"

**本 skill 应对**：

| 应对维度 | 具体位置 |
|---|---|
| **brainstorm 报告 §6.2 业务风险**强制列"隐藏依赖" | [`report_template.md`](../../../report_template.md) §六.2 |
| **找人地图**（A.团队组织）引导明确 owner 清单 | [`A. 组织职责与找人地图/团队组织.md`](../A.%20组织职责与找人地图/团队组织.md) |
| 跨域接口协议变更专栏（边界协议） | [`skill_request_brainstorm/SKILL.md`](../../../skills/skill_request_brainstorm/SKILL.md) §第二步：抗干扰 + 边界协议 |

**自检**：✅ 部分覆盖（核心思路：让 brainstorm 报告**显式列出涉及 owner**，提醒 RD 主动同步）

---

### P-A1：代码库交接 / 局部细节不熟

**痛点描述**：张新健（社区） — "代码库交接，局部细节不熟，评估不足，容易在 Coding / 自测 阶段方案回炉"

**本 skill 应对**：

| 应对维度 | 具体位置 |
|---|---|
| 整个 knowledge_repository 就是为新人 / 交接者设计 | knowledge_repository/ |
| Z.0 知识地图 §二 "新人推荐入门读法" | [`Z.0 知识地图.md`](./Z.0%20知识地图.md) §二.1 |
| 误区纠正 8 条 | [`误区纠正.md`](../A.%20组织职责与找人地图/误区纠正.md) |

**自检**：✅ 部分覆盖（注意：本 skill scope 是创作 agent 链路，社区代码库不在 scope 内）

---

### P-A3：局部屎山纠结重不重构

**痛点描述**：张新健 — "局部屎山代码会纠结重不重构，交付压力大时往往放弃"

**本 skill 应对**：

| 应对维度 | 具体位置 |
|---|---|
| ⚠️ **不直接覆盖**：本 skill 是 brainstorm（决策辅助）非"重构辅助" |
| 但 brainstorm 报告 §7 给出"改动定位建议"——可隐式表达"建议局部不重构，走扩展点" | [`report_template.md`](../../../report_template.md) §七 |
| **不在本 skill 主线 scope** | — |

**自检**：⚠️ 不直接覆盖（决策"重构 vs 不重构"**不属于代码探索 + 知识汇聚**范畴；属另一个 skill 范畴，如 refactoring assistant）

---

### P-林宜葳：文档库维护 & 协同迭代方式

**痛点描述**：林宜葳 — "讨论了中心化管理、feat 分支合入&消费 / main 分支合入&消费。暂无明确倾向，只要不乱就行"

**本 skill 应对**：

| 应对维度 | 具体位置 |
|---|---|
| **隐式覆盖**：5 个 locator 顶部统一 frontmatter（`last_synced_commit` + `verified_against_branch`） | locators/*.md frontmatter |
| **Z.0 知识地图**统一入口 | [`Z.0 知识地图.md`](./Z.0%20知识地图.md) |
| **维护责任**显式说明（每 2-4 周拉飞书白板 JSON） | [`A. 组织职责与找人地图/团队组织.md`](../A.%20组织职责与找人地图/团队组织.md) §七 |

**自检**：✅ 隐式覆盖（提供了"中心化管理 + 静态快照 + 飞书原文档为权威"的方案）

---

### 共性 C1-C4：AI 单测 / CR 标准 / 字节云平台串联 / E2E 评测

**这些是访谈共性问题，但不属"代码探索 + 知识汇聚"范畴**：
- C1 AI 单测 → 单测工具 / coverage 优化 skill 范畴
- C2 CR 标准 → CR 自动化 skill 范畴
- C3 字节云串联 → 平台集成自动化范畴
- C4 E2E 评测 → 评测工具范畴

**自检**：❌ **明示 scope 外**

详见调研报告 `research.md` §7.2 "范围层取舍"（位于本仓库 worktree 之外的主仓 `stone/devclaw_skills_center/feat-dev/2026/04/27/183429-doubao-agent-scene-brainstorm-skill/research.md`）。

---

## 三、覆盖度总结

| 优先级 | 痛点数 | ✅ 强覆盖 | ✅ 覆盖 | ⚠️ 部分 / 范围外 / 不覆盖 |
|---|---|---|---|---|
| P0 | 3 (P-A2, P-B1, P-B2) | 3 | 0 | 0 |
| P1 | 3 (P-B3, P-D1, P-D2) | 1 (D2) | 2 | 0 |
| P2 | 5 (P-D3, P-D4, P-E1, P-A1, P-A3) | 0 | 0 | 5 |
| P-林宜葳 | 1 | 0 | 1（隐式） | 0 |
| 共性 | 4 | — | — | 4（**明示 scope 外**） |

**结论**：

1. **P0 痛点（3 条）：100% 强覆盖** — 核心目标达成
2. **P1 痛点（3 条）：100% 覆盖**（其中 1 条强覆盖，2 条覆盖 + 部分依赖外部能力）
3. **P2 痛点（5 条）**：3 条部分覆盖（隐式）+ 2 条明示范围外
4. **共性问题（4 条）**：100% 明示 scope 外（与本 skill 性质不符）

---

## 四、暴露的潜在改进项（v2 候选）

如果未来要扩展，按优先级：

| # | 候选 | 来自痛点 | 工作量 |
|---|---|---|---|
| V1 | `skill_libra_change`（write 侧） | P-D4（王瑶） | 中 |
| V2 | locator 漂移自动检测 hook | P-林宜葳 | 小 |
| V3 | brainstorm 报告"涉及 owner 清单"自动生成 | P-E1 | 小 |
| V4 | knowledge 与代码 dual-track 持续验证 | P-A1 / P-林宜葳 | 中 |

---

## 五、维护建议

- 每次 skill 更新时**复审本表**：是否新增痛点？是否有痛点应对失效？
- 每 sprint 末跟豆包-创作团队同步：本表能解决你们的问题吗？
- 鼓励 RD 用本 skill 时把"踩到的坑"反馈到对应痛点条目下（v2）
