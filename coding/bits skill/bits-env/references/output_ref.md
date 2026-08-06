# Output Reference — 标准化输出格式模板

> 本文件定义 PLAN 阶段和 COMPLETE/FAILED 阶段的所有标准输出格式。
> LLM 按实际场景选用对应模板，不得随意改变格式结构。

---

## PLAN 阶段输出模板

**确认规则**：
- 简单任务（单服务单集群）：直接执行，无需等待确认
- 复杂任务（≥ 3 个操作 或 多服务）：**必须**呈现计划并等待用户确认
- 高风险操作（ROLLBACK）：**始终**等待确认
- **BATCH_DEPLOY**：**始终**以表格形式汇总全部配置，等待用户确认

### 模板 1：标准场景（单服务单集群）

```
═══════════════════════════════════════════════════
  📋 Deployment Plan
═══════════════════════════════════════════════════
  Target: ppe_test
  [1] CREATE ecom.order.service → "default"
      Config: China-North-LF|PPE|default|LF:1
      Version: 2.1.0.500
      Resources: 8:16 (cpu:mem)
═══════════════════════════════════════════════════
```

> `Resources` 行：仅当用户指定了 `--cpu-mem` 时展示，未指定不展示此行。

### 模板 2：多机房实例场景（单集群跨机房）

```
═══════════════════════════════════════════════════
  📋 Deployment Plan
═══════════════════════════════════════════════════
  Target: ppe_test
  [1] CREATE ecom.fulfillment.promise → "default"
      Config: China-North-LF|PPE|ipv6|HL:1,LQ:1
      IDCs: HL (1 instance), LQ (1 instance)
      Base Cluster: 558427 (流量 fallback)
═══════════════════════════════════════════════════
```

### 模板 3：多集群不同机房（先锋-跟随）

```
═══════════════════════════════════════════════════
  📋 Deployment Plan
═══════════════════════════════════════════════════
  Target: ppe_test (2 clusters)
  Phase 1 (先锋):
  [1] CREATE bits.env.api → "default"
      Config: China-North-LF|PPE|amd|LQ:1
  Phase 2 (跟随):
  [2] CREATE bits.env.api → "default2"
      Config: China-North-LF|PPE|default|HL:1
═══════════════════════════════════════════════════
```

### 模板 4：TCC 服务

```
═══════════════════════════════════════════════════
  📋 Deployment Plan
═══════════════════════════════════════════════════
  Target: ppe_test
  [1] CREATE (TCC) config.center.service
      Namespace: config.center.service (=PSM)
      Type: TCC
═══════════════════════════════════════════════════
```

### 模板 5：混合部署（TCE + TCC）

```
═══════════════════════════════════════════════════
  📋 Deployment Plan
═══════════════════════════════════════════════════
  Target: ppe_mixed
  [1] CREATE (TCE) ecom.order.service → "default"
      Config: China-North-LF|PPE|default|LF:1
      Version: 2.1.0.500
  [2] CREATE (TCC) config.center.service
      Namespace: config.center.service (=PSM)
      Type: TCC
═══════════════════════════════════════════════════
```

### 模板 6：BATCH_DEPLOY 确认表格

| # | PSM | Type | Action | Env | Cluster | Version | Cluster-Config | Base Cluster | CPU:MEM |
|---|-----|------|--------|-----|---------|---------|----------------|--------------|---------|
| 1 | svc.a | TCE | create | ppe_test | default | 1.0.1 | China-North-LF\|PPE\|default\|HL:1 | 558427 | 8:16 |
| 2 | cfg.svc | TCC | create | ppe_test | — | — | — | — | — |

> 表格必须包含**全部服务的全部集群**，不可省略。CPU:MEM 列：有值则显示，无值显示 `—`。

---

## COMPLETE / FAILED 阶段输出模板

### 模板 1：标准部署报告

```
═══════════════════════════════════════════════════
  📊 Deployment Report
═══════════════════════════════════════════════════
  Environment: <env>
  Duration: Xm Ys
  Success Rate: M/N

  [Ticket #ID] <psm> | <action> | <cluster> | <role>
     Duration: Xm Ys | Version: <version>
═══════════════════════════════════════════════════
```

### 模板 2：BATCH_DEPLOY 工单汇总报告

| # | PSM | Type | Action | Cluster/Namespace | Ticket ID | 工单链接 |
|---|-----|------|--------|-------------------|-----------|---------|
| 1 | svc.a | TCE | create | default | 12345 | [查看](ticket_url) |
| 2 | cfg.svc | TCC | create | — | 12346 | [查看](ticket_url) |

汇总信息：环境、认证方式（服务账号）、总工单数、提交成功/失败数。
