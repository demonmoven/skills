# Z.2 PRD vs 实现 Gap 表

> ⚠️ 本表是 brainstorm skill 的"诚实"表——把 PRD 设计与本期 diff 实测的 gap 显式列出，避免 RD 写方案时**误以为 PRD 设计已落地**。
>
> 维护原则：每次同步代码时更新；新发现的 gap 加进来；落地的 gap 状态从 `gap` 改为 `done`，但保留记录。

---

## 状态枚举

- **gap**：PRD 设计了但**完全未实现**
- **partial**：PRD 设计了，本期**部分实现**，仍有改进空间
- **regression**：本期改动与 PRD 设计**反向**（劣化）
- **silent_breaking**：本期 IDL / 协议改动可能破坏存量调用方但**未明示**
- **undocumented_change**：本期改了但 PRD 没明说
- **done**：已完成（仅供历史追溯）

---

## 一、Gap 项总览

| # | Gap 项 | 状态 | 影响范围 | 详情 |
|---|---|---|---|---|
| G1 | 多租户隔离 | **gap** | Agent 逻辑层全部 | § 二 |
| G2 | 错误码统一（flow/ocerr） | **gap** | 8 仓库全部 | § 三 |
| G3 | SP 管理收敛 | **regression** | creation_agent | § 四 |
| G4 | DAG 异步转同步内部逻辑 | **undocumented_change** | aigc_tool | § 五 |
| G5 | ToolInfo 协议简化 | **silent_breaking** | alice_idl 上下游 | § 六 |
| G6 | 工具 schema JSON 化 | **partial / undocumented** | aigc_management | § 七 |
| G7 | 上屏 Block 流式块级管理 | **undocumented_change** | agent_phase_26/writer/ | § 八 |
| G8 | Memory 大 key 倾向 | **partial / known tech-debt** | Abase Memory | § 九 |
| G9 | RAG / Skills 设计 | **gap (P1, 后续)** | agent_phase_26/agent/ | § 十 |

---

## 二、G1 多租户隔离

**PRD 设计**：
- 「设计时需要考虑多租户的逻辑，抽象出对应的模块，支持按照租户维度进行配置 or 豁免」
- 「举例：AB 参数的解析处理逻辑需要收敛，同时需要给出后续固化推全的 SOP（支持多租户隔离 or 通用的配置）」

**实际**：
- `agent_phase_26/agent/sp/sp.md` 自述："**当前有意不考虑** bizID/tenantID，仅实现单租户 SP 选择"
- 例外：`aigc_tool/infra/tcc.go` 新增了租户级超时配置（少数落地点）
- 路由层 / 权限切面的多租户能力**未见明确代码**

**对 RD 的影响**：
- 如果你的需求需要"按租户分发不同 SP / Tool / Block" → **必须自行补隔离逻辑**
- 不要假设 sp_manager / Tool / aigc_management 已经支持租户级配置（除 aigc_tool/infra/tcc.go 例外）

---

## 三、G2 错误码统一（flow/ocerr）

**PRD 设计**：
- 「统一定义好错误码规范，维护在 flow/ocerr 仓库中，每个 function 对外抛出 Error 时尽量使用错误码，避免使用文本的方式」

**实际**：
- 8 仓库 diff 中**未见** `flow/ocerr` 的 import
- 错误透传仅通过 IDL 新增的 `biz_err_status_code` / `biz_err_status_msg` 字段
- 各模块（dag / tool / management）可能各有局部错误定义，**未做统一汇聚**

**对 RD 的影响**：
- 不要 `import "flow/ocerr"`
- 走当前的 `biz_err_status_code/msg` 透传
- 错误码语义需要与下游对齐（评审会显式确认）

---

## 四、G3 SP 管理收敛 ★（regression）

**PRD 设计**：
- 「SP 管理逻辑收敛，并且支持实验等方式动态配置」
- 整体流程：Picasso > AB > TCC > Fornax 4 层加载

**实际**：
- 本期 SP 加载**多增一处**，现在散落在 **3 处**：
  - 旧：`internal/rpc/agent_util/sp.go::LoadSPWithDefault`
  - phase3：`internal/rpc/phase3/agent_runtime/sp_manager.go::SystemPromptManager`
  - 新（本期）：`agent_phase_26/agent/sp/agent_sp.go`
