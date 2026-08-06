---
name: charge-atom-progress-render
description: 计费域研发流程实时进展看板渲染原子。子流程在每个 stage 节点调用本原子写 JSON 快照,优先经 web-hosting 永久态发布(需 DATA_AGENT_TITAN_PASSPORT_ID),失败回退到 mira_runtime / CLI 上传 progress.json + progress.html 并把公网 JSON URL 回注 HTML 模板,前端 5s 轮询稳定刷新。提供 init / patch / upload / render / publish 五个子命令,与 progress-facade 契约对齐。
---

# charge-atom-progress-render

> 计费域研发流程 **实时进展看板** 渲染原子。每个子流程在每个 stage 节点调用本原子,把最新状态注入 JSON 快照;前端 HTML 模板 5s 轮询同目录 JSON,自动重渲染。无需后端,无需 WebSocket,纯静态文件 + 轮询。

## 目标

为 auto-develop SKILL(原 charge-dev-router)的每一次任务运行产出一个**可对外分享的实时看板 URL**:

- 左侧固定 **9-step 时间线**(s0~s8,状态徽章 + 进度条)
- 顶部 hero stats(耗时 / 文件数 / 回环次数 / 流水线)
- 右侧 9 个 stage 详情卡(meta 字段 + 命令 + 链接 + 阻塞节点)
- 自动每 5s 轮询 `progress.json`,无需刷新

> ⚠️ v2(2026-05 起):骨架从 7 stage 升级到 9 stage(s0 启动 / s8 报告 新增,s3 改为子流程标签,s4 改为知识检索,s5 合并 编码+push)。
> 与 auto-develop SKILL v6 的 8 节点生命周期完全对齐。

## 输入数据模型(progress.json)

```jsonc
{
  "task": {
    "title": "<PRD 标题>",
    "prd_url": "https://bytedance.larkoffice.com/wiki/...",
    "repo": "bytepay/bytepay_charge",
    "branch": "feature/charge-monthly-business-scene-code",
    "owner": "<飞书姓名>",
    "session_id": "120130628115",
    "started_at": 1778850808,
    "updated_at": 1778850808
  },
  "hero": [
    {"k":"总耗时","v":"38","unit":"min","fill":0.6,"tone":"default"},
    {"k":"变更文件","v":"4","unit":"files · +312 / -3","fill":0.25},
    {"k":"回环次数","v":"1","unit":"loops","tone":"default","fill":0.33},
    {"k":"流水线状态","v":"RUNNING","tone":"run","fill":0.57}
  ],
  "steps": [
    {"id":"s0","name":"启动","sub":"鉴权 & 看板初始化","status":"done"},
    {"id":"s1","name":"PRD 校验","sub":"飞书 wiki","status":"done"},
    {"id":"s2","name":"仓库路由","sub":"L1/L2 关键词","status":"done"},
    {"id":"s3","name":"子流程标签","sub":"complexity=mid","status":"done"},
    {"id":"s4","name":"知识检索","sub":"基线 + insearch","status":"done"},
    {"id":"s5","name":"编码 + push","sub":"Coco 25min · 编译 PASS","status":"done"},
    {"id":"s6","name":"BITS 自检流水线","sub":"loop 1/3 running","status":"run"},
    {"id":"s7","name":"MR 合入","sub":"等流水线","status":"pend"},
    {"id":"s8","name":"变更报告","sub":"待出报告","status":"pend"}
  ],
  "stages": [
    {
      "id":"s1",
      "title":"PRD 校验",
      "subtitle":"飞书 wiki/docs · lark-wiki & lark-doc SKILL",
      "status":"done",  // done | run | warn | err | pend
      "panels":[
        {
          "title":"校验结果",
          "type":"kv",
          "items":[
            {"k":"PRD 标题","v":"..."},
            {"k":"原文档","v":"<a href=...>飞书 wiki</a>","html":true}
          ]
        },
        {
          "title":"BusinessSceneCode 4 行映射",
          "type":"raw_html",
          "html":"<div>...</div>"
        }
      ]
    }
    // ... 其它 stage
  ]
}
```

## 状态语义

