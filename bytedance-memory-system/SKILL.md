---
name: bytedance-memory-system
description: |
  ByteDance Workspace + Memory + OpenViking 三层记忆架构管理工具。

  帮助用户建立「可读、可找、可回滚、可复盘」的Workspace目录规范，
  整合手动核心记忆（`MEMORY.md`）与OpenViking自动记忆（8类自动分类），
  实现人主导的重要决策记忆 + AI主导的碎片记忆自动沉淀。

  **当以下情况时使用此 Skill**：
  (1) 需要初始化或优化OpenClaw Workspace目录结构
  (2) 需要配置OpenViking自动记忆系统
  (3) 需要建立三层记忆架构（Workspace规范 + 手动记忆 + 自动记忆）
  (4) 需要管理记忆的定期维护、归档、清理
  (5) 用户提到 workspace整理 、 记忆管理 、 openviking 、 目录规范 
example_query: |
  帮我检查三层记忆架构的健康状态
  初始化 Workspace 标准目录结构
  查看 OpenViking 自动记忆列表
  执行每周记忆维护任务
homepage: https://github.com/volcengine/OpenViking
author: suying.1111 <suying@bytedance.com>
version: 1.2.2
metadata:
  openclaw:
    emoji: 🧠
    requires:
      bins: [ git ,  python3 ]
      services: [ openviking-server ]
---


# ByteDance Memory System (三层记忆架构)

> 版本：v1.2.1 | 作者：suying.1111 | 适配：OpenClaw + OpenViking

---

## 一、三层记忆架构概述

ByteDance Memory System 采用**三层架构**设计，从短期到长期逐层递进，确保 Agent 在不同时间跨度下都能高效管理上下文信息：

| 层级 | 名称 | 作用 | 存储周期 | 技术实现 |
| --- | --- | --- | --- | --- |
| **L1 - 会话记忆** | Session Memory | 当前对话的上下文窗口，维持单轮/多轮对话连贯性 | 单次会话 | OpenClaw 内置（LLM Context Window） |
| **L2 - 短期记忆** | Short-term Memory | 跨会话的近期交互摘要，保留用户偏好和任务状态 | 数天~数周 | OpenClaw 内置（本地存储） |
| **L3 - 长期记忆** | Long-term Memory | 持久化的知识库，跨应用、跨平台、跨智能体共享记忆 | 永久 | **OpenViking**（外挂记忆体） |

