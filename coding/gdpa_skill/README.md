# GDPA Agent Skills

GDPA 团队的 Claude Code Skills 集合，提供 GDP 服务开发、部署、测试等全栈技能支持。

## 快速开始

**推荐安装 `backend` 插件**，一次性获取后端开发所需的全部技能。

### AI Native 安装（推荐）

直接把以下内容发送给你的 AI 助手，让它自动完成安装：

**Claude Code：**

> 帮我安装 GDPA Skills 插件。执行以下命令：
> 1. `/plugin marketplace add git@code.byted.org:tiktok/gdpa_skills.git`
> 2. `/plugin install backend@gdpa-skills`

**Cursor：**

> 帮我安装 GDPA Skills 插件到 Cursor。执行以下步骤：
> 1. `git clone git@code.byted.org:tiktok/gdpa_skills.git ~/.cursor/plugins/local/_gdpa_repo`
> 2. `ln -s ~/.cursor/plugins/local/_gdpa_repo/plugins/backend ~/.cursor/plugins/local/gdpa-backend`
> 安装完成后提醒我重启 Cursor 或执行 Developer: Reload Window。

**Trae / Gemini CLI：**

> 帮我安装 GDPA Skills。执行：`npx skills add git@code.byted.org:tiktok/gdpa_skills.git --agent <trae|trae-cn|gemini-cli> --skill '*'`

---

### 手动安装

<details>
<summary>Claude Code</summary>

```bash
# 1. 注册 Marketplace
/plugin marketplace add git@code.byted.org:tiktok/gdpa_skills.git

# 2. 安装插件（推荐 backend，包含全部 84 个技能）
/plugin install backend@gdpa-skills

# /plugin install backend-knowledge@gdpa-skills   # 仅知识文档（29 个）
# /plugin install backend-tools@gdpa-skills       # 仅开发工具（56 个）
# /plugin install gdp@gdpa-skills                 # GDP 框架 + RAL（2 个）
```

</details>

<details>
<summary>Cursor</summary>

```bash
# 1. Clone 仓库到 Cursor 本地插件目录
git clone git@code.byted.org:tiktok/gdpa_skills.git ~/.cursor/plugins/local/_gdpa_repo

# 2. 创建插件 symlink（推荐 backend，包含全部技能）
ln -s ~/.cursor/plugins/local/_gdpa_repo/plugins/backend ~/.cursor/plugins/local/gdpa-backend
```

然后重启 Cursor 或执行 `Developer: Reload Window`。

```bash
# 更新：进入仓库目录 git pull 即可
cd ~/.cursor/plugins/local/_gdpa_repo && git pull
```

> 其他插件：将 `backend` 替换为 `backend-knowledge`、`backend-tools` 或 `gdp`。

</details>

<details>
<summary>Trae / Gemini CLI</summary>

```bash
# Trae（国际版）
npx skills add git@code.byted.org:tiktok/gdpa_skills.git --agent trae --skill '*'

# Trae CN（国内版）
npx skills add git@code.byted.org:tiktok/gdpa_skills.git --agent trae-cn --skill '*'

# Gemini CLI
npx skills add git@code.byted.org:tiktok/gdpa_skills.git --agent gemini-cli --skill '*'
```

</details>

