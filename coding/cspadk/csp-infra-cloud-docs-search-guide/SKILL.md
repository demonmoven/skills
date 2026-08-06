---
name: csp-infra-cloud-docs-search-guide
description: 搜索字节云平台文档的指南，提供计算、存储、数据库、中间件等平台的分类索引和搜索关键词。当用户询问 TCE、RDS、BMQ、Slardar、Tea、SCM、Ark、Grafana、Netlink、TOS 等平台文档时，引导通过 bytedance-cloud-docs 查询官方文档。
version: 1.0.0
metadata:
  patterns:
    - tool-wrapper
  domain: csp-infra
  i18n_level: 0
  prompt_version: "1.0.0"
  agent_support:
    - claude-code
    - cursor
---

# 云平台文档搜索指南

提供字节云平台文档搜索的关键词索引，帮助用户和 AI Agent 快速定位官方文档。

## 触发场景

- 用户询问特定平台的文档（如"TCE 怎么用"、"RDS 文档在哪"）
- 其他 skill 需要查询平台知识时
- AI Agent 在实现过程中遇到未知平台概念

## 使用方法

调用 `/bytedance-cloud-docs` skill，传入关键词进行搜索：

```
/bytedance-cloud-docs search --keyword "<platform-keywords>"
```

## 平台分类索引

### 1. 计算 (Compute)

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 容器 | container, k8s, kubernetes, pod, deployment | 容器部署、资源限制、调度策略 |
| 云引擎 TCE | TCE, tce, 云引擎 | TCE 部署、服务创建、配置管理 |
| Release Manager | release, 发布管理 | 版本发布、回滚策略 |
| 离线调度 Megatron | Megatron, 离线调度 | 任务调度、依赖管理 |
| 云服务器 | ECS, 云服务器 | 实例创建、配置规格 |
| 云主机 EC2 | EC2, 云主机 | 虚拟机管理、网络配置 |
| 开发机 Devbox | Devbox, 开发机 | 开发环境、远程连接 |
| 安卓云 AiC | AiC, 安卓云 | 安卓设备、云手机 |
| 数据计算 | 数据计算 | 数据处理、计算任务 |
| 云原生计算 Ray | Ray, 云原生计算 | 分布式计算、Ray集群 |
| Serverless | serverless, 无服务器 | 函数部署、事件触发 |
| 函数计算 ByteFaaS | ByteFaaS, 函数计算 | 函数开发、触发器配置 |
| 轻量级函数 FaaSWorker | FaaSWorker, 轻量函数 | 轻量任务、快速部署 |
| 定时任务 CronJob | CronJob, 定时任务 | 定时调度、任务管理 |
| 前端部署平台 GoofyDeploy | GoofyDeploy, 前端部署 | 前端发布、预览环境 |
| 资源分发 Gecko | Gecko, 资源分发 | 静态资源、CDN分发 |
| 工作流引擎 ByteFlow | ByteFlow, 工作流 | 流程编排、任务流 |
| 插件管理平台 Lego | Lego, 插件管理 | 插件开发、插件配置 |

### 2. 存储 (Storage)

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 对象存储 TOS | TOS, 对象存储, bucket | 存储桶管理、文件上传下载 |
| 离线存储 HDFS | HDFS, 离线存储 | 大文件存储、分布式存储 |
| 文件存储 ByteNAS | ByteNAS, 文件存储, NAS | 共享存储、文件系统 |
| 块存储 TBS | TBS, 块存储 | 云盘、磁盘管理 |

### 3. 数据库 (Database)

#### 关系型数据库

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 关系型数据库 RDS | RDS, MySQL, PostgreSQL | 数据库创建、连接配置、性能优化 |
| HTAP数据库 ByteNDB | ByteNDB, HTAP | 混合事务分析、实时查询 |

#### NoSQL数据库

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 高可用 NoSQL Abase | Abase, NoSQL | KV存储、高可用配置 |
| 文档数据库 ByteDoc | ByteDoc, MongoDB | 文档存储、查询语法 |
| 图平台 ByteGraph | ByteGraph, 图数据库 | 图查询、关系存储 |
| 键值数据库 ByteKV | ByteKV, KV | 高性能KV、缓存 |
| 多模态数据库 TokaDB | TokaDB, 多模态 | 多模态数据、向量检索 |
| 表格存储 HBase | HBase, 表格存储 | 大表存储、列族 |

