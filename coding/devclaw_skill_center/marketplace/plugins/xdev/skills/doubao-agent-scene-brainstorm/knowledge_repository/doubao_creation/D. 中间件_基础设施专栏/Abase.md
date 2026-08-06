# Abase

> 字节内部的**KV 存储**（类似 Redis，但持久化能力更强）。

---

## 一、用途（豆包-创作链路）

| 用途 | 关键 key |
|---|---|
| **短期记忆持久化** | `MsgCreationAgentMemoryInfo`（按 `MsgID` 索引） |
| 异步任务结果暂存 | `AsyncInfo`、`ActionBarGenVideoParams` |
| 跨服务请求缓存 | （视具体业务） |

---

## 二、Memory 存储结构（PRD § Memory § 协议定义）

```go
type MsgCreationAgentMemoryInfo struct {
    MsgID                   int64
    Content                 string
    AsyncInfo               AsyncInfo                `json:"async_info"`
    ActionBarGenVideoParams *ActionBarGenVideoParams `json:"action_bar_gen_video_params"`
    IsAsyncMemory           bool                     `json:"is_async_memory"`
    // 其他字段...
}

// 运行时（不持久化）
type RuntimeMemory struct {
    AgentMemory     MsgCreationAgentMemoryInfo  // 持久化部分（→ Abase）
    TracerInfo      CreationTracer              // Tracer 上报部分
    RuntimePlanInfo CreationRuntimePlan         // 运行时规划，不持久化
}
```

> ⚠️ PRD 自述："现有的短期记忆结构过于冗杂，已出现部分大 key 倾向，待梳理无用字段"。**已知 tech-debt**。

---

## 三、读写函数

| 函数 | 作用 |
|---|---|
| `NewRuntimeMemory(...)` | 创建新的运行时 Memory（含 with 链式参数） |
| `InsertMemoryToAbase()` | 持久化必要信息到 Abase |
| `TransferMemoryForMessage([]agent.Message)` | 根据 ContextMessage 查 Memory，与 Message 合并 |
| `LoadRuntimeMemory()` | Tool Call 完成后加载运行时 Memory（用于 LLM 继续执行） |

---

## 四、代码位置

| 链路 | 位置 |
|---|---|
| 老 Plan-Act / phase3 | `creation_agent/internal/rpc/phase3/value_obj/memory.go` + `phase3/agent_runtime/memory/memory_process.go` |
| 新 ReAct（agent_phase_26） | `creation_agent/agent_phase_26/agent/memory/runtime_memory/` + `agent_phase_26/model/memory/` |

> Memory 模块**新老两处**，但本期 PRD 显式说"为了保证新老链路对 Memory 读取逻辑的一致性，本期仍复用 `value_obj.MsgCreationAgentMemoryInfo` 整体结构"——所以**存储结构是同一个**，但读写代码是两套。

---

## 五、与 Block 上屏的关系

`AgentMemory.AsyncInfo` 标记异步上屏（一次性 vs 流式），影响 `IsAsyncReach`。

详见 [`../../../locators/block_locator.md`](../../../locators/block_locator.md) §六。

---

## 六、写需求时的检查

- [ ] 改 `MsgCreationAgentMemoryInfo` 结构时**评估存量数据兼容**（反序列化时 unknown 字段忽略 vs 报错）
- [ ] 加新字段建议 `optional`，避免老数据反序列化失败
- [ ] 不要急于"清理大 key"，等 PRD § Memory § TODO 真正实施再做
- [ ] 新老链路读写两套，验证两边都改
