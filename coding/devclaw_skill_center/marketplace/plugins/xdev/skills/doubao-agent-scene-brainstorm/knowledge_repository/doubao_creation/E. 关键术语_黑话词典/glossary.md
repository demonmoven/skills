# 关键术语 / 黑话词典（豆包-创作 Agent 链路）

> 当你在 PRD / 代码里看到一个不认识的词，先来这里查。
>
> 编号 = 类别字母 + 序号（如 M1, T3, P5）。

---

## M. 模型 / 训练

| # | 术语 | 含义 | 出处 |
|---|---|---|---|
| M1 | **doubao-seed-1.8-23b** | 豆包 Seed 系列大模型，本期 ReAct 采用版本 | PRD 协议示例 |
| M2 | **doubao_think_v1** | think 标签解析器，把 `<think>...</think>` 切出 ThinkingBlock | PRD § Agent 执行 |
| M3 | **doubao_fc_xml_v2** | XML function call 解析器，把 `<function_..._=tool>` 切出 ToolCall | 同上 |
| M4 | **ArkModel** | Ark 平台高性能模型实现 | `agent_phase_26/agent/runtime/model/ark_model/` |
| M5 | **ModelAPI** | 通用 ModelAPI 客户端实现 | `model/model_api/` |
| M6 | **stone.llm.gateway** | butterfly 大模型网关 PSM | go.mod |
| M7 | **ic.cv.hamlet** | CV 模型网关 PSM | go.mod |
| M8 | **LLM Hub** | LLM 调度统一入口（IDL 中体现为 `model_api://` URL 协议） | PRD 协议示例 model 字段 |
| M9 | **pixelDance** | 视频生成模型（i2v 接入） | arch.md |
| M10 | **TT music** | 字节音乐生成模型（文生音乐用） | arch.md |
| M11 | **VLM** | 视觉 + 语言多模态模型（识图） | arch.md 找人地图 |

---

## T. 工具 / Tool

| # | 术语 | 含义 | 出处 |
|---|---|---|---|
| T1 | **host** | ReAct 的"宿主"工具，专门发 `display_text`（"正在为您生成..."）伴随消息 | PRD 白板 04_model |
| T2 | **image_gen** | 文生图工具（输入 prompts / ratio / output_image_ids） | PRD § Tool 配置协议 |
| T3 | **image_edit** | 图编辑工具（输入 edit / input_image_ids / output_image_ids） | 同上 |
| T4 | **image_to_video / i2v** | 图生视频工具 | 同上 |
| T5 | **text_to_video / t2v** | 文生视频工具 | 同上 |
| T6 | **saliency_segmentation** | 显著性分割（本期 diff 新增） | aigc_dag/biz/handler_sync/doubao/ |
| T7 | **doubao_image_upload** | 图片上传逻辑（本期 diff 新增） | 同上 |
| T8 | **Hold** | 等待 / 持续状态控制（多步任务中间态） | PRD 白板 04_model |
| T9 | **Quota** | 额度 / 积分 / 会员校验工具 | 同上 |
| T10 | **Safety** | 内容安全审核工具 | 同上 |
| T11 | **DAG Tool** | 通过 DAG 编排的工具基类（Image DAG Tool 等） | PRD § Agent 执行 § Tools |
| T12 | **Image DAG Tool** | 图片相关的 DAG Tool 实现（解析 + 调度 dag tool + 上屏 loading） | PRD 行 1590+ |

---

## P. 协议 / 数据结构

