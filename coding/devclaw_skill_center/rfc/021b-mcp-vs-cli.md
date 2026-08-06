# 2.1.2 MCP vs CLI

> **本节目标**：理解 MCP 和 CLI 这两种外部能力接入方式的定位、区别和选型策略，以及各自的实现最佳实践。

---

## 概念与定位

### 什么是 MCP

**MCP（Model Context Protocol）** 是 Anthropic 于 2024 年底发布的开放协议，为 AI Agent 提供了一种**标准化的方式连接外部数据源和工具**。

一句话定位：**MCP 是 Agent 连接外部世界的标准协议——类似 USB 接口，让任何 Agent 都能以统一方式接入任何外部服务**。

MCP 的职责边界：

```text
MCP  = Agent ↔ 工具/数据（纵向连接）
       Agent 通过 MCP 访问外部 API、数据库、SaaS 服务
```

### 什么是 CLI（命令行工具）

**CLI** 是另一种让 Agent 调用外部能力的方式——Agent 直接通过 Bash 执行命令行工具，解析 stdout 输出。

```text
CLI 方式:  Agent → Bash("gh pr list --json title") → 解析 JSON 输出
MCP 方式:  Agent → MCP Client → MCP Server(GitHub) → 返回结构化数据
```

LLM 天然擅长 CLI——训练数据中包含数百万条 Unix 命令和 pipe 链，模型对 CLI 的模式识别深嵌于权重中。

---

## MCP vs CLI：核心对比

```text
                ┌─────────────┐
                │   Agent     │
                └──────┬──────┘
           ┌───────────┴───────────┐
           │                       │
    ┌──────▼──────┐         ┌──────▼──────┐
    │    CLI      │         │    MCP      │
    │  (Bash)     │         │  (Protocol) │
    └──────┬──────┘         └──────┬──────┘
           │                       │
    ┌──────▼──────┐         ┌──────▼──────┐
    │  命令行工具  │         │ MCP Server  │
    │ gh/curl/jq  │         │  (进程/远程) │
    └──────┬──────┘         └──────┬──────┘
           │                       │
    ┌──────▼──────────────────▼──────┐
    │   外部数据源 / API / 服务       │
    │   GitHub / DB / Slack / ...    │
    └────────────────────────────────┘
```


| 维度          | CLI 的优势                | MCP                            |
| ----------- | ---------------------- | ------------------------------ |
| **调用方式**    | `Bash("command args")` | MCP 协议 → MCP Server            |
| **上下文开销**   | 极低（1K-9K tokens/次）     | 高（MCP schema 占 30K-80K tokens） |
| **可靠性**     | ~100%（命令要么成功要么报错）      | ~72%（协议开销、Server 稳定性）          |
| **成本**      | 10K 次/月 ~$3            | 10K 次/月 ~$55                   |
| **LLM 友好度** | 极高（训练数据包含海量 CLI 示例）    | 中（需要理解 MCP tool schema）        |
| **安全性**     | 需防范命令注入，但有沙箱           | 需防范恶意 Server（CVE-2025-6514）    |
| **适用场景**    | 本地文件、Git、API 调用、数据处理   | SaaS 集成、数据库、需身份认证的服务           |
| **发现性**     | Agent 需要知道有哪些 CLI      | MCP Server 自动暴露 tool 列表        |
| **有状态性**    | 无状态（每次调用独立）            | 可有状态（Server 维护连接）              |


### 为什么 CLI 在 2026 年崛起

1. **Token 效率碾压**：CLI 每次调用 1K-9K tokens vs MCP 的 30K-80K tokens schema 预加载
2. **训练数据优势**：LLM 在数百万 Unix pipe 链上训练过，天然会组合 CLI 工具
3. **零基础设施**：不需要运行 Server 进程，直接 `Bash` 调用
4. **可组合性**：Unix 管道哲学天然支持 `cmd1 | cmd2 | cmd3` 链式处理

---

## 选型策略

### 核心原则：增量 CLI 优先，存量MCP逐步迁CLI

**新建场景一律 CLI 优先**

**存量 MCP 逐步用 CLI 替代**——业界已出现明确的 MCP → CLI 迁移趋势：


| 迁移案例                    | 说明                                                                            |
| ----------------------- | ----------------------------------------------------------------------------- |
| **飞书 MCP → lark-cli**   | 字节内部的飞书 MCP Server 已升级为 `lark-cli` 命令行工具，覆盖文档、消息、日历、审批等全场景                    |
| **字节云 MCP → bytedcli**  | TCE、日志、AGW、认证等 57 个字节云域的 MCP 已统一封装为 `bytedcli` CLI，且 MCP Server 本身也复用 CLI 命令树 |
| **GitHub MCP → gh CLI** | 社区从 93 tool / 55K tokens 的 GitHub MCP Server 迁移到零开销的 `gh` CLI                 |


