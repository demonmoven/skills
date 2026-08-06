> **Action: `feishu-sync`** — 由 `/xdev:harness-kit feishu-sync` 路由调用。
> 原 skill: `harness-feishu-sync` (author: guoshuai.030, version: 1.4)

# Harness Feishu Sync — 飞书文档与仓库双向同步

## 核心理念

> **Agent 读不到的信息，等于不存在。**
> Google Docs 里的讨论、Slack 里的对话、飞书文档里的设计决策——
> 如果不写进仓库，对 Agent 来说就好像从来没发生过。
> — OpenAI Harness Engineering

本技能解决 Harness Engineering 中的关键痛点：**团队知识散落在飞书文档中，Agent 无法访问**。
通过双向同步，让飞书上的知识以版本化 Markdown 的形式进入仓库，成为 Agent 可读的上下文。

**两个方向**：
- **飞书 → 仓库**：读取飞书文档内容，Agent 转为 Markdown，写入 `docs/`
- **仓库 → 飞书**：将仓库 Markdown 发布为飞书文档，供非工程角色阅读

## 工具选型

### 推荐：lark-cli（飞书官方 CLI）

> **首选方案。** 由 [larksuite](https://github.com/larksuite/cli) 团队维护，MIT 协议，
> 200+ 命令覆盖 14 大业务域，22 个 AI Agent Skills 开箱即用。

**安装**：

安装 CLI：

```bash
npm install -g @larksuite/cli
```

安装 CLI Skills（必需）：

```bash
npx skills add larksuite/cli -y -g
```

配置应用凭证（交互式，仅需一次）：

```bash
lark-cli config init
```

登录授权：

```bash
lark-cli auth login --recommend
```

**文档读写**（`lark-doc` skill）：

读取飞书文档（输出 Markdown）：

```bash
lark-cli docs +read --doc-id <DOC_ID>
```

创建飞书文档：

```bash
lark-cli docs +create --title "周报" --markdown "# 本周进展\n- 完成了 X 功能"
```

搜索文档：

```bash
lark-cli docs +search --query "设计方案"
```

**其他常用能力**：

日历议程：

```bash
lark-cli calendar +agenda
```

发消息：

```bash
lark-cli im +messages-send --chat-id "oc_xxx" --text "Hello"
```

知识库：

```bash
lark-cli wiki +search --query "架构文档"
```

任意飞书 API：

```bash
lark-cli api GET /open-apis/docx/v1/documents/<DOC_ID>/blocks
```

详见：https://github.com/larksuite/cli/blob/main/README.zh.md

### 备选：内置脚本（临时方案）

当 lark-cli 不可用（如无法安装 npm、网络受限等）时，可使用本技能内置的 Python 脚本作为临时替代：

- [references/api.py](references/api.py) — 飞书 Open API 封装（Python，stdlib only，零依赖）
- [references/md2feishu.py](references/md2feishu.py) — Markdown → 飞书文档 CLI 工具
- [references/block-types.md](references/block-types.md) — Block 类型速查表

内置脚本的局限：
- 仅覆盖文档读写，不支持日历/消息/表格等业务域
- 读取文档返回原始 block JSON，需要 Agent 手动转换
- 需要手动配置 `FEISHU_APP_ID` / `FEISHU_APP_SECRET` 环境变量
- 权限管理较原始（无交互式授权流程）

**如果 lark-cli 可用，应始终优先使用 lark-cli。**

## 前置条件

### 使用 lark-cli 时

```bash
npm install -g @larksuite/cli
npx skills add larksuite/cli -y -g
lark-cli config init
lark-cli auth login --recommend
```

### 使用内置脚本时

```bash
export FEISHU_APP_ID="cli_xxx"        # 飞书应用 App ID
export FEISHU_APP_SECRET="your_secret" # 飞书应用 App Secret
```

应用权限：

| 操作 | 所需权限 |
|------|---------|
| 读文档 | `docx:document:readonly` |
| 写文档 | `docx:document:write` |
| 上传图片 | `drive:drive` |
| 删除文档 | `drive:drive` 或 `space:document:delete` |
| 查邮箱→open_id | `contact:contact.base:readonly` |
| 转移 ownership | `drive:drive` |

---

## 方向一：飞书 → 仓库（读取并转为 Markdown）

### 方案 A：使用 lark-cli（推荐）

读取飞书文档（直接输出 Markdown）：

```bash
lark-cli docs +read --doc-id <DOC_ID>
```

搜索飞书文档：

```bash
lark-cli docs +search --query "设计方案"
```

读取知识库文档：

```bash
lark-cli wiki +search --query "架构"
```

`lark-cli docs +read` 直接输出 Markdown 格式，Agent 可以直接写入仓库文件，无需手动转换。

`doc_id` 从飞书文档 URL 中提取：`https://xxx.feishu.cn/docx/{DOC_ID}`

### 方案 B：使用内置脚本（临时方案）

当 lark-cli 不可用时，使用内置 Python API 读取 block JSON，由 Agent 转换为 Markdown。

#### Step 1: 获取 Token

```python
from references.api import get_token
token = get_token()  # 读取 FEISHU_APP_ID / FEISHU_APP_SECRET 环境变量
```

#### Step 2: 读取文档 Block 列表

获取文档元信息（标题）和所有 block（自动分页）：

```python
from references.api import list_blocks, read_document

meta = read_document(token, doc_id)
title = meta["document"]["title"]

blocks = list_blocks(token, doc_id)
```

#### Step 3: Agent 将 Block JSON 转为 Markdown

`list_blocks()` 返回的是 block JSON 数组。Agent 根据以下映射规则转换：

**Block Type → Markdown 映射速查**：

| block_type | 名称 | Markdown 转换 |
|------------|------|--------------|
| 1 | Page | 文档根节点，不输出 |
| 2 | Text | 直接输出文本内容，拼接 `text_run` 的 `content` |
| 3 | Heading1 | `# {content}` |
| 4 | Heading2 | `## {content}` |
| 5 | Heading3 | `### {content}` |
| 6 | Heading4 | `#### {content}` |
| 7–11 | Heading5–9 | `#####`–`#########` |
| 12 | Bullet | `- {content}`（看 `children` 判断嵌套层级） |
| 13 | Ordered | `1. {content}` |
| 14 | Code | ` ```{language}\n{content}\n``` ` |
| 15 | Quote | `> {content}` |
| 17 | Todo | `- [ ] {content}` 或 `- [x] {content}` |
| 22 | Divider | `---` |
| 27 | Image | `![image]({file_token})` — 需单独下载图片 |
| 31 | Table | Markdown 表格语法 |

**提取文本内容的方法**：

大多数文本类 block 的内容在 `block.{type_key}.elements` 数组中，每个 element 是一个 `text_run`：

```json
{
  "block_type": 2,
  "text": {
    "elements": [
      {"text_run": {"content": "Hello ", "text_element_style": {"bold": true}}},
      {"text_run": {"content": "World"}}
    ]
  }
}
```

- 拼接所有 `text_run.content` 得到纯文本
- `text_element_style.bold` → `**text**`
- `text_element_style.italic` → `*text*`
- `text_element_style.strikethrough` → `~~text~~`
- `text_element_style.inline_code` → `` `text` ``
- `text_element_style.link.url` → `[text](url)`

**type_key 对照**（block_type → JSON 中的 key）：

| block_type | JSON key |
|------------|----------|
| 2 | `text` |
| 3–11 | `heading1`–`heading9` |
| 12 | `bullet` |
| 13 | `ordered` |
| 14 | `code` |
| 15 | `quote` |
| 17 | `todo` |

**代码块特殊处理**：

```json
{
  "block_type": 14,
  "code": {
    "elements": [{"text_run": {"content": "print('hello')"}}],
    "language": 49
  }
}
```

language 枚举 → 语言名映射见 [references/block-types.md](references/block-types.md)。
常用：`7=Bash` `22=Go` `29=Java` `30=JavaScript` `49=Python` `53=Rust` `63=TypeScript` `67=YAML`

**表格处理**：

Table block (type 31) 的子节点是 TableCell (type 32)。需要：
1. 从 table block 读取 `table.property.row_size` 和 `table.property.column_size`
2. 对每个 cell 调用 `get_block()` 或从 `children` 关系中找到 cell 的内容
3. 按行列顺序拼成 Markdown 表格

#### Step 4: 写入仓库

将转换后的 Markdown 写入仓库约定位置：

| 文档类型 | 建议路径 |
|----------|---------|
| 产品需求/PRD | `docs/specs/` 或 `docs/product-specs/` |
| 设计文档 | `docs/design-docs/` |
| 会议纪要/决策记录 | `docs/decisions/` 或 `docs/adr/` |
| 参考资料 | `docs/references/` |
| 操作手册 | `docs/guidance/` |

文件头部建议添加元信息：

文件头部使用 HTML 注释记录来源，正文以一级标题开始：

    <!-- 
      source: https://xxx.feishu.cn/docx/{DOC_ID}
      synced: 2025-07-01T10:00:00+08:00
      note: 从飞书文档同步，修改请同步回飞书或标注为仓库版本
    -->
    
    # 文档标题
    
    ...

---

## 方向二：仓库 → 飞书（发布 Markdown）

### 方案 A：使用 lark-cli（推荐）

创建飞书文档并写入 Markdown 内容：

```bash
lark-cli docs +create --title "架构文档" --markdown "$(cat ARCHITECTURE.md)"
```

### 方案 B：使用内置脚本（临时方案）

```bash
FEISHU_APP_ID=xxx FEISHU_APP_SECRET=xxx \
  python3 references/md2feishu.py doc.md --email user@company.com
```

这会：
1. 创建飞书文档
2. 将 Markdown 内容转为飞书 block 并写入（descendant API 批量写入）
3. 授权用户 `full_access` 权限
4. 将文档 ownership 转移给用户

### 编程式发布

创建文档，写入 Markdown 内容，然后授权（必须！应用创建的文档默认只有应用能看到）：

```python
from references.api import get_token, create_document, convert_markdown_to_doc, grant_permission

token = get_token()

doc_id, url = create_document(token, "文档标题")

convert_markdown_to_doc(token, doc_id, "path/to/doc.md")

grant_permission(token, doc_id, member_id="ou_xxx", perm="full_access")

print(f"文档已发布: {url}")
```

### 支持的 Markdown 元素

| 元素 | 支持 | 说明 |
|------|------|------|
| 标题 H1–H6 | ✅ | 映射到 block_type 3–8 |
| 段落 | ✅ | block_type 2 |
| 无序列表 | ✅ | block_type 12 |
| 有序列表 | ✅ | block_type 13 |
| 代码块 | ✅ | block_type 14，自动检测语言 |
| 表格 | ✅ | block_type 31，通过 descendant API |
| 图片 | ✅ | block_type 27，三步上传 |
| 引用 | ✅ | block_type 15 |
| 分割线 | ✅ | block_type 22 |
| 粗体/斜体 | ✅ | text_element_style |
| 行内代码 | ✅ | inline_code style |
| Mermaid 图 | ✅ | block_type 43，需额外权限 |

### 性能参考

| 方法 | ~220 blocks | API 调用数 |
|------|-------------|-----------|
| children API（逐条） | ~77s | 220 |
| descendant API（批量） | ~15s | ~40 |

默认使用 descendant API 批量写入，代码块自动回退到 children API。

---

## 使用场景

### 场景 A：拉取产品 PRD 到仓库

用户说："把这个飞书 PRD 拉到仓库里"

```
1. 用户提供飞书文档 URL
2. 从 URL 提取 doc_id
3. lark-cli docs +read --doc-id <DOC_ID>  （或 list_blocks 回退方案）
4. Agent 整理为 Markdown
5. 写入 docs/specs/feature-name.md（带 source 元信息）
6. 告知用户完成
```

### 场景 B：发布架构文档到飞书

用户说："把 ARCHITECTURE.md 发到飞书给 PM 看"

```
1. lark-cli docs +create --title "架构文档" --markdown "$(cat ARCHITECTURE.md)"
   （或 python3 references/md2feishu.py ARCHITECTURE.md --email pm@company.com）
2. 返回飞书文档 URL
```

### 场景 C：同步设计决策

用户说："团队在飞书上讨论了这个设计方案，帮我同步到仓库"

```
1. lark-cli docs +read --doc-id <DOC_ID>  读取飞书文档
2. Agent 整理为 Markdown
3. 写入 docs/decisions/ 或 docs/design-docs/
4. 添加 source 元信息和同步时间戳
```

### 场景 D：批量同步

当需要同步多个飞书文档时（lark-cli 逐个读取，或使用内置脚本）：

```python
doc_ids = ["doc_id_1", "doc_id_2", "doc_id_3"]
for doc_id in doc_ids:
    meta = read_document(token, doc_id)
    blocks = list_blocks(token, doc_id)
    # Agent 逐个转换并写入
```

---

## API 速查

### 认证

```python
token = get_token()  # 有效期 7200s
```

### 读取

```python
meta = read_document(token, doc_id)          # 文档元信息（标题等）
blocks = list_blocks(token, doc_id)           # 所有 block（自动分页）
block = get_block(token, doc_id, block_id)    # 单个 block 详情
```

### 写入

```python
doc_id, url = create_document(token, title)                     # 创建空文档
convert_markdown_to_doc(token, doc_id, md_path)                 # 写入 Markdown
insert_text(token, doc_id, text, block_type=2)                  # 插入文本/标题
insert_code_block(token, doc_id, code, language=49)             # 插入代码块
insert_table_with_data(token, doc_id, [["A","B"],["1","2"]])    # 插入表格
insert_image(token, doc_id, image_path)                         # 插入图片
```

### 权限

```python
grant_permission(token, doc_id, member_id="ou_xxx", perm="full_access")
```

### 删除

```python
delete_blocks(token, doc_id, parent_id, start_index, end_index)  # 删除 block
api_delete(token, f"/drive/v1/files/{doc_id}?type=docx", {})     # 删除文档
```

---

## 常见错误

| 错误码 | 原因 | 处理 |
|--------|------|------|
| `2890004` | Token 过期 | 重新 `get_token()` |
| `1770001` | Block 结构错误 | 检查 `children: []` 是否缺失；代码块不要用 descendant API |
| `1770013` | 图片上传 parent 错误 | 用 image block_id 而非 doc_id 作为 parent_node |
| `1061003` | 文档不存在 | 检查 doc_id |
| `1061004` | 无权限 | 检查应用权限配置 |
| `403` | 用户无法打开文档 | 创建后必须 `grant_permission()` |

---

## 与其他 Harness 技能的衔接

| 技能 | 衔接方式 |
|------|---------|
| `harness-analysis` | 分析报告中发现"知识散落在飞书"时，建议使用本技能同步 |
| `harness-doc-init` | 建立文档体系后，用本技能把已有飞书文档迁入 `docs/` |
| `harness-doc-gardener` | 巡检时发现"文档引用飞书链接但仓库无副本"，用本技能补充 |
| `harness-exec-plan` | 执行计划中涉及"拉取外部文档"步骤时调用 |
