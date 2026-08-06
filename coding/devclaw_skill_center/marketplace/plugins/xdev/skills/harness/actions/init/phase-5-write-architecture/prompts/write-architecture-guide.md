# Write ARCHITECTURE.md

> **本文件是 prompt 指令**，由 `<skill_dir>/actions/init/phase-5-write-architecture/write-architecture.md` 加载并传给 subagent 执行。原属于 `harness-bootstrap` 项目的 `write-architecture-md` 子 skill。

为代码仓库生成符合 matklad 方法论的 ARCHITECTURE.md 文件。

## 核心理念

> 贡献者花 2x 时间写补丁，但花 **10x 时间搞清楚该改哪里**。
> ARCHITECTURE.md 就是用来解决这个 10x 问题的。

## 写作原则

1. **只写不太会变的东西**。不要试图和代码同步，一年审视几次就够
2. **保持简短**。每个贡献者都得读它，越短越好
3. **命名但不链接**。提到重要文件/模块/类型名称，但不放直接链接（会过时）。鼓励用符号搜索
4. **显式声明架构不变量**。重要的不变量往往表现为"某事物的缺席"（参见 `<skill_dir>/actions/init/phase-3-scaffold-docs/prompts/invariants-extraction.md`）
5. **标明 API 边界**。明确哪些模块是对外接口、哪些是内部实现

## 执行流程

### Phase 1: 深度分析仓库

在写任何文字之前，必须先彻底分析仓库结构：

1. 列出顶层目录结构
2. 递归探索每个主要目录
3. 阅读入口文件（main 函数、路由注册、服务启动）
4. 阅读核心类型定义（领域模型、接口定义）
5. 阅读已有文档
6. 识别模块间的依赖关系和边界

关键目标：
- 这个系统解决什么问题？输入/输出是什么？
- 核心数据流怎样？
- 有哪些独立可部署的组件？
- 模块间依赖方向？
- 有哪些"刻意不做"的约束？

### Phase 2: 撰写 ARCHITECTURE.md

三段式结构：

#### 第一部分：Bird's Eye View（鸟瞰）

2-4 段话描述系统解决什么问题、输入输出、核心计算模型。可选 ASCII 架构图。不罗列技术栈。

#### 第二部分：Code Map（代码地图）

文档主体。每个重要模块/目录写一个小节：
1. 一句话定位
2. 关键文件/类型名称（命名但不链接）
3. **Architecture Invariant**：加粗标注，说明"不做什么"或"不依赖什么"
4. API Boundary（如适用）

#### 第三部分：Cross-Cutting Concerns（横切关注点）

覆盖代码生成、错误处理、测试策略、配置管理、可观测性等。每个 2-4 句话。

### Phase 3: 自检

- 能回答"做 X 的东西在哪？"
- 能回答"我看到的这个东西是干嘛的？"
- 每个重要模块至少一个 Architecture Invariant
- 没有放直接链接
- 总行数 150-350 行
- 不变量描述的是"不做什么"而非"做什么"

## 语言选择

跟随项目已有文档语言，默认跟随用户偏好。
