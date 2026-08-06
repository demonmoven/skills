---
name: run
description: SDD 开发入口。根据原始需求（PRD/飞书链接/用户输入）生成前端功能规格说明书 feature_spec.md。Use when user asks to 生成feature spec, 生成功能规格, generation feature spec, 写spec, create feature specification, SDD开发.
---

# 用户 args 解析

优先从 args 里解析相应的参数，如果不存在，则按以下规则处理：

1. **RepoPath**：前端仓库根目录路径（绝对路径）。用户未提供时，**默认使用当前工作目录（`pwd`）**，无需询问。
2. **PrdContent**：需求内容（支持飞书链接、本地文件路径或直接文本）。用户未提供时**必须询问**。
3. **ConstitutionFile**：constitution 规约文档文件路径。用户未提供时**必须询问**。

> PrdContent 和 ConstitutionFile 每个变量单独询问，确保用户明确输入后再继续下一步。

# Generation Feature Spec

SDD（Spec Driven Development）开发入口。根据原始需求生成完整的功能规格说明书（feature_spec.md）。

## 参数

| 参数 | 必填 | 说明 | 示例 |
|------|------|------|------|
| `RepoPath` | ❌ 选填 | 前端仓库根目录路径，默认为当前工作目录（`pwd`） | `/path/to/frontend-repo` |
| `PrdContent` | ✅ 必填 | 需求内容（三种形式，见下方说明） | 飞书链接 / 本地文件路径 / 直接文本 |
| `ConstitutionFile` | ✅ 必填 | constitution 规约文档文件路径 | `/path/to/constitution.md` |

### PrdContent 三种形式

`PrdContent` 支持以下三种输入方式，agent 需要**自动识别**并处理：

1. **飞书文档链接** — URL 中包含 `feishu.cn` 或 `lark` 或 `/docx/` 或 `/wiki/`
   - 使用 `feishu_doc` 工具读取文档内容
   - 提取 doc_token（URL 中 `/docx/XXX` 或 `/wiki/XXX` 部分）
   - 读取完整文档内容作为 PRD

2. **本地文件/目录路径** — 以 `/` 或 `~` 或 `.` 开头的路径
   - 如果是文件：直接读取文件内容
   - 如果是目录：读取目录下所有 `.md` / `.txt` 文件内容

3. **直接文本内容** — 用户直接输入的需求描述
   - 直接使用文本内容作为 PRD

### 参数校验

- `RepoPath` 为**选填**，用户未提供时默认使用当前工作目录（`pwd`）。
- `PrdContent` 和 `ConstitutionFile` 为**必填**，用户未提供时**必须询问获取**。

### 询问模板

如果用户未提供必填参数，询问：

```
需要以下信息来生成 Feature Spec：
1.  需求内容 (PrdContent) — 可以是：
   - 飞书文档链接（自动读取内容）
   - 本地文件路径（如 /path/to/prd.md）
   - 直接粘贴需求文本
2. 📜 规约文档路径 (ConstitutionFile) — constitution 规约文档文件路径？

💡 RepoPath 未提供，将默认使用当前工作目录：{pwd}
```

## 前置处理

### 1. 解析 PrdContent，获取需求文本

根据 PrdContent 的形式读取内容：
- 飞书链接 → `feishu_doc(action=read, doc_token=...)` 或 `feishu_wiki(action=get, token=...)`
- 本地路径 → `read(path=...)` 
- 直接文本 → 直接使用

将获取到的完整需求文本记为 `prdText`。

### 2. 提取特性关键字，创建 SpecsDir

从 `prdText` 中提取一个简短的英文关键字或短语（snake_case，不超过 30 字符），作为特性目录名 `feature_summary`。

例如：
- 需求是"添加评测集管理功能" → `dataset_management`
- 需求是"用户登录流程优化" → `login_flow_optimization`
- 需求是"数据看板" → `data_dashboard`

创建目录：

```bash
mkdir -p {RepoPath}/{feature_summary}
```

`SpecsDir` = `{RepoPath}/{feature_summary}`

### 3. 写入 prd.md

将 `prdText` 写入 `{SpecsDir}/prd.md`。
- 如果传入了本地链接，则说明 prd 已经存在，直接复制到下面的文件即可，无需写入！

### 4. 询问是否需要新建 Feature 分支

PRD 写入完成后，**必须询问用户**是否需要新建一个 feature 分支来实现本次需求。使用以下模板：

```
🌿 是否需要为本次需求新建 feature 分支？

A. ✅ 需要 — 请提供以下信息：
   - 分支名称（如不提供，将自动使用 `feat/{feature_summary}`）
   - 基于哪个分支创建（默认 master）
B. ❌ 不需要 — 跳过
```

- 如果用户选择 **A**：
  1. 确定分支名称：用户指定的名称 或 默认 `feat/{feature_summary}`
  2. 确定基础分支：用户指定的 或 默认 `master`
  3. 在 `RepoPath` 目录下执行：
     ```bash
     cd {RepoPath}
     git fetch origin
     git checkout -b {branch_name} origin/{base_branch}
     git push -u origin {branch_name}
     ```
  4. 告知用户：`✅ 已创建并推送分支：{branch_name}（基于 {base_branch}）`
