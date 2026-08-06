# 结构化用户访谈问题库

本文档是 Phase 1 第二轮"结构化用户访谈"的问题库和条件触发规则。
详细执行流程参见 `deep-analysis-guide.md`。

## 必问问题

### Q1: 核心业务场景

**问法**：
> "这个项目解决什么问题？请用 1-2 段话描述核心业务场景，帮助 AI 理解你们在做什么。"

**信息去向**：AGENTS.md 的项目简介、ARCHITECTURE.md 的 Bird's Eye View

### Q2: 常见痛点

**问法**：
> "团队开发中最常遇到的问题或 bug 是什么类型的？"

**选项模板**（根据技术栈调整）：
- 构建/编译配置问题
- 状态管理 bug（数据不一致、竞态条件）
- API 兼容性问题（前后端不匹配）
- 性能问题（内存泄漏、慢查询）
- 环境差异（本地 vs 线上行为不同）
- 依赖冲突/版本问题
- 测试不稳定（flaky test）

**信息去向**：debugging-playbook.md、common-pitfalls.md

### Q3: 关键设计决策

**问法**：
> "有哪些关键的设计决策是你希望 AI 一定要知道的？比如为什么选择了某个框架、某种架构模式、或者某个技术方案。"

**信息去向**：docs/reference/adr/（创建 ADR 骨架）

## 条件触发问题

### Q4: DI/IoC 流程

**触发条件**：代码模式采样中检测到以下模式之一：
- `@Injectable`、`@Inject`、`@ImplementsPort`（装饰器注入）
- `wire.Build`、`wire.NewSet`（Go Wire）
- `container.register`、`container.resolve`（DI 容器）
- `bind<Interface>().to<Impl>()`（接口绑定）

**问法**：
> "我检测到项目使用了 [具体 DI 模式]。注册新服务/新 Port 的标准流程是什么？有哪些必须遵守的约定？"

**信息去向**：code-patterns.md 的"常见开发模式"章节

### Q5: 多区域行为差异

**触发条件**：检测到以下特征之一：
- 多个 `.env.*` 文件（如 `.env.cn`、`.env.i18n`、`.env.boe`）
- 代码中有 `IS_CN`、`IS_ROW`、`region` 等区域标志
- 配置中有多套 API endpoint

**问法**：
> "我检测到项目有多区域/多环境部署配置。各区域/环境之间有哪些行为差异？"

**信息去向**：runtime-behavior.md、integrations.md

### Q6: 实时通信机制

**触发条件**：检测到以下特征之一：
- `EventSource`、`SSE`、`text/event-stream`
- `WebSocket`、`ws://`、`wss://`
- `grpc.Stream`、`ServerStream`
- 消息队列 client（Kafka、RabbitMQ、Redis Pub/Sub）

**问法**：
> "我检测到项目使用了 [SSE/WebSocket/gRPC stream]。实时通信的重连策略和错误恢复机制是什么？断开连接时用户会看到什么？"

**信息去向**：runtime-behavior.md 的"异步与实时通信"章节

### Q7: 高频 Revert 原因

**触发条件**：Git 考古发现某文件被 revert ≥2 次

**问法**：
> "Git 历史显示 [文件名] 被 revert 了 [N] 次。这个文件为什么容易出问题？有什么需要特别注意的？"

**信息去向**：common-pitfalls.md、debugging-playbook.md

### Q8: 代码生成注意事项

**触发条件**：检测到以下特征之一：
- `generated/` 目录或 `DO NOT EDIT` 标记
- `.proto`、`.thrift`、`.graphql` schema 文件
- BAM/swagger/OpenAPI 配置
- `sqlc.yaml`、`ent/schema/`

**问法**：
> "我检测到 [目录/文件] 是自动生成的代码。生成命令是什么？生成前需要注意什么？"

**信息去向**：code-patterns.md、invariants.md（代码生成边界不变量）

### Q9: 核心数据流

**触发条件**：检测到复杂状态管理——以下条件满足其二：
- ≥3 个 store/state 文件
- 使用 Zustand/Redux/MobX/Vuex
- 有跨组件通信模式（Context Provider、Event Bus）

**问法**：
> "项目的核心数据流是怎样的？比如用户执行 [主要操作] 时，数据从哪里开始、经过哪些处理、最终怎么反映到 UI 上？"

**信息去向**：runtime-behavior.md 的"核心数据流"章节

### Q10: Monorepo 包协调

**触发条件**：检测到 monorepo 结构（≥3 个 package.json/go.mod/Cargo.toml）

**问法**：
> "子包之间的发布依赖和版本协调方式是什么？修改了底层包后，怎么确保上层包也跟着更新？"

**信息去向**：code-patterns.md、release-guide.md

### Q11: 外部服务降级

**触发条件**：外部集成点分析发现 ≥3 个外部服务

**问法**：
> "我检测到项目集成了 [服务列表]。当某个外部服务故障时，有降级策略吗？哪些服务是关键路径上的？"

**信息去向**：integrations.md 的"故障降级"列
