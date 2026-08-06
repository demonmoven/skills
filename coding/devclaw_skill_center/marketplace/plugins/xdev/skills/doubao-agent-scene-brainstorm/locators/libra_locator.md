---
name: libra_locator
description: 7 处 Libra/AB 切流散落点全清单 + 上下游策略差异 + "看到 Libra 改动应查啥"导览。访谈 P0 痛点核心
last_synced_commits:
  creation_agent:    "651a643215e1bd2f713aee504ec46dce40672a72"
  creation_access:   "b9422413f759d72f354f6ed7cddac61422085828"
  aigc_dag:          "81ff22867a8004b0610ff199a00f9a5fda73d2c1"
  aigc_tool:         "cc1e44cc5d177baa19d8b2012139a1f0ec8ab7d0"
verified_against_branch:
  creation_agent:    "feature/agent_26_lyw"
  creation_access:   "feature/agent_26"
  aigc_dag:          "feat/agent26"
  aigc_tool:         "feat/agent26"
last_synced_date: "2026-04-27"
---

# Libra Locator — 切流点散落全清单（P0 痛点核心）

> 访谈中**唐晔晨 / 杨守亮 / 王瑶**都明确提及："代码逻辑中散落较多实验切流逻辑，每次方案过程中需要反复手动 Libra 平台上观察切流进度，会影响技术决策"。
>
> **本文件就是为这条痛点设计的**：把 7 处散落点一次性列出，避免 RD 漏查。

---

## 一、为什么 Libra 痛？

| 痛点子项 | 来源访谈 | 本 locator 的应对 |
|---|---|---|
| 切流逻辑散落各处 | 唐晔晨 / 杨守亮 / 王瑶 | § 三 给出 7 处全清单 |
| 方案过程要反复跳 Libra 平台 | 唐晔晨 / 杨守亮 | § 五 给出"看到 Libra 改动应查哪些代码"导览 |
| 上下游切流策略不统一 | （隐性） | § 四 上下游切流策略差异 |
| 想要 Libra 增 / 删的标准化 skill | 王瑶 | § 七 未来扩展点（write 侧 skill_libra_change） |

---

## 二、概念清晰化

| 概念 | 含义 | 出现位置 |
|---|---|---|
| **Libra** | 字节内部实验切流平台（外部 SDK） | 上游服务直接引用 |
| **AB 实验参数** | 通过 Libra 下发的字段 | `creation_*/constant/` 目录 |
| **TCC 切流** | 配置中心的开关型切流（不走 Libra） | 下游 aigc 三件套 |
| **一致性 hash 切流** | 按 user_id 等维度做稳定切流 | aigc_tool DAG topic 切流 |

> **关键观察**：上游服务（creation_*）走 Libra；下游服务（aigc_*）走 TCC + 一致性 hash。**两条链路的切流策略不一样**。

---

## 三、7 处散落点全清单（按密集度 / 关键度排序）

| # | 位置 | 切流维度 | 类型 |
|---|---|---|---|
| **1** ★ | `creation_agent/handler/handler.go::SupportAgent26` | **灰度新老架构**（最重要） | option / AB |
| **2** | `creation_agent/handler/handler.go::SupportAgent26Ark` | 模型实现选择（Ark vs ModelAPI） | option / AB |
| **3** | `creation_access/constant/libra.go` | Libra 常量（`LibraToken`、`LibraDoubaoAppID`） | Libra SDK |
| **4** | `creation_evaluation/constant/ab_params.go` | 评测专用 AB 维度 | AB 参数常量 |
| **5** ★ | `creation_agent/internal/rpc/phase3/agent_runtime/sp_manager.go` | SP 来源选择（联动 [`sp_locator.md`](./sp_locator.md) 4 层加载） | Libra → SP |
| **6** | `aigc_dag/biz/common/ab.go` | DAG 节点级 AB（5 个 key：缩略图格式、图片格式、分辨率等） | TCC 内嵌 AB |
| **7** ★ | `aigc_tool/dal/dag_switch.go::SwitchToNewTopic()` | DAG topic 切流 | TCC + 一致性 hash |

★ = brainstorm 时**最容易踩坑**的位置。

### 3.1 # 1：架构代际灰度（**最影响判断**）

```go
// creation_agent/handler/handler.go
if strings.Contains(req.Query.ContentForModel, "2026_bh") || option.SupportAgent26(ctx) {
    // 走新 ReAct（agent_phase_26）
    reactAgent := runtime.NewReactAgent(...)
    return reactAgent.Run(ctx, req)
}
// 否则：老 Plan-Act 链路
err = dispatchResult.AgentHandlerInter.Execute(...)
```

**含义**：**任何分析创作 agent 链路的需求都必须先确认这个开关的当前切流状态**。否则方案落到老链路，新链路漏改。

### 3.2 # 2：模型实现选择

```go
if option.SupportAgent26Ark(ctx) {
    // ArkModel
} else {
    // ModelAPIModel
}
```

### 3.3 # 3-4：上游 Libra 常量

`creation_access/constant/libra.go`：定义 LibraToken / LibraDoubaoAppID 等接入 Libra SDK 所需常量。