- 如果用户选择 **B**（或类似否定回复）：跳过此步骤

### 5. 询问是否需要后端 API 变更

PRD 写入完成后，**必须询问用户**是否本次需求涉及后端 API 变更或新增。使用以下模板：

```
📡 本次需求是否涉及后端 API 变更/新增？

A. ✅ 需要 — 请提供 API 文档内容（支持以下形式）：
   - 飞书文档链接（自动读取）
   - 本地文件路径（如 /path/to/api-doc.md）
   - 直接粘贴 API 文档文本
B. ❌ 不需要 — 跳过，直接生成 Feature Spec
```

- 如果用户选择 **A**：按照与 PrdContent 相同的方式解析用户提供的 API 文档内容（飞书链接 → feishu_doc 读取 / 本地路径 → read / 直接文本 → 使用），将内容写入 `{SpecsDir}/api-doc.md`
- 如果用户选择 **B**（或类似否定回复）：跳过此步骤，不创建 api-doc.md

### 6. 确认参数

至此，后续流程所需的参数已就绪：
- `InputsDir` = `{SpecsDir}`（因为 prd.md 已写入该目录）
- `SpecsDir` = `{RepoPath}/{feature_summary}`
- `api-doc.md`（如有）已写入 `{SpecsDir}/api-doc.md`

## 生成流程

### 1. 读取原始需求和模板

从 `{SpecsDir}/prd.md` 读取 PRD 内容，完整理解需求。
读取 `references/spec_template.md` 作为 Feature Spec 模板。

### 2. 生成规格内容

按以下主流程生成规格：

1. **读取原始需求**：从 `{SpecsDir}/prd.md` 读取 PRD
2. **处理不明确项**：
   - 根据上下文和行业标准做出合理推测
   - 仅在显著影响功能范围/用户体验/安全合规时标记 `[待澄清]`
   - 澄清优先级：范围 > 安全/隐私 > 用户体验 > 技术细节
3. **填写用户故事**（遵守 MECE 原则：相互独立、完全穷尽）
4. **生成内容**：
   - 交互流程（Mermaid flowchart TD）
   - 验收场景（markmap）
   - 功能需求（FR 编号，FR-001, FR-002...，禁止嵌套编号）
   - 检测并生成可选章节（业务逻辑规则、状态流转定义、数据实体定义）
   - 检测表单场景并生成表单规格
5. **写入规格**：将规格写入 `{SpecsDir}/feature_spec.md`

### 3. 规格质量验证

1. 在 `{SpecsDir}/requirements.md` 生成质量检查清单
2. 对照检查清单审查规格（内容质量、需求完整性、功能就绪状态）
3. 失败项直接修复，最多 3 次迭代
4. 若存在 `[待澄清]` 标记，逐个向用户展示（格式："问题 1/{TOTAL}"），等待回复后再展示下一个

### 4. 报告完成

报告 specs 目录、规格文件路径、检查清单状态、迭代次数、剩余待澄清问题。

## 🚫 禁止事项

| # | 禁止事项 | 正确做法 |
|---|----------|----------|
| 1 | 使用嵌套需求ID（FR-001-1） | 使用 FR-001, FR-002 |
| 2 | 猜测PRD缺失的逻辑 | 使用 `[待澄清]` 标记 |
| 3 | 一次展示所有澄清问题 | 逐个展示："问题 1/5" |
| 4 | 在规格中包含实现细节 | 仅描述用户需求和业务逻辑 |
| 5 | 重复定义共享功能 | 作为前置条件引用 |

## 用户故事 MECE 原则

- **相互独立**：每个故事涵盖独特的、不重叠的用户旅程
- **完全穷尽**：所有故事加在一起覆盖完整功能范围
- **顺序兼容**：前置故事不应包含或预设后置故事的内容
- 故事 N 的实现不应依赖故事 N+1 的内容
- 开发故事 N+1 时不应需要修改故事 N 的代码

## 输出说明

| 输出文件 | 说明 |
|----------|------|
| `{SpecsDir}/prd.md` | 原始需求文档（前置处理阶段写入） |
| `{SpecsDir}/feature_spec.md` | 完整的功能规格说明书 |
| `{SpecsDir}/api-doc.md` | 后端 API 文档（用户提供时才有） |
| `{SpecsDir}/requirements.md` | 规格质量检查清单 |

## 完成总结（必须）

所有流程完毕后，**必须输出一份简短总结**，包含：
1. **仓库路径**：`RepoPath`
2. **输出目录**：`SpecsDir` 的完整路径
3. **生成文件**：prd.md、feature_spec.md、requirements.md
4. **质量状态**：检查清单通过情况
5. **待澄清数量**：剩余 `[待澄清]` 标记数

# 直接开始下一步：review-feature-spec
直接按照下面的指令开始后续的工作！

```prompt
执行下一步 devclaw-sdd-fe review-feature-spec
  RepoPath: {RepoPath}
  SpecsDir: {SpecsDir}
  ConstitutionFile: {ConstitutionFile}
```
