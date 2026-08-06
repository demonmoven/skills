# AGW IDL 更新流程

## 用途

TCE 部署后，同步 AGW 网关的 IDL 配置到 BOE 泳道环境。确保网关路由规则与最新的接口定义一致。

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
- 无结果时输出提示信息，跳过后续 IDL 更新步骤，不阻断主流程

### Step 2: IDL 更新 + 发布

在迭代循环首轮（Stage 1 部署后、Stage 2 测试前）执行。

- 使用 bytedcli agw 的 IDL 更新命令，传入 service-id、泳道环境名、发布模式
- `changed: false` 表示 IDL 已是最新版本，无需发布（命令具有幂等检测，重复调用安全）

## 错误处理策略

**非阻断（warn and continue）**：AGW IDL 更新失败不应阻断主流程。

原因：
- 部分服务可能没有 AGW 配置
- AGW 配置不影响 TCE 服务本身的运行
- 测试可能不依赖 AGW 网关路由

失败时输出警告信息，继续执行 Stage 2 测试。
