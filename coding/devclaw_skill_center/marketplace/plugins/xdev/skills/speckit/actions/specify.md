# 需求规格生成

根据用户提供的需求描述（文本或文档链接）生成完整的功能规格说明书。

## 参数

- `ARG`（可选）：需求描述文本或文档链接

---

## 入参校验

读取并执行 `<skill_dir>/prompts/resolve_feature_name.md`，传入以下参数：
- `ACTION_TYPE = "specify"`
- `ARG = {用户传入的参数}`

执行完成后获得 `FEATURE_NAME`、`SPECS_DIR`、`IS_NEW_FEATURE`、`PRIMARY_REPO`。

> 注：resolve_feature_name 内部已经调用了 resolve_workspace.md，自动完成了多仓库扫描 / 主仓库选择 / 分支切换。

校验通过后，推导内部变量（使用 `PRIMARY_REPO` 作为工作目录，而非 `$(pwd)`）：

```text
CWD = {PRIMARY_REPO}
FEATURE_DOC = {SPECS_DIR}/feature.md
PRD_DOC = {SPECS_DIR}/prd.md
SPEC_DOC = {SPECS_DIR}/spec.md
CONSTITUTION_DOC = {CWD}/constitution/constitution.md
```

继续校验：

1. `CONSTITUTION_DOC`（必需）文件存在
   - 不存在 → 提示用户："`CONSTITUTION_DOC` 文件不存在：{路径}，请在项目根目录下准备 constitution/constitution.md 文件"
2. `FEATURE_DOC`（必需）文件存在
   - 不存在 → 提示用户："`FEATURE_DOC` 文件不存在：{路径}，请确认 feature.md 文件是否已生成"
   - 注：如果是新建 feature（IS_NEW_FEATURE = true），feature.md 已在 resolve_feature_name 阶段创建

全部校验通过后，进入 PRD 内容获取阶段。

## PRD 内容获取

> **核心原则**：`prd.md` 必须包含**绝对完整**的需求信息，当用户输入包含 URL 链接时，读取下来的原始文档内容**必须完整保留、不做任何删减**，原封不动写入 `prd.md`。

读取 `FEATURE_DOC`（feature.md）获取用户原始输入 `{USER_INPUT}`，然后解析其中的链接并读取内容：

1. **检测飞书文档链接**：匹配 `https://*.feishu.cn/docx/*`、`https://*.feishu.cn/wiki/*`、`https://*.feishu.cn/docs/*`、`https://*.larkoffice.com/docx/*` 等格式
   - 匹配到飞书链接 → 尝试调用飞书 MCP 工具读取文档内容（**完整读取，不遗漏任何内容**）
   - 如果飞书 MCP 不可用 → 提示用户："飞书文档链接检测到，但飞书 MCP 未配置。请手动将文档内容贴入，或按以下方式配置飞书 MCP：`npx -y @larksuiteoapi/lark-mcp mcp -a <APP_ID> -s <APP_SECRET>`"，**终止流程**
2. **检测其他 URL**：匹配 `http://` 或 `https://` 格式
   - 使用 `WebFetch` 工具读取内容（**完整读取，不遗漏任何内容**）
3. **纯文本**：直接保留

将所有内容按以下结构写入 `PRD_DOC`（prd.md）：

```markdown
# 需求描述

## 用户原始输入

{USER_INPUT 原文，但移除其中所有 URL 链接（飞书链接、HTTP 链接等），仅保留纯文本描述部分。如果用户输入仅为 URL 无其他文本，则写入"（见下方引用文档内容）"}

## 引用文档内容

### {文档标题}

{完整的文档内容，不做任何删减}

### {文档标题}

{完整的文档内容，不做任何删减}

（如有多个链接则重复此结构；如无链接则省略"引用文档内容"章节）
（标题优先使用文档自身标题，无标题时使用内容摘要概括，不得使用 URL）
```

> **重要**：`prd.md` 中**不得出现任何 URL 链接**。所有链接的内容已展开写入"引用文档内容"章节，URL 本身不再保留。

确认 `PRD_DOC` 文件已生成后，进入执行阶段。

## 输出参数

1. `FEATURE_NAME`：功能特性名称
2. `SPEC_DOC`：`{SPECS_DIR}/spec.md`，生成的功能规格说明书

## 执行方式

使用 Agent 工具派发 subagent 执行规格生成：

```
Agent(subagent_type="general-purpose", prompt="
你需要执行一个需求规格生成任务。

以下是你的输入变量（均为绝对路径）：
PRD_DOC={PRD_DOC}
SPEC_TEMPLATE_DOC=<skill_dir>/resources/spec_template.md
SPECS_DIR={SPECS_DIR}
SPEC_DOC={SPEC_DOC}

请读取并严格按照以下指令文件执行：<skill_dir>/prompts/specify_prompt.md

以下参数已全部校验通过，直接使用，无需校验、无需向用户提问。
执行完成后，报告产出文件路径。
")
```

## 执行完成后输出

```
Feature 规格生成完毕！

关键变量汇总：
- FEATURE_NAME={FEATURE_NAME}
- SPECS_DIR={SPECS_DIR}
- SPEC_DOC={SPEC_DOC}
```
