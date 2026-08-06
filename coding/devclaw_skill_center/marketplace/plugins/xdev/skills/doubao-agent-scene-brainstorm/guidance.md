<!-- @format -->

# 豆包-创作 Agent 链路 Brainstorm 执行流程

## 执行原则

> **当你被要求基于 PRD / 需求分析豆包-创作 agent 链路时，必须严格按本文件要求设计执行阶段和执行计划，阶段是宏观大步骤，每个阶段内你必须按 sub-skill 的指导规划子执行流程**
> **请你必须严格按照以下步骤拆分阶段执行，每次聚焦在当前阶段的产出，前一阶段的任务没有结束前，严禁执行下一阶段的内容**
> **严禁一次性加载所有 sub-skill，必须进入某阶段后才加载该阶段对应的 sub-skill**
> **每个 sub-skill 加载完成后，必须立即执行一次战术反思，明确本阶段的具体执行计划（要做什么、按什么顺序做、产出是什么）**
> **阶段完成后可以按 token 消耗量选择是否进行压缩**

## 输入

- **用户原始 PRD / 需求**：文本、飞书文档链接、或工作区下 `workspace/downloads/` 的下载件
- **用户的部门、业务线、子方向**：原始问题中应该有这个信息（用于在阶段一判定 scope）

## 工作前置：scope 自检

进入阶段一之前，**必须先确认需求落在豆包-创作 agent 链路**：

- 需求关键词命中：生图 / 生视频 / AI 修图 / 分身写真 / 文生音乐 / Action Bar / 闲聊指令集 / 多模态创作 / Plan-Act / ReAct / Picasso 评测 / Libra 切流
- 需求触达的 PSM 在 8 仓库范围内（见 [`SKILL.md`](./SKILL.md) "覆盖的 8 个仓库"）

如果不在 scope 内，**主动告知用户**："此需求似乎不在豆包-创作 agent scope，建议改用 XX skill / 直接对话"，**不要强行套用本 skill**。

---

## 工作流程

### 阶段一：需求拆解

**执行步骤**：

1. **加载 sub-skill**：`skills/skill_request_brainstorm/SKILL.md`
2. **战术反思**（加载完后立即做）：
   - 本次需求涉及的创作能力是什么？（文生图 / 图生图 / AI 修图 / 分身写真 / 视频 / 音乐）
   - 需求落在哪条入口？（C 端 AgentStream / Picasso ExecuteAgent）
   - 是否触达 SP / Libra / Tool / Block 任意一条敏感线？
   - 预计拆解出多少功能点？
   - 执行顺序是什么？
3. **执行**：严格按 sub-skill 指示的子流程
4. **验收**：所有功能点都已拆解 + 都有 5 维度定位线索 + 都有复杂度定级

**阶段输出**：

- `workspace/user/story.md`：含
  - 需求名 / 业务线 / 子方向
  - 功能清单（按"模块 ｜ 动作 ｜ 标题"格式）
  - 每个功能点的：变更背景 / 核心逻辑 / 影响分析 / **5 维度分析线索**（PSM / SP / Tool / Libra / Block）
  - 复杂度定级（CL1 / CL2 / CL3）

---

### 阶段二：链路探索（核心阶段）

**执行步骤**：

1. **加载 sub-skill**：`skills/skill_link_explore/SKILL.md`
2. **战术反思**（加载完后立即做）：
   - 本次需求涉及哪些功能点？按什么顺序做链路探索？
   - 每个功能点要跑完 6 个子动作（entry_locate / capability_match / sp_inventory / libra_inventory / tool_chain_trace / block_protocol_check）中哪几个？
   - 哪个子动作是本次重头戏？
   - 预计产出多少份 link_analysis 文档？
3. **执行**：严格按 sub-skill 指示，**按功能点粒度逐一**做链路探索
4. **验收**：反思是否所有功能点都覆盖、是否所有触发的子动作都给出 mermaid 图 + 定位清单 + 风险点

**阶段要求**：