### 什么不该做成 MCP/CLI

MCP 和 CLI 本质上都是"让 Agent 调用外部能力"的通道，以下场景不应该走这条路：


| 不该做的                                 | 应该用什么                    | 理由                                            |
| ------------------------------------ | ------------------------ | --------------------------------------------- |
| 把一段 prompt / 工作流指令包装成 MCP Tool 或 CLI | **Skill**                | Skill 是指令注入，MCP/CLI 是能力调用——两者定位不同，Skill 零协议开销 |
| 把纯本地文件操作做成 MCP/CLI                   | **Agent 内置工具**           | Agent 自带 Read/Write/Bash/Glob/Grep，不需要外部通道    |
| 每个细小功能都独立做一个 MCP Server 或 CLI 命令     | **合并为域级 CLI + Skill 路由** | 参考 bytedcli 做法：一个 CLI 统一 57 个域，Skill 按域拆分做路由  |
| 返回巨量未经筛选的数据                          | **分页 + 筛选 + `--json`**   | 无论 MCP 还是 CLI，返回数据都要精简——上下文溢出后 Agent 什么都干不了   |


---

## 各 Coding Agent 的配置方式

### MCP 配置

MCP Server 有两种接入方式：

| 方式 | 说明 | 适用场景 |
|------|------|----------|
| **stdio（本地进程）** | Agent 启动一个子进程作为 MCP Server，通过 stdin/stdout 通信 | 本地工具、CLI 包装 |
| **URL（远程服务）** | Agent 通过 HTTP + SSE 连接远程 MCP Server | 团队共享服务、云端部署 |

#### 各 Agent 配置总览

| Agent | 配置文件 | CLI 安装命令 |
|-------|----------|-------------|
| **Claude Code** | `.claude/settings.json` | `claude mcp add` |
| **Codex CLI** | `.codex/config.toml` | `codex mcp add` |
| **Coco / TRAE CLI** | `.coco/coco.yaml` | YAML 手动编辑 |
| **Trae / Trae CN (IDE)** | IDE Settings | GUI 配置 |

#### Claude Code

```bash
# 方式一：stdio（本地进程）
claude mcp add --scope project brave-search -- npx -y @anthropic-ai/mcp-server-brave-search

# 带环境变量
claude mcp add --scope project brave-search \
  --env BRAVE_API_KEY=your-key \
  -- npx -y @anthropic-ai/mcp-server-brave-search

# 方式二：URL（远程服务）
claude mcp add --transport http --scope project remote-server https://mcp.example.com/sse

# 管理命令
claude mcp list              # 列出所有已配置的 Server
claude mcp get brave-search  # 查看某个 Server 的配置
claude mcp remove brave-search  # 移除
```

scope 说明：`--scope user`（全局）/ `--scope project`（项目级）

#### Codex CLI

```bash
# 方式一：stdio（本地进程）
codex mcp add brave-search -- npx -y @anthropic-ai/mcp-server-brave-search

# 带环境变量
codex mcp add brave-search --env BRAVE_API_KEY=your-key -- npx -y @anthropic-ai/mcp-server-brave-search

# 方式二：URL（远程服务）
codex mcp add remote-server --url https://mcp.example.com --bearer-token-env-var MCP_TOKEN

# 管理命令
codex mcp list              # 列出
codex mcp remove brave-search  # 移除
```

#### Coco / TRAE CLI

手动编辑 `.coco/coco.yaml`：

```yaml
mcp:
  servers:
    brave-search:
      command: npx
      args: ["-y", "@anthropic-ai/mcp-server-brave-search"]
      env:
        BRAVE_API_KEY: your-key
```

#### Trae / Trae CN (IDE)

通过 IDE Settings → MCP 界面 GUI 配置，支持 stdio 和 URL 两种方式。

### CLI 工具配置

CLI 工具不需要额外配置——Agent 通过 Bash 直接调用：

```bash
# GitHub CLI
gh pr list --json title,number,state

# 飞书 CLI
lark-cli docs +fetch --doc "https://..."

# 字节云 CLI（57 个域统一入口）
bytedcli tce service list --json
bytedcli log search --psm xxx --query "error" --json

# cURL + jq（万能组合）
curl -s https://api.example.com/users | jq '.data[] | {id, name}'
```

