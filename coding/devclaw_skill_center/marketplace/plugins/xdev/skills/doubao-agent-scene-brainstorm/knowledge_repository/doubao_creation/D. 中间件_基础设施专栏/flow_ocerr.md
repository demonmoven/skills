# flow/ocerr

> 字节 flow 业务的**统一错误码仓库**。

---

## 一、定位

| 维度 | 内容 |
|---|---|
| 仓库 | `code.byted.org/flow/ocerr`（独立仓库，不在本期 8 仓库之内） |
| 用途 | 提供统一的 Error 类型、错误码枚举 |
| PRD 计划 | "统一定义好错误码规范，维护在 flow/ocerr 仓库中，每个 function 对外抛出 Error 时尽量使用错误码，避免使用文本的方式" |

---

## 二、本期 Gap（**重要**）

> ⚠️ **当前未集成**。详见 [`../Z. 主索引/Z.2 PRD vs 实现 Gap 表.md`](../Z.%20主索引/Z.2%20PRD%20vs%20实现%20Gap%20表.md)。

PRD 设计：
- 每个 function 对外抛错时使用错误码
- 替代当前的 文本 / 自定义 error 模式

实际 diff：
- 8 仓库 diff 中**未见** `flow/ocerr` 的 import
- 错误透传仅通过 IDL 新增的 `biz_err_status_code` / `biz_err_status_msg` 字段（在 `SyncInvokeToolResponse` 等 Response 结构里）
- 各模块（dag / tool / management）可能各有局部错误定义，**未做统一汇聚**

---

## 三、IDL 中的错误码字段

```thrift
struct SyncInvokeToolResponse {
    1: string output
    2: string task_id
    3: optional i32 biz_err_status_code   // 业务错误码
    4: optional string biz_err_status_msg // 业务错误信息
}
```

也有类似结构在其他 Response 里。

> 这是**透传机制**而非"统一错误码"。下游服务返什么码、上游对什么码做什么处理，**目前没有统一约定**。

---

## 四、写需求时的检查

- [ ] 你的需求需要返回错误码？走当前的 `biz_err_status_code/msg`
- [ ] 不要假设 flow/ocerr 已集成 → 不要 `import "ocerr"`
- [ ] 错误码语义需要与下游对齐（在评审会显式确认）
- [ ] 如果你的需求是"错误码统一"本身 → 这是一个独立的大项目，不在本 brainstorm scope

---

## 五、未来

PRD 设计还在，但落地未启动。RD 写方案时如果触达错误处理：
- **建议**：引用 PRD § 整体目标，标注"flow/ocerr 集成是后续工作"
- **不建议**：假设已集成 / 主动接入（会与其他模块的局部 error 模式冲突）