#### 搜索数据库

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 在线搜索 ByteES | ByteES, ES, Elasticsearch | 全文检索、索引管理 |
| 分析引擎 ES | ES分析, 聚合查询 | 日志分析、聚合统计 |
| 向量数据库 Milvus | Milvus, 向量检索 | 向量搜索、相似度匹配 |

#### 数据库管理工具

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 数据传输 ByteDTS | ByteDTS, 数据迁移 | 数据同步、迁移任务 |
| 异构数据同步 Dsyncer | Dsyncer, 数据同步 | 异构数据、同步配置 |
| 数据库工作台 DBW | DBW, 数据库管理 | SQL执行、数据查询 |

### 4. 中间件 (Middleware)

#### 云消息队列

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 消息队列 Kafka & BMQ 版 | Kafka, BMQ, 消息队列 | 消息生产消费、Topic管理 |
| 消息队列 RocketMQ 版 | RocketMQ, 消息队列 | 消息发送、消费组 |
| 异步消息平台 AsyncCloud | AsyncCloud, 异步消息 | 异步通信、消息回调 |

#### 云原生可观测

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 观测诊断 Argos | Argos, 链路追踪 | Trace查询、调用链 |
| 监控打点 Metrics | Metrics, 监控打点 | 指标上报、监控配置 |
| 监控打点 FE 版 | Metrics FE, 前端监控 | 前端性能、异常监控 |
| 变更回溯平台 Eventchange | Eventchange, 变更追踪 | 变更记录、事件查询 |
| 主机监控 Vela | Vela, 主机监控 | 服务器监控、资源告警 |
| 观测中心 Grafana | Grafana, 监控大盘 | 仪表盘、可视化 |

#### 流量调度与治理

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 服务治理平台 Neptune Neo | Neptune, 服务治理 | 服务发现、熔断降级 |
| 服务发现 ByteSD | ByteSD, 服务发现 | 服务注册、发现 |
| 统一插件平台 ByteExt | ByteExt, 插件 | 插件管理、扩展 |
| 混合云调度平台 ByteTraffic | ByteTraffic, 流量调度 | 流量管理、灰度发布 |
| 流量身份标识 ByteTIM | ByteTIM, 流量标识 | 流量染色、链路标识 |

#### 应用集成

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 事件总线 EventBus | EventBus, 事件 | 事件驱动、消息广播 |
| 数据一致性校验 DataEyes | DataEyes, 一致性 | 数据校验、对账 |
| 数据分发 ByteP2P | ByteP2P, 数据分发 | P2P传输、文件分发 |
| 状态机服务 ByteState | ByteState, 状态机 | 状态流转、工作流 |
| 实时数据同步 ByteSync | ByteSync, 实时同步 | 数据同步、CDC |

### 5. 网络与CDN

#### 网络

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 流量服务平台 Netlink | Netlink, DNS, TLB | 域名解析、负载均衡 |

#### 网关

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 业务网关平台 Janus | Janus, API网关 | API管理、限流熔断 |
| 长连接网关 Frontier | Frontier, 长连接 | WebSocket、推送连接 |

#### CDN

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 内容分发网络 CDN | CDN, 内容分发 | 缓存配置、加速节点 |

### 6. 数据服务 (Data Services)

#### 数据开发与服务

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| ByteHouse | ByteHouse, OLAP | 数据分析、SQL查询 |
| 数据安全 Triton | Triton, 数据安全 | 数据脱敏、权限管理 |
| 指标平台 Nuwa | Nuwa, 指标 | 指标定义、数据指标 |
| 数据采集 DataIngestion | DataIngestion, 数据采集 | 数据接入、ETL |
| 数据开发 Dorado | Dorado, 数据开发 | 离线任务、数据加工 |
| 数据地图 Coral | Coral, 数据地图 | 元数据管理、数据血缘 |
| 数据质量平台 Manta | Manta, 数据质量 | 质量规则、数据校验 |
| 数据治理平台 Pontus | Pontus, 数据治理 | 数据标准、治理流程 |
| 数据服务 OneService | OneService, 数据服务 | API服务、数据接口 |
| DataOps | DataOps, 数据运维 | 数据运维、自动化 |

#### 数据应用与可视化

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 风神 Aeolus | Aeolus, BI | 数据报表、可视化分析 |
| 实验评估平台 Libra | Libra, 实验 | A/B测试、实验分析 |
| 行为分析 Tea | Tea, 行为分析 | 用户行为、埋点分析 |
| 画像平台 CDP | CDP, 用户画像 | 用户标签、画像分析 |
| Gaia | Gaia | 数据平台 |

