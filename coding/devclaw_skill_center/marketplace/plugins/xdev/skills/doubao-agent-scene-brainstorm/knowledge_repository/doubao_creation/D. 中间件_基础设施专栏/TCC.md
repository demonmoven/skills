# TCC

> 字节内部的**静态 / 动态配置中心**（"Toutiao Config Center"）。

---

## 一、用途（豆包-创作链路）

| 用途 | 位置 |
|---|---|
| 开关 / 阈值 / 兜底参数 | 大量分散在各仓库 |
| Fornax SP 兜底（DowngradePlanConf.SpBackup） | `creation_agent/internal/util/tcc/base.go` |
| AB 实验下发参数容器（与 Libra 配合） | 各 `constant/ab_params.go` |
| 模型降级 / 资源模板 / 工具版本默认 | 同上 |
| **下游切流（DAG topic）** | `aigc_tool/dal/dag_switch.go` 用 `tool_switch_${tenantID}_${toolName}` 配置 |
| **租户级超时** | `aigc_tool/infra/tcc.go`（本期新增） |

---

## 二、代码引用方式

### 2.1 老链路（creation_agent 主）

```go
// creation_agent/internal/util/tcc/base.go
import "code.byted.org/.../tccclient"

var (
    fornaxConf      = tccclient.NewClient(...).GetConfig("fornax_config_key")
    abTesting       = tccclient.NewClient(...).GetConfig("ab_test_key")
    downgradePlan   = tccclient.NewClient(...).GetConfig("downgrade_plan_key")
    // ... 40+ Getter
)

// 读取：
val := tcc.GetDowngradePlanConf(ctx).SpBackup
```

### 2.2 新链路（agent_phase_26）

```go
// creation_agent/agent_phase_26/tcc/
// 独立的 TCC 配置客户端（与老的不共享！）
```

> **重要**：`internal/util/tcc/base.go` 与 `agent_phase_26/tcc/` 是**两套独立的客户端**。改 TCC 配置时**两边都要 grep 验证**。

### 2.3 下游 aigc 三件套

```go
// aigc_dag/biz/common/ab.go
// 5 个 AB 参数 key：缩略图格式、图片格式、分辨率等
val := tcc.GetAB(ctx, "thumbnail_format")

// aigc_tool/dal/dag_switch.go
config := tcc.Get("tool_switch_${tenantID}_${toolName}")
// 一致性 hash 切流
```

---

## 三、命名规则（推断）

> ⚠️ 推断，以平台实际为准。

| 命名模式 | 用途 |
|---|---|
| `creation_*` | 创作业务 |
| `agent_creation_*` | Agent 主服务 |
| `evaluation_*` | 评测 |
| `aigc_*_*` | aigc 三件套 |
| `tool_switch_<tenant>_<tool>` | 工具切流 |

---

## 四、TCC vs Libra 对比

| 维度 | TCC | Libra |
|---|---|---|
| 类型 | 静态 / 动态配置 | 实验平台 |
| 适合 | 长期固化、租户级开关、降级兜底 | 短期 A/B 对照、灰度发布 |
| 代码引用 | `tccclient.GetConfig` | `libra_sdk.GetParam` / `option.XXX` |
| 推送实时 | ✅ 实时 | ✅ 实时 |
| 上游使用 | ✅ 大量 | ✅ 大量 |
| 下游使用 | ✅ 大量 | ❌ 罕见（统一走 TCC） |

---

## 五、本期改动相关

- 本期**新增** `agent_phase_26/tcc/`（独立客户端）→ 现在 TCC 散落在两处
- 本期**新增** `aigc_tool/infra/tcc.go` 的租户级超时 Getter（PRD 多租户支持的少数落地点之一）
- 本期**未集成** flow/ocerr 错误码（gap，详见 [`./flow_ocerr.md`](./flow_ocerr.md)）

---

## 六、写需求时的检查

- [ ] 改 TCC 配置时**新老两处客户端都验证**？
- [ ] 是固化配置 vs 实验切流？前者走 TCC，后者走 Libra
- [ ] 命名是否符合 `<scope>_<purpose>_<version>` 约定？
- [ ] 下游切流走 `tool_switch_${tenant}_${tool}` 还是新建命名空间？