| # | 术语 | 含义 | 出处 |
|---|---|---|---|
| P1 | **AgentPayload** | Agent 入参结构（含 query / context / 资源 / 配置等） | alice_idl agent.thrift |
| P2 | **AgentStream** | C 端服务器流式接口 | agent_creation.thrift |
| P3 | **AgentDuplex** | 双向流式接口（多轮对话） | 同上 |
| P4 | **CanvasStream** | 画布交互流式接口 | 同上 |
| P5 | **ContextMessage** | 4 路 message 容器（agentMsg / planMsg / preMsg / reviewMsg） | PRD § 上下文管理 |
| P6 | **WriterPacket** | 上屏数据包（含 BlockInfo / PacketType 等） | PRD § 上屏模块 |
| P7 | **PacketType** | `packet_type_incr`（增量） / `packet_type_full`（全量） | 同上 |
| P8 | **CreationBlock**（2074） | 多模态消息 Block（图 / 视频） | PRD § 下行协议 |
| P9 | **LoadingBlock**（10101） | "开始创作..."占位 Block | 同上 |
| P10 | **ThinkingBlock**（10040） | 思考态标题 Block（ReAct 配套） | 同上 |
| P11 | **TextBlock** | 文字 Block | 同上 |
| P12 | **ButtonBlock**（10103） | 通用跳转按钮 | 同上 |
| P13 | **CreditBlock**（10057） | 积分 Block（**不在本期**） | 同上 |
| P14 | **LoginBlock** | 强制登录 Block（**不在本期**） | 同上 |
| P15 | **StreamPacket** | Agent 流式回包通用结构 | alice_idl agent.thrift |
| P16 | **BlockMeta** | Block 元数据 | PRD |
| P17 | **PacketBlockInfo** | 包级 Block 信息（BlockType / BlockContent / BlockMeta） | PRD |
| P18 | **MsgCreationAgentMemoryInfo** | 短期 Memory 持久化结构（Abase） | PRD § Memory |
| P19 | **RuntimeMemory** | 运行时 Memory（含 AgentMemory / TracerInfo / RuntimePlanInfo） | 同上 |
| P20 | **ToolInfo** | Tool 参数定义（本期从 map 简化为 ToolParamInfo 对象） | alice_idl creation_tool.thrift |
| P21 | **ToolParamInfo** | 单个工具参数信息 | 同上 |
| P22 | **ReactAgentConfig** | ReAct Agent 配置（含 ChatModelConfig / PromptConfig / Tools / DAGTools） | creation_evaluation.thrift |
| P23 | **DoubaoAgentConfigV2Data** | ReAct 升级后的配置容器（picasso 注入用） | 同上 |

---

## I. ID / 游标 / 资源

| # | 术语 | 含义 | 出处 |
|---|---|---|---|
| I1 | **image_gen_X** | 文生图输出图片的逻辑 ID（X = 1, 2, ...） | PRD 协议示例 |
| I2 | **image_edit_X** | 图编辑输出图片的逻辑 ID | 同上 |
| I3 | **video_X** | 视频输出的逻辑 ID | 同上 |
| I4 | **MsgID** | 消息 ID（与 Memory 持久化绑定） | PRD § Memory |
| I5 | **canvas_id / canvas_sub_id** | 画布 ID 与子 ID | agent_creation.thrift |
| I6 | **seq_start** | 当前 block 起始 sequence_id | 同上 |
| I7 | **local_id** | 客户端本地 ID（与 canvas_sub_id 一一对应） | 同上 |
| I8 | **ResourceID** | 跨层游标（模型 ↔ 工具 ↔ Memory ↔ 上屏一致） | tool_locator §六 |
| I9 | **ImageUri** | ImageX 资源的 URI 标识 | PRD § Resource |
| I10 | **Vid** | Smart Player 视频 ID | 同上 |
| I11 | **TaskId** | 异步任务 ID（aigc_tool 异步调用用） | idl tool.thrift |
| I12 | **execution_id** | Picasso 评测调用时传入下游的 trace ID | creation_evaluation.thrift |

---

## B. 业务黑话

