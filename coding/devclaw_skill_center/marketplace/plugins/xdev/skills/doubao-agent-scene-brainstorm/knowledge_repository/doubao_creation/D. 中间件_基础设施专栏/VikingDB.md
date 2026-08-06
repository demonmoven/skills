# VikingDB

> 字节内部的**向量数据库**（用于 RAG / 长期记忆 / 召回）。

---

## 一、用途（豆包-创作链路）

| 用途 | 引用方式 |
|---|---|
| **长期记忆**（用户画像 / 历史偏好） | `creation_agent/common/vikingdb/`（main.go line 17 init） |
| RAG（检索增强生成） | `creation_agent/internal/rpc/phase3/agent_runtime/rag/` |
| 动态 SP（基于 RAG 命中下发不同 SP） | `creation_agent/internal/rpc/phase3/agent_runtime/rag/engine/dynamic_sp/` |

---

## 二、代码位置

| 模块 | 位置 |
|---|---|
| 客户端初始化 | `creation_agent/common/vikingdb/`（启动时 init） |
| RAG 引擎 | `creation_agent/internal/rpc/phase3/agent_runtime/rag/engine/` |
| 动态 SP（基于 RAG 命中） | `creation_agent/internal/rpc/phase3/agent_runtime/rag/engine/dynamic_sp/` |
| 新链路（agent_phase_26） | （部分复用 `common/vikingdb/`；具体 RAG 引擎入口在 agent_phase_26/agent/...） |

---

## 三、与 SP 4 层加载的关系

VikingDB 的 RAG 命中可以**动态影响 SP**（下发不同版本）。这与 [SP locator](../../../locators/sp_locator.md) §一的 4 层有交叉：

- 4 层主要描述"加载来源"（Picasso / Libra / TCC / Fornax）
- VikingDB / RAG 是 **横向**，对加载结果再做"动态 SP 替换"

> 写需求触达 SP 时，除了查 4 层加载，还要考虑：是否会被 RAG 命中后动态替换？

---

## 四、与 Memory 的关系

| 维度 | 短期 Memory（Abase） | 长期 Memory（VikingDB） |
|---|---|---|
| 存储 | KV | 向量 |
| 检索 | 按 MsgID | 按语义召回 |
| 时效 | 单次会话 / 短期 | 长期沉淀 |
| 用例 | 当前对话上下文 | 用户偏好 / 历史 |

---

## 五、本期改动相关

- 本期**没有大改 VikingDB / RAG**
- agent_phase_26 部分复用 `common/vikingdb/` 的客户端
- RAG 引擎在新链路中如何重组待考察（PRD § Skills（后续设计）留白，目前 P1）

---

## 六、写需求时的检查

- [ ] 是否触达"用户画像 / 长期偏好" → 走 VikingDB
- [ ] 是否触达"动态 SP" → 走 RAG 引擎 dynamic_sp
- [ ] 阶段二 sp_inventory 必须考虑 RAG 命中是否会替换 SP
- [ ] PRD § Skills（后续设计）显示本期是 P1，**短期可能不动**
