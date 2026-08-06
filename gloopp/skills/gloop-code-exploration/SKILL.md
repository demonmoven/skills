---
name: gloop-code-exploration
version: 1.0.0
description: "代码探索方法论：用分层代码理解模型减少重复扫代码。教你如何高效使用代码地图（codebase map）、工作区上下文和搜索工具，从顶层架构逐步下钻到具体代码。适用场景：新任务启动时想快速了解项目结构、需要定位某个功能的代码位置、不确定该读哪些文件时。不适用：已经明确知道改哪里的小改动。"
metadata:
  class: both
  category: info
  kind: capability
  related_skills:
    - name: gloop-user-context
      type: depends_on
      description: codebase_map 等维度由 context 系统提供，L2 层核心依赖
    - name: gloop-self-awareness
      type: related
      description: activity_snapshot 维度通过 self-awareness 查询
    - name: gloop-note-keeping
      type: related
      description: 探索发现建议用 note add 固化
    - name: gloop-quest-execution
      type: related
      description: 代码探索是执行流程中的一个子阶段
  requires:
    bins: ["gloop"]
    cliHelp: "gloop context --help"
---

# gloop-code-exploration

代码探索的分层方法论。目标是**用最少的 token 找到最相关的代码**，避免每个任务都从 `ls -R` 和全量 grep 开始。

## Trigger Examples

| 类型 | 场景 | 说明 |
|---|---|---|
| 适用 | 新任务启动，对项目结构不熟悉 | 先看地图再动手 |
| 适用 | 需要定位某个功能的代码位置 | 分层下钻，不盲目搜索 |
| 适用 | 不确定改动影响范围 | 从地图推断波及面 |
| 适用 | 评审他人代码时快速建立认知 | 架构图 + 关键模块 |
| 不适用 | 已经明确知道改哪个文件的小改动 | 直接读文件就行 |
| 不适用 | 代码地图已经读过且项目没变化 | 不要重复加载 |

## 分层代码理解模型

Gloop 提供四层代码认知能力，从上到下逐层深入。**永远从上层开始，不够时再往下钻**。

```
L3 Activity Snapshot      — 当前循环中还有哪些任务在进行、近期做了什么（全局视角）
L2 Codebase Map    — 代码地图：目录树、核心模块、关键接口、架构模式（鸟瞰）
L1 Search Tools    — 搜索工具：grep、符号跳转、文件定位（精准查找）
L0 Source Code     — 源代码：逐行阅读（深入细节）
```

各层特点：
- **L3 Activity Snapshot**：跨任务视角，知道最近改了什么、还有什么在进行。通过 `gloop context show activity_snapshot` 获取。
- **L2 Codebase Map**：项目级索引，告诉你"有什么模块、在哪里、干什么"。通过 `gloop context show codebase_map` 获取。
- **L1 Search Tools**：精准定位工具，告诉你"具体位置在哪里"。优先用 `gloop code search`（结构化符号搜索），备选 shell grep / find / editor native tools。
- **L0 Source Code**：实际代码文件，告诉你"具体怎么实现的"。通过文件读取获取。

v0.4 边界：Activity Snapshot 是全局活动摘要；Loop State Spine 是单个 quest 的执行记忆。代码探索可以用 Activity Snapshot 判断近期改动，但不要把它当当前 phase 的状态源；当前 quest 的 phase、Human Exception、Loop State Spine 走 gloop-self-awareness / quest detail。

### 分层原则

1. **先上后下**：优先使用高层信息，信息不足时才向下一层探索。
2. **按需下钻**：不需要理解整个项目，只探索与当前任务相关的部分。
3. **带目的搜索**：每次下钻都要有明确的问题，不要漫无目的地读代码。
4. **复用记忆**：同一次任务中已经读过的模块不要再重读，记在脑子里或笔记里。

## 标准探索流程

### 第 1 步：全局定位（L3 + L2）

先建立全局认知，不要一上来就 grep。

```bash
# 1. 看循环状态（最近在改什么，有没有相关任务）
gloop context show activity_snapshot

# 2. 看代码地图（项目结构、核心模块、架构模式）
gloop context show codebase_map

# 3. 如有需要，补看工作区上下文（技术栈、项目契约）
gloop context show workspace
```

读完后回答三个问题：
- 这个项目是干什么的？主要模块有哪些？
- 我要改的功能大概在哪个模块？
- 有没有近期改动可能影响我？

