# 链路探索方法论 — 豆包-创作 agent 链路

> 对标 ecom-buy `skill_code_analysis/resources/普通架构代码分析工具.md`，但全部豆包化。

---

## 一、核心方法论：「定位入口 → 双轨追踪 → 7 维提取」

```
Step 0  前置确认
Step 1  入口与代际定位（D1+D2）
Step 2  命中动线（D3）
Step 3  双轨追踪（新老链路同时走）
Step 4  关键模块逐项盘点（D4 SP / D5 Libra / D6 Tool / D7 Block）
Step 5  风险与遗漏点
```

### 核心原则

1. **确定性坐标优先**：先定位 PSM + 入口接口，建立分析边界
2. **双轨同时走**：agent_phase_26（新）+ phase3 / 老 Plan-Act（老）—— 不能只看一条
3. **5 个 locator 是地图**：禁止跳过 locator 直接 grep
4. **链路追踪**：关注数据流转（AgentPayload → ContextMessage → ToolCall → Block），而非孤立函数
5. **工具约束**：grep 必须限定 path 参数；type=go；禁止全工作空间泛搜

---

## 二、Step-by-Step

### Step 0：前置确认

- [ ] 已读 `workspace/user/story.md` 中的功能点和 5 维度分析线索
- [ ] 8 仓库 worktree 路径已记下（来自 `feat-design/background/info.md`）
- [ ] 5 个 locator 都打开（`../../locators/*`）
- [ ] B.3 ReAct 五层架构 + B.5 新老并存策略 已读

如果上述未完成，**回到阶段一或读 knowledge_repository**，不要硬上。

---

### Step 1：入口与代际定位（子动作 entry_locate）

#### 1.1 流量入口判定（D1）

来自 [`../../../locators/psm_locator.md`](../../../locators/psm_locator.md)：

```mermaid
flowchart TD
    A[需求关键词] --> B{命中"评测/picasso/离线"?}
    B -- 是 --> C[Picasso 入口<br/>creation_access::ExecuteAgent]
    B -- 否 --> D[C 端入口<br/>flow.agent.creation::AgentStream]
    C --> E[转发 → flow.agent.creation_evaluation]
    D --> F{handler.go 顶部判断}
    E --> F
    F -- SupportAgent26 命中 --> G[新 ReAct<br/>agent_phase_26]
    F -- 未命中 --> H[老 Plan-Act<br/>handler/phase3handle]
```

#### 1.2 架构代际判定（D2）

判定当前需求会落到新链路、老链路、还是双轨都触达：

| 触达 | 何时 |
|---|---|
| 仅新（agent_phase_26） | 灰度 100% 已完成（少见，可能 PRD 描述强制走新链路） |
| 仅老（phase3 / 老 Plan-Act） | 改 bug 修老链路 |
| **双轨都触达** | **常态**：灰度未 100% 时新需求 |

#### 1.3 输出

```markdown
## 1. 入口与代际定位

| 维度 | 落点 |
|---|---|
| 流量入口 | C 端（AgentStream）/ Picasso（ExecuteAgent）/ 双侧 |
| 架构代际 | 新 ReAct / 老 Plan-Act / **双轨** |
| 当前灰度比例 | X%（去 Libra / TCC 平台查） |
| 触发条件 | 例如 `option.SupportAgent26(ctx) && tenant_id == X` |
```

mermaid 流程图必出（参考上面 1.1）。

---

### Step 2：命中创作动线（子动作 capability_match）

#### 2.1 关键词命中

按需求关键词查 [`../../../knowledge_repository/doubao_creation/C. 创作能力动线专栏/0. 动线总览.md`](../../../knowledge_repository/doubao_creation/C.%20创作能力动线专栏/0.%20动线总览.md) 矩阵：

