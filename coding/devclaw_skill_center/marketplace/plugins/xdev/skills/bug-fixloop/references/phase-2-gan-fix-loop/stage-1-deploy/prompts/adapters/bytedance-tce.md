# TCE 服务部署

当内场服务需要部署到 TCE 时使用此流程。

## 前置条件

1. **内网环境** — bytedcli 需要内网访问权限

## 命令参考

bytedcli TCE 部署相关命令的完整用法，请读取已安装的 bytedcli skill 文件：

- **调用方式和全局参数**：`$BYTEDCLI_SKILLS_DIR/bytedance-tools/references/invocation.md`
- **TCE 子命令详情**：`$BYTEDCLI_SKILLS_DIR/bytedance-tce/SKILL.md`
- **认证方式**：`$BYTEDCLI_SKILLS_DIR/bytedance-auth/SKILL.md`

执行任何命令前，先通过 `--help` 确认当前版本的参数。

## 部署流程

### Step 1: 确认部署信息

需要以下参数：

| 参数 | 必填 | 说明 | 默认值 |
|------|------|------|--------|
| `psm` | 是 | TCE 服务标识 | - |
| `env` | 否 | 目标泳道名 | 自动生成 `boe_costudio_<随机4位>` |
| `branch` | 否 | 部署分支 | `master` |

### Step 2: 检查泳道是否已有实例，决定 action

通过 bytedcli TCE 的实例查询命令（参考 bytedance-tce skill），查询目标泳道的实例列表。

> **重要**：BOE 环境下所有 TCE 命令都必须指定 BOE 站点，否则会返回 404 service not found。

**判断逻辑**（解析 JSON 响应）：
- `status` 为 `"success"` 且 `data` 非空 → 后续部署使用 upgrade 动作
- `status` 为 `"error"` 或 `data` 为空 → 后续部署使用 create 动作（首次部署）

### Step 3: 执行部署

通过 bytedcli TCE 的泳道部署命令执行部署。

**关键参数**（通过 --help 确认具体 flag 名）：
- 泳道名
- 标准环境（如 boe）
- PSM
- 部署分支
- 部署动作（create 或 upgrade，根据 Step 2 判断）
- SCM 基线（一般为 prod）

### Step 4: 轮询实例状态

每 15 秒查询实例列表，直到出现就绪状态的实例（超时 20 分钟）。

**状态匹配容错规则**：

即使使用 JSON 输出，实例状态值可能是中文或英文，取决于 CLI 版本和 locale 设置。必须同时匹配两种形式：

| 就绪（成功） | 等待中 | 失败 |
|-------------|--------|------|
| `Running` 或 `运行中` | `Pending` 或 `待处理` | `Failed` 或 `失败` |
| | `Creating` 或 `创建中` | `CrashLoopBackOff` |
| | `Building` 或 `构建中` | |
| | `Restarting` 或 `重启中` | |

**注意**：解析返回的 JSON 时，先查看实际返回结构确定实例列表的字段名（可能是 `pods`、`instances` 等），不要假设固定字段名。

### Step 5: 确认结果

部署成功后，记录泳道名供后续测试使用。

## 部署失败修复流程

当 TCE 部署失败时（编译错误、启动崩溃等），按照以下流程修复。

### 适用场景

- TCE deploy-lane 返回失败
- 部署后实例状态持续为 Failed/Pending
- 容器启动后立即崩溃（CrashLoopBackOff）

### 修复步骤

#### 步骤 1：获取错误信息

如果部署命令直接返回错误，使用其输出。如果实例状态异常，查看实例详情获取错误日志。

#### 步骤 2：分析错误

仔细阅读错误信息，判断错误类别：

**编译错误（构建阶段失败）**：
- 语法错误、类型错误、未定义引用、包导入错误
- 识别出错的文件和行号

**运行时错误（容器启动后崩溃）**：
- panic 堆栈：查找 `panic:`、`goroutine`、`runtime error`
- 错误日志：查找 `fatal`、`error`、`failed to`
- 常见原因：nil pointer、配置缺失、端口绑定失败

#### 步骤 3：定位问题文件

根据错误信息中的文件路径和行号，用 Read 工具查看相关代码。

#### 步骤 4：实施修复

**最小改动原则**，只修复导致部署失败的错误。

允许的操作：
- 修复语法错误、类型不匹配
- 添加缺失的 import
- 修复未定义的变量/函数引用
- 修复接口未实现的方法
- 修复运行时 panic（nil pointer、越界等）
- 修复配置错误或缺失的默认值

禁止的操作：
- 不删除功能代码
- 不修改测试文件
- 不重构代码
- 不添加新功能

#### 步骤 5：验证编译

```bash
cd $BUSINESS_REPO_PATH && go build ./...
```

#### 步骤 6：推送并重新部署

```bash
cd $BUSINESS_REPO_PATH
git add -A
git commit -m "fix: resolve deployment/compilation error"
git push origin $BRANCH
```

然后重新执行部署步骤。

**部署失败修复最多重试 3 次**。如果 3 次修复后部署仍然失败：
1. 在 `$OUTPUT_DIR/iteration_$ITERATION/fix_summary.md` 中记录详细的部署失败信息
2. 终止当前迭代循环，输出最终摘要（状态为失败）
3. 通知用户需要人工介入

#### 步骤 7：记录修复

将修复简要写入 `$OUTPUT_DIR/iteration_$ITERATION/fix_summary.md`：
1. 问题的根本原因
2. 做了什么修改
3. 为什么这个修改能解决问题

## 注意事项

- 首次部署需 5-10 分钟（构建+启动），复用已有泳道约 15 秒
- 临时泳道使用完毕后需手动清理，避免资源浪费
- bytedcli 需要内网访问权限（VPN 或办公网络）


---

# AGW IDL 同步（合并自原 stage-1.5）

> 以下内容原为 stage-1.5 独立文件 agw-idl-sync.md。在 bug-fixloop 中，因为 AGW 同步只在 PROFILE=bytedance-tce 时执行，所以合并到本 adapter 文件。

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