如果三个问题都能回答，进入下一步；否则继续看 L2 的细节。

### 第 2 步：精准定位（L1）

用搜索工具从模块级缩小到文件级/符号级。

常用方法：
- **符号定义搜索**：用 `code_search` 工具的 definition 模式，直接定位函数/类型/接口定义
- **引用搜索**：用 `code_search` 的 reference 模式，快速找调用方和被调用方
- **文件定位**：按目录结构 + 命名约定猜文件路径
- **入口追踪**：从 CLI 入口 / main 函数 / API endpoint 顺着调用链往下走

搜索技巧：
- 先搜符号定义（definition 模式噪音最少），再搜引用
- 用 language 参数过滤，减少无关结果
- 搜到太多结果时，加更多限定词；太少时，放宽条件或切到 all 模式
- 优先看接口和类型定义，不急着看实现
- code_search 不够用时，再用 shell grep/find 做兜底

### 第 3 步：深入细节（L0）

找到相关文件后，再读具体代码。

读代码策略：
- 先读接口/类型定义，理解数据结构
- 再读关键函数的签名和注释，理解行为
- 最后才读实现细节
- 边读边在脑子里（或笔记里）画调用链和数据流

## Codebase Map 使用指南

Codebase map 是 L2 层的核心产物，由 `auto_codebase_map_refresh` automation 定期生成。

### 内容结构

| 章节 | 用途 |
|---|---|
| **Scope** | 仓库基本信息、扫描范围、更新时间 |
| **Directory Tree** | 顶层目录树 + 每个目录的一句话职责 |
| **Core Modules** | 核心模块列表：职责、关键类型、与其他模块关系 |
| **Key Interfaces** | 对外接口、CLI 命令、入口文件 |
| **Architecture Patterns** | 架构模式、关键数据流/调用链 |
| **Build & Test** | 构建和测试命令、关键配置 |
| **Knowledge Gaps** | 未覆盖区域和不确定项 |

### 正确读法

1. 先扫 **Directory Tree** 和 **Core Modules**，建立模块清单
2. 根据任务目标，圈出 2-3 个最相关的模块
3. 读 **Key Interfaces** 了解入口点
4. 读 **Architecture Patterns** 理解整体设计
5. 带着具体问题进入 L1 搜索

### 常见误区

- ❌ 把 codebase map 当事实权威——它是摘要，可能过时或有遗漏，关键信息要到源码里确认
- ❌ 每次都全量重读——只看与当前任务相关的部分
- ❌ 发现 map 过时就停下来更新——先完成任务，有空闲再建议刷新

## 减少重复扫描的技巧

1. **善用笔记**：探索过程中把关键发现记在 quest notes 里（`note_add` 工具），后面返工或评审时不用重扫
2. **按模块工作**：一次专注一个模块，读完一个再下一个，避免来回跳
3. **增量探索**：第一轮先建立高层认知，第二轮再深入细节，不要试图一次搞懂所有
4. **利用历史**：activity_snapshot 里提到的近期完成的 quest，可能已经探索过相关代码，可以直接读它们的交付物

## 与其他 Skill 的关系

- **gloop-user-context**：教你如何使用全局上下文系统（包括 codebase map）。本 skill 专注于代码探索方法论。
- **gloop-self-awareness**：教你查询当前 quest 的状态和历史。本 skill 专注于代码结构探索。
- **gloop-note-keeping**：教你如何记录笔记。探索过程中的发现建议用笔记固化下来。
- **gloop-quest-execution**：教你整体执行方法论。代码探索是执行的一个子阶段。

## CLI Contract

本 skill 覆盖代码探索相关的 gloop CLI 命令，包括上下文查询、笔记记录和结构化代码搜索三大类。

### 命令一览

| 命令 | 用途 |
|---|---|
| `gloop context show <dim>` | 查看指定维度的上下文（L3/L2 层） |
| `gloop context list` | 列出所有可用的上下文维度 |
| `gloop note add` | 追加笔记到当前委托的事件流 |
| `gloop code search` | 结构化代码符号搜索（L1 层推荐工具） |

### 详细说明

#### gloop context show

查看指定维度的上下文信息，用于在 L3/L2 层建立全局认知。

