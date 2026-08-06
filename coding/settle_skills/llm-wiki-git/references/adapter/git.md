# Git Adapter — Git-only 模式参考

Git-only 模式用于把计费/结算 LLM Wiki 完全放到 Git 仓库管理：

- `raw/` 只保存原始资料引用。
- `wiki/` 全部为本地 Markdown。
- `wiki/registry/` 是机器入口。
- 飞书/Lark 仅作为 raw 原始资料读取来源。
- 禁止创建飞书目录、知识库节点、Drive shortcut、Wiki shortcut。
- 禁止把 `wiki/`、`REGISTRY`、`INDEX`、`LOG` 写入飞书。

## 仓库结构

```text
<repo>/
├── AGENTS.md
├── raw/
│   ├── 需求文档/
│   ├── 技术方案/
│   ├── ADR决策/
│   ├── 系统白皮书/
│   ├── 数据文档/
│   ├── 接口文档/
│   ├── 值班记录/
│   ├── 复盘报告/
│   ├── 团队规约/
│   ├── 风险防控/
│   └── 日常分享/
├── wiki/
│   ├── INDEX.md
│   ├── LOG.md
│   ├── registry/
│   │   ├── REGISTRY.md
│   │   ├── REGISTRY-Sources-技术方案.md
│   │   ├── REGISTRY-Modules.md
│   │   ├── REGISTRY-Scenarios.md
│   │   ├── REGISTRY-PlatformCapabilities.md
│   │   ├── REGISTRY-Applications.md
│   │   ├── REGISTRY-Data.md
│   │   ├── REGISTRY-Implementations.md
│   │   ├── REGISTRY-Maps.md
│   │   └── ...
│   ├── sources/
│   ├── modules/
│   ├── scenarios/
│   ├── platform-capabilities/
│   ├── applications/
│   ├── data/
│   ├── implementations/
│   ├── maps/
│   ├── overviews/
│   ├── comparisons/
│   └── query_feedback/
└── .llm-wiki/                    # 可选
    └── config.json               # 可选元数据，不是入口
```

## 空目录保留规则

Git 不跟踪空目录。`modules/`、`scenarios/`、`platform-capabilities/`、`implementations/`、`maps/`、`overviews/`、`comparisons/`、`query_feedback/` 等固定 schema 目录即使暂无页面，也必须保留 `.gitkeep`。清理页面时只删除业务页面，不删除目录和 `.gitkeep`。

## 仓库识别规则

一个目录可被识别为 LLM Wiki Git 仓库的最小条件：

- `raw/` 存在
- `wiki/registry/` 存在
- `wiki/INDEX.md` 或 `wiki/registry/REGISTRY.md` 至少存在一个

`.llm-wiki/config.json` 不是必须入口。缺失时从目录结构自发现：

- `raw/` 下一级目录 = raw 分类
- `wiki/registry/REGISTRY.md` = 机器入口
- `wiki/INDEX.md` = 人工入口
- `wiki/LOG.md` = 日志

## 本地脚本

Git-only adapter 提供以下本地脚本，均不写入飞书：

```bash
# 初始化仓库骨架
python3 scripts/init_git.py --repo <REPO_ROOT> --wiki-name <WIKI_NAME> --with-billing-settlement-defaults

# 旧参数兼容：等价于 --with-billing-settlement-defaults
python3 scripts/init_git.py --repo <REPO_ROOT> --wiki-name <WIKI_NAME> --with-bytepay-defaults

# 导入 raw 引用
python3 scripts/import_raw.py --repo <REPO_ROOT> --title "<标题>" --raw-category 技术方案 --source-kind lark_wiki --lark-url "<URL>" --slug <slug>

# 批量导入 Feishu raw 引用（最多 10 个）
python3 scripts/import_raw.py --repo <REPO_ROOT> --source-kind lark_auto --lark-url "<URL1>" --lark-url "<URL2>"

# 创建 Source 骨架并注册到 REGISTRY-Sources-<分类>
python3 scripts/ingest_source.py --repo <REPO_ROOT> --raw raw/技术方案/<slug>.md

# 批量创建 Source 骨架并注册到 REGISTRY-Sources-<分类>（最多 10 个）
python3 scripts/ingest_source.py --repo <REPO_ROOT> --raw raw/技术方案/<文档1>.md --raw raw/技术方案/<文档2>.md

# 本地只读结构检查
python3 scripts/lint_git.py --repo <REPO_ROOT> --json
```

脚本负责稳定的文件创建、注册和结构检查；批量模式下脚本只检测数量、路径、重复来源、raw/Source 对齐等结构冲突。LLM 负责读取原始资料、抽取事实、识别批次内/既有 wiki 语义冲突、更新业务分层页面和语义 lint；发现明显冲突时必须阻塞语义写入并让用户确认。

