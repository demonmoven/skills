# 3.1.1.1 exec-plan

> **本节目标**：理解 exec-plan 的定位——最轻量的 SDD，Vibe Coding / Plan Mode 的工程化替代。

---

## 概念与定位

**exec-plan** 源自 OpenAI 在 Harness Engineering 实践中提出的"计划是一等公民"理念。

> 参考：[Using PLANS.md for multi-hour problem solving](https://cookbook.openai.com/articles/codex_exec_plans)（OpenAI Cookbook）

### 核心定位：最轻量的 SDD

exec-plan 是 SDD 工具谱系中**最轻量**的一个——只有一个文档（plan.md），流程最短，适合敏捷小需求。它填补了 Vibe Coding / Plan Mode 和完整 SDD（OpenSpec / Spec Kit）之间的空白：

```text
Vibe Coding        exec-plan           OpenSpec / Spec Kit
(无文档)        (1 个文档 plan.md)     (多个文档: spec + design + tasks)
  ↑                  ↑                      ↑
  最快               快                     最完整
  最不可控           可控                   最可控
  适合一行改动        适合小需求(≤1人天)      适合大需求(>2人天)
```

### 为什么要替代 Vibe Coding / Plan Mode

| 方式 | 问题 |
|------|------|
| Vibe Coding (Write Mode) | 无任何文档痕迹，Agent 做完就忘，无法复盘 |
| Plan Mode | 计划在对话上下文中，session 结束就丢失，无法中断恢复 |
| **exec-plan** | 计划持久化为 plan.md，可中断恢复、可追溯、可协作 |

---

## 工作流

exec-plan 的流程极简——只有 3 步：

```text
Step 1: 创建 plan.md
  用户描述需求 → Agent 生成 plan.md（含目标、milestone、验收标准）

Step 2: 按 milestone 执行
  Agent 逐个完成 milestone → 每完成一个立即更新 Progress

Step 3: 完成归档
  全部 milestone 完成 → 自动归档到 completed/
```

### 文件存放

```text
docs/plans/
├── proposal/    # 草案：待确认
├── active/      # 执行中
└── completed/   # 已归档
```

---

## Slash Command

| 命令 | 说明 | 产物 |
|------|------|------|
| `/exec-plan run "描述"` | 创建新的 plan 并开始执行 | `docs/plans/active/{name}.md` |
| `/exec-plan continue` | 继续当前活跃计划 | 更新已有 plan.md 的 Progress 段 |

### 可选：pre-push 守护

配合 Git Hook，在推送前检查 active plans 的完成度：

```bash
# .hooks/check-active-plans.sh（pre-push 阶段）
# - 发现 [ ] → 阻止推送
# - 全部 [x] → 自动归档到 completed/
```

---

## 中间产物：plan.md

> **核心思想**：单文档即全部——一个 plan.md 同时承载需求、方案、任务、进度，实现"自包含 + 可中断恢复"。

### 模板

```markdown
# Plan: {功能名称}

## Purpose / Big Picture
<!-- 为什么要做这件事，解决什么问题 -->

## Plan of Work
### Concrete Steps
- [ ] M1: {任务描述}（验收：{量化标准}）
- [ ] M2: {任务描述}（验收：{量化标准}）

## Validation and Acceptance
<!-- 全局验收标准 -->
- {验收条件 1}
- {验收条件 2}

## Progress
<!-- 每完成一个 Milestone 立即记录 -->
### M1 (完成)
- {做了什么}
- {关键发现}

## Surprises & Discoveries
<!-- 意外发现随时写入，沉淀为项目知识 -->

## Decision Log
<!-- 关键技术决策及理由 -->
```

### 模板规范

| 维度 | 要求 |
|------|------|
| 自包含性 | Plan 必须完全自包含，无项目上下文也能理解 |
| 弱实现、强验收 | Milestone 只写"做什么 + 验收标准"，不写"怎么实现" |
| 实时更新 | 每完成一个 milestone 立即记录 Progress |
| 幂等恢复 | 中断后可从任意 milestone 恢复执行 |
| Surprises 沉淀 | 意外发现随时写入，沉淀为项目知识 |

### 示例

```markdown
# Plan: 用户头像上传功能

## Purpose / Big Picture
支持用户上传和裁剪头像，存储到 TOS

## Plan of Work
### Concrete Steps
- [x] M1: API 接口定义（POST /api/v1/user/avatar）
- [x] M2: 后端实现 + 单测
- [ ] M3: 前端 UI + 裁剪交互

## Validation and Acceptance
- 上传 ≤5MB 的 jpg/png 成功
- 超大文件返回 413 错误码
- 单测覆盖率 ≥ 80%

## Progress
### M1 (完成)
- 定义了 multipart/form-data 上传接口
- 响应结构：{ url, width, height }

### M2 (完成)
- 实现了 TOS 上传 + 图片校验

## Surprises & Discoveries
- TOS SDK v3 的 Content-Type 自动检测在 multipart 场景下失效，需手动指定
```

---

## 适用场景

| 场景 | 是否适用 | 说明 |
|------|----------|------|
| 小需求（≤1 人天） | 最适合 | 替代 Vibe Coding，多了可追溯的 plan.md |
| 快速修复 + 想留痕 | 适合 | 比 Vibe Coding 多了文档，比完整 SDD 轻 |
| 跨模块大重构 | 可以但不是最优 | 建议用 OpenSpec / Spec Kit（多文档更清晰） |
| 一行配置改动 | 不适用 | 直接 Vibe Coding 即可 |

---

[下一节：OpenSpec →](031a2-openspec.md) | [返回上级：SDD](031a-sdd.md)