**用法：**
```bash
gloop context show <dim>
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `<dim>` | 是 | string | 上下文维度，支持 `activity_snapshot`、`codebase_map`、`workspace` |

**返回：**

对应维度的结构化上下文内容。

**注意：**
- `activity_snapshot`：L3 层，展示当前循环中还有哪些任务在进行、近期做了什么（全局视角）
- `codebase_map`：L2 层，由 `auto_codebase_map_refresh` automation 定期生成，包含目录树、核心模块、关键接口、架构模式等项目级索引
- `workspace`：补充维度，包含技术栈、项目契约等工作区上下文
- 同一次任务中，每个维度读过一次即可，不要重复加载

#### gloop context list

列出所有可用的上下文维度。

**用法：**
```bash
gloop context list
```

**参数：** 无参数

**返回：**

可用上下文维度列表。

#### gloop note add

追加笔记到当前委托的事件流，用于固化探索过程中的关键发现。

**用法：**
```bash
gloop note add "<内容>"
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `<内容>` | 是 | string | 笔记正文，append-only，跨阶段可见 |

**返回：**

无（纯副作用命令）

**注意：**
- 笔记是 append-only 的事件流，后续所有阶段和返工轮次均可见
- 探索过程中建议记录：关键模块位置、架构发现、已知坑点、待验证假设

#### gloop code search

结构化代码符号搜索，L1 层的推荐工具，输出结构化、噪音少、所有 agent 都能用。

**用法：**
```bash
gloop code search [options] "<查询关键词>"
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `--mode` | 否 | string | 搜索模式：`definition`（只搜符号定义，默认，噪音最少）、`reference`（搜引用，排除定义行）、`all`（全部匹配，含注释、字符串中的引用） |
| `--language` | 否 | string | 按语言过滤结果，如 `go`、`python` 等 |
| `--include-tests` | 否 | flag | 是否包含测试文件，默认不包含 |
| `<查询关键词>` | 是 | string | 要搜索的符号名或关键词 |

**返回：**

结构化的搜索结果列表：

```json
[
  {
    "file": "文件路径",
    "line": 123,
    "symbol_type": "function / type / interface / const / var / other",
    "symbol_name": "识别出的符号名",
    "signature": "定义行签名",
    "snippet": "该行原文"
  }
]
```

**注意：**
- 优先使用 `definition` 模式，噪音最少
- 先搜符号定义，再搜引用
- 用 `--language` 参数过滤，减少无关结果
- 搜到太多结果时，加更多限定词；太少时，放宽条件或切到 `all` 模式
- 优先看接口和类型定义，不急着看实现

#### Shell 命令备选（非 gloop CLI，兜底用）

`gloop code search` 不可用时，以下 shell 命令可按需使用（法师请走白名单或自行确认副作用等级）。

**用法：**
```bash
# 符号搜索
grep -rn "符号名" --include="*.go" ./

# 文件查找
find . -name "*.go" -path "*/module/*"

# Git 历史
git log --oneline -20
git diff main --stat
```

**注意：**
- 以上不是 gloop CLI 命令，仅作为 code_search 的兜底方案
- 优先使用 `gloop code search`，输出更结构化

### 通用约定

- CLI 是本地 agent 调用 gloop 平台能力的主路径
- 平台注入 `GLOOP_*` 环境变量，通常不需要手填 qid/sid
- **分层探索原则**：从 L3 → L2 → L1 → L0 逐步深入，永远从上层开始，不够时再往下钻
- **使用节奏**：任务开始时 `context show codebase_map` + `context show activity_snapshot` 各 1 次即可；探索过程中用搜索工具做 L1/L0 下钻；关键发现用 `note add` 固化
- 不要每轮都重新加载 codebase map，或无目的地全量 grep

## Discipline

1. **分层下钻**：从 L3 → L2 → L1 → L0 逐步深入，禁止跳过上层直接读代码
2. **带目的搜索**：每次探索都要有明确的问题，不要"随便看看"
3. **不重复加载**：同一次任务中，codebase map 和 context 维度读过一次就够了
4. **关键信息二次确认**：codebase map 是摘要不是事实，涉及改动的关键信息要到源码里确认
5. **发现过时不阻塞任务**：如果发现 codebase map 明显过时，记下来但不要停下当前任务去刷新
6. **探索有节**：不要在代码探索阶段花太多 token，明确够用了就开始动手
7. **固化发现**：重要的探索结果用 `note_add` 记下来，方便后续阶段和返工复用
