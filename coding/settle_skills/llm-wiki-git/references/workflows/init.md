# Init — 初始化 Git 知识库

## 目标

创建一个计费/结算 Git-only LLM Wiki 仓库骨架。init 不创建任何飞书目录、飞书文档、知识库节点或快捷方式。

## 前置条件

- 用户提供本地仓库路径，或要求初始化一个新目录。
- 不需要 `lark-cli`。
- 若目录已存在，必须先检查 `git status --short`，避免覆盖用户已有变更。

## 步骤

可直接使用本地脚本初始化：

```bash
python3 scripts/init_git.py \
  --repo <REPO_ROOT> \
  --wiki-name <WIKI_NAME> \
  --with-billing-settlement-defaults
```

旧参数 `--with-bytepay-defaults` 作为兼容 alias 保留，效果等同于 `--with-billing-settlement-defaults`。如目录不是 Git 仓库且用户明确希望初始化 Git，可追加 `--git-init`。脚本只写本地文件，不调用飞书。

### 1. 确定仓库路径

- 用户提供一个 Git 仓库本地路径，或要求初始化一个新目录。
- 若目录不存在，则创建目录并执行 `git init`。
- 若目录存在：
  - 如果不是 Git 仓库，询问是否执行 `git init`。
  - 如果是 Git 仓库，检查 `git status --short`。
  - 有未提交变更时，继续写入前说明风险；不得覆盖同名文件。

Git-only 模式入口就是仓库路径，不依赖 `~/.llm_wiki.setting.json`。

### 2. 确认 raw/ 分类

默认 raw 分类：

```text
需求文档, 技术方案, ADR决策, 系统白皮书, 数据文档, 接口文档, 值班记录, 复盘报告, 团队规约, 风险防控, 日常分享, 代码仓库
```

必须向用户展示配置摘要并等待确认：

```text
📋 初始化配置确认：
  - 知识库名称：<WIKI_NAME>
  - 仓库路径：<REPO_ROOT>
  - 存储模式：git-only
  - 默认平台系统：计费, 结算
  - raw/ 子目录：需求文档, 技术方案, ADR决策, 系统白皮书, 数据文档, 接口文档, 值班记录, 复盘报告, 团队规约, 风险防控, 日常分享, 代码仓库

请确认或修改 raw/ 子目录后继续。
```

### 3. 创建目录结构

创建：

```text
AGENTS.md
raw/<分类>/
wiki/INDEX.md
wiki/LOG.md
wiki/registry/
wiki/sources/
wiki/modules/
wiki/scenarios/
wiki/platform-capabilities/
wiki/applications/
wiki/code-components/
wiki/data/
wiki/implementations/
wiki/maps/
wiki/overviews/
wiki/comparisons/
wiki/query_feedback/
.llm-wiki/config.json   # 可选
```

Git 不跟踪空目录。对于允许暂时为空、但属于固定 schema 的目录（如 `modules/`、`scenarios/`、`platform-capabilities/`、`implementations/`、`maps/`、`overviews/`、`comparisons/`、`query_feedback/`），初始化或清理后必须保留 `.gitkeep`，避免提交后目录丢失。

### 4. 创建 REGISTRY 分册

创建：

```text
wiki/registry/REGISTRY.md
wiki/registry/REGISTRY-Sources-<分类>.md
wiki/registry/REGISTRY-Modules.md
wiki/registry/REGISTRY-Scenarios.md
wiki/registry/REGISTRY-PlatformCapabilities.md
wiki/registry/REGISTRY-Applications.md
wiki/registry/REGISTRY-CodeComponents.md
wiki/registry/REGISTRY-Data.md
wiki/registry/REGISTRY-Implementations.md
wiki/registry/REGISTRY-Maps.md
wiki/registry/REGISTRY-Overviews.md
wiki/registry/REGISTRY-Comparisons.md
wiki/registry/REGISTRY-QueryFeedback.md
```

### 5. 创建默认业务骨架

如用户采用计费 / 结算默认平台系统，可初始化：

```text
wiki/modules/charge.md                 # 计费
wiki/modules/settlement.md             # 结算
wiki/maps/module-scenario-map.md
wiki/maps/scenario-platform-map.md
wiki/maps/platform-application-map.md
wiki/maps/application-data-map.md
wiki/maps/application-business-map.md
wiki/maps/implementation-codecomponent-map.md
wiki/maps/implementation-data-map.md
wiki/maps/module-capability-map.md
wiki/maps/end-to-end-billing-settlement-flow.md
```

如果用户不希望内置业务模块，只创建空目录和空注册表。

### 6. 写入初始文件

- `AGENTS.md`：使用 [templates/init.md](../templates/init.md) 中 AGENTS 模板。
- `wiki/INDEX.md`：人工导航，不承载全量 Source。
- `wiki/LOG.md`：操作日志。
- `wiki/registry/REGISTRY.md`：登记所有 REGISTRY 分册入口和统计。
- `.llm-wiki/config.json`：可选，只保存 `wiki_name`、`storage_type: git`、`created_at`、`raw_subdirs` 等元数据。

### 7. 报告结果

输出：

- 仓库路径
- 创建/保留的目录与文件
- 是否已有 Git 仓库
- `git status --short`
- 后续建议：`import -> ingest -> review diff -> git commit`

init 完成后不自动 commit。

## 注意事项

- 不写 `~/.llm_wiki.setting.json`。
- 不调用 `lark-cli` 创建任何飞书资源。
- 不覆盖已有同名文件；如文件存在，保留并报告。