- 4 层优先级在新旧实现都是各自独立写的，不复用

**对 RD 的影响**：
- 改 SP **必须三处都验证**（不能假设统一）
- 详见 [`../../../locators/sp_locator.md`](../../../locators/sp_locator.md)
- 想做"SP 收敛"本身的需求 → 这是一个独立的大项目，不是顺手做

---

## 五、G4 DAG 异步转同步内部逻辑（undocumented）

**PRD 设计**：未提及。

**实际**：
- IDL `tool/service.thrift` 的 `SyncInvokeTool` 接口注释明确："当前接口内部有包含 'DAG异步转同步' 的逻辑"
- 这是性能优化，但**没有独立设计文档**

**对 RD 的影响**：
- 选 sync vs async 时要知道：sync 内部其实是把 async 等回来
- 如果 DAG 执行很慢，不要选 sync（等不动）

---

## 六、G5 ToolInfo 协议简化（silent_breaking）

**PRD 设计**：未提及。

**实际**：
- `alice_idl/thrift/alice_creativity/creation_tool.thrift` 把 `ToolInfo.properties` 从 `map<string, ToolParamInfo>` 简化为 `ToolParamInfo` 对象
- 旧客户端如果依赖 `properties` 是 map → **解析破坏**

**对 RD 的影响**：
- 本期改动可能影响第三方调用 ToolInfo 的代码
- 如果你的需求消费 ToolInfo → 检查是否仍按 map 取值

---

## 七、G6 工具 schema JSON 化（partial）

**PRD 设计**：「工具的 schema 配置（管理 + 部署）」

**实际**：
- 新增 8+ 个 doubao 工具的 JSON schema 文件（doubao_text2image / doubao_image2image / doubao_saliency_segmentation 等）
- 总计 1000+ 行 JSON
- 但**没有设计文档说明**为何选 JSON schema 而非其他序列化（如 YAML / thrift / protobuf）

**对 RD 的影响**：
- 新增工具时遵循"加 JSON schema"惯例
- 不要试图引入新的 schema 格式

---

## 八、G7 上屏 Block 流式块级管理（undocumented）

**PRD 设计**：上屏模块协议有 WriterPacket / Block 结构。

**实际**：
- `agent_phase_26/writer/block_build.go` 引入"块级单位的递增输出"模式（think block → text block → finish packet）
- 改变了老的"一次性返回"输出模型
- 但这个**架构层语义改动在 diff 里只作为"实现细节"**，没有突出强调

**对 RD 的影响**：
- 写涉及上屏的需求时，**默认假设是流式块级**，不要回退到一次性返回模式
- 详见 [`../../../locators/block_locator.md`](../../../locators/block_locator.md)

---

## 九、G8 Memory 大 key 倾向（known tech-debt）

**PRD 自述**："**TODO**：现有的短期记忆结构过于冗杂，已出现部分大 key 倾向，待梳理无用字段在新老链路中进行去除"

**对 RD 的影响**：
- 不要新加大字段到 `MsgCreationAgentMemoryInfo`（除非必要）
- 改 Memory 结构时考虑"是否能顺手清理一些"

---

## 十、G9 RAG / Skills 设计（gap, P1）

**PRD 设计**：「Skills（P1）」+ Skills 的 RAG + Tool 拆分

**实际**：
- `agent_phase_26/agent/skills/`（如有）只是占位
- RAG 引擎仍在 `internal/rpc/phase3/agent_runtime/rag/`（老链路）
- 新链路如何使用 RAG 待后续

**对 RD 的影响**：
- 不要假设新链路有 RAG / Skills 模块（短期内沿用 phase3）
- 如果你的需求触达 RAG 动态 SP → 看 `phase3/agent_runtime/rag/engine/dynamic_sp/`

---

## 十一、Gap 跟踪建议

| 频率 | 动作 |
|---|---|
| 每个 sprint 末 | 复查本表，更新 gap 状态 |
| 每次本 skill 维护时 | 同步 |
| RD 真踩到 gap 时 | 反馈，加新行 |
