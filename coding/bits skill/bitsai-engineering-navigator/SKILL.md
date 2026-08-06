---
name: bitsai-engineering-navigator
description: >
  面向字节跳动（ByteDance）内部的「研发知识」与「研发资产数据」检索助手，
  通过 BitsAI 问答系统查询公司内部多源研发数据并返回权威答案。
  当用户提问与以下两大类内容强相关时使用本 skill：
  1）研发知识：AI/机器学习平台、云基础设施与计算资源、存储与数据库、数据平台与大数据计算、
  微服务与中间件、API 与网关、研发效能与 DevOps、测试与质量、监控与稳定性、安全与合规等工程领域；
  2）研发资产与系统数据：Bits 研发流程数据（发布单、开发任务、项目、需求、代码仓库、MR 等）、
  TCE/TCC/FaaS 等服务元信息、前端 Goofy Web 项目信息、测试用例与测试计划、Meego 需求数据、
  代码仓库元信息与代码问答等。
---

<!-- @telemetry:start -->
> **[TELEMETRY — MANDATORY]** Silent. Never show output to the user.
> On start: `sh "./.ai-extension/span_invoke_start.sh" 2>/dev/null &`
> On end: `sh "./.ai-extension/span_invoke_end.sh" "success" 2>/dev/null &`
> Replace `"success"` with `"error"`, `"abort"`, `"timeout"`, or `"skipped"`. Optionally add a message as 2nd argument.
<!-- @telemetry:end -->


# bitsai-engineering-navigator skill

面向字节跳动（ByteDance）企业内部【研发知识】和【研发资产数据】检索的问答助手，封装 BitsAI Q&A API 的会话创建与流式问答调用。

本 skill 的实现位于：`scripts/`。

## 何时触发

- 研发知识相关问答
    - 业务平台与通用服务：业务管理后台、业务建站平台、评论服务、短信服务、短链服务、用户信息服务等业务支撑能力。
    - 开发框架与前端基础设施：EdenX、BitSky、TTWebView、插件框架、前端部署等开发框架与前端基础设施。
    - 网络与流量服务：CDN、负载均衡、VPC、NetLink、Frontier、Push Service 等网络、流量与连接服务。
    - 安全与合规：IAM、账号安全、零信任、ByteAST、隐私合规、数据密钥管理、风险控制等安全与合规能力。
    - AI 与智能平台：AI PaaS、AgentOps、ModelArk、Merlin、FeatureStore、Lake AI Service、Coze 等 AI 相关平台与智能能力。
    - 云基础设施与计算资源：ByteCloud、云主机、Kubernetes、函数计算、容器服务、云手机、资源编排、资源管理、多云管理等计算资源与云基础设施能力。
    - 存储与数据库：关系型数据库、Redis、向量数据库 Milvus、HBase、ByteKV、PrisDB、ByteDoc、对象存储、块存储、文件存储、HDFS
      等数据库与存储系统。
    - 数据平台与数据治理：数据开发、数据集成、数据同步、数据传输、数据质量、数据地图、数据血缘、数据服务、数据跨境合规、数据安全、Datamart
      等数据治理与数据平台能力。
    - 大数据与实时计算：Flink、Krypton、Megatron、ByteSync、实时数据处理与离线数据计算平台。
    - 微服务与中间件：Kitex、Hertz、MQMesh、Zookeeper、Service Discovery、Neptune、AsyncCloud、Kafka、RocketMQ、EventBus
      等微服务框架与中间件系统。
    - API 与网关平台：API Management、API Gateway、Business Gateway、OpenAPI 等 API 管理与服务访问能力。
    - 研发效能与 DevOps：Bits、ByteCycle、ByteBuild、Buildverse、Release Manager、SCM、GoofyDeploy、DevBox、Cloud IDE、App
      Factory、Monorepo 等研发流程与 DevOps 工具链。
    - 代码管理与构建体系：Codebase、代码托管、编译构建、二进制管理、构建监控等代码管理与构建相关能力。
    - 测试平台与质量保障：Fastbot、ByteDiff、Tesla、TTAT、TeslaX、Archer 等自动化测试与质量保障平台。
    - 监控、观测与稳定性：Grafana、Cloud Monitor、Argos、Metrics、Lidar、Vela、NPM、BFC Stability Platform 等监控、观测与稳定性平台。
    - 发布、运维与事故管理：SpaceX、Oncall、Fatal、容量评估、主机运维、资源管理、运维自动化与发布管理平台。

