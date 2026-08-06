# charge-atom-code-gen — 计费域编码原子 SKILL

## 定位

计费域统一编码入口。子流程只需调用本 SKILL 传入需求描述和仓库信息，本 SKILL 内部根据**变更复杂度**自动选择最优编码策略：

| 策略 | 别名 | 适用场景 | 耗时预期 |
|---|---|---|---|
| **Plan A: Mira 直写** | `mira_patch` | 简单/模式化变更 | ~8 min |
| **Plan B: Coco 全量** | `coco_full` | 复杂/探索性变更 | ~20 min |

## 输入参数（由子流程传入）

```yaml
repo_id: 849505                          # Codebase 仓库 ID
repo_path: bytepay/bytepay_charge_offline_engine  # 仓库路径
base_branch: master                      # 基线分支
new_branch: feature/<slug>               # 新建分支名
target_branch: master                    # MR 目标分支
prd_title: "渠道离线计费新增退款计算器"       # PRD 标题
prd_link: "https://..."                  # PRD 飞书链接
prd_requirements: |                      # PRD 解析后的实现要求（结构化）
  - 新增 FixedRefundCalculator
  - 新增 RefundFeeModeEnum.FIXED_REFUND
  - 注册到 CalculatorAdapter
  - 单测覆盖
change_type: "new_calculator"            # 变更类型标签
reference_files:                         # 需要参考的文件路径列表（可选）
  - src/main/java/.../CalculatorAdapter.java
  - src/main/java/.../RefundFeeModeEnum.java
  - src/main/java/.../refund/PartRefundCalculator.java
mr_title_prefix: "[fee-rule]"            # MR 标题前缀
```

## 输出（返回给子流程）

```yaml
strategy_used: mira_patch | coco_full    # 实际使用的策略
coco_task_id: "774328771729120"          # Coco 任务 ID
commit_id: "c6c733d7..."                 # 40位 commit sha
branch: feature/fixed_refund_calculator_v2
changed_files:                           # 变更文件列表
  - path: src/.../FixedRefundCalculator.java
    type: new
  - path: src/.../RefundFeeModeEnum.java
    type: modified
bypass_changes:                          # 旁路改动（非业务需求引起）
  - path: src/.../ChargeDataSourceTest.java
    reason: "放宽无外部配置时的断言以通过 mvn test"
compile_pass: true
test_pass: true
```

## 策略路由逻辑（核心）

### 判定规则

**简单 → Plan A (mira_patch)**：
1. 变更类型为已知模式：`new_calculator`、`new_enum_value`、`config_change`、`add_field`
2. 影响文件数 ≤ 6（可从 PRD requirements 推断）
3. 有明确的参考实现（同模块已有同类代码）
4. 不涉及跨模块/跨服务调用链改动
5. 不涉及数据库 schema 变更

**复杂 → Plan B (coco_full)**：
1. 变更类型为未知模式或标记为 `complex`
2. 影响文件数 > 6 或无法预估
3. 需要理解跨模块调用链（如引擎调度逻辑改动）
4. 涉及新增 RPC/HTTP 接口
5. PRD 中含"重构"、"架构调整"、"迁移"等关键词
6. 子流程或用户显式指定 `strategy: coco_full`

**降级规则**：
- Plan A 执行中若 `mvn compile` 失败（非环境问题）→ 自动降级到 Plan B
- 降级时传入 Plan A 生成的代码作为 Coco 的参考

### 策略覆盖

子流程或用户可通过 `force_strategy: mira_patch | coco_full` 强制指定策略。

---

## Plan A 执行流程：Mira 直写 + Coco 轻量应用

### A1. 读取参考文件

通过 `bytedcli codebase repo file` 读取 `reference_files` 中列出的文件：

```bash
bytedcli --json codebase repo file "<path>" --repo-id <repo_id> --revision <base_branch>
```

- 响应中 `data.file.Content` 为 base64 编码，需解码
- 若某文件 NotFound → 记录警告，不阻塞

### A2. Mira 本地生成代码

基于读取到的参考文件 + PRD 要求，Mira 在本地生成所有需要 新增/修改 的文件：
- 遵循现有代码风格
- 生成完整文件内容（非 diff）
- 同时生成单测文件

### A3. 组装 Coco 轻量任务消息

精简模板：任务说明 + 仓库信息 + 完整文件内容 + 约束（仅 apply+compile+test+push）

### A4. 发送 Coco 任务并监控

```bash
bytedcli --json coco task send --agent-name sandbox --message "$MSG"
```

混合监控：后台 subscribe + 主线程每 60s task get 轮询

### A5. 降级检查

- compile PASS + test PASS + commit_id 存在 → 成功
- compile FAIL → 触发降级到 Plan B

---

## Plan B 执行流程：Coco 全量编码

### B1. 组装六段全量消息（见 charge-dev-router SKILL）
### B2. Coco Round-1（代码+测试）+ 监控
### B3. Coco Round-2（git push）

降级场景额外附加 Plan A 代码作参考。

---

## 通用约束

- 不操作生产环境
- 不暴露 token
- 不使用 Codebase PAT
- Coco agent 统一用 sandbox
- Maven IPv6 必带
- 旁路改动必须记录

## 异常处理

| 异常 | 处理 |
|---|---|
| codebase repo file NotFound (参考文件) | 警告不阻塞 |
| Plan A compile 失败 | 降级 Plan B |
| Plan B Status=failed | 重试 ≤ 1 次 |
| git push 被拒绝 | 返回错误 |
| Coco AUTH_REQUIRED | 报错,不创建 PAT |
| 超时(>30min) | 中断 |

