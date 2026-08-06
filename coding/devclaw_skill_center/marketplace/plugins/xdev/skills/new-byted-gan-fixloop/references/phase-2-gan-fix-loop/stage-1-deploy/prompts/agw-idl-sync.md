# AGW IDL 同步流程

## 用途

TCE 部署后，同步 AGW 网关的 IDL 配置（及路由规则）到 BOE 泳道环境。确保网关路由规则与最新的接口定义一致。

## 命令参考

AGW 相关命令的完整用法，请读取已安装的 bytedcli skill 文件：

- `$BYTEDCLI_SKILLS_DIR/bytedance-agw/SKILL.md`
- `$BYTEDCLI_SKILLS_DIR/bytedance-tools/references/invocation.md`

执行前先通过 `--help` 确认当前版本参数。

## 工作流

### Step 1: Service-ID 查找

在前置检查阶段执行一次，结果缓存到 `AGW_SERVICE_ID` 变量。

- 使用 bytedcli agw 的 service 搜索命令，传入 PSM 作为关键字
- 从返回结果中提取第一个非空的 `service_id`
- 无结果时输出提示信息，跳过后续 IDL 同步步骤，不阻断主流程

### Step 2: IDL 同步 + 发布

在迭代循环首轮（Stage 1 部署后、Stage 2 测试前）执行。

1. 读取 `$BYTEDCLI_SKILLS_DIR/bytedance-agw/SKILL.md`，了解所有可用的 IDL 相关命令
2. **分析 IDL 变更内容，选择合适的命令**：
   - 对比 `IDL_BRANCH` 与基线分支的 diff，判断本次变更的性质
   - **涉及新增接口、修改路由路径等路由变更** → 选择同时更新 IDL 和路由的命令（确保网关路由规则与接口定义同步生效）
   - **仅修改已有接口的字段定义（入参/出参结构变更）** → 选择仅更新 IDL 的命令即可
   - 具体命令名以 bytedance-agw SKILL.md 文档为准，不硬编码
3. 调用选定的命令，传入 service-id、泳道环境名、IDL 分支名（IDL_BRANCH）、发布模式
4. `changed: false` 表示 IDL 已是最新版本，无需发布（命令具有幂等检测，重复调用安全）

### Step 3: 错误处理与泳道注册

AGW IDL 同步失败时，先检查是否为泳道环境未注册错误，若是则自动注册后重试。

**错误检测**：检查命令的错误输出，匹配泳道未注册相关关键字：

```
泳道未注册|env.*(not found|not registered)|lane.*(not found|not registered)|environment.*not.*exist
```

**泳道未注册 → 自动注册并重试**：

1. 读取 `$BYTEDCLI_SKILLS_DIR/bytedance-agw/SKILL.md` 获取泳道/环境注册命令的用法
2. 执行 `--help` 确认当前版本参数
3. 调用注册命令，传入 `AGW_SERVICE_ID` 和 `TCE_LANE`
4. 注册成功后，重新执行 Step 2 的步骤 3（调用 IDL 同步命令）
5. 如果重试仍然失败，阻断主流程并报告错误

**其他错误 → 阻断主流程**：非泳道注册错误直接阻断，停止执行并报告错误。

## 错误处理策略

**阻断（block on failure）**：AGW IDL 同步失败时阻断主流程，停止执行并向用户报告错误，不继续进入 Stage 2 测试。

AGW IDL 与网关路由规则的一致性是有效测试的前提条件。唯一的自动恢复路径是泳道环境未注册时的自动注册重试（见 Step 3）。