### 7. 人工智能 (AI)

#### 模型平台与服务

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 方舟 Ark | Ark, 模型平台 | 大模型、模型服务 |
| AgentOps 平台 Fornax | Fornax, AgentOps | Agent开发、智能体 |
| AI PaaS | AI PaaS | AI开发平台 |
| MCP Market | MCP, 模型上下文 | 模型协议、工具集成 |
| 记忆库 MemoryBase | MemoryBase, 记忆库 | 对话记忆、上下文存储 |

#### 人工智能平台

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 深度学习训练 Arnold | Arnold, 训练平台 | 模型训练、GPU资源 |
| 深度学习推理 Bernard | Bernard, 推理平台 | 模型部署、推理服务 |
| 机器学习平台 MLX | MLX, 机器学习 | ML流程、特征工程 |

### 8. 中台云 (Platform Services)

#### 用户中台

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 账号服务 Passport | Passport, 账号 | 用户登录、认证 |
| 个人实名认证 | 实名认证 | 身份验证 |
| 企业认证 | 企业认证 | 企业身份 |
| 账号安全风控 | 安全风控 | 风控策略 |
| 用户隐私与合规 Privacy | Privacy, 隐私合规 | 数据隐私、合规 |
| 资料服务 Profile | Profile, 用户资料 | 用户信息、资料管理 |
| 行业账号 | 行业账号 | B端账号 |
| 用户数据托管服务 UserInfo | UserInfo, 用户数据 | 数据存储 |
| 用户服务 | 用户服务 | 用户管理 |
| 设备中台 Device | Device, 设备 | 设备管理 |

#### 社交互动

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 分享平台 | 分享 | 社交分享 |

#### 用户触达

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 推送服务 Push | Push, 推送 | 消息推送、通知 |
| 短信服务 SMS | SMS, 短信 | 短信发送、验证码 |
| 邮件服务 Email | Email, 邮件 | 邮件发送 |
| 短链服务 ShortLink | ShortLink, 短链 | URL缩短 |

#### 其他中台

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| API 网关 AGW | AGW, API网关 | API管理、接口网关 |
| 内容分发服务 CDS | CDS, 内容分发 | 内容推送 |

### 9. 视频云 (Video Cloud)

| Platform | Keywords | Common Search Scenarios |
|----------|----------|------------------------|
| 视频点播 VOD | VOD, 点播 | 视频上传、转码 |
| 图片服务 ImageX | ImageX, 图片 | 图片处理、CDN加速 |
| 云制播 Liveactivity | Liveactivity, 制播 | 直播制作 |
| 实时音视频 TT | TT, RTC | 实时通信、音视频 |
| 字节投屏 ByteCast | ByteCast, 投屏 | 投屏服务 |
| 直播服务架构平台 TTLIVEARCH | TTLIVEARCH, 直播架构 | 直播架构 |
| 多媒体智能处理平台 MIPP | MIPP, 多媒体 | 智能处理 |

## 工作流程

1. **识别平台**: 用户提及平台名称或概念
2. **查找关键词**: 从上方索引中找到相关搜索关键词
3. **搜索文档**: 调用 `/bytedance-cloud-docs` skill 传入关键词
4. **汇总结果**: 向用户呈现相关文档内容

## 常见搜索场景

| 用户问题 | 建议关键词 |
|----------|-----------|
| TCE 怎么部署服务 | TCE, 部署, 服务创建 |
| RDS 如何创建数据库 | RDS, 数据库创建, 连接配置 |
| BMQ 消息怎么消费 | BMQ, 消息消费, Topic |
| Argos 怎么查链路 | Argos, Trace, 链路追踪 |
| Grafana 怎么建大盘 | Grafana, 仪表盘, 监控 |
| Netlink 域名怎么解析 | Netlink, DNS, 域名解析 |
| TOS 怎么上传文件 | TOS, 文件上传, bucket |
| Ark 模型怎么调用 | Ark, 模型调用, API |

## 常见陷阱

1. **使用错误的平台名称**: 某些平台有多个名称（如 "TOS" vs "对象存储"）。请查看关键词列表。
2. **遗漏相关平台**: 一个功能可能跨多个平台（如数据管道同时使用 Dorado 和 ByteHouse）。
3. **关键词过时**: 平台文档结构会随时间变化。如果某个关键词没有结果，尝试列表中的其他关键词。

## Output Language

Detect the user's input language and respond in the same language.
- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).