## raw 与 wiki 分层

- `raw/` 按材料类型组织：需求文档、技术方案、ADR决策、系统白皮书、数据文档、接口文档、值班记录、复盘报告、团队规约、风险防控、日常分享等。
- `wiki/` 按计费/结算业务架构模型组织：平台系统、场景、平台能力/规则/流程、应用、数据、技术实现、跨层映射。

默认业务拆解链路：

```text
业务架构: Module/System -> Scenario
应用架构: PlatformCapability -> Application
存储结构: Application/Implementation -> Data（仅 MySQL 表、Redis Key、索引、字段等明确存储证据）
技术架构: PlatformCapability/Application -> Implementation（领域模型与 PSM 交互链路）
证据链路: Implementation -> Source -> RawRef
```

默认且仅允许的顶层平台系统：

- 计费：`wiki/modules/charge.md`
- 结算：`wiki/modules/settlement.md`

## 可选 `.llm-wiki/config.json`

仅用于保存仓库内元数据，不替代目录扫描。

```json
{
  "wiki_name": "billing-settlement-llm-wiki",
  "storage_type": "git",
  "raw_subdirs": ["需求文档", "技术方案", "ADR决策", "系统白皮书", "数据文档", "接口文档", "值班记录", "复盘报告", "团队规约", "风险防控", "日常分享"],
  "created_at": "YYYY-MM-DD HH:mm"
}
```

## raw 引用文件

raw 文件不是原文副本，而是原文索引。

```markdown
---
type: raw_ref
title: "抖音支付计收费-白皮书"
raw_category: "系统白皮书"
source_kind: "lark_doc"
lark_url: "https://..."
doc_id: "xxx"
wiki_token: "yyy"
obj_type: "docx"
modules:
  - 计费
architecture_layers:
  - 业务架构
  - 应用架构
  - 数据架构
business_scenarios:
  - 费用项计费
  - 账单生成
platform_capabilities:
  - 计费规则
  - 账单生成
systems:
  - caijing.bytepay.charge
imported_at: "YYYY-MM-DD HH:mm"
updated_at: "YYYY-MM-DD HH:mm"
---

# 抖音支付计收费-白皮书

- 原始飞书地址：<https://...>
- 备注：本文件只保存 raw 引用，不保存原文。
```

## Source 页面

```markdown
---
type: source
title: "Source：抖音支付计收费-白皮书"
raw_ref: "../../../raw/系统白皮书/抖音支付计收费-白皮书.md"
raw_category: "系统白皮书"
modules:
  - 计费
architecture_layers:
  - 业务架构
business_scenarios:
  - 费用项计费
platform_capabilities:
  - 计费规则
systems:
  - caijing.bytepay.charge
created_at: "YYYY-MM-DD HH:mm"
updated_at: "YYYY-MM-DD HH:mm"
---

# Source：抖音支付计收费-白皮书

## 元数据

- 原始来源：[抖音支付计收费-白皮书](../../../raw/系统白皮书/抖音支付计收费-白皮书.md)
- Raw 分类：`raw/系统白皮书`

## 摘要

...
```

## AGENTS 默认协作约定

若业务没有更具体说明，默认包含：

1. 业务定制输出风格、面向特定团队的提示词和报告格式，优先放在包装本 skill 的外层业务 skill；知识库内 AGENTS 只放知识库结构、分类、事实优先级和质量规则。
2. 不读取、不要求 `~/.llm_wiki.setting.json`。
3. 禁止创建飞书目录、飞书 wiki 节点、Drive shortcut、Wiki shortcut。
4. 禁止把 wiki 层写入飞书。
5. 金额、币种、主体、账期、费用项、费率、规则版本、结算周期、账单/结算单状态等事实必须回溯 Source/raw。

## 推荐 maps 页面

- `wiki/maps/module-scenario-map.md`
- `wiki/maps/scenario-platform-map.md`
- `wiki/maps/platform-application-map.md`
- `wiki/maps/application-data-map.md`
- `wiki/maps/module-capability-map.md`
- `wiki/maps/end-to-end-billing-settlement-flow.md`

## 常用检查

```bash
git status --short
rg -n "raw_category|raw_ref|modules|architecture_layers|business_scenarios|platform_capabilities|systems" wiki raw
find wiki/registry -maxdepth 1 -type f -name 'REGISTRY*.md'
find wiki/modules wiki/scenarios wiki/platform-capabilities wiki/applications wiki/data wiki/implementations wiki/maps -type f -name '*.md'
```
