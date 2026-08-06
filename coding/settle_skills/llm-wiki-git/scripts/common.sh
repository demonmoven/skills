#!/bin/bash
# common.sh — init.sh 共用函数库
# 使用方式：source "$(dirname "$0")/common.sh"

WIKI_SUBDIRS="sources modules scenarios platform-capabilities applications data implementations maps comparisons overviews query_feedback registry"

# ---------- 模板函数 ----------

agents_markdown() {
  cat <<'MD'
# AGENTS.md — LLM Wiki Git-only 协作规则

本文档只记录这套 LLM Wiki 的结构、规则和例外，不记录具体业务事实。具体业务知识应沉淀到 Source / Module / Scenario / PlatformCapability / Application / Data / Implementation / Map / Overview / Comparison / Query Feedback。

当前知识库采用 Git-only 模式：raw 只保存原始资料引用，wiki 全部为本地 Markdown，INDEX 给人看，REGISTRY 给机器读。飞书只作为 raw 原始资料读取来源，不作为 wiki 存储后端。

## 核心规则

- INDEX 是人工导航页，不承担全量注册；REGISTRY 及其分册才是机器侧全量入口。
- Source 分类唯一以 raw 为准：每个 Source 的分类必须与原始素材所在的 `raw/分类` 一致。
- Source 页面必须保留原始文档名：标题统一为 `Source：<原始文档名>`，文件名默认使用 `<原始文档名>.md`（仅清理文件系统非法字符）。
- 所有事实性内容必须能回溯到 raw 或 Source；综合判断必须和原始事实区分开。
- 默认增量维护；只有页面严重漂移、结构损坏或重建 REGISTRY 分册时才整页覆盖。
- 先定 raw，再谈 wiki：import、ingest、registry、lint 的上游依据都是 `raw/`，不是 INDEX。
- wiki 是知识合集的索引层 / 关键摘要层，不是 raw 原文对应的飞书文档镜像；只沉淀可查询、可复用、可回溯的关键摘要、结构化关系、证据边界、阅读路径和不确定性。

## Git 模式业务分层默认规则

- `raw/` 按原始材料类型组织，不按业务架构层拆 raw。
- `wiki/` 按业务架构模型组织：平台系统 → 业务场景 → 平台能力 → 应用 → 数据 / 技术实现。
- 默认链路：`Module/System -> Scenario -> PlatformCapability -> Application -> Data/Implementation -> Source -> RawRef`。
- 默认平台系统仅包括：计费、结算；可以更新既有两类，但除非用户明确指定新增 module / 新增平台系统，不得创建第三类 Module。
- 旧专题、旧概念、旧实体层已废弃：长期查询入口归入 `scenarios/`，能力/规则/流程归入 `platform-capabilities/`，PSM/职责对象归入 `applications/`。

## 主干目录职责

- `modules/`：顶层平台系统，仅计费、结算两类；可更新，不默认新增。
- `scenarios/`：计费 / 结算领域的业务场景和长期查询入口，例如计费规则生效、账单生成、账单重算、结算单生成、结算出款、对账差异、差错补偿等。
- `platform-capabilities/`：计费 / 结算平台对外或业务可复用的能力、规则和流程，例如费用项定义、计费规则、账单生成、结算单生成、结算周期、差错处理等。
- `applications/`：PSM 维度的应用职责、边界、上下游、接口、任务和消息；除非用户明确指定新增 PSM，默认不新增 Application。
- `data/`：数据库表结构、索引信息、Redis Key、字段等具体存储结构；只有明确涉及现有表/索引修正或新增表/Redis Key 时才变动。
- `implementations/`：领域模型和系统 PSM 之间的交互链路细节；领域模型需要关联到 `data/` 的具体表/Key/索引，缺少存储证据时标注“待确认”。
- `maps/`：只维护跨层映射。
- `overviews/`、`comparisons/` 是辅助知识层，服务主干，不替代主干；主要由 query 阶段在用户确认后产生，ingest 阶段默认不主动创建。

## 新增与更新阈值

- 业务分层页面是知识合集索引，不是 Source 引用计数器。新 Source 命中已有 Scenario / PlatformCapability / Application / Data / Implementation / Module 时，先判断是否改变该页面的定义、边界、契约、规则、职责、证据边界、长期阅读路径或关键映射；没有改变时，只在 Source、必要的上层页面或 Map 中建立引用，不更新该页面正文。
- 越底层、越稳定的知识索引，更新阈值越高。若新 Source 只是使用或引用已有 Application / Data / Implementation，不改变职责边界、输入输出、适用范围、核心机制、存储结构或关键规则，则不得更新稳定页面正文。
- 新增 Scenario 或 PlatformCapability 前，必须按第一性原理向用户说明并征询确认：为什么现有目录不足以承载、解决什么长期复用问题、覆盖哪些来源、证据是否充分、有哪些不确定性。
- 若 Source 没有明确指定新增 PSM，不新增 Application；若没有明确指定新增平台系统，不新增 Module。

## 命名与格式规则

- 飞书来源导入时，必须优先用 `lark-cli` 只读获取标题和正文摘要，根据内容匹配当前仓库已有 raw 分类；无法读取、内容不足或无法稳定命中分类时，再咨询用户。
- 飞书 raw 文件名默认必须与飞书文档标题一致，仅清理文件系统非法字符；“白皮书”类文档默认归入 `raw/系统白皮书`。
- Source 标题统一为 `Source：<原始文档名>`；Source 文件默认位于 `wiki/sources/<分类>/<原始文档名>.md`。
- PSM Application 文件名使用 PSM 下划线格式，例如 `caijing.bytepay.charge` → `caijing_bytepay_charge.md`。
- Application 页面必须保留 `## 关联数据`、`## 关联平台能力`、`## 证据来源` 三个章节；已废弃层的历史引用应忽略或移除。
- Data 页面应在前置 `## 来源` 章节统一声明表结构来源；金额、币种、主体、账期、费用项、费率、规则版本、结算周期、结算单状态等高风险字段必须能回溯 Source/raw；字段/索引表默认继承该来源，不逐行重复来源，只有字段级来源差异、冲突或待确认时才在备注中说明。

## 默认流程规则

- 默认流程是 `import -> ingest`。除非用户明确说明“只导入不摄入 / 暂不摄入”，否则 import 成功后默认继续 ingest。
- import 的关键结果是把资料放到正确的 `raw/分类`。
- ingest 的关键结果是保证 `raw -> Source -> REGISTRY` 对齐，并在确有长期复用价值和证据变化时维护业务分层页面。
- query 默认读取顺序是：`REGISTRY -> 对应 REGISTRY 分册 -> 具体页面`；必要时回查 raw 原文对应的飞书文档。

## Query 与纠错回流规则

- query 阶段默认只读，不应因为“顺手修一下”直接写回。
- query 中涉及任何页面变更时，原则上都应先获得用户明确确认。
- 唯一例外：用户明确指出现有页面内容有误并要求修正，可以直接修改对应页面。
- Query Feedback 单独维护在 `wiki/query_feedback/`，并注册到 `REGISTRY-QueryFeedback`；按天维度维护。

## 日常分享规则

- 用户只给一句话或几句话、但没有正式文档/附件/链接时，默认先落到 `raw/日常分享`，并按天维度维护。
- 即使内容涉及风险、防控、值班、技术方案等主题，也不能仅按关键词猜到其他 raw 目录。
MD
}
log_init_markdown() {
  cat <<'MD'
## 操作日志

最新操作在最下方。
MD
}