`creation_evaluation/constant/ab_params.go`：定义评测专用 AB 维度（评测平台往 ExecuteAgentRequest.doubao_agent_config_v2_data 注入实验配置时，用这些维度）。

### 3.4 # 5：SP 来源选择（联动 SP locator）

`creation_agent/internal/rpc/phase3/agent_runtime/sp_manager.go::SystemPromptManager.GetValue()` 内部按 [`sp_locator.md`](./sp_locator.md) §一 4 层优先级判定 SP 来源。**这是 Libra 与 SP 的耦合点**。

### 3.5 # 6：DAG 节点级 AB

`aigc_dag/biz/common/ab.go` 定义 5 个 AB 参数 key（缩略图格式、图片格式、分辨率等），在 biz/handler 中通过 NodeCtx 获取参数。

### 3.6 # 7：aigc_tool DAG topic 切流

```go
// aigc_tool/dal/dag_switch.go
func SwitchToNewTopic(ctx, tenantID, toolName, userID) string {
    config := tcc.Get("tool_switch_${tenantID}_${toolName}")  // TCC 配置
    return consistentHash(userID, config.NewTopicRatio)        // 一致性 hash
}
```

---

## 四、上下游切流策略差异（关键）

| 维度 | 上游（creation_*） | 下游（aigc_*） |
|---|---|---|
| 平台 | Libra（实验平台） | TCC（配置中心） |
| 切流粒度 | 用户、设备、AppID、地理 | 租户、工具名、用户（一致性 hash） |
| 是否实时 | 实验灰度可实时调整 | TCC 推送可实时调整 |
| 操作位置 | Libra 平台 | TCC 平台 |
| 代码引用风格 | `option.XXX(ctx)`, `libra_sdk.GetParam()` | `tcc.Get("xxx_${tenant}_${tool}")` + 一致性 hash |
| 适用场景 | 短期实验、A/B 对照 | 长期固化、租户级灰度 |

> **brainstorm 提醒 RD**：如果你的需求是"短期 A/B 对照" → 走 Libra；如果是"租户级永久切换" → 走 TCC。**不要把两者混用**。

---

## 五、"看到 Libra 改动应该查哪些代码"导览

### 5.1 PRD / 需求里出现以下关键词时，必查的位置

| 关键词 | 必查位置 |
|---|---|
| "灰度" / "实验" / "AB" / "Libra" | 全部 7 处 |
| "新老架构切换" / "ReAct 灰度" | # 1, # 5 |
| "模型切换" / "Ark 实验" | # 2 |
| "Picasso 评测注入" | # 4（`ab_params.go`）+ [`sp_locator.md`](./sp_locator.md) 4 层 ① |
| "SP 改动" | # 5 + [`sp_locator.md`](./sp_locator.md) 全部 |
| "图片格式" / "分辨率" / "缩略图" | # 6（aigc_dag ab.go） |
| "工具切流" / "DAG topic 切换" | # 7（dag_switch.go） |

### 5.2 RD 工作流建议

```
1. RD 收到需求 → 看本 locator 决定查哪几处
2. 在每一处确认"当前切流配置"（去 Libra / TCC 平台查）
3. 写到 brainstorm_core_findings.md § 5.2 Libra 切流盘点表
4. 评估改动是否需要新增 / 改动 / 下线切流配置
5. 提醒上下游切流策略差异（防止跨服务时混用）
```

---

## 六、Libra 灰度的"标准动作"模板

> **本期不做**（属未来 skill_libra_change 范畴），但写需求时建议遵循下面动作 checklist。

### 6.1 新增 Libra 实验

- [ ] 命名：`creation_<能力>_<场景>_v<版本>`（如 `creation_watermark_picasso_v1`）
- [ ] 在 Libra 平台创建实验
- [ ] 在代码里加 option / 常量（# 1-4 之一）
- [ ] 在使用点加 if 分支
- [ ] 上线前 dry-run 切流 0%
- [ ] 推全：直至 100%
- [ ] **下线（关键，常被忽略）**：实验结束后清代码 if 分支 → 这是**屎山的源头**

### 6.2 下线 Libra 实验

- [ ] 在代码里 grep 实验 key
- [ ] 删除所有 if 分支
- [ ] 在 Libra 平台关闭实验
- [ ] **避免遗留 dead code**

---

## 七、未来扩展点：`skill_libra_change`

王瑶访谈明确："Libra 相关代码变更逻辑，希望能有一个标准化的 skill—— 增加一个 Libra 或下线一个 Libra，代码变更模式能统一"。

**本期不实现**（属 write 侧 skill），但本 locator 的 § 六 模板就是它的雏形。

---

## 八、写需求时的 Libra 检查清单

- [ ] 本需求**是否涉及切流**？（看 PRD 关键词：灰度 / AB / 实验 / Libra）
- [ ] 切流策略选对了？（上游 Libra vs 下游 TCC + 一致性 hash）
- [ ] 7 处散落点都查过了吗？
- [ ] 是否影响 # 1（架构代际灰度）？如果是，**必须新老双轨写方案**
- [ ] 是否影响 # 5（SP 来源）？如果是，联动 [`sp_locator.md`](./sp_locator.md)
- [ ] 实验 key 命名遵循 `creation_<能力>_<场景>_v<版本>` 约定？
- [ ] 实验下线计划已写入方案？