| 关键词 | 命中动线 |
|---|---|
| 文生图 / 画 / 生图 | 文生图/* |
| 改图 / 图编辑 / 风格化 | 图生图/* |
| 修图 / 扩图 / 消除 | AI 修图/* |
| 分身 / 写真 | 分身写真/* |
| 视频 / 动画 | 文生视频_图生视频/* |
| 音乐 / 歌 | 文生音乐/*（**scope 提醒**） |

#### 2.2 链向具体动线

读对应动线文件的 7 章节，把"用户动线时序图"复制到 link_analysis 作为底图。

#### 2.3 输出

```markdown
## 2. 命中动线

| 命中动线 | 文件 | 该动线本期 PRD 改造 |
|---|---|---|
| 文生图/主bot 闲聊入口 | C/文生图/主bot 闲聊入口.md | ✅ 完整命中 |

（复制该动线的"用户动线时序图"，用红色高亮本功能点改动位置）
```

---

### Step 3：双轨追踪

参考 [`../../../knowledge_repository/doubao_creation/B. Agent 架构专栏/5. 新老架构并存策略.md`](../../../knowledge_repository/doubao_creation/B.%20Agent%20架构专栏/5.%20新老架构并存策略.md) §三 的"各模块新老对照表"，对每个触达模块都写两份位置。

---

### Step 4：关键模块逐项盘点

#### 4.1 子动作 sp_inventory（D4）

参考 [`../../../locators/sp_locator.md`](../../../locators/sp_locator.md)。

输出表（必出）：

```markdown
## 3. SP 触达盘点

### 4 层加载触达
| 层 | 命中 | 验证位置 |
|---|---|---|
| ① Picasso 注入 | ✅/❌ | AgentPayload.doubao_agent_config_v2_data |
| ② Libra 实验 | ✅/❌ | creation_evaluation/constant/ab_params.go |
| ③ TCC 兜底 | ✅/❌ | internal/util/tcc/base.go |
| ④ Fornax 默认 | ✅/❌ | internal/rpc/fornax/ |

### 3 处散落代码（**必查**）
- 旧: internal/rpc/agent_util/sp.go::LoadSPWithDefault
- phase3: internal/rpc/phase3/agent_runtime/sp_manager.go::SystemPromptManager
- 新: agent_phase_26/agent/sp/agent_sp.go

### SP key 候选
- PlannerSpKey / T2IDivergent / ...（按需推断）

### Gap 提醒
- 本期 SP 收敛 = regression（多增一处）
- 多租户 SP 当前 = gap（不支持）
```

#### 4.2 子动作 libra_inventory（D5，**P0 痛点**）

参考 [`../../../locators/libra_locator.md`](../../../locators/libra_locator.md) 7 处散落点。

输出表（**强制 7 行全列**）：

```markdown
## 4. Libra 切流盘点

| # | 散落点 | 命中本需求 | 当前切流配置 |
|---|---|---|---|
| 1 | creation_agent/handler/handler.go::SupportAgent26 | ✅/❌ | 实验 key = ?, 比例 = ? |
| 2 | SupportAgent26Ark | ✅/❌ | ... |
| 3 | creation_access/constant/libra.go | ✅/❌ | ... |
| 4 | creation_evaluation/constant/ab_params.go | ✅/❌ | ... |
| 5 | phase3/agent_runtime/sp_manager.go (SP 来源) | ✅/❌ | ... |
| 6 | aigc_dag/biz/common/ab.go | ✅/❌ | ... |
| 7 | aigc_tool/dal/dag_switch.go | ✅/❌ | ... |

### 上下游切流策略选择
- 上游（Libra）vs 下游（TCC + 一致性 hash）
- 本需求建议走: Libra / TCC
- 实验命名建议: creation_<能力>_<场景>_v<版本>

### 下线计划（**必填**）
- 实验下线条件: ...
- 代码 dead code 清理时机: ...
```

#### 4.3 子动作 tool_chain_trace（D6）

参考 [`../../../locators/tool_locator.md`](../../../locators/tool_locator.md) §二 4 级链路。

输出（按命中工具逐一）：

```markdown
## 5. Tool 链路

### Tool: image_gen
| 级 | 仓库 | 路径 | 改动 |
|---|---|---|---|
| ① schema | aigc_management | service/domain/ability/schema/doubao/doubao_text2image.json | 加 close_watermark 字段 |
| ② 网关 | aigc_tool | method/sync_invoke_tool.go | 不变 |
| ③ DAG topic | aigc_management | （读 ListToolVersions） | 升 version |
| ④ handler | aigc_dag | biz/handler_sync/doubao/text2image.go | 消费 close_watermark |

### Sync vs Async
- 选择: Sync / Async
- 理由: ...

### ResourceID 跨层一致性
- 检查: image_gen_X 在模型/工具/Memory/Block 各层一致 ✅
```

#### 4.4 子动作 block_protocol_check（D7）

参考 [`../../../locators/block_locator.md`](../../../locators/block_locator.md)。

输出：

```markdown
## 6. Block 协议

| Block 类型 | 命中 | 改动 |
|---|---|---|
| CreationBlock (2074) | ✅ | 渲染水印标识 |
| LoadingBlock (10101) | ❌ | — |
| ThinkingBlock (10040) | ❌ | — |
| TextBlock | ❌ | — |
| ButtonBlock (10103) | ❌ | — |

### 同步 vs 异步上屏
- 本功能点: 同步上屏（生图常态）
- IsAsyncReach 标记: false

### 父子 Block 关系
- 不涉及（仅 ReAct ThinkingBlock 有此结构）
```

---

### Step 5：风险与遗漏点

每个 link_analysis 末尾综合：

```markdown
## 7. 风险与遗漏点

### 隐藏依赖
- ...（来自 A.误区纠正 / Z.2 Gap 表 / 历史包袱）

### 新老并存影响
- ...

### 数据兼容
- Memory / 上屏协议存量数据是否受影响

### 强弱依赖变更
- 是否引入新的强依赖下游 PSM
```

---

## 三、子动作通用输出三件套

每个子动作必出：

1. **mermaid 图**（流程图 / 时序图 / 调用拓扑 / 决策树之一）
2. **配置 / 代码定位清单**（表格化，禁止堆砌路径行号）
3. **风险与遗漏点**（≥ 2 条）

---

## 四、防呆检查（必看）

| ❌ 错误 | ✅ 正确 |
|---|---|
| 全工作空间 grep "image_gen" | 限定 path = `creation_agent/agent_phase_26/agent/runtime/tool/` 后 grep |
| 只看 agent_phase_26 不看 phase3 | 双轨都查 |
| SP 只看 1 处 | 3 处都查 |
| Libra 只查关键词 | 7 处全列 |
| 输出"foo.go:L10-L20"堆砌 | 流程图 + 表格清单 |
| 用字母 A/B/C 编号改动点 | 用功能点命名 |

---

## 五、与 ecom-buy `普通架构代码分析工具.md` 的关键差异

| 维度 | ecom-buy | 豆包 link_explore |
|---|---|---|
| 阶段 | 7 步骤 | 5 步骤（重组）|
| 入口定位 | PSM + API + Repo | PSM + 入口 + 代际（3 维） |
| 必查清单 | 配置 / RPC / DB / 中间件 | **3 处 SP + 7 处 Libra**（P0） |
| 双轨分析 | 不存在 | **必须** |
| 创作动线 | 不存在 | C 矩阵 13 条 |
| 模型 / Tool 链路 | 标准 service-dao | 4 级跨仓库链路 |
