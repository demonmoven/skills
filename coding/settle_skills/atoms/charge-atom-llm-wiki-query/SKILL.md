---
name: charge-atom-llm-wiki-query
description: 计费/结算 LLM Wiki 知识检索原子。knowledge-facade 的唯一实现:在 sandbox 内落地知识库代码仓库 bytepay/charge_settle_llm_wiki(repo_id=1069114,默认分支 feat_ai),驱动 settle_skill 的 llm-wiki-git skill 执行 query 操作,产出综合知识上下文。仅供 auto-develop 路由层和子流程调用。
---

# charge-atom-llm-wiki-query

## 定位

`knowledge-facade` 的唯一底层原子。封装 settle_skill 中 **llm-wiki-git** skill 的 `query` 操作,面向计费、结算两个平台型系统提供业务知识检索。

- 知识库**事实源**是 Git 仓库 `bytepay/charge_settle_llm_wiki`(repo_id=`1069114`,默认分支 `feat_ai`),而非飞书/cjadk。
- 飞书仅作为 raw 原始资料读取来源:query 在本地 wiki 摘要不足时,可回查 raw 引用对应的飞书原文。
- 本原子不写回知识库、不 ingest、不 push;只做**只读检索**。

## 调用契约

### 输入(facade 透传)
- `keywords`:PRD 关键词,拼成自然语言 query 的主语
- `requirement_brief`:PRD 摘要,作为 query 的上下文补充
- `scope`:计费域 `charge` / `settle` / `billing`,用于约束检索主题

facade 把三者拼成一句**自包含的自然语言 query**(不要堆砌关键词),例如:
> "在 charge 域下,<requirement_brief>;围绕 <keywords> 涉及哪些平台能力、应用职责、数据表与计价/结算口径?"

### 输出(stdout / facade outputs)
```json
{
  "status": "ok",
  "knowledge_md": "## 综合回答\n...(含内联引用 [[title]](path))",
  "sources": [{"title": "Source：计费白皮书", "path": "wiki/sources/系统白皮书/计费白皮书.md"}],
  "raw_refs": [{"title": "计费白皮书", "ref": "https://bytedance.larkoffice.com/wiki/xxxx"}],
  "used_keywords": ["计费", "费率", "..."],
  "repo_revision": "feat_ai@<commit>",
  "warnings": []
}
```

### status 语义(本原子只如实上报;由 knowledge-facade 按 stop_and_ask 决定是否停流程)
- `ok`:正常命中并综合
- `empty`:仓库落地成功但 query 无命中 → `knowledge_md` 给出"知识库未覆盖"说明
- `unauth`:鉴权失败(JWT/飞书未授权)→ 提示登录命令,knowledge_md 为空
- `error`:其他异常 → warnings 记录原因,knowledge_md 为空

> facade `fallback_strategy = stop_and_ask`:status ∈ {empty, unauth, error} 时停止整条流程等用户,不 fallback 到其他原子。

## 执行流程

### 0. 鉴权(复用 bytedcli ByteCloud SSO → Codebase JWT,禁用 PAT)
```bash
JWT=$(bytedcli --json auth get-codebase-jwt-token | jq -r '.data.jwt')
[ -z "$JWT" -o "$JWT" = "null" ] && { echo '{"status":"unauth"}'; exit 0; }
```
未授权时输出 `status=unauth`,并提示用户:
`! bytedcli auth login --session --feishu`;必要时 `! bytedcli feishu login --force-oauth`。
(facade 据此 stop_and_ask 停流程等用户登录)

### 1. 落地知识库仓库(双路径,按序尝试,任一成功即用)

**A) git clone(首选,整库)** —— `llm-wiki-git query` 需要 `rg`/`find` 遍历 `wiki/` 与 `raw/`,必须有完整工作树:
```bash
REPO_DIR="$HOME/files/charge-knowledge/charge_settle_llm_wiki"
mkdir -p "$(dirname "$REPO_DIR")"
if [ -d "$REPO_DIR/.git" ]; then
  git -C "$REPO_DIR" fetch --depth 1 origin feat_ai && \
  git -C "$REPO_DIR" reset --hard origin/feat_ai
else
  git clone --depth 1 --branch feat_ai \
    "https://oauth2:${JWT}@code.byted.org/bytepay/charge_settle_llm_wiki.git" "$REPO_DIR"
fi
REV="feat_ai@$(git -C "$REPO_DIR" rev-parse --short HEAD 2>/dev/null)"
```
> clone URL 必须用内联 `oauth2:${JWT}@`;裸 clone 与 JWT 走 http.extraHeader Bearer 均会 403/128。

**B) bytedcli 按需取文件(兜底,clone 不可用时;仅适合点读 REGISTRY / 指定页面)**:
```bash
bytedcli --json codebase repo file <path> -R bytepay/charge_settle_llm_wiki --revision feat_ai \
  | jq -r '.data.file.Content' | base64 -d
```
按需先取 `wiki/registry/REGISTRY.md` 与相关 `REGISTRY-*.md`,再按命中拉取具体页面;无法整库遍历时,query 退化为"按 REGISTRY 索引点读"。

### 2. 驱动 llm-wiki-git query
以 `REPO_DIR` 作为知识库入口,按 llm-wiki-git 的 query workflow 执行(LLM 驱动,无独立脚本):
1. 读 `wiki/registry/REGISTRY.md` + `REGISTRY-*.md` 定位候选本地 Markdown
2. 读命中的 `wiki/` 页面(modules/scenarios/platform-capabilities/applications/data/implementations/maps/sources)
3. 本地摘要不足以完整回答时,沿 Source → raw 引用回查 raw 原始资料;raw 为飞书来源时读飞书原文
4. 综合成 `knowledge_md`,关键事实句末内联引用 `[[title]](path)` 或 raw 链接

> 执行 query 前必须读取知识库仓库内 `AGENTS.md`;若其要求先查特定 skill 路由,优先遵循。

### 3. 计费/结算证据纪律
- 金额、币种、主体、账期、费用项、费率、计价口径、规则版本、结算周期、结算单/账单/出款状态等高风险事实,必须能回溯到 Source/raw,并在 `sources`/`raw_refs` 中给出引用。
- 不得用通用支付/清结算经验补全口径;证据不足或来源冲突时在 `knowledge_md` 中显式标"待确认 / UNKNOWN / 来源冲突",不静默合并。

## 下游消费

`tech-solution-facade`(default 实现)消费本 facade 全部 outputs 作为 `knowledge_context`,主要取 `knowledge_md` + `sources` 注入技术方案上下文。由 knowledge-facade 的 `stop_and_ask` 保证:进入下游时 `status=ok` 且 `knowledge_md` 非空(否则上游已停)。

## 前置依赖
- `bytedcli`(已登录 ByteCloud SSO,可取 Codebase JWT)
- `git`、`jq`、`base64`、`rg`/`find`(sandbox 默认具备)
- settle_skill 的 `llm-wiki-git` skill(query workflow 由其定义)
