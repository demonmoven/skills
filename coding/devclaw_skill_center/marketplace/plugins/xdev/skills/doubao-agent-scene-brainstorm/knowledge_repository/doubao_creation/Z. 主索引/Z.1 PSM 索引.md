# Z.1 PSM 索引

> 6 PSM × 关键代码路径 / build / IDL 速查表。是 [`../../../locators/psm_locator.md`](../../../locators/psm_locator.md) 的浓缩版。

---

## 一、6 PSM 速查表

| # | PSM | RUN_NAME / kitex ServiceName | 仓库 worktree | IDL |
|---|---|---|---|---|
| 1 | `flow.alice.creation_access` | `creation_access/build.sh` + `kitex_info.yaml` | `creation_access` | `alice_idl/thrift/alice_creativity/creation_access.thrift` |
| 2 | `flow.agent.creation` | `creation_agent/build.sh::RUN_NAME=flow.agent.creation` | `creation_agent` | `alice_idl/thrift/alice_creativity/agent_creation.thrift` |
| 3 | `flow.agent.creation_evaluation` | `creation_agent/main.go::isPSMCreationEvaluation()` 切分 | `creation_agent/creation_evaluation/`（同仓库） | `alice_idl/thrift/alice_creativity/creation_evaluation.thrift` |
| 4 | `flow.aigc.management` | `aigc_management/build.sh + kitex_info.yaml` | `aigc_management` | `idl/flow/service/management/` |
| 5 | `flow.aigc.dag` | `aigc_dag/build.sh + kitex_info.yaml` | `aigc_dag` | `idl/flow/service/dag/` |
| 6 | `flow.aigc.tool` | `aigc_tool/build.sh + kitex_info.yaml` | `aigc_tool` | `idl/flow/service/tool/` |

---

## 二、入口 handler 索引

| PSM | handler.go 暴露的关键接口 |
|---|---|
| `flow.alice.creation_access` | `EvaluateImageTask` / `ExecuteAgent` / `GetDoubaoAgentConfig` v1/v2 / `ConvertToDoubaoImages` / `GetDoubaoAgentToolVersion` |
| `flow.agent.creation` | `AgentStream` / `AgentDuplex` / `AgentStream4Search` / `CanvasStream` / `CreationCache` / `GetCrowdTestTask` |
| `flow.agent.creation_evaluation` | （creation_evaluation/handler/）路由 ExecuteAgent 到本地 agent runtime |
| `flow.aigc.management` | `CreateTool` / `UpdateTool` / `ListTool` / `ListToolVersions` / `QueryAgentToolList` / `CreateDAGDraft` / `PublishDAGDraft` / `QueryDAGMeta` / `GetDAGConfig` / `ExecDAGTopic` / `GetAbilityConfig` / `GetTenantToolConf` / `UpdateTenantToolConf` |
| `flow.aigc.tool` | `AsyncInvokeTool` / `SyncInvokeTool` / `InvokeTool` / `GetTaskResult_` |
| `flow.aigc.dag` | `Exec` / `GetResult_` |

---

## 三、子目录索引（按代码"主战场"）

### 3.1 creation_access（接入层）

| 子目录 | 用途 |
|---|---|
| `handler/picasso/` | Picasso 5 个接口的 handler |
| `handler/evaluate_image_task/` | EvaluateImageTask 唯一非 picasso 接口 |
| `constant/libra.go` | Libra 接入常量 |
| `service/` | 业务服务 |

### 3.2 creation_agent（C 端 Agent 主服务）

| 子目录 | 用途 |
|---|---|
| `handler/` | 主链路 handler（含 phase3handle） |
| `handler/handler.go` | **顶部 SupportAgent26 灰度判断** |
| `agent_phase_26/` | **本期新增**新 ReAct 全套 |
| `creation_evaluation/` | Picasso 评测子单体（独立 PSM） |
| `internal/rpc/agent_util/sp.go` | 老 SP 加载 |
| `internal/rpc/phase3/` | 第 ④ 代代码（Plan-Act） |
| `internal/util/tcc/base.go` | 老 TCC 客户端 |
| `common/vikingdb/` | VikingDB 客户端 |
| `internal/tracer/` | 老 tracer 模块 |