> 更多安装选项见 [其他 AI 工具安装](#其他-ai-工具安装) 章节。

## Plugins

| Plugin | Skills | Description |
|--------|--------|-------------|
| **backend** | 84 | 后端开发全栈技能集（推荐），一站式安装 |
| **backend-knowledge** | 29 | 后端知识文档：框架、SDK、规范、平台 |
| **backend-tools** | 56 | 后端开发工具：CLI、查询、部署、工作流 |
| **gdp** | 2 | GDP 框架核心 + RAL 资源访问层 |

### backend-knowledge 子分类

| Category | Skills | Description |
|----------|--------|-------------|
| **framework-knowledge** | 5 | GDP、RAL、Kitex、Hertz、Overpass |
| **sdk-knowledge** | 18 | 存储、配置、监控、日志、测试、认证、AI 等 SDK |
| **guideline-knowledge** | 6 | TikTok 规范、代码审查、Go 实践、MD3C、集成测试、Monorepo |

#### framework-knowledge（5 个）

| Skill | Description |
|-------|-------------|
| gdp-knowledge | GDP 框架开发规范和最佳实践 |
| ral-knowledge | RAL 资源访问层初始化与流量调度 |
| kitex-knowledge | Kitex RPC 框架开发与错误排查 |
| hertz-knowledge | Hertz HTTP 框架开发指南 |
| overpass-knowledge | Overpass RPC 调用代码生成平台 |

#### sdk-knowledge（18 个）

| Skill | Description |
|-------|-------------|
| storage-knowledge | 存储 SDK（GORM/Redis/MongoDB/TOS/MQ 等）|
| tcc-knowledge | TCC 动态配置中心 SDK |
| byteconf-knowledge | ByteConf 配置管理平台 SDK |
| confx-knowledge | Confx 多数据源统一配置管理 |
| metrics-knowledge | Metrics 监控打点（v2/v3/v4）|
| metricx-knowledge | Metricx 监控打点库 |
| streamlog-knowledge | StreamLog 流式日志采集 |
| bytedtrace-knowledge | BytedTrace 链路追踪指标 |
| testing-knowledge | Golang 单元测试与 Mockey Mock |
| bytetim-knowledge | ByteTIM 流量身份标识与票据校验 |
| dolphin-knowledge | Dolphin 动态决策平台 SDK |
| frontier-knowledge | Frontier 长连接网关 SDK |
| imagex-knowledge | veImageX 图片服务 SDK |
| kms-knowledge | KMS 密钥管理与加解密 |
| libra-knowledge | Libra A/B 实验平台 SDK |
| rtc-knowledge | RTC 实时音视频服务端 API |
| fornax-knowledge | Fornax AI Agent Ops 平台 SDK |
| anycache-knowledge | AnyCache 缓存组件（函数缓存、批量缓存、多层缓存）|

#### guideline-knowledge（6 个）

| Skill | Description |
|-------|-------------|
| tiktok-guidelines-knowledge | TikTok API/代码规范 |
| refactor-review-knowledge | 代码审查与重构指南 |
| golang-knowledge | Go 语言通用实践 |
| md3c-knowledge | MD3C 多数据中心兼容与 3V 架构 |
| integration-testing-knowledge | 集成测试与 PPE 环境 |
| monorepo-knowledge | TikTok Bazel Monorepo 开发指南 |

### backend-tools 技能列表

| Skill | Description |
|-------|-------------|
| argos-query | Argos 日志查询 |
| argos-dashboard | Manage Argos custom dashboards — list / get / save / delete dashboards by id, or quickly build a multi-metric dashboard from a list of metric names. Use whenever the user wants to browse their dashboards, create / update / delete an Argos custom dashboard, set up a metrics dashboard for a service, or fetch the JSON of an existing dashboard. Also trigger when the user mentions Argos dashboard, 自定义看板, dashboard URL, list dashboards, 我的看板, or wants to wrap a few metric names into a quick monitoring view. |
| bam-api | BAM API 接口查询 |
| bam-query | BAM 接口测试（RPC/HTTP） |
| base-workflow | 基础工作流 |
| bits-devops | BITS DevOps 工作流管理 |
| bytefaas | ByteFaaS 服务与版本查询 |
| codebase | 代码库分析 |
| decc-desrpc | decc-desrpc |
| devflow | 开发工作流 |
| edit-idl | IDL 编辑 |
| env | Env Platform BOE/PPE 环境查询与部署 |
| eventbus | EventBus 消息查询、搜索与发送 |
| gdpa-cli | GDPA CLI 命令行工具 |
| iam | IAM 权限管理 |
| meego-query | Meego 项目查询 |
| metrics | Metrics 监控指标查询 |
| neptune-acl | Neptune ACL 权限查询 |
| neptune-stability | Neptune 超时/稳定性配置查询 |
| overpass-tool | Overpass IDL/代码生成 |
| post-coding-verify | 编码后验证流程 |
| rds | RDS 数据库 SQL 执行 |
| rds-query | RDS 数据库元数据查询 |
| repo2psm | 代码仓库 PSM 定位 |
| repotalk | 代码仓库分析查询 |
| scm | SCM 版本管理 |
| tcc-deploy | TCC 配置部署到 PPE |
| tcc-query | TCC 配置中心查询 |
| tce | TCE 容器平台 |
| user-jwt | 用户 JWT/用户名获取 |
| workflow-builder | 自定义工作流构建器 |
| aeolus | Aeolus 本地凭证管理 |
| bytecloud-doc | ByteCloud 文档查询 |
| meego-manage | Meego 工作项管理 |
| trace-query | Trace 查询 |
| abase | ABase 相关能力与开发支持 |
| bpm | BPM 流程/平台相关能力 |
| dorado | Dorado 相关能力与工具 |
| learning-capture | 学习沉淀/经验捕获工作流 |
| libra | Libra 平台相关工具能力 |
| redis | Redis 相关工具能力 |
| tos | TOS 对象存储相关工具能力 |

## 目录结构

```
gdpa_skills/
├── skills/                           # 所有 skill 平铺（单一数据源）
├── plugins/                          # Plugin 分组（通过软链接引用 skills，支持 Claude Code 和 Cursor）
│   ├── backend/skills/               # 全栈技能集 = backend-knowledge + backend-tools（83 个）
│   ├── backend-knowledge/skills/     # 知识文档类（29 个）
│   │   ├── framework-knowledge/      # 核心框架（5 个）
│   │   │   ├── gdp-knowledge         # GDP 框架开发规范和最佳实践
│   │   │   ├── ral-knowledge         # RAL 资源访问层初始化与流量调度
│   │   │   ├── kitex-knowledge       # Kitex RPC 框架开发与错误排查
│   │   │   ├── hertz-knowledge       # Hertz HTTP 框架开发指南
│   │   │   └── overpass-knowledge    # Overpass RPC 调用代码生成平台
│   │   ├── sdk-knowledge/            # SDK 集成（18 个）
│   │   │   ├── storage-knowledge     # 存储 SDK（GORM/Redis/MongoDB/TOS/MQ 等）
│   │   │   ├── tcc-knowledge         # TCC 动态配置中心 SDK
│   │   │   ├── byteconf-knowledge    # ByteConf 配置管理平台 SDK
│   │   │   ├── confx-knowledge       # Confx 多数据源统一配置管理
│   │   │   ├── metrics-knowledge     # Metrics 监控打点（v2/v3/v4）
│   │   │   ├── metricx-knowledge     # Metricx 监控打点库
│   │   │   ├── streamlog-knowledge   # StreamLog 流式日志采集
│   │   │   ├── bytedtrace-knowledge  # BytedTrace 链路追踪指标
│   │   │   ├── testing-knowledge     # Golang 单元测试与 Mockey Mock
│   │   │   ├── bytetim-knowledge     # ByteTIM 流量身份标识与票据校验
│   │   │   ├── dolphin-knowledge     # Dolphin 动态决策平台 SDK
│   │   │   ├── frontier-knowledge    # Frontier 长连接网关 SDK
│   │   │   ├── imagex-knowledge      # veImageX 图片服务 SDK
│   │   │   ├── kms-knowledge         # KMS 密钥管理与加解密
│   │   │   ├── libra-knowledge       # Libra A/B 实验平台 SDK
│   │   │   ├── rtc-knowledge         # RTC 实时音视频服务端 API
│   │   │   ├── fornax-knowledge      # Fornax AI Agent Ops 平台 SDK
│   │   │   └── anycache-knowledge    # AnyCache 缓存组件（函数缓存、批量缓存）
│   │   └── guideline-knowledge/      # 规范与实践（6 个）
│   │       ├── tiktok-guidelines-knowledge  # TikTok API/代码规范
│   │       ├── refactor-review-knowledge    # 代码审查与重构指南
│   │       ├── golang-knowledge      # Go 语言通用实践
│   │       ├── md3c-knowledge        # MD3C 多数据中心兼容与 3V 架构
│   │       ├── integration-testing-knowledge  # 集成测试与 PPE 环境
│   │       └── monorepo-knowledge    # TikTok Bazel Monorepo 开发指南
│   └── backend-tools/skills/         # 工具类（55 个）
│       ├── gdpa-cli                  # GDPA CLI 命令行工具安装
│       ├── bam-api                   # BAM API 接口定义查询
│       ├── bam-query                 # BAM 接口测试（RPC/HTTP 请求）
│       ├── overpass-tool             # Overpass IDL 信息与代码生成
│       ├── repotalk                  # 代码仓库结构与依赖分析
│       ├── argos-query               # Argos 服务日志搜索
│       ├── rds-query                 # RDS 数据库元数据查询
│       ├── tcc-query                 # TCC 配置中心配置查询
│       ├── scm                       # SCM 仓库与版本管理
│       ├── edit-idl                  # IDL 文件智能编辑
│       ├── base-workflow             # 完整研发流程驱动
│       ├── codebase                  # 代码库/分支/MR 管理
│       ├── devflow                   # 开发任务管理
│       ├── iam                       # IAM 权限申请
│       ├── tce                       # TCE 容器服务查询
│       ├── metrics                   # Metrics 监控数据查询
│       ├── rds                       # RDS 数据库 SQL 查询
│       ├── tcc-deploy                # TCC 配置部署
│       ├── user_jwt                  # 用户 JWT/用户名获取
│       ├── meego-query               # Meego 工作项查询
│       ├── bits-devops               # BITS DevOps 工作流管理
│       ├── bytefaas                  # ByteFaaS 服务查询
│       ├── decc-desrpc               # DECC DES-RPC 通道/规则查询
│       ├── neptune-acl               # Neptune ACL 服务治理查询
│       ├── neptune-stability         # Neptune 稳定性规则查询
│       ├── neptune-traffic           # Neptune 流量规则查询
│       ├── post-coding-verify        # 编码后验证流程
│       ├── repo2psm                  # 代码仓库 PSM 定位
│       ├── workflow-builder          # 自定义工作流构建器
│       ├── env                       # Env Platform BOE/PPE 环境查询与部署
│       └── eventbus                  # EventBus 消息查询、搜索与发送
├── .claude-plugin/marketplace.json
├── .cursor-plugin/marketplace.json   # Cursor plugin marketplace config
└── CLAUDE.md
```

## 安装方式

### Claude Code

**注册 Marketplace**

```bash
/plugin marketplace add git@code.byted.org:tiktok/gdpa_skills.git
```

**安装插件**

```bash
# 推荐：一键安装全部
/plugin install backend@gdpa-skills

# 按需安装
/plugin install backend-knowledge@gdpa-skills   # 仅知识文档
/plugin install backend-tools@gdpa-skills       # 仅开发工具
/plugin install gdp@gdpa-skills                 # GDP 框架 + RAL
```

**通过 UI 安装：**

1. 选择 `Browse and install plugins`
2. 选择 `gdpa-skills`
3. 选择需要的插件
4. 选择 `Install now`

### Cursor

**方式一：本地安装（推荐）**

```bash
# Clone 到 Cursor 本地插件目录，再 symlink 具体插件
git clone git@code.byted.org:tiktok/gdpa_skills.git ~/.cursor/plugins/local/_gdpa_repo
ln -s ~/.cursor/plugins/local/_gdpa_repo/plugins/backend ~/.cursor/plugins/local/gdpa-backend
```

重启 Cursor 或执行 `Developer: Reload Window` 生效。更新时 `cd ~/.cursor/plugins/local/_gdpa_repo && git pull`。

**方式二：npx 安装**

```bash
# 项目级安装
npx skills add git@code.byted.org:tiktok/gdpa_skills.git -a cursor

# 全局安装
npx skills add git@code.byted.org:tiktok/gdpa_skills.git -g -a cursor
```

### Trae / Gemini CLI

```bash
# Trae（国际版）
npx skills add git@code.byted.org:tiktok/gdpa_skills.git --agent trae --skill '*'

# Trae CN（国内版）
npx skills add git@code.byted.org:tiktok/gdpa_skills.git --agent trae-cn --skill '*'

# Gemini CLI
npx skills add git@code.byted.org:tiktok/gdpa_skills.git --agent gemini-cli --skill '*'
```

> 更多选项见 [其他 AI 工具安装](#其他-ai-工具安装) 章节。

## 使用方式

安装插件后，Skills 会在对话中自动提供相关领域的专业知识和最佳实践指导。

### 自动触发

Claude Code 会根据对话内容自动识别并加载相关技能，无需手动干预。例如：

- 当你讨论 RPC 调用时，会自动加载 `kitex-knowledge`
- 当你使用存储 SDK 时，会自动加载 `storage-knowledge`
- 当你编写单元测试时，会自动加载 `testing-knowledge`

### 示例

> "帮我写一个使用 BytedGORM 访问 MySQL 的示例代码"

Claude Code 会自动加载 `storage-knowledge` 技能并提供符合字节内部规范的代码示例。

## 其他 AI 工具安装

除了 Claude Code 的 `/plugin` 命令，本项目也支持其他主流 AI 编程工具。

### 通用安装命令

```bash
# 安装到特定工具
npx skills add git@code.byted.org:tiktok/gdpa_skills.git -a <agent-name>

# 全局安装（对所有项目生效）
npx skills add git@code.byted.org:tiktok/gdpa_skills.git -g -a <agent-name>

# 安装特定技能
npx skills add git@code.byted.org:tiktok/gdpa_skills.git --skill <skill-name> -a <agent-name>
```

### 支持的 AI 工具

| 工具 | Agent 参数 | 项目路径 | 全局路径 |
|------|-----------|---------|---------|
| **Claude Code** | `claude-code` | `.claude/skills/` | `~/.claude/skills/` |
| **Cursor** | `cursor` | `.cursor/skills/` | `~/.cursor/skills/` |
| **Trae** | `trae` | `.trae/skills/` | `~/.trae/skills/` |
| **Trae CN** | `trae-cn` | `.trae/skills/` | `~/.trae-cn/skills/` |
| **Gemini CLI** | `gemini-cli` | `.gemini/skills/` | `~/.gemini/skills/` |

### 安装示例

**Cursor**

推荐通过 Cursor Plugin Marketplace 安装（见[快速开始](#cursor)）。也可使用 npx：

```bash
# 项目级安装
npx skills add git@code.byted.org:tiktok/gdpa_skills.git -a cursor

# 全局安装
npx skills add git@code.byted.org:tiktok/gdpa_skills.git -g -a cursor
```

**Trae / Trae CN**

```bash
# Trae（国际版）
npx skills add git@code.byted.org:tiktok/gdpa_skills.git --agent trae --skill '*'

# Trae CN（国内版）
npx skills add git@code.byted.org:tiktok/gdpa_skills.git --agent trae-cn --skill '*'
```

**Gemini CLI**

```bash
npx skills add git@code.byted.org:tiktok/gdpa_skills.git -a gemini-cli
```

**多工具同时安装**

```bash
# 同时安装到 Cursor 和 Trae
npx skills add git@code.byted.org:tiktok/gdpa_skills.git -a cursor -a trae
```

**非交互式安装（CI/CD）**

```bash
npx skills add git@code.byted.org:tiktok/gdpa_skills.git -a cursor -y
```

### 其他命令

```bash
npx skills list              # 列出已安装技能
npx skills remove            # 删除技能
npx skills update            # 更新技能
```

- `bytedoc-query`: 读取和搜索 ByteDoc 文档与元信息。
- `bytest-case`: 查询和检查 ByteTest 用例与相关测试资产。
- `tea-query`: 查询 TEA 埋点/事件元信息与相关定义。

## 向 gdpa-cli 贡献 Skill

本仓库是 `gdpa-cli` 的默认 skill 插件仓。要新增一个被 gdpa-cli 加载的 skill：

1. **Clone 本仓库**（无需主仓 `gdpa-agents` 权限）：
   ```bash
   git clone git@code.byted.org:tiktok/gdpa_skills.git
   cd gdpa_skills
   ```

2. **新增 skill 目录**，路径遵循 `plugins/backend-knowledge/skills/<category>/<skill-name>/SKILL.md`：
   ```bash
   mkdir -p plugins/backend-knowledge/skills/<category>/my-skill
   $EDITOR plugins/backend-knowledge/skills/<category>/my-skill/SKILL.md
   ```

3. **本地集成测试**（可选，需要主仓 `gdpa-agents` 已 clone 在本地）：
   ```bash
   # 在主仓中将 submodule 临时指向本地 clone
   cd <path-to-gdpa-agents>
   git -C pkg/skills/gdpa_skills fetch <path-to-this-clone> feat/your-branch
   git -C pkg/skills/gdpa_skills checkout FETCH_HEAD
   ./build.sh
   ./output/gdpa-cli skills | grep my-skill
   ```

4. **提交 MR 到本仓库**。合入后，主仓 `gdpa-agents` 会在下次发版时 bump submodule 引用。

### Plugin Manifest

`.gdpa-plugin/plugin.json` 是 gdpa-cli 识别本仓库的元数据自述。字段说明见主仓 `docs/superpowers/specs/2026-04-14-repo-split-for-collaboration-design.md` §3.3。

| argos-dashboard | Argos Dashboard 仪表盘与监控面板查询 |

| bytees | ByteES 检索与索引能力查询 |

| ems | EMS 事件管理与处置操作 |

| tiktok-event-center | TikTok Event Center 事件定义与配置管理 |
