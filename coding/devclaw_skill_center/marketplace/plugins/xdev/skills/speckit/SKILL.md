---
name: speckit
description: "Spec Kit - SDD 后端引擎，规格驱动开发全流程。支持 specify / review-spec / tech-guidance / tech-design / review-tech-design / dev / run（全流程）。"
argument-hint: "<action> [参数]"
---

# speckit

Spec Kit — SDD（Specification-Driven Development）后端引擎，统一入口。

> 原名：`spkx` / `devclaw-sdd-be`。

## 用户输入

```text
$ARGUMENTS
```

## 参数格式

`$ARGUMENTS` 格式：`<action> [参数]`

第一个词为 `action`，决定执行哪个 SDD 步骤。后续部分为该 action 的参数（可选）。

| action | 参数 | 说明 | 类型 |
|--------|------|------|------|
| `specify` | `[需求描述或链接]` | 生成功能规格说明书 | 自动化 |
| `review-spec` | `[FEATURE_NAME]` | 审阅规格（HITL） | 交互式 |
| `tech-guidance` | `[FEATURE_NAME]` | 技术指导输入（HITL） | 交互式 |
| `tech-design` | `[FEATURE_NAME]` | 技术方案设计 | 自动化 |
| `review-tech-design` | `[FEATURE_NAME]` | 审阅技术方案（HITL） | 交互式 |
| `dev` | `[FEATURE_NAME]` | 任务拆分 + 逐 Phase 开发 | 自动化 |
| `run` | `[需求描述或链接]` | 全流程串联（specify → ... → dev） | 混合 |

**参数说明**：
- `specify` 和 `run`：带参数 = 新建 feature（参数为需求描述或文档链接），不带参数 = 续接已有 feature
- 其他 action：带参数 = 指定 FEATURE_NAME，不带参数 = 自动识别（当前分支名匹配或交互选择）

---

## 执行方式

> **`<skill_dir>` 约定**：以下路径中 `<skill_dir>` 指代本 SKILL.md 所在目录的绝对路径。

### 1. 解析 action

从 `$ARGUMENTS` 中提取第一个词作为 `action`，剩余部分作为该 action 的参数。

### 2. 路由分派

根据 `action` 的值，读取对应的 action 文件并执行：

```
action = "specify"            → 读取并执行 <skill_dir>/actions/specify.md
action = "review-spec"        → 读取并执行 <skill_dir>/actions/review-spec.md
action = "tech-guidance"      → 读取并执行 <skill_dir>/actions/tech-guidance.md
action = "tech-design"        → 读取并执行 <skill_dir>/actions/tech-design.md
action = "review-tech-design" → 读取并执行 <skill_dir>/actions/review-tech-design.md
action = "dev"                → 读取并执行 <skill_dir>/actions/dev.md
action = "run"                → 读取并执行 <skill_dir>/actions/run.md
其他                          → 输出下方的帮助信息
```

### 3. 执行 action

读取对应的 `actions/{action}.md` 文件后：

1. 将文件内容作为当前任务的执行指令
2. 文件中所有 `<skill_dir>` 均指向本 SKILL.md 所在目录
3. 将剩余参数（去掉 action 后的部分）传递给该 action 的执行逻辑

### 4. 帮助信息（action 未匹配时输出）

```
speckit - Spec Kit / SDD 后端引擎

用法：/speckit <action> [参数]

可用 actions：
  specify [需求描述或链接]    生成功能规格说明书（新建或续接 feature）
  review-spec [FEATURE_NAME]  交互式审阅规格
  tech-guidance [FEATURE_NAME] 交互式收集技术指导
  tech-design [FEATURE_NAME]  生成技术方案设计
  review-tech-design [FEATURE_NAME] 交互式审阅技术方案
  dev [FEATURE_NAME]          任务拆分 + 逐 Phase 开发
  run [需求描述或链接]        全流程串联（推荐）

参数均为可选——不带参数时自动从当前分支或已有 feature 中识别。

示例：
  /speckit run 用户登录流程优化
  /speckit run https://xxx.feishu.cn/docx/xxx
  /speckit run                          # 续接已有 feature
  /speckit specify 新增支付接口
  /speckit review-spec
  /speckit dev
```