| # | 术语 | 含义 | 出处 |
|---|---|---|---|
| B1 | **agent26 / agent_phase_26** | 本期升级专项代码代号（来自 2026 Q2） | 代码目录 |
| B2 | **2026_bh** | query 标签灰度触发开关（任意 query 含此字符即走新链路） | handler.go |
| B3 | **phase3** | 第 ④ 代多模态 agent 探索代码（Plan-Act 架构） | 代码目录 |
| B4 | **Plan-Act** | "规划 + 执行"架构（一次 Plan + N 次 FC + N 次模型调用） | PRD 背景 |
| B5 | **ReAct** | "Reasoning + Acting" 架构（模型自主 think + 自动 fc） | 同上 |
| B6 | **picasso** | 字节内部效果评测平台（创作离线流量入口） | info.md / PRD |
| B7 | **闲聊入口** | 用户在豆包主 bot 自然语言对话触发 | arch.md |
| B8 | **指令集入口** | 用户通过结构化指令 / 模板触发 | 同上 |
| B9 | **Action Bar** | 主 bot 顶部按钮栏入口 | 同上 |
| B10 | **官方多 bot** | 豆包平台提供的专门创作 bot（如"AI图片生成"） | 同上 |
| B11 | **运营多 bot** | 运营创建的主题 bot（如绘画系列） | 同上 |
| B12 | **bot_engine** | 早期 hook 时代用的 bot 调度引擎 | arch.md 演进 |

---

## E. 工程黑话

| # | 术语 | 含义 | 出处 |
|---|---|---|---|
| E1 | **SupportAgent26** | option 包装的 AB 实验函数，决定走新 ReAct or 老 Plan-Act | handler.go |
| E2 | **SupportAgent26Ark** | option 包装的 AB 函数，决定 ArkModel vs ModelAPI | handler.go |
| E3 | **isPSMCreationEvaluation** | main.go 用此函数判断走 evaluation 子单体 or 主服务 | creation_agent/main.go |
| E4 | **tenant_tool_conf** | aigc_management 租户级工具配置 | aigc_management/handler.go |
| E5 | **SwitchToNewTopic** | aigc_tool 的 DAG topic 切流函数（TCC + 一致性 hash） | aigc_tool/dal/dag_switch.go |
| E6 | **LoadSPWithDefault** | 老链路 SP 加载主入口 | internal/rpc/agent_util/sp.go |
| E7 | **SystemPromptManager** | phase3 SP 加载实现 | phase3/agent_runtime/sp_manager.go |
| E8 | **LibraToken / LibraDoubaoAppID** | Libra SDK 接入常量 | creation_access/constant/libra.go |
| E9 | **DowngradePlanConf** | TCC 降级配置（含 SpBackup） | internal/util/tcc/base.go |
| E10 | **BuildContextMessage** | 上下文管理入口函数 | agent_phase_26/agent/context_manage/ |
| E11 | **NewRuntimeMemory** | 创建运行时 Memory（链式 with） | PRD § Memory § 函数表 |
| E12 | **InsertMemoryToAbase** | 持久化 Memory 到 Abase | 同上 |
| E13 | **GetPacketSeq / SendWritePacket** | 上屏顺序锁与发包 | PRD § 上屏 § 函数表 |
| E14 | **GetOnlineDAGTopic** | aigc_tool 调 aigc_management 拿当前线上 DAG topic | aigc_tool/dal/dag.go:50 |
| E15 | **`_never_used_<hash>` token** | 模型独享 token 围栏（防内容污染） | 见 误区纠正 § 误区 7 |

---

## S. 服务 / PSM 速查

| PSM | 一句话 |
|---|---|
| `flow.alice.creation_access` | 接入层（主要 for picasso） |
| `flow.agent.creation` | C 端 Agent 主服务 |
| `flow.agent.creation_evaluation` | Picasso 评测专用（与上者共仓） |
| `flow.aigc.management` | 创作能力配置中心 |
| `flow.aigc.dag` | DAG 执行引擎 |
| `flow.aigc.tool` | 工具调用网关 |

---

## 词条总数：M11 + T12 + P23 + I12 + B12 + E15 + S6 = **91 条**

> 维护建议：每次 brainstorm 阶段二遇到新术语，回填到本表。新词条编号从对应类别的最大值 +1。
