# Fornax

> 字节内部的 **Prompt 配置中心**（专门管 LLM SP / 提示词的版本化配置平台）。

---

## 一、用途

| 用途 | 在豆包-创作 agent 链路中 |
|---|---|
| 管理 SP（System Prompt）的**线上版本** | SP 加载 4 层优先级的兜底层 ④ |
| 支持版本回滚 / 灰度发布 | RD 在平台调整 SP 后无需发版 |
| 支持 PPE（Pre-Production Environment） | 测试环境读 PPE，线上读 release |

---

## 二、代码引用方式

### 2.1 老链路（创作 agent）

```go
// creation_agent/internal/rpc/agent_util/sp.go::LoadSPWithDefault
prompt, err := util.FornaxCli().GetPrompt(ctx, fornaxKey, version)
if err != nil {
    // 失败降级 TCC
    prompt = tcc.GetDowngradePlanConf(ctx).SpBackup
}
```

### 2.2 新链路（agent_phase_26）

```go
// creation_agent/agent_phase_26/internal/rpc/fornax/
client := fornax.GetFornaxClient()
content := client.GetPrompt(ctx, key, version)
```

> 新老两套客户端实现独立。

---

## 三、关键 SP key（来自 phase3 sp_manager.go）

| SP key（常量） | 用途 |
|---|---|
| `PlannerSpKey` | 老 Plan-Act 主 Planner SP |
| `T2IDivergent` | 文生图发散 SP |
| ...（约 15-17 个） | （详见 [`../../../locators/sp_locator.md`](../../../locators/sp_locator.md) §三） |

> Fornax key 与代码常量是一一对应的，但平台上的"key 命名"可能与代码不完全一致；**务必查代码常量定义而不是凭记忆**。

---

## 四、配置 / 接入

| 项 | 位置 |
|---|---|
| 客户端初始化 | `creation_agent/internal/util/tcc/base.go`（老）+ `agent_phase_26/internal/rpc/fornax/`（新） |
| ak/sk 配置 | TCC 命名空间下管理（不在仓库） |
| Fornax 平台地址 | 字节内部，看团队文档 |

---

## 五、与 SP 4 层加载的关系

参考 [`../../../locators/sp_locator.md`](../../../locators/sp_locator.md) §一。Fornax 是**最低优先级的兜底层 ④**：

```
① Picasso 注入 > ② Libra 实验 > ③ TCC 兜底 > ④ Fornax 默认
```

Fornax 失败时无降级（已经是兜底层），抛错。

---

## 六、本期改动相关

- 本期 **没有** 大改 Fornax 接入方式
- 但 `agent_phase_26/internal/rpc/fornax/` 是**全新增**（与老 `util.FornaxCli()` 独立），属于"SP 管理三处分散"问题的一环（详见 [`../A. 组织职责与找人地图/误区纠正.md`](../A.%20组织职责与找人地图/误区纠正.md) § 误区 5）

---

## 七、写需求时的检查

- [ ] 是否新增 SP key？→ 在 Fornax 平台先配置，再加代码常量
- [ ] 是否需要 PPE 测试？→ 走 PPE 入口，不要在 release 上调试
- [ ] 是否需要版本回滚？→ Fornax 平台原生支持，不要自建版本管理