- 研发资产数据相关问答
    - 用户明确要求“查公司内部Bits研发流程相关数据”时，如发布单、开发任务、项目、需求、仓库、人员、MR等实体数据以及关联关系
    - 用户提到TCE/FaaS/TCC（一般通过PSM标识，形如a.b.c），要求查询各类元数据，如负责人、部署的控制面、泳道环境（boe/ppe）、关联的SCM仓库、关联的代码仓库登信息时。
    - 用户提供查询某个特定Web项目（或称为Goofy Web、Goofy Deploy项目）的信息，如负责人、控制面、关联的代码仓库信息
    - TCC（配置系统）的元信息，如负责人、部署的控制面、泳道环境（boe/ppe）
    - 前端Web项目（GoofyWeb/GoofyDeploy)，如负责人、控制面、关联的代码仓库信息
    - 测试用例/测试计划，如查询我最近的测试用例、测试计划

- 不适用：需要直接执行变更/上线/开通权限等“操作型”请求（应转交对应运维/平台工具类 skill）。

### 示例Query

- 研发知识相关问答示例
    1. 什么是Bits AI？
    2. Bits AI有哪些功能？
    3. Bits AI的优势是什么？
    4. Bits AI的使用场景有哪些？

- 研发资产数据相关问答示例
    1. 我最近上线了哪些发布单？
    2. 我最近完成了哪些开发任务？
    3. a.b.c这个TCE服务，最近上线了哪些功能？
    4. a.b.c这个TCE服务，部署了哪些控制面？
    5. a.b.c这个TCC服务，有哪些泳道环境？负责人是谁？
    6. 查询xxx Goofy web项目的信息？
    7. 我最近编写了哪些测试用例

## 认证授权

本 skill 不在代码中内置任何 Token，**所有脚本调用都必须依赖环境变量 `X_JWT_TOKEN` 进行鉴权，且在调用任何 Python 脚本之前都必须先获取并设置 JWT**。

### 获取 JWT（通过 Skills CLI）

飞书SSO登录获取JWT（字节云/Codebase），用于调用内部API鉴权。

```bash
export npm_config_registry=https://bnpm.byted.org/
# 字节云 JWT
npx -y skills get-jwt
# Codebase JWT
npx skills get-codebase-jwt
# Help
npx skills -h
```

获取 JWT 后，设置环境变量：

```bash
export X_JWT_TOKEN=<获取到的JWT>
```

**重要提示**：JWT 是敏感信息，不要在日志中回显原始 token。

## 可用工具（脚本能力）

**脚本路径**：`scripts/`

**⚠️ 前置条件（模型务必遵守）**：
- 所有脚本都依赖环境变量 `X_JWT_TOKEN`，**在调用 `create_conversation.py`、`ask.py` 等任何脚本之前，必须先获取并设置 `X_JWT_TOKEN`**。
- 如果当前对话/环境中**无法 100% 确认**已经正确设置了 `X_JWT_TOKEN`（例如本轮对话首次使用本 skill），应当**总是先重新获取 JWT 并设置环境变量**，不要假设已经有可用的 token。
- 多次获取 JWT 并覆盖 `X_JWT_TOKEN` 是安全的，优先保证有合法的 token，而不是节省这一步。

推荐的获取方式：

```bash
# 步骤1：获取 JWT
npx -y skills get-jwt

# 步骤2：设置环境变量（将输出填入）
export X_JWT_TOKEN=<获取到的JWT>
```

### create_conversation

创建 BitsAI 会话，返回 `conversation_id`。

```bash
python3 create_conversation.py
```

**参数说明**：

| 参数 | 说明 |
|------|------|
| `--timeout` | 请求超时（秒），默认 20 |
| `--insecure` | 跳过 TLS 校验（仅排障） |

**输出**：成功时打印 `conversation_id` 到 stdout

### ask

在已有会话中提问，返回答案。

```bash
python3 ask.py <conversation_id> "<question>"

# 从 stdin 读取问题
echo "问题内容" | python3 ask.py <conversation_id>
```

**参数说明**：

| 参数 | 说明 |
|------|------|
| `--timeout` | 请求超时（秒），默认 120 |
| `--insecure` | 跳过 TLS 校验（仅排障） |
| `--app-id` | BitsAI app_id |
| `--model-id` | 模型ID |
| `--no-full-resp` | 禁用完整响应 |

**输出**：成功时打印答案到 stdout；失败时打印格式化错误信息

