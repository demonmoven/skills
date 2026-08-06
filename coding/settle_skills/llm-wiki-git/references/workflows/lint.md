# Lint — Git 知识库健康检查

## 目标

检查整条知识链路是否一致：

```text
raw -> Source -> REGISTRY -> INDEX
```

同时检查计费/结算业务分层链路是否闭合：

```text
Module/System -> Scenario -> PlatformCapability -> Application -> CodeComponent/Data/Implementation -> Source -> RawRef
```

## 前置条件

- 已确定 Git 仓库路径。
- 已读取 `AGENTS.md`。
- lint 默认只读；写回修复、删除、迁移、重分类必须经用户确认。

## 完整执行要求

可先运行本地结构 lint：

```bash
python3 scripts/lint_git.py --repo <REPO_ROOT> --json
```

脚本只读，不访问飞书、不修改文件；它覆盖 D1-D7 的主要结构检查和部分 D8/D9 链接检查。D9-D13 的语义重复、孤立、矛盾、过时内容、业务分层质量仍必须按本文继续读取页面正文进行 LLM 深检。

lint 必须按四层执行：

### 1. 结构层

读取 / 扫描：

- `AGENTS.md`
- `wiki/INDEX.md`
- `wiki/registry/REGISTRY.md`
- 所有 `wiki/registry/REGISTRY-*.md`
- `raw/*`
- `wiki/sources`
- `wiki/modules`
- `wiki/scenarios`
- `wiki/platform-capabilities`
- `wiki/applications`
- `wiki/code-components`
- `wiki/data`
- `wiki/implementations`
- `wiki/maps`
- `wiki/overviews`
- `wiki/comparisons`
- `wiki/query_feedback`

### 2. 注册层

- 建立实际文件集合和 REGISTRY 链接集合。
- 输出漏注册、重复注册、跨分类注册、registry 中存在但文件不存在。
- 核对 REGISTRY 根页统计和分册统计。
- QueryFeedback 必须单独对账。

### 3. 正文抓取层

逐页读取本地 Markdown 正文。D1 / D2 / D9、矛盾检测、过时内容检测、过时声明检测都不能只用目录和注册表判断。

页面较多时分批读取，并把中间结果写入临时文件，例如：

```text
/tmp/llm_wiki_lint_dirs.json
/tmp/llm_wiki_lint_registry.json
/tmp/llm_wiki_lint_bodies.json
/tmp/llm_wiki_lint_report.json
```

### 4. 内容深检层

检查：

- 空白页。
- 相对 Markdown 链接断链。
- Source frontmatter 的 `raw_ref` 是否存在。
- Source `raw_category` 是否与 raw 目录一致。
- REGISTRY 中相对链接是否存在。
- raw 引用 frontmatter 是否包含 `type/raw_category/title/source_kind`。
- 飞书来源 raw 是否包含 `lark_url` 或 `doc_id/wiki_token`。
- raw 标签是否与 wiki 分层页面互相可追踪。
- Module / Scenario / PlatformCapability / Application / Data / Implementation / Map 是否断链或缺反链。
- Source / Scenario / PlatformCapability / Application / Overview / QueryFeedback 之间事实、定义、结论是否冲突。
- raw / Source / QueryFeedback 更新后，下游页面是否过时。
- 时效性强页面是否缺少最后更新时间、适用范围、待更新 / 可能失效 / 需复核等声明。
- 金额、币种、主体、账期、费用项、费率、规则版本、结算周期、账单/结算单状态等高风险事实是否缺少 Source/raw 证据。

如需验证 raw 引用中的飞书 token/URL 可访问性，才访问飞书；此操作仍是只读。

## 检查维度