| status | 颜色 | 徽章 | 含义 |
|---|---|---|---|
| `done` | ok 绿 | 完成 | 已成功 |
| `run` | run 蓝 | RUNNING | 正在执行,带心跳动画 |
| `warn` | warn 黄 | 需人工 | 阻塞但非失败 |
| `err` | err 红 | 失败 | 终止,需排查 |
| `pend` | pending 灰 | 待执行 | 还没轮到 |

## CLI 用法(render.py)

```bash
# 初始化(子流程开始时调一次,生成空快照 + 自动 publish 上传两个文件 + 写 latest_urls.json)
python3 render.py init \
  --session-id 120130628115 \
  --task-title "<PRD 标题>" \
  --prd-url "<URL>" \
  --repo bytepay/bytepay_charge \
  --branch feature/xxx \
  --owner 黄哲骏 \
  --strict   # ← 推荐带上,任一上传步骤失败立即报错,避免静默吞掉错误

# 增量更新某个 stage(每个步骤完成时调一次,默认会触发完整 publish)
python3 render.py patch \
  --session-id 120130628115 \
  --stage s4 \
  --status done \
  --hero-update "变更文件=4|单元测试=PASS:ok" \
  --panel-json @stage_s4_panels.json \
  --strict

# 强制重新发布(快照已经被外部直接改写时使用)
python3 render.py upload --session-id 120130628115 --strict

# 单次完整渲染(调试用,等价于把所有 patch 一把传完)
python3 render.py render --session-id 120130628115 --data @full.json --strict

# 一站式发布:外部已用 mcp__runtime__upload_file 上传过两个文件,把 URL 注回模板 + 落 latest_urls.json
python3 render.py publish \
  --session-id 120130628115 \
  --html-url "https://.../progress.html" \
  --json-url "https://.../progress.json"
```

执行后输出 JSON:

```json
{
  "session_id":  "120130628115",
  "local_json":  "/home/mira/files/charge-progress/120130628115/progress.json",
  "local_html":  "/home/mira/files/charge-progress/120130628115/progress.html",
  "html_url":    "https://p-mira-img-sign-sgnontt.byteintl.net/.../progress.html",
  "json_url":    "https://p-mira-img-sign-sgnontt.byteintl.net/.../progress.json",
  "updated_at":  1778850808,
  "action":      "patch"
}
```

`--no-publish` 用于纯本地调试(只写文件不上传);`--strict` 用于强约束模式(任一步失败立即非 0 退出),路由层默认应当带 `--strict`。

## 调用约定(子流程接入)

auto-develop SKILL 在 9 个生命周期节点都应**调本原子打卡一次**:

| 阶段 | 时机 | 推荐 patch |
|---|---|---|
| s0 启动 | 子流程刚启动、鉴权握手前 | `init` 即写 `s0=done`(随 init 内置完成) |
| s1 PRD 校验 | 飞书 PRD 读取并校验通过 | `patch s1=done`,panels 带 PRD 标题 / URL / 摘要 |
| s2 仓库路由 | 路由结果(命中仓库列表)产出 | `patch s2=done`,panels 带命中 repo 列表 / L1/L2 关键词 |
| s3 子流程标签 | 复杂度判定 + decisions/open_questions 出来 | `patch s3=done`,panels 带 complexity / decisions |
| s4 知识检索 | 基线知识 + insearch 召回完成 | `patch s4=done`,panels 带 keys / 命中条数 / Top 3 |
| s5 编码 + push | Coco 编码完成 + 本地编译 PASS + push 成功 | `patch s5=done`(失败则 `warn`),panels 带 task_id / commit_id / changed_files |
| s6 BITS 流水线 | 流水线创建后置 `run`;终态置 `done`/`err` | `patch s6=run`(带 dev_id / pipeline_url);终态 `patch s6=done`,带 loop_count |
| s7 MR 合入 | MR 合入主干 | `patch s7=done`,panels 带 MR url + 合入时间 |
| s8 变更报告 | 最终飞书报告写完 | `patch s8=done`,panels 带 飞书报告 URL |

每次调用都会:

1. 读取 `~/files/charge-progress/<session_id>/progress.json`
2. 按 patch 内容深度合并
3. 重写本地 `progress.json` + `progress.html`
4. 输出 stdout(JSON)包含 `local_json` / `local_html` / `updated_at`
5. 默认自动上传 + 落 `latest_urls.json`(可加 `--no-publish` 跳过)

## 上传与稳定 URL(默认由原子内置完成,失败时降级到外部上传)

