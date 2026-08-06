---
name: skill_link_explore
description: "豆包-创作 agent 链路 brainstorm 阶段二：按 6 子动作（entry_locate / capability_match / sp_inventory / libra_inventory / tool_chain_trace / block_protocol_check）做链路代码探索，每个功能点独立产出 link_analysis。新老架构必须双轨分析。"
---

<!-- @format -->

# skill_link_explore — 阶段二：链路探索（核心阶段）

本 sub-skill 处理 `skill_request_brainstorm` 产出的 `story.md`，**按功能点粒度逐一**做链路代码探索，每个功能点独立产出一份 `link_analysis-<功能点>.md`。

## 输入

- `workspace/user/story.md`（含功能点 + 5 维度分析线索）

## 前置条件

- 已完成阶段一（`skill_request_brainstorm`）
- 8 仓库 worktree 在工作空间下可达

## 输出

- `workspace/user/link_analysis-<功能点>.md`：每个功能点一份
  - 入口落点 + 代际选择
  - 命中的创作动线
  - **6 子动作产出**

---

## 执行原则（强约束）

参考 ecom-buy 的"入口优先 + 工具约束"：

- **入口优先**：先用 [`../../locators/psm_locator.md`](../../locators/psm_locator.md) 定位入口（C 端 / Picasso × 新 / 老）
- **禁止泛搜**：未确定入口与目录范围前，不准在工作空间做随意 grep
- **新老双轨**：触达 agent 主链路时，**必须** `agent_phase_26/` 与 `handler/phase3handle/` 都查
- **三处 SP 散落点**：触达 SP 时**必须三处都验证**
- **7 处 Libra 散落点**：涉及灰度时**必须查 7 处全清单**

## 必读资源

执行前**必读**：
- [`./resources/link_explore_methodology.md`](./resources/link_explore_methodology.md)（方法论 + Step-by-Step）
- [`./resources/7_dimension_checklist.md`](./resources/7_dimension_checklist.md)（7 维度检查清单）

---

## 6 子动作

每个子动作对应 7 维度矩阵的一个维度（D1 入口在阶段一已定位，本阶段从 D2 开始）：

| 子动作 | 维度 | 必读 | 产出小节 |
|---|---|---|---|
| `entry_locate` | D1（复盘） + D2（代际） | [`psm_locator`](../../locators/psm_locator.md)、[`B.5 新老架构并存`](../../knowledge_repository/doubao_creation/B.%20Agent%20架构专栏/5.%20新老架构并存策略.md) | "入口与代际定位"小节（双轨） |
| `capability_match` | D3（创作能力动线） | [`B.1 业务架构与场景全景`](../../knowledge_repository/doubao_creation/B.%20Agent%20架构专栏/1.%20业务架构与场景全景.md)、[`C. 创作能力动线专栏`](../../knowledge_repository/doubao_creation/C.%20创作能力动线专栏/) | "命中动线"小节（链向 C 文件） |
| `sp_inventory` | D4（SP 4 层） | [`sp_locator`](../../locators/sp_locator.md)、[`D.Fornax`](../../knowledge_repository/doubao_creation/D.%20中间件_基础设施专栏/Fornax.md)、[`D.TCC`](../../knowledge_repository/doubao_creation/D.%20中间件_基础设施专栏/TCC.md) | "SP 触达"小节（3 处分散点） |
| `libra_inventory` | D5（切流） | [`libra_locator`](../../locators/libra_locator.md) | "Libra 切流盘点"小节（7 处全检） |
| `tool_chain_trace` | D6（工具链路） | [`tool_locator`](../../locators/tool_locator.md) | "Tool 链路"小节（4 级链路） |
| `block_protocol_check` | D7（上屏） | [`block_locator`](../../locators/block_locator.md) | "Block 协议"小节 |

---

## 执行步骤

### 第一步：战术反思

加载本 skill 后**立即反思**：
- 本次需求涉及哪些功能点？按什么顺序做？
- 每个功能点要跑完 6 个子动作中哪几个？（**至少跑 4 个**：entry / sp / libra / capability，其余按需）
- 哪个子动作是本次重头戏？
- 预计产出多少份 link_analysis？

### 第二步：按功能点逐一执行

**关键约束**：一个功能点完整跑完 6 子动作，再进入下一个；**不要把多个功能点的子动作混着跑**。

对每个功能点：