---

## 实现最佳实践

### MCP Server 实现要点

> 官方文档：[Build an MCP Server](https://modelcontextprotocol.io/docs/develop/build-server) | [GitHub: modelcontextprotocol](https://github.com/modelcontextprotocol/modelcontextprotocol)

1. **Tool 数量克制**：每个 Server 暴露的 tool 控制在 10-15 个以内，超过 20 个就应拆分
2. **Tool 描述必须精准**：description 是 Agent 判断是否调用的唯一依据
3. **返回数据精简**：不要返回整张表，做好分页和筛选
4. **凭据不硬编码**：通过 `env` 字段注入，不要写在配置文件中
5. **错误信息要有指导性**：Agent 看到报错后要能知道怎么修

### CLI 实现要点：让 LLM 更容易使用

以 bytedcli 的实践为例，一个 LLM 友好的 CLI 应该做到：`git@code.byted.org:byteapi/bytedcli.git`

**1. 统一输出格式**

```bash
# 默认人类友好文本，-j/--json 输出结构化 JSON
bytedcli tce service list              # 表格输出
bytedcli tce service list --json       # JSON 输出（Agent 消费）
```

Agent 模式下统一使用 `--json`，输出走统一工具函数，不要散落 `console.log`。

**2. 参数全部用 option，不用位置参数**

```bash
# Good: 显式 option，Agent 易理解
bytedcli tce service get --id demo-service --cluster default

# Bad: 位置参数，Agent 容易搞错顺序
bytedcli tce service get demo-service default
```

**3. 错误输出结构化**

```bash
# 统一抛结构化错误（AppError / HttpError），包含:
# - error code
# - human-readable message
# - 建议的修复动作
```

**4. CLI → MCP 低成本复用**

CLI 命令树和 MCP Server 共享同一套 handlers，新增 CLI 命令可零成本暴露为 MCP tool。具体实现参考 bytedcli（`git@code.byted.org:byteapi/bytedcli.git`）：

```text
src/cli/commands/<domain>/   ← Commander 命令注册（参数定义）
src/cli/handlers/<domain>/   ← 业务入口（参数校验 + 调用 + 输出）
src/api/<domain>/            ← API 客户端层
src/mcp/                     ← MCP Server，复用 CLI handlers
```

MCP tool name 自动从命令路径生成（下划线格式），超过 60 字符自动缩写。

**5. Skill 作为 CLI 的"说明书"**

CLI 工具本身只提供能力，Skill 告诉 Agent 何时、如何使用这些能力：

```text
bytedcli (CLI)          → 提供 57 个域的命令行能力
skills/bytedance-tce/   → SKILL.md 告诉 Agent "遇到 TCE 相关任务时用这些命令"
  └── references/       → 各命令的详细参数和示例
```

Skill 的 description 决定了 Agent 是否会自动调用对应的 CLI 命令——这是 CLI 相比 MCP 在"发现性"上的补偿机制。

---

## Good Case vs Bad Case

### Good Case

- 用 `gh` CLI 操作 GitHub（零配置，token 高效）
- 用 `lark-cli` 操作飞书（从 MCP 升级而来，结构化输出）
- 用 `bytedcli` 操作字节云（57 域统一 CLI，Skill 做"说明书"）
- 用 MCP Server 连接需要 OAuth 的 Jira（Server 管理认证态）
- bytedcli 的 CLI + MCP 双模架构（共享 handlers，零成本互转）

### Bad Case


| 错误                          | 后果                                     |
| --------------------------- | -------------------------------------- |
| GitHub 操作用 MCP 而不是 `gh` CLI | 93 个 tool 定义占 ~55K tokens，还没提问就用完半个上下文 |
| 新场景直接做 MCP 而不先考虑 CLI        | 增加不必要的基础设施复杂度和 token 开销                |
| CLI 输出纯文本不支持 `--json`       | Agent 难以解析，需要靠正则猜                      |
| CLI 使用位置参数而非 option         | Agent 容易搞错参数顺序                         |
| CLI 工具没有配套 Skill            | Agent 不知道工具的存在，无法自动调用                  |
| MCP Server 暴露 50+ tool      | schema 占满上下文，Agent 无法有效选择              |
| 把敏感凭据硬编码在 MCP 配置中           | 安全风险                                   |
| 能用 Skill 解决的问题做成 MCP        | 杀鸡用牛刀                                  |


---

[上一节：Agents 文档体系](021a-agents-docs.md) | [下一节：SubAgent →](021c-subagent.md) | [返回上级：上下文控制](021-context-control.md)