---
name: skill_request_brainstorm
description: "豆包-创作 agent 链路 brainstorm 阶段一：从 PRD/需求拆解功能点 + 复杂度定级 + 5 维度定位线索（PSM/SP/Tool/Libra/Block）。当用户带 PRD 进入 brainstorm 流程的第一阶段时使用。"
---

<!-- @format -->

# skill_request_brainstorm — 阶段一：需求拆解

本 sub-skill 处理用户的原始 PRD / 需求，按豆包-创作 agent 链路的特性拆解为功能点，并产出 5 维度定位线索。

## 输入

- 用户原始 PRD / 需求（文本 / 飞书链接 / `workspace/downloads/`）
- 用户的部门、业务线（必须明示）

## 前置条件

- 已进入阶段一（由 `../../guidance.md` 协调）
- 已完成 scope 自检（需求确实落在豆包-创作 agent 链路）

## 输出

- `workspace/user/story.md`：含
  - 需求名 / 业务线 / 子方向
  - 功能清单（按 "模块 ｜ 动作 ｜ 标题" 格式）
  - 每个功能点的：变更背景 / 核心逻辑 / 影响分析 / **5 维度分析线索** / **复杂度定级**

---

## 执行步骤

### 第一步：框架提取

1. 阅读 PRD 全文
2. 阅读 [`../../knowledge_repository/doubao_creation/A. 组织职责与找人地图/团队组织.md`](../../knowledge_repository/doubao_creation/A.%20组织职责与找人地图/团队组织.md) 中的业务域职责定义
3. 提取**原子化且逻辑完整**的功能点

输出格式：

```markdown
## 【需求名】

业务线：豆包-创作 / 子方向：<生图 / 视频 / 修图 / 分身 / ...>

### 功能清单

1. 【模块】｜【动作】｜【功能点标题】
2. ...
```

提取规则：
- **模块**：业务模块（如"文生图主链路"、"图生图风格化"）
- **动作**：必须是新增 / 删除 / 修改 / 其他
- **功能点标题**：30 字以内，**优先取自原文**

> **scope 提醒**：本 skill 仅处理豆包-创作 agent 链路功能点。其他子方向（社区 / 海外 / 分发&消费 / 商业化）应明示"超出本 skill scope"。

### 第二步：内容填充

为每个功能点填充详细描述：

```markdown
### 【模块】｜【动作】｜【功能点标题】

#### 变更背景
为什么要做这个改动。

#### 核心逻辑
详细描述改动规则、公式、流程。

#### 影响分析
对现有功能、数据、关联模块的影响。
```

填充规则：
- **原文保留**：计算公式 / 业务口径 / 复杂逻辑判断**原样保留**，不摘要
- **标 (见原文)**：涉及核心计算公式或状态机流转的，句尾加 `(见原文)`
- **抗干扰**：忽略归属其他业务线 / 子方向的改动

### 第三步：额外信息检索

调用飞书检索工具检索一轮：
- 需求相关的前期沟通（飞书消息）
- 评审会纪要
- 历史相关需求迭代

如果检索到关键背景，向 `story.md` 追加 **背景信息** 章节：

```markdown
### 背景信息
【信息来源】【背景信息内容】
```

### 第四步：5 维度定位线索（**核心改造**，与 ecom-buy 的差异点）

为每个功能点生成 **5 维度定位线索**：

| 维度 | 来源 locator | 给出什么 |
|---|---|---|
| **PSM** | [`psm_locator.md`](../../locators/psm_locator.md) | 触发的入口 PSM + 第一落点 |
| **SP** | [`sp_locator.md`](../../locators/sp_locator.md) | 是否触达 SP？哪些 SP key？是否走 4 层加载 |
| **Tool** | [`tool_locator.md`](../../locators/tool_locator.md) | 是否触达 Tool？哪些工具？schema 路径 |
| **Libra** | [`libra_locator.md`](../../locators/libra_locator.md) | 是否触达 Libra 灰度？哪些散落点？ |
| **Block** | [`block_locator.md`](../../locators/block_locator.md) | 是否触达上屏？哪些 Block 类型？ |