| # | 维度 | 说明 | 严重级别 |
|---|------|------|---------|
| D1 | 空白页面 | 仅有标题或元数据、缺少实质内容 | ERROR |
| D2 | 断链引用 | 相对 Markdown 链接或 raw_ref 不存在 | ERROR |
| D3 | 未注册页面 | wiki/ 中存在但未在 REGISTRY 注册 | ERROR |
| D4 | 错误 Source 分类 | Source 分类与 raw 目录不一致 | ERROR |
| D5 | Source 漏注册 | Source 未注册到对应 `REGISTRY-Sources-*` | ERROR |
| D6 | Source 错注册 / 重复注册 | Source 被注册到错误分类或多个分类 | ERROR |
| D7 | REGISTRY 根页统计不一致 | 根页统计与分册数量不一致 | WARNING |
| D8 | INDEX 角色漂移 | INDEX 错误承载全量 Source 或自称全量注册表 | WARNING |
| D9 | 重复页面 / 孤立页 / 交叉引用缺失 | 结构质量问题 | WARNING |
| D10 | 过时内容 | 下游页面未跟上 raw / Source / QueryFeedback | WARNING |
| D11 | 矛盾检测 | 页面间事实、定义、结论冲突 | ERROR / WARNING |
| D12 | 过时声明缺失 | 可能过时页面缺少显式声明 | WARNING |
| D13 | 业务分层断链 | 主干链路或 maps 断链 / 缺反链 | WARNING / ERROR |
| D14 | 高风险事实缺证 | 金额、币种、主体、账期、费率、结算周期等缺 Source/raw 证据 | ERROR / WARNING |
| D15 | CodeComponent 注册/证据缺失 | CodeComponent 未注册、缺 Application/Source/代码路径/commit 证据 | WARNING / ERROR |
| D16 | 代码仓库依赖语义错误 | Implementation/System dependency 把中间件当作下游系统依赖 | WARNING |
| D17 | 代码仓库分层断链 | Application/Implementation/Data/CodeComponent/Map 之间断链或场景粒度不一致 | WARNING / ERROR |

## 报告要求

最终报告必须包含：

- D1-D14 状态表，状态只能是 PASS / ERROR / WARNING / PARTIAL / BLOCKED。
- raw 分类统计。
- wiki/sources 实际数量 vs REGISTRY-Sources-* 分册数量。
- Source 漏注册、重复注册、跨分类注册列表。
- 业务分层 REGISTRY 与实际目录差异。
- maps 断链、缺反链、raw 标签与 wiki 页面不一致列表。
- QueryFeedback 目录与 REGISTRY-QueryFeedback 差异。
- REGISTRY 根页统计差异。
- 空白页、断链页、缺 raw 回溯字段页。
- 矛盾页面 / 冲突点 / 证据来源 / 建议以哪一侧为准。
- 过时页面 / 上游更新来源 / 是否缺少过时声明。
- 高风险事实缺证列表。
- 未完成项及原因。

## 修复规则

- 可自动修的如补注册、修统计、修明显相对链接，可在用户确认后执行。
- 删除、合并、迁移 raw 分类、重命名页面，必须单独确认。
- Source 与 raw 不一致时，以 raw 为准。
- INDEX 与 REGISTRY 不一致时，以 REGISTRY 为准。


## 代码仓库语义索引检查

当仓库包含 `raw/代码仓库` 或 `wiki/code-components` 时，lint 需要额外检查：

- `REGISTRY-CodeComponents.md` 存在且注册全部 CodeComponent。
- CodeComponent frontmatter `type=code_component`，正文链接到 Application 和 Source。
- Application 作为代码仓库入口时，必须包含 repo/path/revision 或明确标注待确认。
- Source(code_repo) 必须包含 repo URL/path 和 branch/commit/revision 证据边界。
- Implementation 的“系统依赖”不得列 Kitex、MySQL/GORM、Redis、TCC、RocketMQ、Chronos 等中间件。
- Implementation 应按场景/平台能力拆分；粗粒度大杂烩页面应作为 WARNING。
- Data 只记录具体存储证据；领域模型无存储证据不得写入 Data。
- Map 中 Application -> Implementation/Data、Implementation -> CodeComponent、Implementation -> Data 链接必须存在且注册。