- **入口优先**：先用 [`locators/psm_locator.md`](./locators/psm_locator.md) 定位入口（C 端 / Picasso × 新 / 老），再沿调用链读
- **禁止泛化检索**：未确定入口与目录范围前，不准在工作空间做随意 grep
- **新老双轨**：如果功能点涉及 agent 主链路，**必须同时分析** `agent_phase_26/`（新 ReAct）和 `handler/phase3handle/`（老 Plan-Act），避免误判
- **三处 SP 分散点**：如果功能点触达 SP，**必须在三处都验证**（旧 `internal/rpc/agent_util/sp.go` + phase3 `sp_manager.go` + 新 `agent_phase_26/agent/sp/`）
- **7 处 Libra 散落点**：如果功能点涉及灰度，**必须查 [`locators/libra_locator.md`](./locators/libra_locator.md) 7 处全清单**

**阶段输出**：

- `workspace/user/link_analysis-<功能点>.md`：每个功能点一份
  - 入口落点 + 代际选择
  - 命中的创作动线（链向 `knowledge_repository/.../C. 创作能力动线专栏/`）
  - 6 子动作产出（每个含：mermaid 时序图 / 配置代码定位清单 / 风险点）

---

### 阶段三：方案对齐（brainstorm 核心发现）

**执行步骤**：

1. **加载 sub-skill**：`skills/skill_solution_alignment/SKILL.md`
2. **战术反思**（加载完后立即做）：
   - 本次的 link_analysis 总共产出几份？
   - 是否有跨功能点的共性发现？
   - 是否有命中 `Z.2 PRD vs 实现 Gap 表` 的项？
   - 给 RD 的"方案方向建议"应该重点提醒哪些事？
3. **执行**：严格按 sub-skill 指示整合并产出 brainstorm 核心发现
4. **验收**：
   - 文档**不含**"我建议这样写代码 / 你应该改 XX 函数"等代写方案语气
   - 文档**包含**：需求复述 / 入口与代际 / 命中动线 / 链路结论 / 三处分散点盘点 / Gap 与风险 / 方案方向建议
   - 文档对照了 `Z.2 Gap 表`

**阶段要求**：

- **不代写技术方案**。本 skill 角色是"决策辅助"，方案交给 RD。
- **强制对照 Gap 表**：从 `knowledge_repository/.../Z. 主索引/Z.2 PRD vs 实现 Gap 表.md` 取出本次需求可能撞上的 Gap 项（多租户 / 错误码 / SP 收敛等），**显式提醒 RD**。
- **方案方向建议给"方向"，不给"代码"**：例如"建议在 X 模块用扩展点而非修改核心链路"，而非"建议修改 foo.go::bar() 第 N 行"。

**阶段输出**：

- `workspace/artifacts/research/brainstorm_core_findings.md`：brainstorm 核心发现
  - 完整覆盖 [`report_template.md`](./report_template.md) 的 7 章节
  - **独立可读**：不能引用 `workspace/user/` 下的中间产物文档，所有内容已整合

---

## 阶段间的硬约束

| 约束 | 说明 |
|---|---|
| 阶段单向不可逆 | 进入阶段二后不再回头改 story.md（除非用户主动要求） |
| sub-skill 不串读 | 阶段一只读 skill_request_brainstorm；不许一次把 4 个 sub-skill 全读了 |
| 战术反思强制 | 每个 sub-skill 加载后第一件事是反思，**不允许直接执行** |
| 7 维度对齐 | 阶段二的 6 子动作覆盖 D2-D7 维度（D1 入口已在阶段一定位） |
| 防代写 | 阶段三**严禁**输出代码片段；只能输出方向、建议、风险 |

---

## 与 ecom-buy-skills 的执行差异

| 维度 | ecom-buy | doubao-agent-scene-brainstorm |
|---|---|---|
| 阶段名 | story_extraction / code_analysis / solution_design | request_brainstorm / link_explore / solution_alignment |
| 阶段三定位 | 输出技术方案文档 | 输出 brainstorm 核心发现（不代写） |
| 定位粒度 | PSM+API 二维 | 入口×代际×动线 三维 |
| 链路探索维度 | 普通架构 / 交易模式二选一 | 6 子动作（覆盖 7 维度的 D2-D7） |
| 写作模板 | 报告模板（技术方案） | brainstorm 报告模板（决策辅助） |