输出格式：

```markdown
#### 分析线索

- **PSM**:
  - 入口候选: `flow.agent.creation::AgentStream`（C 端）/ `flow.alice.creation_access::ExecuteAgent`（Picasso）
  - 第一落点: `creation_agent/handler.go::AgentServiceImpl`
  - 置信度: 高
  - 定位依据: 关键词"文生图"+"主bot 闲聊"匹配 → C 端入口
- **SP**:
  - 是否触达: ✅
  - SP key 候选: T2IDivergent / 主 ReAct SP
  - 4 层加载点: 改 SP 必须 3 处验证
  - 置信度: 中
- **Tool**:
  - 是否触达: ✅
  - 候选: image_gen
  - schema 路径: aigc_management/.../schema/doubao/doubao_text2image.json
  - 置信度: 高
- **Libra**:
  - 是否触达: ✅（灰度需求）
  - 候选散落点: SupportAgent26（# 1）、aigc_dag/biz/common/ab.go（# 6）
  - 切流策略选择: 上游 Libra vs 下游 TCC
  - 置信度: 高
- **Block**:
  - 是否触达: ❓
  - 候选: CreationBlock / LoadingBlock
  - 置信度: 中
```

**置信度规则**：
- **高**：locator 中关键词直接匹配 + 业务场景明确
- **中**：locator 中关键词部分匹配 + 业务场景需推断
- **低**：locator 中无关键词匹配 + 需多候选

### 第五步：复杂度定级

参考 ecom-buy 的 CL1-CL3 标准（豆包业务可适当调整阈值）：

| 等级 | 定义 | 判定标准 |
|---|---|---|
| **CL1** 简单 | 单模块改动 | ≤ 3 个改动点；≤ 1 条创作动线；不触达架构代际灰度 |
| **CL2** 中等 | 跨模块改动 | 4-10 改动点；2-3 条动线；触达 Libra 单一切流 |
| **CL3** 复杂 | 核心链路重构 | > 10 改动点；> 3 条动线；触达架构代际或多 Libra 切流 |

把定级结果写到 `story.md` 第一行：

```markdown
需求定级：CL2 中等需求
```

---

## 质量检查清单

输出前必须检查：

**基础**：
- [ ] 所有功能点已提取并填充
- [ ] 功能点描述尊重原文，未不当改写
- [ ] 格式规范（模块 ｜ 动作 ｜ 标题）
- [ ] 每个功能点都有 5 维度分析线索

**5 维度专项**：
- [ ] PSM 候选明确（C端 / Picasso 双入口至少标注其一）
- [ ] SP 触达评估明确（高/中/低 + 理由）
- [ ] Tool 候选给出至少一个
- [ ] **Libra 必查**（即使是"无 Libra"也要明确写"❌ 不触达"）
- [ ] Block 触达评估给出
- [ ] 置信度标注完整

**写作约束**：
- [ ] 计算公式 / 业务口径已原样保留
- [ ] 涉及核心逻辑的句尾有 (见原文)
- [ ] 信息全部来自 PRD / 检索结果，无自行脑补
- [ ] 业务线 / 子方向明示

**定级**：
- [ ] 定级有明确依据
- [ ] 定级标准与表格定义一致

---

## 错误处理

- 如果 PRD 不在 doubao-agent scope（如属社区 / 海外）→ 提示用户改用对应 skill / 直接对话
- 如果 PRD 极简（单文档不足以拆解功能点）→ 主动检索补充
- 如果 5 维度任一无法判断 → 标"待确认"+ 列多候选

---

## 与 ecom-buy 的关键差异

| 维度 | ecom-buy story_extraction | doubao request_brainstorm |
|---|---|---|
| 第四步 | 1 维（API 定位） | **5 维（PSM/SP/Tool/Libra/Block）** |
| 业务域职责依据 | A.团队组织架构 | A.团队组织 + 误区纠正 |
| Libra 关注度 | 一般 | **P0 痛点核心** |
| 输出 story.md 章节 | 4 段 | 5 段（多了"分析线索"） |
