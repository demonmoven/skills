# ENV 环境预检流程

本文档用于 `api-test` 在 BOE/PPE/测试/预览环境执行接口测试前，检查服务在目标 ENV/泳道中的集群和实例状态，必要时通过 `bits-env` skill 部署服务，并产出后续接口测试所需的 `<CLUSTER>` 与 `<VDC>`。

## 适用范围

- 仅适用于 BOE、PPE、测试环境、预览环境或用户未明确说明环境类型的场景。
- 如果用户指定生产环境、线上环境、prod 环境，不执行本文档流程，继续使用 `SKILL.md` 中生产环境安全确认逻辑。

## 变量说明

- `<ENV>`: 用户指定或确认后的 ENV/泳道名称。
- `<psm>`: 用户指定或确认后的服务 PSM。
- `<CLUSTER>`: 本流程最终选择的集群名称。
- `<VDC>`: 本流程最终选择的 IDC/VDC。

## 强制依赖

- 本流程是接口测试前的**强制检查点**：进入接口测试前，必须先确认 `<ENV>` 中 `<psm>` 的集群与实例状态，**不得跳过、不得“先测试再排查环境”**。
- 环境、泳道、服务、集群、实例查询，以及服务部署，**优先调用 `bits-env` skill** 完成。
- **降级路径（`bits-env` skill 未安装/不可用时）**：禁止因依赖缺失而跳过本流程，必须改用 `bitscli env` CLI 完成只读的环境/实例状态预检（详见下方「降级路径：bitscli env CLI」），完成预检后再继续。
- 部署等写操作不在降级路径范围内：若需部署且 `bits-env` skill 不可用，提示用户安装 `bits-env` skill 或手动部署，不得用 CLI 自行编造部署参数。
- 如果 `bits-env` 或 `bitscli env` 提示缺少认证信息，提示用户先执行 `bytedcli login` 或 `agentbuddy login`，完成后重试当前步骤。
- 禁止复用从历史流量推荐参数里拿到的JWT来做环境查询认证

### 安装与更新

若 `bits-env` skill 或底层环境 CLI（`bitscli env` / `bits_env_cli`）不存在（`command not found`）或版本过旧，以及部署写操作所需的 `agentbuddy` 安装与 JWT 获取、登录鉴权失败处理，统一参考 [runtime-support.md](../support/runtime-support.md)。

### 降级路径：bitscli env CLI

仅在 `bits-env` skill 不可用时，按以下命令完成环境状态只读预检（命令仅做查询，不产生写操作）：

| 用途 | 命令 |
|------|------|
| 按关键词/泳道搜索环境，确认 `<ENV>` 是否存在 | `bitscli env env-search --keyword <ENV>` |
| 查询 `<ENV>` 中服务 `<psm>` 的集群/实例信息 | `bitscli env instance-meta --env <ENV> --psm <psm>` |

示例：

```bash
bitscli env env-search --keyword ppe_hzw_itest
bitscli env instance-meta --env ppe_hzw_itest --psm env.t.api
```

依据上述返回信息，按「服务集群状态定义」判断状态并选择 `<CLUSTER>` 与 `<VDC>`；若需要部署，回到 `bits-env` skill 主路径处理。

## 服务集群状态定义

| 状态 | 判定条件 |
|------|----------|
| 未部署 | 环境不存在、服务在该环境下未部署，或集群不存在 |
| 部署成功 | 服务在该环境下已部署，且存在集群，且至少有一个实例状态为 `运行中` |
| 部署未成功 | 服务在该环境下已部署，且存在集群，但没有实例或实例状态均不是 `运行中` |

## 执行流程

### Step A: 查询服务集群状态

1. 调用 `bits-env` skill 查询环境 `<ENV>` 中服务 `<psm>` 的集群和实例信息；若 `bits-env` skill 不可用，改用上方「降级路径：bitscli env CLI」的 `env-search` 与 `instance-meta` 命令查询。
2. 将查询结果摘要展示给用户，至少包含环境、服务、集群、IDC/VDC、实例数量、运行中实例数量、状态，并保留 `bits-env` 或 `bitscli env` 返回的关键原始字段或错误信息，便于用户排查。
3. 根据查询结果判断服务集群状态：
   - `未部署`: 进入 Step B。
   - `部署成功`: 进入 Step D。
   - `部署未成功`: 进入 Step B。

### Step B: 推断下一步操作

当状态为 `未部署` 或 `部署未成功` 时，根据 `bits-env` 返回内容推断下一步：

- 如果环境名 `<ENV>` 或服务名 `<psm>` 明显错误，提示用户修正；用户确认后返回 Step A。
- 如果返回结果明确适合部署，向用户说明原因，并确认是否部署服务 `<psm>` 到环境 `<ENV>`。
- 仅当用户已经明确要求自动部署，或在展示部署原因后明确确认继续部署，才进入 Step C。
- 如果用户拒绝部署，退出当前流程，并提示无法从该 ENV 产出 `<CLUSTER>` 与 `<VDC>`。

### Step C: 部署服务

1. 调用 `bits-env` skill 将服务 `<psm>` 部署到环境 `<ENV>`。
2. 等待部署执行完成。
3. 如果部署成功，返回 Step A 重新查询并确认集群实例状态。
4. 如果部署失败，展示失败信息和 `bits-env` 返回的关键原因，由用户决定是否重试、修正环境/服务名或退出流程。

### Step D: 展示集群与实例信息

以表格形式展示 `bits-env` 返回的集群和实例信息：

| ENV | PSM | Cluster | IDC/VDC | 实例 | 实例状态 | 是否候选 |
|-----|-----|---------|---------|------|----------|----------|
| `<ENV>` | `<psm>` | `<cluster name>` | `<idc>` | `<instance>` | `<status>` | `<yes/no>` |

### Step E: 选择 `<CLUSTER>` 与 `<VDC>`

1. 优先选择包含 `运行中` 实例的集群。
2. 如果只有一个满足条件的集群，默认选择该集群，设置 `<CLUSTER>` 与 `<VDC>`。
3. 如果多个集群满足条件，必须展示候选集群表格并让用户选择；只有存在明确优先级规则或用户已指定目标集群时，才可自动选择并说明依据。
4. 如果无法获取 `<CLUSTER>` 或 `<VDC>`，展示失败原因并由用户决定是否重试查询、修正参数或退出流程。

## 输出

流程成功后，必须明确记录以下变量供 Step 5 使用：

- `<ENV>`
- `<CLUSTER>`
- `<VDC>`