1. **entry_locate**（D1+D2）：定位入口和代际
2. **capability_match**（D3）：命中哪条创作动线（链向 C 文件）
3. **sp_inventory**（D4）：SP 触达评估（3 处分散点）
4. **libra_inventory**（D5）：Libra 切流盘点（7 处全检）
5. **tool_chain_trace**（D6）：Tool 链路（4 级链路）
6. **block_protocol_check**（D7）：Block 协议

每个子动作输出格式（见 [`./resources/link_explore_methodology.md`](./resources/link_explore_methodology.md) §三）：
- 1 张 mermaid（时序图 / 调用拓扑 / 决策树之一）
- 1 个配置 / 代码定位清单（表格）
- 1 段"风险与遗漏点"

### 第三步：产出 link_analysis 文档

**文件命名**：`workspace/user/link_analysis-<功能点>.md`

**内容结构**：

```markdown
# link_analysis: <功能点标题>

> 关联 story.md @ commit
> 复杂度定级: CL<X>

## 1. 入口与代际定位（entry_locate）
（mermaid + 双轨清单）

## 2. 命中动线（capability_match）
（链向 C 文件 + 该动线的本期改造影响）

## 3. SP 触达盘点（sp_inventory）
（4 层 ① ② ③ ④ 触达表 + 3 处散落代码 + SP key 列表）

## 4. Libra 切流盘点（libra_inventory）
（7 处散落表 + 当前切流配置 + 上下游策略选择）

## 5. Tool 链路（tool_chain_trace）
（4 级链路：schema → 网关 → DAG → handler）

## 6. Block 协议（block_protocol_check）
（命中 Block 类型 + 同步异步 + WriterPacket 协议影响）

## 7. 风险与遗漏点
（综合 6 子动作的风险点）
```

### 第四步：反思补充

执行完所有功能点后，做一次综合反思：
- 是否所有功能点都覆盖？
- 是否有跨功能点的共性发现？（如多个功能点都触达 SP regression）
- 是否所有触发的子动作都给出 mermaid + 清单 + 风险点？

---

## 质量检查清单

输出前必须检查：

**基础**：
- [ ] 文件命名 `link_analysis-<功能点>.md`
- [ ] 所有功能点都有独立 link_analysis
- [ ] 不同功能点未混杂在同一个文档

**入口定位**：
- [ ] C 端 / Picasso 双入口至少明确标注
- [ ] 新老架构双轨分析（agent_phase_26 + phase3 / 老 Plan-Act）
- [ ] 当前灰度比例已查（去 Libra / TCC 平台）

**SP 专项**：
- [ ] 4 层加载触达情况评估完整
- [ ] **3 处分散代码**位置都已确认
- [ ] SP key 列表给出（推断 / 待验证标注）

**Libra 专项（P0）**：
- [ ] **7 处散落点**全数覆盖（即使无关也明示"不触达"）
- [ ] 上下游切流策略差异（Libra vs TCC）已显式说明
- [ ] 实验 key 命名建议遵循 `creation_<能力>_<场景>_v<版本>`

**Tool 专项**：
- [ ] 4 级链路（schema → 网关 → DAG → handler）至少 1 个 Tool 完整
- [ ] Sync vs Async 决策给出
- [ ] ResourceID 跨层引用一致性已检查

**Block 专项**：
- [ ] 命中的 Block 类型清单
- [ ] 同步 / 异步上屏决策
- [ ] 父子 Block 关系（如有 ThinkingBlock）

**写作**：
- [ ] 严禁堆砌"路径+行号"调试信息（按写作指导整合）
- [ ] mermaid 渲染通过
- [ ] 表格化清单 / 流程图 / 风险点三件套齐全

---

## 与 ecom-buy `skill_code_analysis` 的关键差异

| 维度 | ecom-buy code_analysis | doubao link_explore |
|---|---|---|
| 粒度 | PSM + API | 功能点（融合 PSM + 入口 + 代际） |
| 子动作数 | 4 步骤（PSM 定位 / 代码梳理 / 产出 / 反思） | 6 子动作（entry / capability / sp / libra / tool / block） |
| 必查的散落点 | 配置项 / RPC 依赖 / 模型 | **3 处 SP / 7 处 Libra**（P0 痛点） |
| 双轨分析 | 不需要（无新老并存） | **必须**（agent_phase_26 + phase3） |
| 创作动线 | 不存在 | C 类目 13 条动线必查 |