**默认路径(推荐)**:`init / patch / upload / render` 都会自动触发 `publish` 流程:

1. 上传 `progress.json` 到 TOS,拿到 `json_url`
2. 把 `json_url` 注入 HTML 的 `__JSON_URL__` 占位符,重写本地 `progress.html`
3. 上传新的 `progress.html`,拿到 `html_url`
4. 把 `{html_url, json_url, updated_at}` 写入 `~/files/charge-progress/<session_id>/latest_urls.json`

`render.py` 内置三条上传通道,按优先级依次尝试:

1. **web-hosting 永久态**(优先,需要环境变量触发):上游(auto-develop SKILL / charge-dev-router)若设置了 `DATA_AGENT_TITAN_PASSPORT_ID`,原子会调用 `hosting deploy` 把当前快照部署成 `https://data.bytedance.net/apps/<id>` 的内网永久 URL,产物里 `progress.json` 也得到形如 `https://tosv.byted.org/.../aeolus_web_application/<id>/progress.json` 的永久公开链接。可附加可选环境变量 `HOSTING_CLI_BIN`(指定 CLI 可执行路径)、`HOSTING_DEPLOY_NAME`(自定义页面名,默认 `charge-progress-<session_id>`)。env 缺失或 deploy 失败 → 自动回退到 2/3,**调用方零感知**。
2. **mira_runtime python SDK**(沙箱内默认可用)
3. **mira / bytedcli CLI**(任一可用即可)

任意一条命中即返回。三条都失败时:

- 默认行为:仅在 stdout 输出 `warning` 字段,不阻塞主链路
- `--strict`:任一步失败立即非 0 退出(推荐路由层默认带上)

**降级路径**:当原子内置上传都失败时,由 auto-develop SKILL 主链路接管:

1. 调用 `python3 render.py patch ... --no-publish` 只写本地
2. 路由层用 `mcp__runtime__upload_file` 工具分别上传 `progress.json` / `progress.html`
3. 调用 `python3 render.py publish --session-id <sid> --html-url <h> --json-url <j>` 把 URL 注入 HTML + 落 `latest_urls.json`
4. 因为第 3 步会重写 HTML,需要再上传一次新 HTML,再 publish 一次写回最终 URL
5. 飞书报告 / IM 通知**始终从 `latest_urls.json` 读取 `html_url`**,链接永远是当下最新的

> TOS 每次上传得到 immutable hash URL,所以"稳定"指的是
> **`latest_urls.json` 永远指向当下最新的 hash URL**。
> 已经分享过的旧链接打开仍能看到当时那一刻的快照(HTML 自带 inline data),
> 不会显示空白,语义自洽。

## 实现文件

- `SKILL.md` — 本文档
- `render.py` — 入口,init/patch/upload/render/publish 5 个子命令
- `uploader.py` — **上传通道抽象层**(2026-05-30 新增)。定义 `UploadChannel` 协议 +
  registry,内置 3 个通道(web-hosting / mira-runtime / mira-cli),按 priority
  顺序遍历;新增平台(如 Coze / 扣子 / 其他 Agent runtime)只需新增一个子类
  并 `register()`,**无需改 render.py 主流程**。本抽象层是支持 atom 跨平台运行的核心。
- `template.html` — 静态 HTML 模板,内嵌 fetch 轮询逻辑
- `default_payload.py` — 9 个 stage 的默认骨架(避免调用方每次重复传)

## 设计原则

- **静态文件 + 轮询**:不依赖任何后端服务,不需要 BITS 域内常驻;HTML 每 5s GET 同目录 `progress.json`,通过 `If-None-Match` 自动避免抖动。
- **patch 优先**:子流程不需要每次重复传完整 7-stage,只需要传"这次变了什么";原子内部维护快照。
- **失败不影响主流程**:render 失败只 warn,不阻塞 auto-develop 主链路(`--strict` 模式下硬阻断)。
- **数据可重放**:`progress.json` 本身保留全部历史字段,可作为本次任务的"事件溯源"产物。

## 与既有 SKILL 的关系

- 不替代 `report` 原子(那个是飞书文档,本看板是 web 实时分享)
- 由 `auto-develop` SKILL 在子流程注册时统一调用,子流程内部不必关心
- 输出 URL 会写入最终飞书报告的"实时进度"区块

