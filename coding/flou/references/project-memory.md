# 项目记忆搭建

收集项目相关信息并保存到项目记忆文件，为后续开发流程提供上下文支持。

## 目标

收集并保存项目的关键信息，包括仓库地址、PSM、集群信息、中间件配置等。

## 记忆文件位置

项目记忆分为两类，存储在 `${PROJECT_DIR}/.flou/memory/` 目录下：

| 文件 | 用途 | 管理方式 |
|------|------|----------|
| `runtime.yaml` | 部署记忆：PSM、集群、中间件等运行时配置 | 自动初始化 |
| `archive.json` | 研发记忆：开发经验、技术决策等 | CLI 工具管理, 不可直接读，很长 |

### runtime.yaml - 部署记忆

记录项目部署相关的静态配置，包括仓库地址、PSM、集群信息、中间件配置等。供部署、运维场景使用。

### 研发记忆

记录开发过程中的经验积累、技术决策、知识沉淀等。

## 前置依赖

本流程依赖以下技能和工具，详见 [SKILL.md](../SKILL.md#依赖技能)：

- **bytedcli 技能包**：搜索 TCE 服务、检索代码仓库
- **DeepWiki MCP**：关联仓库的 DeepWiki 文档

## 记忆检索流程

### 1. 加载部署记忆 (runtime.yaml)
- 检查 `${FLOU_DIR}/memory/runtime.yaml` 是否存在
- 如果存在，加载并解析项目信息、服务配置、中间件配置等

### 2. 加载研发记忆
- 通过flou-cli工具检索记忆

### 3. 从 DeepWiki MCP 检索项目历史记忆
- 使用已配置的 DeepWiki MCP 工具
- 搜索与项目相关的历史文档和知识
- 获取项目历史变更记录、技术决策文档等
- 补充本地记忆文件中缺失的信息

### 4. 记忆合并与验证
- 合并本地记忆和 DeepWiki 检索结果
- 验证信息的完整性和一致性
- 标注信息来源（本地文件/DeepWiki）
- 对于冲突信息，优先使用 DeepWiki 的最新信息


## 记忆生成执行流程

### 1. 查找配置文件确定可能的psm列表

1. 在项目中查找配置文件，包括但不限于：
- `build.sh`
- `app.yaml` / `app.conf`
- `config.yaml` / `config.json`
- `application.yaml` / `application.properties`
- `deployment.yaml`
- Dockerfile
- go.mod / package.json / pom.xml 等

2. 从配置文件中查找 `psm`、`service_name` 等字段

### 3. 收集项目信息

使用合适的工具从配置文件和代码中提取以下信息：

#### 3.1 仓库地址
- 从 git remote 获取
- 从配置文件中查找

#### 3.2 PSM（服务唯一标识）

**查找方式**：

1. **从配置文件查找**
   - 从配置文件中查找 `psm`、`service_name` 等字段
   - 从代码中搜索 PSM 相关配置

2. **使用 bytedcli 技能搜索 TCE 服务**
   - 调用 bytedance-tce 技能搜索 TCE 服务
   - 根据仓库名搜索对应的 PSM
   - **注意**：可能存在单仓库对应多个 PSM 的情况
   - 将搜索结果提供给用户选择
   - 对于每个 PSM，可能存在多个集群（cluster）

**访问示例**：
```
搜索 TCE 服务，仓库名 <仓库名>
```

#### 3.3 定位代码仓库

在确定 PSM 后，使用 bytedcli 技能检索代码仓库信息。

**执行步骤**：

1. **使用 bytedcli codebase 检索仓库**
   - 调用 bytedcli 技能的 codebase 功能
   - 根据每个 PSM 检索对应的代码仓库
   - 获取仓库地址、分支等信息
   - **注意**：多个 PSM 可能对应同一个仓库

2. **验证仓库信息**
   - 确认仓库地址正确性
   - 检查仓库可访问性

**访问示例**：
```
检索代码仓库，PSM <PSM名称>
```

#### 3.4 中间件信息
- **RDS**：数据库连接配置、实例信息（使用 PSM 作为唯一键）
- **Redis**：缓存配置、集群信息（使用 PSM 作为唯一键）
- **Kafka**：消息队列配置、topic 信息（使用 PSM 作为唯一键）
- 其他中间件：MQ、ES 等

#### 3.5 其他信息
- 服务端口
- 依赖服务
- 环境变量

### 4. 关联 DeepWiki

在确定仓库名后，使用 DeepWiki MCP 搜索并关联对应的 DeepWiki 文档。

1. **配置 DeepWiki MCP**
   - 在 MCP 配置文件中添加 DeepWiki 服务器配置（见前置依赖部分）
   - 将 `MCP_SERVER_CALL_TOOL_PARAMS` 中的 `repo_name` 替换为实际仓库名

### 5. 保存到记忆文件

将收集到的信息保存到 `$FLOU_DIR/memory/runtime.yaml`。

## 记忆文件格式

```yaml
project:
  name: <项目名称>
  repository:
    url: <仓库地址>
    branch: <分支>
  idl: 
    path: <项目协议路径, 可以没有>
    type: <协议类型： local,overpass>
  deepwiki:
    repo_name: <仓库名称>
    url: <DeepWiki文档URL>

services:
  - psm: <服务唯一标识>
    clusters:
      - name: <集群名称>
        dc: 
          - lf
          - hl
        region: <区域>
      - name: <集群名称>
        region: <区域>

middleware:
  rds:
    - psm: <RDS实例PSM>
  redis:
    - psm: <Redis实例PSM>
  kafka:
    - psm: <Kafka集群PSM>
      topics:
        - <topic名称>

```

## 注意事项

- 信息收集应尽可能全面，但避免敏感信息泄露
- 如果某些信息无法获取，标注为 `unknown` 或留空
- 记忆文件应在项目根目录创建
- DeepWiki 关联应在发现仓库地址后立即进行

## 研发记忆管理

在开发过程中，随时将重要的技术决策、问题解决方案、最佳实践等记录到研发记忆中。

### 归档时机

- 完成一个功能模块后
- 解决了一个复杂问题
- 进行了重要的技术选型
- 代码评审中发现了值得记录的经验

### 注意事项

- 归档时注意拆分记忆，每个记忆应包含独立的信息，避免合并多个记忆

### 使用 CLI 管理

```bash
flou-cli --project-dir <项目目录> [--project-name <项目名>] memory <子命令> [参数]
# 归档文件：$FLOU_DIR/memory/archive.json
# 索引与哈希：/tmp/flou/memory/<项目名>/
# 未显式传入 --project-name 时，默认使用 basename(<项目目录>)
```

### 命令示例

```bash
# 初始化数据库
flou-cli memory init --project-dir /path/to/project --project-name myproject
# 或省略 --project-name（默认为 project 目录名）
# flou-cli --project-dir /path/to/project memory init

# 归档记忆管理
./flou-cli memory add "修复了连接池泄漏" --tags bug,性能 --project-dir /path/to/project --project-name myproject
./flou-cli memory recall 连接池 --limit 10 --project-dir /path/to/project --project-name myproject
./flou-cli memory list --limit 50 --project-dir /path/to/project --project-name myproject
./flou-cli memory delete <archive_id> --project-dir /path/to/project --project-name myproject
```