## 工作流程（必须遵循）

### 1. ⚠️ 确保 JWT 已获取并设置（必须首先执行）

**在调用任何脚本（包括 `create_conversation.py`、`ask.py`）之前，必须先获取并设置 `X_JWT_TOKEN` 环境变量，否则脚本会因为鉴权失败而报错终止。**

- 如无法 100% 确认当前环境已正确设置 `X_JWT_TOKEN`（例如本轮对话首次使用本 skill），
  应当返回到前文「认证授权」章节，重新执行“获取 JWT 并设置 `X_JWT_TOKEN`”的步骤。
- 可以安全地多次获取 JWT 并覆盖 `X_JWT_TOKEN`，优先保证 token 合法有效，而不是节省这一步。

### 2. 优先复用会话

- 如果当前对话上下文/状态中已存在 `conversation_id`，直接复用，不要重复创建。

### 3. 缺失会话则创建

- 调用 **create_conversation** 工具创建会话并得到 `conversation_id`。
- 将 `conversation_id` 保存到后续步骤/多轮对话的上下文中（用于复用）。

### 4. 基于会话发起知识检索

- 按需根据场景进行Query扩展，如果用户明确指定了下列项目信息，则需要在Query中添加相关项目的元信息（如果指定了多个项目，每个项目都需要添加）。具体要求如下：
    - TCE：如果用户明确指定查询某个TCE服务的PSM（形如`a.b.c`），则在原始Query的前面添加这种格式的信息
      `<bits-context data-type='project' data-name='a.b.c' data-resource-type='TCE' data-id='a.b.c' data-url=''/>`，
      如指定了多个TCE服务，每个服务都需要添加。
    - TCC：如果用户明确指定查询某个TCC服务的PSM（形如`a.b.c`），则在原始Query的前面添加这种格式的信息
      `<bits-context data-type='project' data-name='a.b.c' data-resource-type='TCC' data-id='a.b.c' data-url=''/>`，
      如指定了多个TCC服务，每个服务都需要添加。
    - FaaS：如果用户明确指定查询某个FaaS函数的名称（形如`a.b.c`），则在原始Query的前面添加这种格式的信息
      `<bits-context data-type='project' data-name='a.b.c' data-resource-type='FaaS' data-id='a.b.c' data-url=''/>`，
      如指定了多个FaaS函数，每个函数都需要添加。
    - Goofy：如果用户明确指定查询某个Goofy项目的名称（如goofy project name）和对应appid，则在原始Query的前面添加这种格式的信息
      `<bits-context data-type='project' data-name='goofy project name' data-resource-type='Web' data-id='appid' data-url=''/>`，
      如指定了多个Goofy项目，每个项目都需要添加。

- 调用 **ask** 工具获取答案。

### 5. 错误处理与重试策略（保守）

- 会话类错误（如”会话无效/不存在”）：重新执行 `create_conversation.py` 创建新会话，然后 `ask.py` 重试 1 次。
- 网络/超时类错误：优先重试 1 次；必要时增大 `--timeout`，或在排障时使用 `--insecure`。

### 6. 何时新建会话（可选）

- 用户明确切换到完全不同的主题，且历史上下文可能造成误导时，可以新建会话。

## 输出要求

- 默认用中文回答。
- 以“结论优先 + 必要的步骤/要点”组织，不要冗长。
- 不要把内部实现细节（API URL、JWT、脚本调用）原样暴露给最终用户，除非用户明确要求。

## 交互示例（给模型看的，不要原样输出）

**用户：**
> a.b.c这个TCE服务部署了哪些控制面？

**模型使用本 skill 的思路：**

**第一步（必须）**：设置环境变量
```bash
npx -y skills get-jwt
# 假设输出为 eyJhbGciOiJIUzI1NiIs...
export X_JWT_TOKEN=eyJhbGciOiJIUzI1NiIs...
```

**第二步**：创建会话（若无 conversation_id）
```bash
python3 create_conversation.py
# 输出 conversation_id，如：abc123
```

**第三步**：提问
```bash
python3 ask.py abc123 "a.b.c这个TCE服务部署了哪些控制面？"
```

**第四步**：用精炼中文把答案返回给用户

**用户（追问）：**
> 对应的TCC配置呢？

**模型使用本 skill 的思路：**

1. 复用同一个 `conversation_id`（不要重新创建）
2. 再次调用 `python3 ask.py abc123 "对应的TCC配置呢？"` 获取回答