# ---------- INDEX 全量内容 ----------
# 依赖环境变量（由 run_init 在调用前设置）：
#   WIKI_NAME, STORAGE_TYPE, SPACE_ID（wiki 模式）
#   ROOT_TOKEN, RAW_TOKEN, RAW_TOKEN_TABLE, WIKI_TOKEN
#   TOKEN_sources/modules/scenarios/platform-capabilities/applications/data/implementations/maps/comparisons/overviews/query_feedback/registry
#   AGENTS_DOC_ID, LOG_DOC_ID, TODAY

build_index_markdown() {
  local space_id_value="${SPACE_ID:--}"
  cat <<MD
<callout emoji="📚" background-color="light-blue">

LLM Wiki 索引 — 所有页面的注册表和导航入口。

</callout>

## 目录配置

> Token 列：云盘模式存 folder_token，知识库模式存 node_token。

| 目录 | Token |
|------|-------|
| root (${WIKI_NAME}) | ${ROOT_TOKEN} |
| raw | ${RAW_TOKEN} |
${RAW_TOKEN_TABLE}| wiki | ${WIKI_TOKEN} |
| wiki/sources | ${TOKEN_sources} |
| wiki/modules | ${TOKEN_modules} |
| wiki/scenarios | ${TOKEN_scenarios} |
| wiki/platform-capabilities | ${TOKEN_platform_capabilities} |
| wiki/applications | ${TOKEN_applications} |
| wiki/data | ${TOKEN_data} |
| wiki/implementations | ${TOKEN_implementations} |
| wiki/maps | ${TOKEN_maps} |
| wiki/comparisons | ${TOKEN_comparisons} |
| wiki/overviews | ${TOKEN_overviews} |
| wiki/query_feedback | ${TOKEN_query_feedback} |
| wiki/registry | ${TOKEN_registry} |

## Wiki 配置

| 键 | 值 |
|---|---|
| wiki_name | ${WIKI_NAME} |
| storage_type | ${STORAGE_TYPE} |
| space_id | ${space_id_value} |
| 创建时间 | ${TODAY} |
| 最后更新 | ${TODAY} |
| 页面总数 | 0 |
| AGENTS doc_id | ${AGENTS_DOC_ID} |
| LOG doc_id | ${LOG_DOC_ID} |

> - \`storage_type\`：\`drive\`（云盘，默认）或 \`wiki\`（知识库）
> - \`space_id\`：仅知识库模式需要，云盘模式填 \`-\`

## 页面注册表

| 标题 | 类型 | Doc ID | Doc | 目录 | 最后更新 | 关联 |
|------|------|--------|-----|------|---------|------|
MD
}

# ---------- LOG 初始化条目 ----------

build_log_entry() {
  local raw_count
  local wiki_count
  raw_count=$(echo "$RAW_SUBDIRS" | wc -w | tr -d ' ')
  wiki_count=$(echo "$WIKI_SUBDIRS" | wc -w | tr -d ' ')
  cat <<MD

### ${TODAY} INIT

- 操作: 初始化知识库
- 存储模式: ${STORAGE_TYPE}
- raw/ 子目录: ${RAW_SUBDIRS}
- 创建文件夹: ${raw_count} raw 子目录 + ${wiki_count} wiki 子目录
MD
}

# ---------- 主初始化流程 ----------
# 要求调用前已定义：
#   _create_dir "$name" "$parent"   → 输出 token 字符串
#   _create_doc "$title" "$parent_token" "$markdown" → 输出 JSON（含 .data.doc_id 和 .data.doc_url）
#   STORAGE_TYPE, WIKI_NAME, PARENT_TOKEN, RAW_SUBDIRS
#   SPACE_ID（wiki 模式必填）

run_init() {
  # --- [1/9] 根目录 ---
  echo "=== [1/9] 创建根目录: $WIKI_NAME ==="
  ROOT_TOKEN=$(_create_dir "$WIKI_NAME" "$PARENT_TOKEN")
  echo "ROOT_TOKEN=$ROOT_TOKEN"

  # --- [2/9] raw/ 和 wiki/ ---
  echo "=== [2/9] 创建 raw/ 和 wiki/ ==="
  RAW_TOKEN=$(_create_dir "raw" "$ROOT_TOKEN")
  WIKI_TOKEN=$(_create_dir "wiki" "$ROOT_TOKEN")
  echo "RAW_TOKEN=$RAW_TOKEN  WIKI_TOKEN=$WIKI_TOKEN"

  # --- [3/9] raw/ 子目录 ---
  echo "=== [3/9] 创建 raw/ 子目录 ==="
  RAW_TOKEN_TABLE=""
  for subdir in $RAW_SUBDIRS; do
    echo "  创建 raw/$subdir ..."
    token=$(_create_dir "$subdir" "$RAW_TOKEN")
    echo "  raw/$subdir => $token"
    RAW_TOKEN_TABLE+="| raw/${subdir} | ${token} |"$'\n'
  done

  # --- [4/9] wiki/ 子目录 ---
  echo "=== [4/9] 创建 wiki/ 子目录 ==="
  for subdir in $WIKI_SUBDIRS; do
    echo "  创建 wiki/$subdir ..."
    token=$(_create_dir "$subdir" "$WIKI_TOKEN")
    echo "  wiki/$subdir => $token"
    var_name="TOKEN_${subdir//-/_}"
    declare "$var_name=$token"
  done

  # --- [5/9] AGENTS.md ---
  echo "=== [5/9] 创建 AGENTS.md ==="
  local agents_result
  agents_result=$(_create_doc "AGENTS" "$ROOT_TOKEN" "$(agents_markdown)")
  read -r AGENTS_DOC_ID AGENTS_DOC_URL < <(
    echo "$agents_result" | jq -r '[.data.doc_id, (.data.doc_url // "")] | @tsv'
  )
  echo "AGENTS_DOC_ID=$AGENTS_DOC_ID"

  # --- [6/9] INDEX ---
  echo "=== [6/9] 创建 INDEX ==="
  local index_result
  index_result=$(_create_doc "INDEX" "$WIKI_TOKEN" " ")
  read -r INDEX_DOC_ID INDEX_DOC_URL < <(
    echo "$index_result" | jq -r '[.data.doc_id, (.data.doc_url // "")] | @tsv'
  )
  echo "INDEX_DOC_ID=$INDEX_DOC_ID"

  # --- [7/9] LOG ---
  echo "=== [7/9] 创建 LOG ==="
  TODAY=$(date "+%Y-%m-%d %H:%M")
  local log_result
  log_result=$(_create_doc "LOG" "$WIKI_TOKEN" "$(log_init_markdown)")
  read -r LOG_DOC_ID LOG_DOC_URL < <(
    echo "$log_result" | jq -r '[.data.doc_id, (.data.doc_url // "")] | @tsv'
  )
  echo "LOG_DOC_ID=$LOG_DOC_ID"

  # --- [8/9] 更新 INDEX ---
  echo "=== [8/9] 更新 INDEX（填入所有 token）==="
  lark-cli docs +update --as user --doc "$INDEX_DOC_ID" \
    --mode overwrite --markdown "$(build_index_markdown)"

  # --- [9/9] 追加 LOG 条目 ---
  echo "=== [9/9] 追加 LOG 初始化条目 ==="
  lark-cli docs +update --as user --doc "$LOG_DOC_ID" \
    --mode append --markdown "$(build_log_entry)"

  # --- 保存配置 + 输出摘要 ---
  echo "=== 初始化完成 ==="
  echo ""
  echo "INIT_RESULT_JSON:"
  SPACE_ID="${SPACE_ID:-}"
  ROOT_URL="${ROOT_URL:-}"
  export WIKI_NAME STORAGE_TYPE SPACE_ID ROOT_TOKEN ROOT_URL \
         INDEX_DOC_ID INDEX_DOC_URL AGENTS_DOC_ID AGENTS_DOC_URL \
         LOG_DOC_ID LOG_DOC_URL RAW_SUBDIRS TODAY
  python3 "$SCRIPT_DIR/save_config.py"
}