**OpenViking 的核心价值**​：作为 L3 长期记忆层，OpenViking 解决了 AI Agent 在长周期任务中管理海量、动态上下文的难题。通过集成 OpenViking，OpenClaw 的任务完成率提升 **43%**​，输入 token 成本降低 **91%**[[1]](https://bytedance.larkoffice.com/docx/DmLFdvBnDo0T1Mx5e7Rc5o66nsh)。

## 二、前置条件

| 组件 | 版本要求 | 检查命令 | 一键安装 |
| --- | --- | --- | --- |
| Python | >= 3.10 | `python3 --version` | `brew install python@3.10` 或 `apt install python3.10` |
| Node.js | >= 22 | `node --version` | `brew install node` 或 `apt install nodejs` |
| Git | 任意版本 | `git --version` | `brew install git` 或 `apt install git` |
| cmake | >= 3.15 | `cmake --version` | `brew install cmake` 或 `apt install cmake` |
| OpenViking | >= 0.2.4 | `openviking-server --version` | **一键安装脚本自动安装** |
| 火山方舟 API Key | — | 用于 Embedding 和 VLM 模型调用 | |
| OpenClaw | >= 2.10 | 推荐 2026.3.2（3.12+ 版本存在兼容性问题） | |

### 2. 方舟平台权限配置

需要开通以下模型（内部方舟）：
- **向量模型**：`skylark-embedding-vision`或等效模型（记忆向量化召回）
- **VLM模型**：`seed-2-0-mini` 或等效模型（自动提取记忆）

**获取 Endpoint ID 和 API Key**：
1. 访问内部方舟控制台（https://ark-ap-southeast.byteintl.net/api/v3/）
2. 创建 Embedding Endpoint（选择 `skylark-embedding-vision`）
3. 创建 VLM Endpoint（选择 `seed-2-0-mini` 或等效模型）
4. 在「API Key 管理」中生成新的 API Key

**💡 开发者优惠 - Coding Plan**：
- 火山方舟提供 [Coding Plan](https://www.volcengine.com/activity/codingplan) 方案
- 支持主流 Code 模型：Doubao-Seed-Code、GLM、Kimi、DeepSeek 等
- 适合个人项目、学习实践、工具搭建等编码任务
- **注意**：OpenViking 的记忆功能仍需配置方舟 Embedding 模型（`skylark-embedding-vision`）

**⚠️ 方舟 API 环境区域说明**：

| Region | API Base | 适用场景 |
|--------|----------|----------|
| **SG（新加坡）** | `https://ark-ap-southeast.byteintl.net/api/v3` | 内部方舟 SG |
| **VA（美东）** | `https://ark-i18n-tt.byteintl.net/api/v3` | 内部方舟 VA |
| **CN（中国）** | `https://ark-cn-beijing.bytedance.net/api/v3` | 内部方舟 CN |
| **CN（中国）** | `https://ark-cn-beijing.volces.com/api/v3` | 外部方舟 CN |

**⚠️ 踩坑提示**：
- **确保使用正确的 API Base**：配置 `ov.conf` 时，`api_base` 必须与控制台区域匹配
- **常见错误**：在新加坡控制台创建的 Endpoint，却使用了国内 API Base，会导致 404 或 403 错误
- **检查 Endpoint 状态**：确保 Endpoint 状态为「运行中」才能正常使用
- **API Key 有效期**：Key 有过期时间，过期后需要重新生成
- **401 错误排查**：检查 API Key 是否正确复制（不要有多余空格）、API Base 是否正确
- **错误码查询**：调用失败时，可参考 [方舟错误码文档](https://cloud.byteintl.net/docs/ark/docs/664afad9e16ff302cb5c0706/23eaa2fbda399c32b933128f?x-bc-region-id=bytedance) 进行排查

### 3. 配置文件位置

**OpenViking 配置**：`~/.openviking/ov.conf`

**OpenClaw 配置**：`~/.openclaw/config.json`

**⚠️ 踩坑提示**：
- 配置文件使用 JSON 格式，注意逗号和引号的语法
- 如果配置修改后不生效，检查是否有多余的逗号或缺少引号
- 建议修改前备份原配置文件

---

## 🎯 核心能力

### 三层记忆架构

```text
┌─────────────────────────────────────────────────────────┐
│  第一层：Workspace 规范（目录结构 + 版本控制 + 清理策略）        │
├─────────────────────────────────────────────────────────┤
│  第二层：手动核心记忆（人主导：重要决策、项目状态、经验总结）     │
├─────────────────────────────────────────────────────────┤
│  第三层：OpenViking 自动记忆（AI主导：对话碎片、工具经验）       │
└─────────────────────────────────────────────────────────┘
```

### 记忆分类体系

| 类型 | 归属 | 说明 | 示例 |
|------|------|------|------|
| **profile** | User | 用户身份/属性（始终合并到单文件） | 「用户是后端工程师」 |
| **preferences** | User | 用户偏好设定（按主题聚合） | 「用户喜欢深色模式」 |
| **entities** | User | 与用户相关的人物/项目/概念/组织 | 「Sensight是字节内网服务」 |
| **events** | User | 用户的事件（过去、当前、未来） | 「用户计划下周review」 |
| **cases** | Agent | Agent 遇到具体问题+解决方案 | 「Sensight连不上时提示需要内网」 |
| **patterns** | Agent | Agent 总结可复用流程/方法模板 | 「Workspace优化流程」 |
| **tools** | Agent | Agent 工具使用记忆（调用统计、优化策略） | 「公网搜索用coze-web-search」 |
| **skills** | Agent | Agent 技能执行记忆（工作流、策略） | 「安装技能先检查网络」 |

---

## 📋 快速开始

### 场景 1：初始化 Workspace 三层架构

```bash
# 运行初始化脚本
npx ts-node {baseDir}/scripts/init-workspace.ts
```

**功能**：
- 创建标准目录结构（memory/ .learnings/ state/ logs/ tmp/）
- 生成核心配置文件（DIRECTORIES.md, HEARTBEAT.md, EVOLUTION.md）
- 初始化 Git 追踪
- 配置 OpenViking 连接
  
### 场景 2：检查记忆系统健康状态

```bash
# 检查三层记忆健康度
npx ts-node {baseDir}/scripts/health-check.ts
```

**输出指标**：
- Workspace 目录结构完整性
- Git 提交状态
- OpenViking 服务连通性
- 记忆文件数量统计
  
### 场景 3：查看自动记忆

```bash
# 列出用户偏好记忆
ov ls viking://user/$(whoami)/memories/preferences/

# 查看实体记忆树
ov tree viking://user/$(whoami)/memories/entities/

# 读取具体记忆内容
ov read viking://user/$(whoami)/memories/preferences/theme.md

# 查看摘要
ov abstract viking://user/$(whoami)/memories/

# 查看概述（用于导航）
overview viking://user/$(whoami)/memories/
```

### 场景 4：每周记忆维护

```bash
# 执行维护脚本
npx ts-node {baseDir}/scripts/weekly-maintenance.ts
```

**执行内容**：
1. 检查 `MEMORY.md` 长度（警告：>40行）
2. 归档 7 天前的日志文件
3. 清理过期 state 文件
4. 提交 Git 变更
  
---

## 📖 工作原理

### 记忆提取流程 (agent_end hook)

```text
对话结束
↓
VLM 自动分析对话内容
↓
提取 8 类候选记忆
↓
向量预筛（找相似记忆）
↓
LLM 决策：SKIP / CREATE / MERGE
↓
写入 AGFS + 向量化
```

### 记忆召回流程 (before_agent_start hook)

```text
用户发送消息（长度 >= 5）
↓
健康检查（TCP 探测端口是否可达）
↓
并发搜索 viking://user/memories 和 viking://agent/memories（双源）
↓
合并结果，按 URI 去重，只保留 leaf 节点（level=2）
↓
后处理排序（基础语义分数 + 多因子加权）：
- leaf boost (+0.12)
- 时间关键词 boost (+0.10)
- 偏好匹配 boost (+0.08)
- 词汇重叠 boost (+0.20)
↓
选取 top N 条（默认 6 条），逐条读取 L2 正文
↓
注入 <relevant-memories> 上下文块到 System Prompt
↓
Agent 基于完整上下文回复
```

### 三层内容模型

| 层级 | 名称 | 用途 | Token 成本 |
|------|------|------|-----------|
| L0 | Abstract | 快速判断相关性 | 极低 |
| L1 | Overview | 规划阶段决策 | 低 |
| L2 | Details | 深度阅读时加载 | 按需 |

---

## ✅ 最佳实践

### DO
- **小步提交**：一次 commit 只做一类事
- **及时记录**：对话中重要的点立即说「记住」
- **定期整理**：每周花 10 分钟 review 记忆
- **分层存储**：核心结论放 MEMORY.md，细节放日志
- **使用 CLI**：`ov ls/read/abstract/overview` 管理记忆

### DON'T
- ❌ 不要把一切都写进 MEMORY.md（会导致膨胀）
- ❌ 不要把日志当数据源（难以复现）
- ❌ 不要手动修改 OpenViking 自动生成的记忆文件
- ❌ 不要把临时文件乱放根目录

### 健康指标

| 指标 | 健康 | 危险 |
|------|------|------|
| `git status` 输出 | 短、可解释 | 几百个未跟踪文件 |
| `MEMORY.md` 长度 | < 40 行 | > 100 行 |
| `logs/` 文件数 | < 10 | > 30 |
| OpenViking 记忆 | 定期清理 | 无限增长 |

---

## 🔍 故障排查

### 常见问题速查表

| 问题 | 可能原因 | 解决方案 |
|------|----------|----------|
| **OpenViking 连接失败** | 服务未启动 | `openviking-server` 启动服务 |
| **401 AuthenticationError** | API Key 无效 | 检查方舟控制台 key 是否过期 |
| **记忆不召回** | 向量模型问题 | 检查 embedding endpoint 状态 |
| **记忆提取失败** | VLM 模型问题 | 检查 vlm endpoint 状态 |
| **Git 提交失败** | 未配置 git | `git config --global user.name/email` |
| **配置修改后不生效** | 未重启服务 | 修改 `ov.conf` 后需重启 OpenViking |
| **Endpoint ID 错误** | 使用了模型名 | 应使用 Endpoint ID（以 ep- 开头） |

### 错误码查询

调用失败时，可参考 [方舟错误码文档](https://cloud.byteintl.net/docs/ark/docs/664afad9e16ff302cb5c0706/23eaa2fbda399c32b933128f?x-bc-region-id=bytedance) 进行排查。

常见错误码：

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 400 | 请求参数错误 | 检查请求参数是否符合 API 文档要求 |
| 401 | 认证失败 | 检查 API Key 是否正确、是否过期 |
| 403 | 权限不足 | 检查是否有权限访问该 Endpoint |
| 404 | 资源不存在 | 检查 Endpoint ID 或模型名称是否正确 |
| 429 | 请求过于频繁 | 降低请求频率，或申请更高配额 |
| 500 | 服务器内部错误 | 稍后重试，或联系方舟技术支持 |
| 503 | 服务暂时不可用 | 稍后重试 |

### 诊断命令

```bash
# 检查 OpenViking 服务状态
curl http://127.0.0.1:1933/health

# 检查 ov CLI 是否可用
which ov && ov --version

# 检查配置语法
cat ~/.openviking/ov.conf | python3 -m json.tool

# 查看服务日志（如果使用 systemd）
journalctl -u openviking -f
```

---

## ❓ 常见问题 (FAQ)

### Q1: OpenViking 的「3层记忆架构」是什么？

**A:** OpenViking 采用「双向三层」记忆架构：

**横向双源召回（2源）：**
- **User 记忆**：`profile/preferences/entities/events` — 用户身份、偏好、相关实体、事件
- **Agent 记忆**：`cases/patterns/tools/skills` — Agent 经验、可复用方法、工具使用、技能策略
  
**纵向三层内容模型（3层）：**
- **L0 (Abstract)**：一句话摘要，快速判断相关性（Token 成本极低）
- **L1 (Overview)**：概述信息，帮助 Agent 规划决策（Token 成本低）
- **L2 (Details)**：完整原始内容，深度阅读时加载（Token 成本按需）
  
> **一句话总结**：既召回用户相关记忆，也召回 Agent 经验记忆；每条记忆都包含摘要(L0)、概述(L1)、详情(L2)三层，召回时智能排序并注入 Prompt，实现信息密度与调用成本的平衡。

### Q2: OpenViking 是否必须配置方舟 Embedding 和 VLM？Codex/Claude 用户能否使用？

**A:**

| 组件 | 是否必须 | 说明 | Codex/Claude 用户 |
| --- | --- | --- | --- |
| **Embedding 模型** | ✅ **必须** | 记忆向量化检索。目前主要支持方舟 `skylark-embedding-vision`、`doubao-embedding-vision`等 | 必须通过方舟配置，暂无替代方案 |
| **VLM 模型** | ⚠️ **强烈建议** | 自动提取记忆、生成 L0/L1/L2 三层内容 | 可选，没有时仍可存储/检索记忆，只是无法自动从对话提取 |

**解耦说明**：
- Codex/Claude 作为 **Agent 层**，调用 OpenViking **记忆层** 时，只关心 HTTP 接口，不关心 OpenViking 内部用什么模型
- **Embedding 目前绑定方舟**：这是 OpenViking 的硬性依赖，所有用户（包括 Codex/Claude 用户）都必须配置方舟的 embedding 模型
- **VLM 可降级使用**：如果不想用方舟 VLM，可以关闭自动提取功能，只用 OpenViking 的存储+检索能力，自己用 Claude API 做记忆提取
  
### Q3: 记忆召回的完整流程是怎样的？

**A:** 记忆召回流程如下：

```text
用户发送消息（长度 >= 5）
    ↓
1. 健康检查（TCP 探测端口是否可达）
    ↓
2. 并发搜索 viking://user/memories 和 viking://agent/memories（双源）
    ↓
3. 合并结果，按 URI 去重，只保留 leaf 节点（level=2）
    ↓
4. 后处理排序（基础语义分数 + 多因子加权）：
    - leaf boost (+0.12)
    - 时间关键词 boost (+0.10)
    - 偏好匹配 boost (+0.08)
    - 词汇重叠 boost (+0.20)
    ↓
5. 选取 top N 条（默认 6 条），逐条读取 L2 正文
    ↓
6. 注入 <relevant-memories> 上下文块到 System Prompt
    ↓
Agent 基于完整上下文回复
```

---

# OPENVIKING配置详解

## 三、Linux（ECS）部署指南

> 适用于火山引擎 ECS 实例，系统通常为 Ubuntu/Debian。

### 3.1 创建虚拟环境

ECS 实例根目录部署有限制，不能直接用 `pip` 装全局包，需先创建虚拟环境：

```bash
# 1. 安装依赖
apt update && apt install python3-pip -y
apt install python3.12-venv

# 2. 创建虚拟环境
python3 -m venv myenv

# 3. 激活虚拟环境（每次操作前需先激活）
source myenv/bin/activate
```

### 3.2 安装 OpenViking

推荐使用 npm 一键安装助手：

```bash
# npm 安装（推荐，全平台通用）
npm install -g openclaw-openviking-setup-helper
ov-install
```

安装过程中按照提示依次配置，内置了 VLM 和 Embedding 模型，无需修改直接按回车，填入 API Key 即可。

安装成功后会显示：

```text
═══════════════════════════════════════════════════════════
  Installation complete!
═══════════════════════════════════════════════════════════
[INFO] You can edit the config freely: /root/.openviking/ov.conf
```

### 3.3 配置 OpenViking

#### OpenViking 配置 (~/.openviking/ov.conf)

```json
{
  "server": {
    "host": "127.0.0.1",
    "port": 1933,
    "cors_origins": ["*"]
  },
  "storage": {
    "workspace": "~/.openviking/data",
    "vectordb": {
      "name": "context",
      "backend": "local"
    }
  },
  "embedding": {
    "dense": {
      "provider": "volcengine",
      "api_key": "YOUR_API_KEY",
      "model": "YOUR_ENDPOINT_ID",
      "api_base": "https://ark-ap-southeast.byteintl.net/api/v3",
      "dimension": 1024,
      "input": "multimodal"
    }
  },
  "vlm": {
    "provider": "volcengine",
    "api_key": "YOUR_API_KEY",
    "model": "YOUR_ENDPOINT_ID",
    "api_base": "https://ark-ap-southeast.byteintl.net/api/v3",
    "temperature": 0.1,
    "max_retries": 3
  }
}
```

**⚠️ 配置注意事项**：
- `YOUR_API_KEY`：替换为从方舟控制台获取的真实 API Key
- `YOUR_ENDPOINT_ID`：替换为实际的 Endpoint ID（以 `ep-` 开头）或 Model ID
  - **Endpoint ID**（推荐）：从方舟控制台「Endpoint 管理」获取，例如 `ep-xxxxxxxxxxxxxxxxx`
  - **Model ID**：例如 `doubao-embedding-vision-251215`、`skylark-embedding-vision`
- `api_base`：根据方舟区域选择正确的 API Base URL：
  - 字节内部 SG：`https://ark-ap-southeast.byteintl.net/api/v3`
  - 字节内部 VA：`https://ark-i18n-tt.byteintl.net/api/v3`
  - 外部方舟 CN：`https://ark-cn-beijing.volces.com/api/v3`
- 修改配置后需要重启 OpenViking 服务：`pkill openviking-server && openviking-server`
  
**配置示例 - 外部方舟 CN（个人版本）**：

```json
{
  "server": {
    "host": "127.0.0.1",
    "port": 1933
  },
  "embedding": {
    "dense": {
      "provider": "volcengine",
      "api_key": "YOUR_API_KEY",
      "model": "ep-xxxxxxxxxxxxxxxxx",
      "api_base": "https://ark.cn-beijing.volces.com/api/v3",
      "dimension": 1024
    }
  },
  "vlm": {
    "provider": "volcengine",
    "api_key": "YOUR_API_KEY",
    "model": "doubao-seed-2.0-code",
    "api_base": "https://ark.cn-beijing.volces.com/api/v3"
  }
}
```

> 💡 提示：`model` 参数支持传入 **Endpoint ID**（如 `ep-20260309140511-xdlqm`）或 **Model ID**（如 `doubao-embedding-vision-251215`）。推荐使用 Endpoint ID，更稳定可控。

#### OpenClaw 插件配置 (~/.openclaw/config.json)

```json
{
  "plugins": {
    "entries": {
      "memory": {
        "enabled": true,
        "package": "memory-openviking",
        "config": {
          "serverUrl": "http://127.0.0.1:1933",
          "apiKey": "YOUR_API_KEY",
          "agentId": "openclaw-main"
        }
      }
    }
  }
}
```

编辑配置文件：

```bash
vim ~/.openviking/ov.conf
```

参考配置格式（按 `i` 进入编辑，`Esc` 退出编辑，`:wq` 保存退出）：

```json
{
  "server": {
    "host": "127.0.0.1",
    "port": 1933
  },
  "storage": {
    "workspace": "/home/yourname/.openviking/data",
    "vectordb": { "backend": "local" },
    "agfs": { "backend": "local", "port": 1833 }
  },
  "embedding": {
    "dense": {
      "backend": "volcengine",
      "api_key": "你的-ark-api-key",
      "model": "doubao-embedding-vision-251215",
      "api_base": "https://ark.cn-beijing.volces.com/api/v3",
      "dimension": 1024,
      "input": "multimodal"
    }
  },
  "vlm": {
    "backend": "volcengine",
    "api_key": "你的-ark-api-key",
    "model": "doubao-seed-1-8-251228",
    "api_base": "https://ark.cn-beijing.volces.com/api/v3",
    "temperature": 0.1,
    "max_retries": 3
  }
}
```

**关键字段说明：**

| 字段 | 说明 |
| --- | --- |
| `server.port` | OpenViking 记忆服务监听端口 |
| `storage.workspace` | 存储记忆数据和索引的本地路径（必须使用**绝对路径**，不支持`~`） |
| `embedding.dense.api_key` | 火山方舟 Embedding API Key |
| `vlm.api_key` | 火山方舟 VLM API Key（通常与 embedding 的 key 相同） |
| `vlm.model` | 用于从对话中提取记忆点的 VLM 模型 |

加载环境变量：

```bash
# Linux
source /root/.openclaw/openviking.env

# macOS
source /Users/$USER/.openclaw/openviking.env
```

### 3.4 安装 tmux

```bash
apt update && apt install tmux -y
```

### 3.5 启动服务（tmux 分屏）

```bash
# 1. 进入 tmux
tmux

# 2. 分屏：Ctrl+B，然后按 %
```

**左窗口 — 启动 OpenViking Server：**

```bash
source myenv/bin/activate
source /root/.openclaw/openviking.env
python -m openviking.server.bootstrap
```

看到 `OpenViking HTTP Server is running on 127.0.0.1:1933` 即为成功。

**右窗口 — 启动 Web Console：**（按 `Ctrl+B` 然后按 `→` 切换到右窗口）

```bash
source myenv/bin/activate
source /root/.openclaw/openviking.env
python -m openviking.console.bootstrap --host 0.0.0.0 --port 8020 --openviking-url http://127.0.0.1:1933
```

看到 `Uvicorn running on http://0.0.0.0:8020` 即为成功。

> **安全提示：** 如需临时写入权限，可在命令后加 `--write-enabled`，并限制访问来源。长期使用建议配置 `root_api_key` 进行权限控制。

**退出 tmux：** 按 `Ctrl+B`，再按 `D`，服务继续在后台运行。

### 3.6 访问 Web Console

确保 ECS 安全组已在入向规则中开放 **8020 端口**，然后在浏览器中访问：

```text
http://你的服务器公网IP:8020
```

### 3.7 tmux 常用操作速查

| 操作 | 快捷键/命令 |
| --- | --- |
| 进入 tmux | `tmux` |
| 竖分屏 | `Ctrl+B`然后按`%` |
| 切换左窗口 | `Ctrl+B`然后按`←` |
| 切换右窗口 | `Ctrl+B`然后按`→` |
| 创建新终端 | `Ctrl+B`然后按`C` |
| 退出 tmux（服务保留） | `Ctrl+B`然后按`D` |
| 回到 tmux 会话 | `tmux a`或`tmux attach` |
| 关闭 tmux 会话 | `tmux kill-session` |

## 四、macOS 部署指南

> 适用于 macOS（Apple Silicon / Intel），使用 Homebrew 和 pyenv 管理环境。macOS 无 `apt` 命令，所有包管理通过 `brew` 完成。

### 4.1 基础准备（Python 环境修复）

macOS 系统自带 Python 3.9，需安装 Python 3.11.12 且不影响系统版本。

#### 步骤一：退出旧虚拟环境 + 安装 pyenv

```bash
# 1. 退出当前所有虚拟环境（避免干扰）
deactivate 2>/dev/null || true

# 2. 安装 pyenv（字节内网优先）
brew install pyenv || curl https://mirror.bytedance.com/pyenv/install.sh | bash

# 3. 配置 pyenv 环境变量（适配 zsh）
echo 'export PYENV_ROOT="$HOME/.pyenv"' >> ~/.zshrc
echo 'command -v pyenv >/dev/null || export PATH="$PYENV_ROOT/bin:$PATH"' >> ~/.zshrc
echo 'eval "$(pyenv init -)"' >> ~/.zshrc
source ~/.zshrc
```

#### 步骤二：安装 Python 3.11.12（保留系统 3.9）

```bash
# 1. 字节内网镜像加速安装
export PYTHON_BUILD_MIRROR_URL="https://mirror.bytedance.com/pyenv/"
pyenv install 3.11.12

# 2. 验证已安装版本（系统 3.9 + 新增 3.11.12）
pyenv versions

# 3. 设置全局默认版本
pyenv global 3.11.12

# 4. 验证
python --version      # 输出 Python 3.11.12
python3.9 --version   # 输出 Python 3.9.x（系统版本保留）
```

#### 步骤三：创建 Python 3.11.12 专属虚拟环境

```bash
# 1. 创建虚拟环境
pyenv local 3.11.12
python -m venv ~/myenv311

# 2. 激活虚拟环境
source ~/myenv311/bin/activate

# 3. 验证版本（必须输出 3.11.12）
python --version

# 4. 升级 pip（字节内网源）
python -m pip install --upgrade pip --trusted-host mirror.bytedance.com
```

#### 步骤四：多版本切换（按需）

```bash
# 临时切换到系统 3.9
pyenv local system
python --version   # 输出 3.9.x

# 切回 3.11.12
pyenv local 3.11.12
python --version   # 输出 3.11.12

# 激活 3.11 虚拟环境（开发用）
source ~/myenv311/bin/activate
```

#### 兜底方案：brew 安装 Python（可选）

```bash
# 安装 brew 版 3.11
brew install python@3.11

# 用 brew 版创建虚拟环境
/opt/homebrew/bin/python3.11 -m venv ~/myenv311_brew
source ~/myenv311_brew/bin/activate
python --version   # 验证 3.11.x
```

### 4.2 安装 OpenViking

确保 Python 环境已配置完成后，执行安装：

```bash
# 激活虚拟环境
source ~/myenv311/bin/activate

# npm 一键安装
npm install -g openclaw-openviking-setup-helper
ov-install
```

按提示依次配置，内置了 VLM 和 Embedding 模型，直接按回车并填入 API Key 即可。

### 4.3 配置 OpenViking

```bash
vim ~/.openviking/ov.conf
```

配置内容与 Linux 一致（参见 3.3 配置 OpenViking），注意 `storage.workspace` 使用 macOS 绝对路径，例如：

```text
/Users/yourname/.openviking/data
```

加载环境变量到 OpenClaw（​**macOS 路径**​）：

```bash
source /Users/$USER/.openclaw/openviking.env
```

### 4.4 安装 tmux（macOS 专用）

macOS 没有 `apt` 命令，使用 Homebrew 安装 tmux：

```bash
# 1. 更新 brew 源（字节内网加速）
brew update

# 2. 安装 tmux
brew install tmux

# 3. 验证安装
tmux -V   # 输出 tmux 版本号即成功（如 tmux 3.4）
```

### 4.5 启动服务（tmux 分屏）

#### 步骤一：基础准备

```bash
# 退出旧虚拟环境
deactivate 2>/dev/null || true

# 确保虚拟环境已创建（若未创建）
# /opt/homebrew/bin/python3.11 -m venv ~/myenv311
```

#### 步骤二：进入 tmux 并分屏

```bash
# 启动 tmux 会话
tmux

# 分屏（竖分左右）：先按 Ctrl+B 松开，再按 %（英文输入法）
```

#### 步骤三：左窗口 — 启动 OpenViking Server

```bash
# 切换到左窗口：Ctrl+B 然后按 ←
source ~/myenv311/bin/activate
source /Users/$USER/.openclaw/openviking.env #注意这里是自己路径，我的路径是 source /Users/bytedance/.openclaw/openviking.env
python -m openviking.server.bootstrap
```

看到 `OpenViking HTTP Server is running on 127.0.0.1:1933` 即为成功。

#### 步骤四：右窗口 — 启动 Web Console

```bash
# 切换到右窗口：Ctrl+B 然后按 →
source ~/myenv311/bin/activate
source /Users/$USER/.openclaw/openviking.env
python -m openviking.console.bootstrap --host 0.0.0.0 --port 8020 --openviking-url http://127.0.0.1:1933
```

看到 `Uvicorn running on http://0.0.0.0:8020` 即为成功。

> **可选：** 如需写入权限，启动命令改为：`python -m openviking.console.bootstrap --host 0.0.0.0 --port 8020 --openviking-url http://127.0.0.1:1933 --write-enabled`

#### 步骤五：退出 tmux

按 `Ctrl+B`，再按 `D`，弹出 `[detached (from session x)]`，服务继续后台运行。

### 4.6 访问 Web Console

#### 方式一：本地直接访问（macOS 本机运行服务时）

```text
http://127.0.0.1:8020
```

#### 方式二：SSH 隧道访问（服务运行在远程 ECS，从 Mac 访问）

当远程服务器公网端口未开放 8020 时，在本地 Mac 新终端执行：

```bash
# 建立 SSH 端口转发隧道
ssh -L 8020:127.0.0.1:8020 用户名@你的服务器IP

# （可选）后台运行隧道（关闭终端不中断）
ssh -fN -L 8020:127.0.0.1:8020 用户名@你的服务器IP
```

然后在浏览器访问 `http://127.0.0.1:8020` 即可。

### 4.7 macOS 常用操作速查

| 操作目标 | 指令/快捷键 |
| --- | --- |
| 启动 tmux | `tmux` |
| 竖分屏 | `Ctrl+B`然后按`%` |
| 切换左窗口 | `Ctrl+B`然后按`←` |
| 切换右窗口 | `Ctrl+B`然后按`→` |
| 退出 tmux（服务保留） | `Ctrl+B`然后按`D` |
| 回到 tmux 会话 | `tmux a` |
| 关闭 tmux 会话（停服务） | `tmux kill-session` |
| 关闭 SSH 隧道 | 执行隧道的终端按`Ctrl+C`，或后台隧道用`ps aux | grep ssh`查进程后`kill` |

## 五、快速启动命令（日常使用）

### Linux（ECS）

```bash
source myenv/bin/activate && source /root/.openclaw/openviking.env && openclaw gateway restart
```

### macOS

```bash
source ~/myenv311/bin/activate && source /Users/$USER/.openclaw/openviking.env && openclaw gateway restart
```

看到以下输出即为成功：

```text
[gateway] listening on ws://127.0.0.1:18789
[gateway] memory-openviking: local server started (http://127.0.0.1:1933, config: ...)
```

检查插件状态：

```bash
openclaw status
# Memory 行应显示：enabled (plugin memory-openviking)
```

## 六、验证记忆功能

### 6.1 检查安装

打开 OpenClaw 对接的 IM 助手，或使用 `openclaw tui` 启动文本界面，发送：

> 当前使用的记忆方案是什么？

正常应回答 `OpenViking`。

### 6.2 测试记忆写入

发送一条需要记忆的信息：

> 请记住：我最喜欢的编程语言是 Python。

### 6.3 测试记忆读取

新开一个对话，提问：

> 我最喜欢的编程语言是什么？

正常应回答 `Python`。

### 6.4 通过 Web Console 验证

访问 `http://你的服务器IP:8020`（或通过 SSH 隧道访问 `http://127.0.0.1:8020`），在 Web Console 界面可直接查看记忆写入和读取记录。

### 6.5 通过日志验证（Linux）

```bash
# 找到日志路径
openclaw logs

# 检查记忆注入日志（将日期替换为实际日期）
grep -i inject /tmp/openclaw/openclaw-2026-03-13.log
```

## 七、常见问题

### Q1：如何关闭/重新启用 OpenViking 记忆集成？

```bash
# 关闭
openclaw config set plugins.slots.memory none

# 重新启用
openclaw config set plugins.slots.memory memory-openviking

# 修改后重启 gateway 生效
openclaw gateway restart
```

### Q2：安装失败（Download failed）

重试安装命令即可。网络不稳定时可能需要多次重试：

```bash
ov-install
```

或使用 curl 方式：

```bash
curl -fsSL https://raw.githubusercontent.com/volcengine/OpenViking/main/examples/openclaw-plugin/install.sh | bash
```

### Q3：如何验证 OpenViking 进程是否在运行？

```bash
ps | grep openviking
```

### Q4：macOS 上 apt 命令不可用？

macOS 使用 Homebrew 替代 apt，所有包管理命令替换为：

| Linux (apt) | macOS (brew) |
| --- | --- |
| `apt update` | `brew update` |
| `apt install tmux -y` | `brew install tmux` |
| `apt install python3-pip -y` | `brew install python@3.11` |

### Q5：OpenClaw 3.12+ 版本兼容性问题？

近期更新的 OpenClaw（3.12+）与当前 OpenViking 插件存在兼容性问题，遇到相关报错可先回退至旧版本，推荐 2026.3.2。

## 八、更新记录

### v1.2.1 (2026-03-16)
- ✅ **重构附录 F/G**​：整合 Mac & Linux 双平台安装指南，清晰区分 `brew` vs `apt`、`pyenv` vs `venv`
- ✅ **隐藏敏感信息**​：将公网 IP 替换为 `YOUR_SERVER_IP` 占位符
- ✅ **补充 tmux 安装**​：Mac 使用 `brew install tmux`，Linux 使用 `apt install tmux`
  
### v1.2.0 (2026-03-15)
- ✅ **新增一键安装能力**​：支持 curl 和 npm 两种安装方式
- ✅ **完善安装文档**​：添加详细的安装步骤、环境要求、验证方法
- ✅ **修复方舟 API 区域说明表格**​：优化表格显示格式
- ✅ **添加错误码查询链接**​：包含方舟错误码文档链接和常见错误码对照表
- ✅ **优化前置条件说明**​：添加一键安装命令和环境安装指南
- ✅ **确保安装/配置/使用一致性**​：统一文档风格和术语使用
  
### v1.1.3 (2026-03-13)
- ✅ 新增「🧪 测试用例与效果验证」完整章节
- ✅ 修正脚本中的硬编码路径为 `process.cwd()`
- ✅ 沉淀完整辅助文档
- ✅ 创建可安全发布到字节内网的版本
- ✅ 版本号更新至 v1.1.3
  
### v1.1.2 (2026-03-09)
- ✅ 新增「常见问题 (FAQ)」章节
- ✅ 补充「2层记忆召回」完整解释（横向双源 + 纵向双层）
- ✅ 补充「方舟 Embedding/VLM 是否必须」详细说明
- ✅ 补充「Codex/Claude 用户能否使用」解耦说明
- ✅ 版本号更新至 v1.1.2
  
### v1.1.1 (2026-03-09)
- ✅ 新增「方舟 API 环境区域说明」表格
- ✅ 补充 API Base URL 对照表
- ✅ 版本号更新至 v1.1.1
  
### v1.1.0 (2026-03-09)
- ✅ 新增「踩坑指南」章节
- ✅ 扩充「方舟平台权限配置」详细步骤
- ✅ 新增「配置文件位置」说明
- ✅ 扩充「故障排查」章节
- ✅ 版本号更新至 v1.1.0
  
### v1.0.1 (2026-03-09)
- ✅ 修正 SKILL 名称
- ✅ 更新 homepage URL
- ✅ 添加 `example_query` 字段
- ✅ 版本号更新至 v1.0.1
  
## 九、参考资料

- [OpenViking GitHub](https://github.com/volcengine/OpenViking)
- [OpenClaw 集成 OpenViking，升级长期记忆](https://bytedance.larkoffice.com/docx/DmLFdvBnDo0T1Mx5e7Rc5o66nsh)
- [OpenViking 配置文档](https://github.com/volcengine/OpenViking/blob/main/docs/zh/guides/01-configuration.md)
- [OpenViking 安装指南 INSTALL-ZH](https://github.com/volcengine/OpenViking/blob/main/examples/openclaw-memory-plugin/INSTALL-ZH.md)
- [快速部署 OpenClaw 火山引擎](https://www.volcengine.com/docs/6396/2189942?lang=zh)
- [OpenClaw：如何保持干净的 workspace 目录](https://bytedance.larkoffice.com/docx/DWqsdTZgAo0XGIxSjWhcUYeJnsb)


_本文档最后更新于：2026-03-16_
_版本：v1.2.1_