### 3.3 aigc_management（配置中心）

| 子目录 | 用途 |
|---|---|
| `handler/` | tool / dag / ability / tenant 4 类接口 |
| `service/domain/tool/` | Tool 元数据 DAO |
| `service/domain/ability/` | 能力配置 |
| `service/domain/ability/schema/` | **JSON schema 文件**（doubao 工具 8+ 个本期新增） |
| `service/domain/tenant_tool_conf/` | 租户配置 |

### 3.4 aigc_tool（工具调用网关）

| 子目录 | 用途 |
|---|---|
| `handler.go` | sync / async / 通用 InvokeTool |
| `method/` | 各种 invoke 方法 handler |
| `service/tool.go:50` | **调 aigc_dag.Exec / GetResult** |
| `dal/dag.go:50` | **调 aigc_management.GetOnlineDAGTopic** |
| `dal/dag_switch.go::SwitchToNewTopic` | TCC + 一致性 hash 切流 |
| `infra/tcc.go` | 租户级 TCC 配置（**本期新增**） |

### 3.5 aigc_dag（DAG 执行引擎）

| 子目录 | 用途 |
|---|---|
| `handler.go` | Exec / GetResult |
| `engine/` | 通用调度框架（dag.go / executor.go / node_executor*） |
| `biz/handler_sync/doubao/` | **本期新增** doubao 同步工具 handler |
| `biz/handler_async/` | 异步工具 handler |
| `biz/derived_func/` | 衍生函数库 |
| `biz/common/ab.go` | 5 个 AB 参数 key |

---

## 四、IDL 双源速查

### 4.1 alice_idl（创作业务自有）

| thrift 文件 | namespace | 关键内容 |
|---|---|---|
| `agent_creation.thrift` | `flow.agent.creation` | service AgentService（C 端入口） |
| `creation_access.thrift` | `flow.alice.creation_access` | service CreationAccessService（picasso 入口） |
| `creation_evaluation.thrift` | `alice_creativity` | ExecuteAgentRequest / ReactAgentConfig / DoubaoSubAgentConfig / ChatModelConfig / PromptConfig |
| `creation_tool.thrift` | — | DAGTool / **ToolInfo（本期简化）** |
| `creation_push.thrift` 等 | — | 推送 / 指令集业务能力 |

### 4.2 idl (butterfly)（aigc 三件套公共）

| 路径 | 服务 | diff 关键点 |
|---|---|---|
| `flow/service/management/` | AIGCManagementService | **新增 ListToolVersions / QueryAgentToolList**；Tool 新增 version / is_sync / tool_info；Ability 新增 is_sync |
| `flow/service/tool/` | AIGCToolService | **新增 SyncInvokeTool**；新增 InvokeToolExtraAipReqParam |
| `flow/service/dag/` | AIGCDAGService | 主结构稳定 |

---

## 五、模型 / 资源类下游 PSM

| PSM | 用途 |
|---|---|
| `stone.llm.gateway` | LLM 网关（butterfly） |
| `ic.cv.hamlet` | CV 模型网关 |
| `flow.alice.model` | 模型管理（alice 业务） |
| `flow.alice.resource` / `_center` / `_item` | 资源管理 |
| `flow.alice.config_center` | 配置中心 |
| `flow.alice.creativity` | 创作业务能力 |
| `flow.alice.message` | 消息 |
| `flow.notify.push` | 推送 |
| `flow.commerce.credit_rpc` | 积分（CreditBlock 关联） |
| `ocean.cloud.review` | 内容审核 |
| `ocean.cloud.profile` | 用户档案 |
| `bytedance.videoarch.imagex_url` | 图片资源（ImageX） |
| `toutiao.videoarch.smart_player` | 视频资源（Smart Player） |
