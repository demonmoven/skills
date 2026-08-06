---
name: auto-develop
description: 自动研发 AutoDevelop — 端到端研发流程入口路由,支持多团队复用(配置化接入)。用户提供 PRD/BRD 飞书文档链接 **或** Meego 工单链接(双向解析,任选其一),需要分析需要做哪些系统变更并自动执行端到端研发流程时使用。各团队在自己 skill 仓库根目录提供 config/team_profile.yaml 即可接入(财经计费域 = bytepay/settle_skill@feature/ai_native 是首批接入实例)。触发词:auto-develop、AutoDevelop、自动研发、计费需求分析、charge 研发流程、settle_skill 流程、PRD 自动执行、需求路由。即使用户没有明确说"路由",只要给出飞书 PRD 或 Meego 链接并希望"分析+执行变更"也应触发本 SKILL。
---

# AutoDevelop 研发流程入口路由(v7)

## 定位
本 SKILL 是**多团队复用**的端到端研发流程入口。架构:**SKILL 公共 + 团队配置化**。
财经计费域(charge / settle)是首批接入团队,本 SKILL.md 的所有团队相关示例均以该域为例。
其它团队接入只需:① fork 一份自己的 skill 仓库;② 在仓库根创建 `config/team_profile.yaml`(schema 见步骤 0.2);③ 在调用 auto-develop 前注入 `AUTO_DEVELOP_PROFILE_PATH` / `AUTO_DEVELOP_BOOTSTRAP_REPO` / `AUTO_DEVELOP_BOOTSTRAP_BRANCH` 三个环境变量。

本 SKILL 的核心责任:
0. **平台兼容自适应**(2026-06-01 起强制):路由启动时探测当前运行平台,若**非 Mira 平台**(例如纯 sandbox 容器、Codex、Claude Code 本地、CI worker 等)→ 自动把"实时进展看板(progress-facade)"全链路降级为 `skip`,**不阻塞**主流程,也不视为路由失败。看板相关的"必须 emit""硬阻断"约束只在 Mira 平台下生效。详见「步骤 0.4:平台能力探测」与「横切关注点:实时进展看板」。
1. 加载用户输入(飞书 PRD URL **或** Meego 工单 URL,任选其一,**步骤 1 自动双向解析**)
2. 加载团队 `config/team_profile.yaml`(步骤 0.2,fail-fast)
3. **每次必须**同步最新的子流程仓库(参数从 profile 读)
4. **每次必须**读取最新的领域知识库
5. 对 PRD 进行变更分类,产出变更项清单
6. **基于子流程仓库的 `config/repo_routing.yaml` 路由出目标仓库列表**(可多命中,多仓库并发执行)
7. 路由到对应子流程并编排执行(每个仓库独立派发一条编排链路)
   - **simple 复杂度**:编码节点走 Coco sandbox 自动编码(老路径,保持现状)
   - **complex 复杂度**(2026-06-01 起强制默认):编码节点切换为本地交付(`local_handoff`)— 路由层只产出技术方案 + handoff 交付包,**暂停**等用户在本地 IDE / Cursor / Codex / Claude Code 完成编码 + push 后回填 `commit_id` 即可恢复(直接进 MR + 流水线节点)。可通过 `team_profile.complex_coding_strategy: auto_coco` 显式覆盖回老路径
8. 创建 BITS 自检流水线时,**如果**步骤 1 解析到了 Meego URL 则带上 `--meego <url|id>`;**未解析到时省略该参数,不阻塞流水线创建**(`bytedcli bits develop create` 允许 `--meego` 缺省)
9. 汇总结果写回飞书报告文档(无飞书 Bot 授权时降级为 Markdown 文件 + 后续提示)

## 通用能力来源:bytedcli + Coco
计费域研发流程的通用能力**统一通过 Mira SKILL 完成**,不再使用本地自建原子。

| 能力 | 走哪个 SKILL |
|---|---|
| **代码生成 / 编辑 / 审查 / 修复** | **`bytedance-coco`** |
| 代码 Q&A、读代码、查 diff、查 MR 元数据 | `bytedcli` / `bytedance-codebase` 或 `bytedance-bitsai` |
| 提交 / 创建 MR / 发布任务 | `bytedcli` / `bytedance-codebase`、`bytedance-scm`、`bytedance-devflow` |
| 知识检索 / 读飞书文档 | `bytedcli` / `bytedance-insearch`、`bytedance-feishu`、`bytedance-cloud-docs` |
| 打包构建(SCM) | `bytedcli` / `bytedance-scm`(`scm repo build`) |
| 部署 | `bytedcli` / `bytedance-tce`、`bytedance-faas`、`bytedance-goofy-deploy` |
| 自动化测试 | `bytedcli` / `bytedance-aitest`、`bytedance-api-test`、`bytedance-test-plan` |
| 写飞书文档报告 | `bytedcli` / `bytedance-feishu` |

**编码任务硬规则**:
- **complexity = simple**(由步骤 4.7 facade 判定):任何涉及代码生成、编辑、审查、修复的子流程步骤,**必须**通过 `bytedance-coco`,而不是手写 LLM 编辑或 codebase write API。Coco 任务结束后,改动**只在 Coco sandbox 内**;后续接 Round-2 task 让 Coco 执行 `git push`,再回到本地 `bytedcli codebase mr create`。
- **complexity = complex**(由步骤 4.7 facade 判定):**默认走本地交付**(handoff)模式 — 路由层**不**触发 Coco 编码,只产出「需求 + 仓库 + 分支建议 + 技术方案 + 变更要点」的交付包,**暂停**等用户在本地 IDE / Cursor / Codex / Claude Code 等环境完成编码并 push 后,回复 commit_id / 分支即可恢复路由(直接进 MR + 流水线节点)。详见「步骤 4.7.2 Case B」与新增章节「公共能力·complex 路径本地交付协议(local-dev-handoff)」。⚠️ 复杂需求 Coco 原子产出物历史返工率高(架构理解 / 跨模块影响判断弱),本地 IDE + 真人复合方式更可靠;此规则可由用户在 `team_profile.yaml` 中显式覆盖(参见下方 `complex_coding_strategy` 字段)。
  - ⛔ **绝对硬规则(2026-07-10 起强制)**:complex 路径下,handoff 交付包里的「技术方案」字段**必须**来自 `prd-tech-design` 编排 Agent 产出的飞书技术方案文档(`tech_design_feishu_url`,通过 `lark-doc` 拉取全文),**禁止** Agent 基于 PRD / 复杂度判定理由 / 仓库路由结果 / 知识基线 / 任何上下文自行手写或脑补任何形式的「技术方案」/「技术拆解」/「实现思路」/「需要做的事」/「Step 1 / Step 2 …」/「预分析」/「初步设计」。Agent 在 complex 路径下对方案内容的**唯一**合法动作 = 通过 Skill 工具调用 `prd-tech-design` 编排 Agent + 拿到 `tech_design_feishu_url` + 用 `lark-doc` 拉全文。任何形式的"我先帮你梳理一下""我先草拟一版""根据 PRD 我理解需要做的事"都属于路由实现错误,产物视为污染必须重做。详见 4.7.2 Step B-1 的「硬纪律」与「prd-tech-design 不可用 = 路由停机,严禁脑补降级」段落。

### Coco / Codebase 认证方式(纯 SSO,不使用 PAT)

bytedcli 0.57+ 提供从 ByteCloud Auth(SSO) → Codebase JWT 的转换通道,**全程无需 PAT**。
路由层每次进入子流程编排前,**必须**完成下面的 JWT 注入握手:

```bash
# 1) 确认 ByteCloud Auth 已就绪(SSO 已登录)
bytedcli auth status   # 期待 ByteCloud Auth: ready

# 2) 用 ByteCloud JWT 换取 Codebase JWT
bytedcli --json auth get-codebase-jwt-token   # 取 .data.jwt

# 3) 注入到 codebase 配置
bytedcli codebase auth config-auth --jwt-token <上一步取到的 jwt>
```

完成后,所有 `codebase` / `coco` 子命令均可正常工作。

**鉴权失败处理**:
- `bytedcli auth status` 报 ByteCloud Auth not ready → 触发 SSO 登录(按以下顺序):
  1. **先尝试 session 复用**:`bytedcli --json auth login --begin --session`。若返回 `reused_existing_session=true` **且** `bytedcli auth status` 立刻通过,跳过扫码进入下一步;否则进入 2。
  2. **device_code 扫码登录**:`bytedcli --json auth login --begin`。从 stdout 同时拿到:
     - `verification_uri_complete`(扫码链接)
     - `qr_image_path`(本地 PNG 路径,如 `/tmp/bytedcli/auth-login-XXX/auth-login.png`)
     - `complete_token`(供后续 `auth login --complete <token>` 使用)
  3. **必须把二维码图片直接渲染给用户**(不只是给本地路径!):
     - 用 `mcp__runtime__upload_file` 上传 `qr_image_path` 拿到公网 URL
     - 在阻断回复中以 `![扫码登录](<url>)` 的 Markdown 图片语法**内联渲染**,Mira 飞书私聊会直接展示图片
     - 同时附上 `verification_uri_complete` 文字链接作为兜底(电脑端用户可直接点开)
     - **禁止**只输出 `qr_image_path` 这种 sandbox 本地路径(用户拿不到)
  4. 用户回复恢复触发词(「继续」「已扫码」「ok」等)后,执行 `bytedcli auth login --complete <complete_token>` 完成握手。
  5. **最多重试 1 次**(challenge 过期 → 重新发一次 begin),仍失败则报错给用户,**不绕过去用 PAT**。
- `get-codebase-jwt-token` 失败但 ByteCloud Auth ready → 报告问题给用户,**不在路由内自动绕过去用 PAT**
- ⛔ 路由层**不允许**在任何场景主动创建/读取/落地 Codebase PAT

**鉴权阻断回复模板**(必须严格按此结构,二维码图片不可省略):

```markdown
## 🚫 路由阻断:bytedcli ByteCloud Auth 鉴权未完成

**任务**:<PRD 标题>
**阻断节点**:步骤 0(bytedcli 鉴权握手)

### 请扫码完成 SSO 登录

![扫码登录](<上传后的二维码公网 URL>)     ← 必填,内联渲染

- **扫码链接(电脑端可点)**:<verification_uri_complete>
- **复杂 token**:<complete_token>(我会在你回复后自动执行 complete)

扫完二维码,**回复「继续」/「已扫码」/「ok」**,我会立刻完成握手并跑完整路由。

> challenge 约 10 分钟过期,请尽快扫码;过期后我会自动重发一次。
```

### Coco 调用规范

调用 `bytedcli coco task send` 时:
1. **`--model-name` 默认不传**(由 Coco 选择默认模型)
2. **`--agent-name` 默认走 `sandbox`**(批量/无人值守任务比 copilot 稳定)
3. 仅当用户在 PRD 或对话中明确指定模型/agent 时,才传 `--model-name <用户指定>` 或 `--agent-name copilot`
4. 任务发送成功后**必须**用混合监控策略:
   - 后台 `coco task subscribe --task-id <id>` 写入 ndjson(SSE 易断)
   - 主线程每 60s 调 `coco task get --task-id <id>` 轮询直到 `Status=completed && (Final 标记或最终 full_message 出现)`
   - SSE 中断时不重启 subscribe 也行,以 `task get` + `task events` 为准

#### Coco 任务消息模板(必须按此六段结构组装)

```
# 任务:<PRD 标题>

## PRD
- 标题:<...>
- 内容:<一两句概述>
- PRD 链接:<飞书链接>

## 仓库
- 仓库:<group/repo> (repo_id=<id>)
- 基线分支:<source>(默认业务仓库 master;Agent 不询问用户)
- 新分支:feature/auto_dev-<slug>(由 Agent 基于 PRD 自行 kebab-case 命名)
- 目标分支:master(2026-05-19 起强制,不再使用 feature/ai_native)

## 实现要求
1. <定位已有同类实现>
2. <核心算法/单位说明>
3. <注册到工厂/SPI/路由表>
4. <类型/code 命名规范>
5. <单测覆盖项>
6. <影响面/不动哪些>

## 约束
- 命名严格使用 <ClassName 或 函数/类型名>(按目标语言风格,Java→PascalCase 类、Go→Pascal 导出/小驼峰未导出 + snake_case 文件、Python→snake_case)
- 不部署 prod
- MR 标题前缀 `[<前缀>]`
- 编码完成即 commit + push 到远端 `<branch>`,作为本编码节点的完成标志(详见下方「编码节点完成判定」)
- ⛔ **不要**在 sandbox 内跑**全量 UT / 全量集成测试**(`mvn test` / `go test ./...` / `pytest` 全跑都不要发起);UT 由后续 BITS / SCM 流水线节点统一执行
- ✅ **必须**在 sandbox 内跑**仓库级编译 + 静态检查作为 push 的前置门禁**(2026-05-20 起强制,详见下方「push 前置编译门禁」)

## 期望产出
- 新文件 + 修改的注册/工厂类 + 新增 UT + changed_files 清单
- push 完成后的远端 commit_id(40 位 sha)与远端仓库地址
- **本地编译 PASS 证据**(编译命令 + 退出码 + 简短 stdout 末尾)
```

> **编码节点完成判定(强制)**:Coco Round-1 任务的"完成"以「**本地编译 PASS** && **远端 `<branch>` 有新 commit(commit_id 与下发任务前的分支 head 不同)**」为联合判据。**不**以 mvn test / go test / pytest 是否 PASS 为判据(全量 UT 留给流水线),**不**以 `coco task get` 返回的 Status 为唯一判据(Status 有时滞后)。路由层在轮询时,除了看 task Status,还必须并行轮询 `bytedcli codebase ...` 取分支 head sha,与启动前记录的 head sha 比对,一旦不一致即视为编码节点完成,可立刻进入 MR + 流水线节点。

> **编码节点的职责边界(2026-05-20 起强制更新)**:编码节点做五件事 — **「基于业务仓库 master 拉新分支 `feature/auto_dev-<slug>` → 读代码 → 改代码 → 本地 commit → 本地编译门禁(必须 PASS) → push 到远端新分支」**。
> ⛔ 编码节点**绝对禁止**做以下事情(违反即视为路由实现错误):
> - **创建 MR / Pull Request**(包括 `bytedcli codebase mr create`、`git push -o merge_request.create` 等任何形式)
> - 跑**全量** UT / 集成测试(`mvn test` / `go test ./...` / `pytest` 全跑;改动文件所在包内的轻量 `go test ./pkg/...` / `mvn -pl <module> test` 允许,但失败不阻塞 push,只在 commit message 注明)
> - 触发 BITS / SCM 流水线
> 理由:① 全量自检拉长编码节点运行时长(常见 1~3h);② 全量 UT 失败本质是流水线节点的职责;③ Coco 跑全量自检时改其它模块测试以"通过"会污染本次提交(已实测发生过);④ MR 描述需要 changed_files / commit_id / 影响面说明,只能在 push 完成后由 MR 节点基于编码节点的最终输出生成。
>
> ✅ 编码节点**必须**做的"门禁级"校验(2026-05-20 起强制):
> - **仓库级编译**(`go build ./...` / `mvn -DskipTests compile` / `tsc --noEmit` / `pnpm build`)→ **退出码必须为 0**,否则禁止 push
> - **基础静态检查**(Go: `go vet ./...`;Java: 编译期 javac 报错;TS: `tsc --noEmit`)→ 退出码必须为 0
> - 编译失败 → Coco 在 sandbox 内**自主修复并重编**,直到 PASS 才 push;**禁止**带着编译错误 push
>
> **MR 创建职责归属**:Coco 编码节点 push 完成后,路由层在 sandbox **外部**用本地 `bytedcli codebase mr create` 创建 MR(详见下文「MR 创建样板」)。这是**独立的 MR 节点**,不是编码节点的一部分。

#### push 前置编译门禁(2026-05-20 起强制)

**Coco 编码任务的 prompt 必须包含以下段落**(按目标仓库 `build_system` 字段渲染对应命令):

```
## Push 前置编译门禁(强制,不通过禁止 push)

完成代码改动 + commit 之后,push 之前,**必须**在仓库根目录执行下列命令,**所有命令退出码必须为 0**:

[Go 仓库]
  go mod tidy
  go build ./...        # 整仓编译
  go vet ./...          # 静态检查

[Java/Maven 仓库]
  mvn -B -DskipTests -T 1C compile      # 整仓编译跳测
  # 不跑 mvn test(留给流水线)

[TypeScript / Node 仓库]
  pnpm install --frozen-lockfile  (或 npm ci)
  pnpm build  (或 tsc --noEmit)

[Python 仓库]
  python -m compileall -q .            # 字节码编译
  ruff check . || flake8 .             # 任一可用即可

任意命令非 0 退出 → **不要 push**;在 sandbox 内继续修复,直至全部 PASS。
push 成功后,在最终消息中输出:
  - commit_id(40 位 sha)
  - 编译命令清单 + 各自退出码 = 0 的证据(每条命令最后 5 行 stdout 即可)
```

> 路由层在 Coco prompt 末尾**强制注入**该段落;不允许由 Agent 临时拼接。
> 路由层判定编码节点完成时,除轮询远端 head sha 外,还要在 Coco subscribe 流里搜索关键字 `BUILD SUCCESS` / `^ok\s+\S+` / 编译命令的退出码标记;若 Coco 跳过编译就 push,**视为编码节点失败**,自动回环重发(规则参见下方「失败自动回环」)。

#### Coco 编码任务的 push 段落(必须并入消息末尾)

Coco sandbox **不预装 bytedcli**,无法自创 MR;但 push 必须由 Coco 自己完成(编码节点 = 编码 + push)。在上面六段 prompt 之后,**必须**附加下面这段固定指令:

```
## Push 与完成判定(必带)

完成代码编写后,在 sandbox 内执行:
1. cd <workspace>
2. git status
3. 若有未 commit 改动:
   git add -A
   git commit -m "[<前缀>] <PRD 标题>"
4. git push -u origin <branch>
5. 在最终消息中**必须**返回:
   - 远端 commit_id(40 位 sha,`git rev-parse HEAD`)
   - 远端分支:origin/<branch>
   - changed_files 完整清单(`git diff --name-status <base_sha>..HEAD`)

⛔ 不要尝试编译 / 不要跑测试 / 不要创建 MR。本节点完成的唯一判据是"push 成功 + 返回 commit_id"。
```

push 完成后,路由层从最终 full_message 中正则抽取 commit_id(`[0-9a-f]{40}`),并以「远端分支 head sha 与发起前不同」作为编码节点完成的最终判据,然后进入 MR 创建步骤。

#### MR 创建样板(本地 bytedcli)

```bash
BODY=$(cat <body.md>)
bytedcli --json codebase mr create \
  --repo-id <id> \
  --head <branch> \
  --base <target,默认 master> \
  --title "[<前缀>] <PRD 标题>" \
  --body "$BODY"
# 解析 .data.MergeRequest.Number → URL: https://code.byted.org/<group>/<repo>/merge_requests/<Number>
```

MR body 模板必须包含:背景(PRD 链接)、实现要点、changed_files、自检结论、影响面、commit、来源(Coco Task ID)。

#### SCM 打包样板(MR 创建后)

渠道离线引擎使用 SCM 打包(不使用 BITS 流水线):

```bash
# 1. 触发 offline 构建
bytedcli --json scm repo build caijing/bytepay/charge_offline_engine \
  --branch <branch> \
  --type offline \
  --message "[<前缀>] <PRD 标题>"

# 2. 查最新版本号+状态(轮询直到 success/failed,每 30s 一次,最多 10min)
bytedcli --json scm repo version list caijing/bytepay/charge_offline_engine

# 3. 构建失败时拉日志
bytedcli --json scm repo build-log caijing/bytepay/charge_offline_engine <version>
```

- SCM repo name = `caijing/bytepay/charge_offline_engine`(scm repo-id = 466820)
- `--type offline` 固定用于离线引擎
- 构建失败 → **不重试**,记录到报告并标注"需人工排查"
- 构建超时(>10min) → 停止轮询,报告中附版本号让用户自查

## 计费域专属原子(仍保留)
以下 SKILL 是计费域独有,bytedcli 无对应:
- **`settle_skill 仓库同步与提交`**:已**内联**进本 SKILL,见下方 `## 公共能力·settle_skill 仓库同步与提交(repo-sync)` 章节;路由层**直接**按章节里的 bash 命令执行,不再调外部 `charge-atom-repo-sync` SKILL
- `charge-atom-knowledge-loader`:**强基线知识加载器**。读取 settle_skill `config/knowledge_sources.yaml` 清单,按 loader 字段调对应拉取器,把文档内容缓存到本地。仓库只持有"文档地址 + 元信息",**内容每次运行实时拉取**
- ~~`charge-subflow-fallback`:仓库不可用时的兜底流程~~(**已弃用**:工具链 / 仓库同步失败一律硬阻断,不再走兜底;此 SKILL 保留仅供存量会话回查,不应再被新会话调用)

## 公共能力·settle_skill 仓库同步与提交(repo-sync)

> 本章节是**自包含**的仓库同步/提交能力,原 `charge-atom-repo-sync` SKILL 的全部能力已内联到此。**不要**再调外部 SKILL,直接按本章节执行 bash。
>
> **设计理由**:
> - Mira 沙箱跨会话不持久化,git clone 每次都要重新认证,带 token 不安全
> - bytedcli 在沙箱启动时已用当前飞书账号 SSO 认证完毕,直接调 `codebase repo file` 即可,**无需任何 PAT**
> - settle_skill 仓库子流程定义不大(几个 yaml/md 文件),按需拉比 clone 整仓库更省
> - 写入侧:Codebase(原 GitLab)的 `/api/v5/repos/{id}/commits` 接口已于 2025-12-31 下线、bytedcli 0.60 没有 `codebase repo file write/update`,因此 push 走 **Coco sandbox**(checkout → 写文件 → commit → push),认证复用 bytedcli 的 SSO → Codebase JWT 链路

### ⚠️ 跨平台兼容:`bytedcli codebase repo file` 必须走"鲁棒拉取" wrapper(2026-05-30 起强制)

**踩坑事件**:在非 Mira 平台上跑本 SKILL 时,框架会强制给 bytedcli 注入 `--json` 全局开关(为了拿结构化错误码),导致 `codebase repo file` 返回 base64 包装的元信息 envelope:

```json
{"status":"success","data":{"file":{"Path":"...","Type":"normal_file","Encoding":"base64","Content":"<base64 of raw bytes>"},"meta":{...},"path":"...","repository":{...},"revision":"..."},"error":null,"context":{...}}
```

如果直接 `> manifest.yaml`,落盘的就是这段 JSON,**不是** YAML 原文 → 后续 yaml.safe_load / frontmatter 解析全炸,SKILL 无法启动。

**更隐蔽的反向陷阱**:**不带** `--json` 时,bytedcli **也不会**直接吐 raw 字节流,而是先打一个 5–10 行的人类可读 ASCII 表头(`File / Field Value / Repo ... / Revision ... / Path ... / Type / Encoding / Size`)再吐内容。这段表头同样会污染落盘文件,只是污染方式不同(YAML 解析时第一行 `File` 被当作普通字段名,可能侥幸不立刻报错但语义全错)。**因此"去掉 `--json` 直接拉"也不行。**

**根因要点**(必须同时考虑,因此**禁止**用单一命令直拉):
1. `--json` 可能由调用方框架强注入,SKILL 内部不可控
2. JSON envelope 里 `Content` / `Encoding` / `Path` 是**首字母大写**(Go struct 序列化风格),许多通用 fetcher 按 `content` / `encoding` 小写 key 取值会取到空
3. 不带 `--json` 时 bytedcli 会前置打印 ASCII 表头,**不能**当 raw 用
4. 部分平台版本 bytedcli 即使没 `--json` 也会把 stderr 混进 stdout

**统一解法**:**所有**拉取仓库文件的地方,**必须**通过下方 `fetch_repo_file` shell 函数,函数**强制走 `--json` + base64 解码**这一条最可控路径,不依赖 CLI 的默认输出格式。**禁止**任何步骤直接写 `bytedcli codebase repo file ... > foo`(无论是否带 `--json`)。

```bash
# === fetch_repo_file: 跨平台鲁棒文件拉取(把内容打到 stdout) ===
# 用法: fetch_repo_file <repo> <revision> <path_in_repo>
# 成功: 文件原文写到 stdout,exit 0
# 失败: 把诊断信息打到 stderr,exit !=0
#
# 实现策略: 强制走 --json + base64 解码,这是唯一不依赖 CLI 默认输出格式的可控路径。
# 不带 --json 的 CLI 输出会被前置 ASCII 表头污染,绝不能当 raw 用。
fetch_repo_file() {
  local repo="$1" rev="$2" path="$3"
  command -v jq >/dev/null 2>&1 || { echo "fetch_repo_file: jq is required" >&2; return 127; }
  command -v base64 >/dev/null 2>&1 || { echo "fetch_repo_file: base64 is required" >&2; return 127; }

  local out status err b64 enc
  out=$(bytedcli --json codebase repo file "$path" -R "$repo" --revision "$rev" 2>/dev/null) || {
    echo "fetch_repo_file: bytedcli call failed for $repo@$rev:$path" >&2
    return 1
  }
  [ -n "$out" ] || { echo "fetch_repo_file: empty response for $repo@$rev:$path" >&2; return 1; }

  status=$(printf '%s' "$out" | jq -r '.status // ""')
  if [ "$status" != "success" ]; then
    err=$(printf '%s' "$out" | jq -r '.error // .message // "unknown"')
    echo "fetch_repo_file: status=$status error=$err for $repo@$rev:$path" >&2
    return 1
  fi

  # 兼容大小写 key(Go: Content/Encoding;某些版本: content/encoding)
  enc=$(printf '%s' "$out" | jq -r '.data.file.Encoding // .data.file.encoding // ""')
  b64=$(printf '%s' "$out" | jq -r '.data.file.Content // .data.file.content // ""')
  if [ -z "$b64" ]; then
    echo "fetch_repo_file: empty Content field for $repo@$rev:$path" >&2
    return 1
  fi

  case "$(printf '%s' "$enc" | tr A-Z a-z)" in
    base64)
      printf '%s' "$b64" | base64 -d
      ;;
    ""|text|utf-8|utf8|raw|plain)
      # 极少数小文件可能不走 base64;此时 Content 就是原文
      printf '%s' "$b64"
      ;;
    *)
      echo "fetch_repo_file: unknown encoding=$enc for $repo@$rev:$path" >&2
      return 1
      ;;
  esac
}

# === assert_file_sane: 拉完后做一次基础合理性校验,避免 envelope 落盘 ===
# 用法: assert_file_sane <local_path> <expect_kind: yaml|markdown|text>
assert_file_sane() {
  local f="$1" kind="$2"
  [ -s "$f" ] || { echo "assert_file_sane: empty file $f" >&2; return 1; }
  # 嗅探 JSON envelope 残留(出现 "data":{"file":{"Content"... 几乎肯定是没解开的包装)
  if head -c 256 "$f" | grep -qE '"data"[[:space:]]*:[[:space:]]*\{[^}]*"file"[[:space:]]*:[[:space:]]*\{[^}]*"Content"'; then
    echo "assert_file_sane: $f looks like a JSON envelope (codebase repo file --json output not decoded)" >&2
    return 1
  fi
  # 嗅探 bytedcli CLI 的 ASCII 表头残留(不带 --json 时会冒出来,首行常为空)
  if head -5 "$f" | grep -qE '^File[[:space:]]*$|^[[:space:]]*Field[[:space:]]+Value[[:space:]]*$|^[[:space:]]*Repo[[:space:]]+\S+/\S+'; then
    echo "assert_file_sane: $f contains bytedcli ASCII header (raw CLI output not stripped)" >&2
    return 1
  fi
  case "$kind" in
    yaml)
      # YAML 不应以 '{' 开头(那是 JSON)
      first=$(head -c 1 "$f")
      if [ "$first" = "{" ]; then
        echo "assert_file_sane: $f starts with '{' but expected YAML" >&2
        return 1
      fi
      ;;
    markdown)
      # SKILL.md / subflow.md 必须以 '---' 开头(YAML frontmatter)
      head -1 "$f" | grep -q '^---$' || {
        echo "assert_file_sane: $f missing YAML frontmatter (no leading '---')" >&2
        return 1
      }
      ;;
  esac
  return 0
}
```

**约束(强制)**:
- 步骤 2 / 步骤 3 / 步骤 4 / 后续步骤 0.2-postsync / 一切拉远端文件的地方,**必须**调 `fetch_repo_file` + `assert_file_sane`,**禁止**直接 `bytedcli codebase repo file ... >`
- 拉完 SKILL.md / subflow.md / *.yaml 后**必须**立刻 `assert_file_sane`,失败 → 同步整体硬阻断,把 stderr 抛给用户;不要等到下游 yaml.safe_load 才崩

### 模式 1:拉取(sync)

**参数来源**:`repo` / `branch` 取自路由上下文 `team_profile.skill_repo` / `team_profile.skill_branch`(本会话 = `bytepay/settle_skill` / `feature/ai_native`,作为示例)。

**步骤 0:加载 team profile**

由于本步本身要拉 `config/team_profile.yaml`,存在"先有鸡还是先有蛋"问题。具体处理:
- 若 `AUTO_DEVELOP_PROFILE_PATH` 环境变量已显式指向某文件 → 步骤 0.2-prefetch 已加载,本步直接用 profile 里的 `skill_repo` / `skill_branch`
- 否则 → 用环境变量 `AUTO_DEVELOP_BOOTSTRAP_REPO` / `AUTO_DEVELOP_BOOTSTRAP_BRANCH`(由调用方注入,如 `bytepay/settle_skill` / `feature/ai_native`)做"引导拉取"先把 yaml 拉下来(**走 `fetch_repo_file`,见上方 wrapper**),再执行步骤 0.2-postsync 加载;之后所有同步动作必须改用 profile 里的值

**步骤 1:确定 cache 目录**
- 在 sandbox 临时目录建 `~/charge-cache/<skill_repo_short>/`(本会话内有效即可,`<skill_repo_short>` 取自 `team_profile.skill_repo` 末段如 `settle_skill`)
- 子目录:`subflows/<name>/`、`atoms/<name>/`、`config/`

**步骤 2:拉取 manifest.yaml**

```bash
SK_REPO="bytepay/settle_skill"
SK_BR="feature/ai_native"
SK_CACHE="$HOME/charge-cache/settle_skill"
mkdir -p "$SK_CACHE"

# 必须走 fetch_repo_file wrapper(本章节顶部定义),不能直接 bytedcli ... >
fetch_repo_file "$SK_REPO" "$SK_BR" "manifest.yaml" > "$SK_CACHE/manifest.yaml"
assert_file_sane "$SK_CACHE/manifest.yaml" yaml || exit 1
```

- 失败 → 整体硬阻断,把 `bytedcli` 真实日志 + `assert_file_sane` 的 stderr 一并抛给用户(独立的"普通回复消息",**禁止**调 lark-im / im-chat-manager)
- 成功 → 解析 yaml,提取 `atomic_skills` 和 `subflows` 两个 section

**步骤 3:按 manifest 列出的 subflow 拉子流程文件**

对 `manifest.subflows[]` 中每个 `name`:

```bash
mkdir -p "$SK_CACHE/subflows/$name"
# 必拉:subflow.md(含 YAML frontmatter 声明 facade_implementations)
fetch_repo_file "$SK_REPO" "$SK_BR" "subflows/$name/subflow.md" \
  > "$SK_CACHE/subflows/$name/subflow.md"
assert_file_sane "$SK_CACHE/subflows/$name/subflow.md" markdown || {
  echo "subflow_md_invalid: $name"; continue;
}
```

- `subflow.md` 拉不到或 `assert_file_sane` 失败 → 该 subflow 标记 `errors: ["subflow_md_missing"]` 或 `["subflow_md_invalid"]`,继续其它
- 路由层解析 frontmatter 中 `facade_implementations` 字段作为声明式调度依据

**步骤 4:同步 atoms / config 目录(按需)**

后续步骤(知识库加载、看板、路由等)用到的原子和配置文件,**必须**走 `fetch_repo_file`(yaml 文件追加 `assert_file_sane <path> yaml`,markdown 追加 `assert_file_sane <path> markdown`),逐个拉到 `$SK_CACHE/atoms/<name>/` 或 `$SK_CACHE/config/`。**禁止**直接 `bytedcli codebase repo file ... >`,即使该平台你测过当前可用——一次 bytedcli 升级或框架包一层 wrapper 就会复现 base64 envelope 落盘事故。

**步骤 5:返回结构化结果(供路由上下文使用)**

```json
{
  "repo": "bytepay/settle_skill",
  "branch": "feature/ai_native",
  "cache_path": "<sandbox>/charge-cache/settle_skill",
  "fetched_at": "<ISO8601>",
  "manifest": { "atomic_skills": [...], "subflows": [...] },
  "subflow_files": {
    "<name>": { "subflow_md": "...", "steps_yaml": "...", "mode": "yaml | natural" }
  },
  "errors": []
}
```

### 模式 2:推送(push,通过 Coco sandbox)

**适用场景**:把 sandbox 本地修改后的 settle_skill 文件,直接 commit + push 到远端分支。**不创建 MR、不创建新分支**。

**输入参数**:
| 参数 | 必填 | 说明 |
|---|---|---|
| `files` | 是 | 提交文件列表,每项含 `path`(仓库相对路径)和 `content`(字符串) |
| `commit_message` | 是 | commit message |
| `branch` | 否 | 目标分支,默认 `feature/ai_native` |
| `repo` | 否 | 目标仓库,默认 `bytepay/settle_skill`(repo_id=987788) |

**步骤 1:组装 Coco task message**

```markdown
# 任务:提交文件到 {branch} 分支

## 仓库
- 仓库:{repo} (repo_id={repo_id})
- 基线分支:{branch}
- 目标:直接 commit + push 到 {branch}(不创建新分支,不创建 MR)

## 实现要求
请将以下文件的内容**完整覆写**到仓库对应路径,然后 commit + push。

### 文件 N: {path}
请用以下内容**完整替换** `{path}`:
```
{file_content}
```
(重复 N 个文件)

## 约束
- **不要**创建新分支,直接 push 到 {branch}
- **不要**创建 MR
- **不要**修改任何其他文件
- commit message: `{commit_message}`

## Push 与完成判定(必带)
完成文件写入后,在 sandbox 内执行:
1. cd <workspace>
2. git add {file_paths_space_separated}
3. git commit -m "{commit_message}"
4. git push origin {branch}
5. 最终消息返回:远端 commit_id(`git rev-parse HEAD`) + changed_files 清单

⛔ 不要尝试编译 / 不要跑测试 / 不要创建 MR。
```

**步骤 2:发送 Coco task**

```bash
bytedcli --json coco task send \
  --message "$MSG" \
  --repo-id "$REPO_ID" \
  --branch "$BRANCH" \
  --agent-name sandbox
# 记录返回的 Task.Id
```

**步骤 3:轮询等待完成**

```bash
# 每 30s 一次,最多 5min
bytedcli --json coco task get --task-id "$TASK_ID"
# 等 Status = "completed" | "failed"
```

**步骤 4:验证提交**

```bash
bytedcli --json codebase commit list -R "$REPO" --revision "$BRANCH"
# 检查最新 commit message 是否匹配
```

- 匹配 → 提取 commit_id,返回成功
- 不匹配 / 状态 failed → 返回 `errors: ["push_failed: ..."]`

### 异常分支(sync + push 共用)

| 情况 | 处理 |
|---|---|
| `bytedcli` 不在 PATH | 用 `NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest` 兜底 |
| 认证失效 / 401 | 返回 `auth_required`,提示用户先在终端跑 `bytedcli codebase auth` |
| `manifest.yaml` 拉不到(sync) | 整体失败,硬阻断 |
| 单个 subflow 文件拉失败(sync) | 标记该 subflow 出错,其它继续 |
| Coco task failed(push) | 返回 errors + task_id 供排查 |
| Coco task 超时 >5min(push) | 返回 timeout error |
| 网络瞬断 | 该操作重试 ≤ 3 次 |

### 关键约束

- **不在日志中打印任何 token**(bytedcli 自管认证,我们也不接触 token)
- **cache 路径在 sandbox 临时区**,每次会话重新拉(配合路由层"每次必须同步"的硬约束)
- **不使用 git clone / git pull**:读取走 bytedcli HTTP API,写入走 Coco sandbox
- **不写文件到用户家目录之外**
- **push 模式默认不创建 MR**:直接 push 到目标分支;如需 MR 由路由层额外调 `bytedcli codebase mr create`
- **push 模式不创建新分支**:直接 push 到已有分支(默认 `feature/ai_native`)

### 已验证用例

| 日期 | commit_message | commit_id | 结果 |
|---|---|---|---|
| 2026-05-19 | `[auto-develop] R1+R2: 决策点分流策略 + ArchMate 恢复轮询协议` | `6a9d0a844bc7e0c5227124e7b661dd9d7e57d3de` | 成功 |
| 2026-05-23 | `[auto-develop] 修正 ArchMate API header 名(X-Jwt-Token, 非 Bearer),恢复自动轮询` | (待提交) | 修复(verified conv_id=1779515423618) |
| 2026-07-10 | `[auto-develop] complex 技术方案原子由 ArchMate(tech-solution-gen)切换为 prd-tech-design 编排 Agent` | (待提交) | 迁移 |

## settle_skill 仓库内的计费域专用 SKILL
通过上面 `repo-sync` 同步后,以下计费域 SKILL 在本地可直接执行:
- `bytedance-jwt`(取计费域 JWT)
- `contract-query`(查端外商户合约)
- `ftools` / `ftools-aime`(查计费结算单据)
- `log-check`(检查计费日志)
- `streamlog-trace-query`(按 logid 查日志)

## 横切关注点:实时进展看板(progress-facade)

> **2026-06-01 起强制**:看板能力仅在 **Mira 平台**下启用。其它平台(Codex、Claude Code、纯 sandbox 容器、CI worker 等)**自动跳过**整条看板链路,**不阻塞**主流程,也**不视为路由失败**。本节描述的"必须 emit""硬阻断"约束**仅当 `dashboard_enabled = true` 时生效**;`dashboard_enabled = false` 时全部按"看板节点 = skip"处理。
>
> **判定方式**(见步骤 0.4):满足以下任一条件即视为 Mira 平台 → `dashboard_enabled = true`:
> - 环境变量 `MIRA_PLATFORM=1` 或 `MIRA_SESSION_ID` 非空
> - `~/files/` 目录可写且能访问 `mcp__runtime__upload_file` 工具
> - 团队 `team_profile.yaml` 显式 `dashboard.force_enable: true`
>
> 否则 `dashboard_enabled = false`,整个 progress-facade 链路按 skip 处理,主流程继续完整执行(步骤 1 → 7 全跑完),最终回复中**省略** `dashboard_url` 字段并在末尾追加一行 `> 当前平台未启用进度看板(non-Mira platform)`。
>
> **配置覆盖**:团队可在 `team_profile.yaml` 中设 `dashboard.force_enable: true` 强制启用(适用于自建 Mira 兼容平台),或 `dashboard.force_disable: true` 强制关闭(便于在 Mira 平台调试主链路时跳过看板)。

**强制规则(仅 dashboard_enabled = true 时生效)**:本路由的**每一次**执行,无论成功 / 中断 / 失败,**最终回复必须包含 `dashboard_url`**(从 `~/files/charge-progress/<session_id>/latest_urls.json.html_url` 读取),并放在飞书报告链接之**前**置顶输出。

**强制规则(看板地址即时回复,仅 dashboard_enabled = true 时生效)**:`dashboard_url` 一旦在步骤 0.5 拿到,**必须立刻**作为一条独立的"普通回复消息"输出给用户(Mira 在飞书侧会把回复呈现到当前用户的私聊里,等同于即时推送)。**不允许**等到下一次用户停顿点或最终回复才出。这是为了避免「Agent 在后台一直跑,用户却没拿到看板地址」的情况。详见步骤 0.5 第 5 条。

> 历史方案曾尝试用 `lark-im` / `im-chat-manager` skill 直发私聊,但 Mira 沙箱对 `open_id` 做 PII 脱敏(占位符 `[ph_USER_ACCOUNT_x_ph]`)而 server 端强校验 `ou_` 前缀,union_id / user_id 均被拒,该链路在当前环境**确认不可用**。**禁止**再调 `lark-im` / `im-chat-manager` 推送看板/报告地址,统一改走"普通回复消息"。

### 看板生命周期(8 个 patch 节点)

> **全局短路约定**:本节列出的所有 `progress-facade.init / patch / publish` 调用,**仅在 `ctx["dashboard_enabled"] = true` 时执行**;`dashboard_enabled = false` 时**全部 no-op**(不调原子、不写文件、不发独立消息),主流程跳过看板直接进入下一步。后续步骤 1 ~ 7 中所有 `progress-facade.patch(...)` 段落同样遵循此短路约定,**无需**在每处重复声明。

| 时机 | progress-facade 操作 | hero / panel 关键字段 |
|---|---|---|
| 步骤 0 完成(鉴权握手 OK) | `init(task_meta)` + 写 `latest_urls.json` | task_title / prd_url / owner / 启动时间 |
| 步骤 1 完成(PRD 校验通过) | `patch s1=done` | PRD 标题 / 链接 / 需求摘要 |
| 步骤 4.5 完成(仓库路由) | `patch s2=done`(per repo) | L1/L2 关键词 / build_system / repo_path |
| 步骤 5 完成(子流程标签) | `patch s3=done` | complexity / decisions / open_questions |
| 步骤 6 知识检索完 | `patch s4=done` | 关键词 / 命中条数 / Top 3 |
| 步骤 6 编码 + push 完 | `patch s5=done` + hero "变更文件=N\|UT=PASS" | strategy_used / coco_task_id / changed_files |
| 步骤 6 流水线终态 | `run` → `done`/`warn`/`err` + hero "流水线状态=PASS:ok:1" | dev_id / pipeline_url / build_log_excerpt |
| 步骤 7 报告生成完 | `patch s7=done`(MR) + `patch s8=done`(report) | MR URL / report URL |

> 每次 patch 后,`progress-facade` 会重新上传 HTML 并更新 `latest_urls.json`;路由层**只从 `latest_urls.json` 读 `html_url`**,不要自己拼 URL。

#### web-hosting 依赖自愈(2026-06-01 起强制)

实时看板推荐优先走 `web-hosting` 永久态发布。进入看板初始化前,必须确保以下两项自愈逻辑已执行;任一步失败不阻断主流程,但必须记录 warning,随后交由 `charge-atom-progress-render` 自动降级到 mira_runtime / CLI 上传通道:

1. **安装 web-hosting skill**:若 `/data/plugins/custom/skills/web-hosting/hosting-cli.tgz` 不存在,执行一次:
   ```bash
   skills add skills.byted.org/default/public --skill web-hosting --version 1.0.3
   ```
   - 若命令不存在或安装失败,记录 `web_hosting_skill_install_failed`,不要重试超过 1 次。
   - 安装成功后,`charge-atom-progress-render` 会通过 bundle 内的 `hosting-cli.tgz` 自动生成 npx wrapper。
2. **注入 Titan Passport ID**:若 `DATA_AGENT_TITAN_PASSPORT_ID` 未设置,优先执行 `get-titan-id` 获取并 `export DATA_AGENT_TITAN_PASSPORT_ID=<解析结果>`。
   - 支持 `get-titan-id` 直接输出 ID、JSON envelope 或提示文本中夹带 ID 的形态。
   - 若 `get-titan-id` 不存在、返回非 0 或解析失败,记录 `titan_passport_id_unavailable`,不要把空值或整段日志写入环境变量。
   - `atoms/charge-atom-progress-render/uploader.py` 内已内置同等兜底:即使路由层漏设,原子也会在判断 web-hosting 可用性时自动尝试获取并注入。

### 看板原子调用(默认路径,**必须优先尝试**)

`progress-facade` 的默认实现是 **`charge-atom-progress-render`** 原子,**已 commit 到 `bytepay/settle_skill : feature/ai_native : atoms/charge-atom-progress-render/`**,通过步骤 2 的内置 `repo-sync` 章节(模式 1)同步到 sandbox 后即可使用。所有 init / patch / publish 操作必须**优先**通过该原子完成,在 sandbox 内**直接 shell 执行**,不要走 Mira Skill 工具的间接层。

#### 推荐路径:原子内置上传(零自由度,最稳)

如果 sandbox 里 `mira_runtime` SDK 或 `mira` / `bytedcli` CLI 任一可用,**直接让原子自己上传**:

```bash
ATOM=<sandbox>/settle_skill/atoms/charge-atom-progress-render/render.py
SID=<session_id>
DASH_DIR=~/files/charge-progress/$SID

# 1) 子流程启动时(看板初始化 + 自动 publish)
python3 $ATOM init \
  --session-id $SID \
  --task-title "..." --prd-url "..." --owner "..." --strict

# 2) 每个 stage 节点(增量 patch + 自动 publish)
python3 $ATOM patch \
  --session-id $SID \
  --stage <s1..s8> --status <done|run|warn|err> \
  --hero-update '变更文件=N|流水线状态=PASS:ok:1' \
  --panel-json @stage.json --strict

# 3) 任何节点之后,从下面这个文件取最新 dashboard URL(永远只读它,不要自己拼)
jq -r '.html_url' $DASH_DIR/latest_urls.json
```

`init / patch` 默认会触发原子内置的 `publish` 流程(上传 JSON → 注入 `__JSON_URL__` → 上传 HTML → 写 `latest_urls.json`),路由层无需关心上传细节。

#### 兜底路径:外部上传 + URL 回注(原子内置上传不可用时)

当原子在 sandbox 里既无 `mira_runtime` SDK 也无可用 CLI 时,`init / patch` 的 stdout 会带 `warning` 字段。此时**必须**由路由层用 Mira agent 工具完成两次上传 + 一次 publish 注入:

```bash
ATOM=<sandbox>/settle_skill/atoms/charge-atom-progress-render/render.py
SID=<session_id>
DASH_DIR=~/files/charge-progress/$SID

# 1) 写本地快照(--no-publish 跳过原子内置上传)
python3 $ATOM patch --session-id $SID --stage s1 --status done --no-publish

# 2) 路由层用 mcp__runtime__upload_file 工具分别上传两个文件
#    JSON_URL = upload_file($DASH_DIR/progress.json).url    # ← 必须先上传 JSON
#    HTML_URL = upload_file($DASH_DIR/progress.html).url    # ← 这一版的 HTML 还是相对路径

# 3) 把两个 URL 回注:原子重新渲染 HTML(把 __JSON_URL__ 替换成真实公网 URL),写 latest_urls.json
python3 $ATOM publish --session-id $SID \
  --html-url "$HTML_URL" --json-url "$JSON_URL"

# 4) 第三步会**重写本地 progress.html**(因为模板已注入新 JSON_URL),
#    所以路由层必须再 upload 一次 HTML 拿到新 URL,然后再 publish 一次:
NEW_HTML_URL=$(upload_file $DASH_DIR/progress.html)
python3 $ATOM publish --session-id $SID \
  --html-url "$NEW_HTML_URL" --json-url "$JSON_URL"

# 5) 之后所有的 patch 都按 1→2→3→4 循环跑(每次 patch JSON 都会变 → 都要重新上传 JSON)
```

> 这条路径**易错**且**链路长**,只在内置上传通道彻底不可用时才走;能走推荐路径就走推荐路径。

#### 关键纪律:`dashboard_url` 永远从 `latest_urls.json` 读

```bash
DASH_URL=$(jq -r '.html_url // empty' ~/files/charge-progress/$SID/latest_urls.json)
```

- **路由层不能自己拼 URL**(TOS 是签名 URL,拼出来必 404)
- **每次 patch / publish 之后**都要重读这个文件,因为 hash URL 每次都会变
- 用户面前展示的 `dashboard_url` 必须是这个文件里的 `html_url`,不是 `progress.html` 的本地路径

调用契约见 `atoms/charge-atom-progress-render/SKILL.md`(子命令、payload schema、字段语义)。

### 看板原子加载失败 = 硬阻断(**禁止降级**,仅 dashboard_enabled = true 时生效)

> **若步骤 0.4 判定 `dashboard_enabled = false`**(非 Mira 平台),本节整段**不适用**,看板原子无需加载,主流程直接进入步骤 1。本节仅在 Mira 平台(或团队显式 force_enable)下生效。

`charge-atom-progress-render` 调用异常(原子文件未同步到 sandbox / render 报错 / 上传失败 / sandbox 缺工具链)时,**直接终止整个路由,向用户报错**,**不允许**走任何 markdown / 文本兜底:

1. **禁止**生成 `dashboard.md` 兜底文件
2. **禁止**用 `mcp__runtime__upload_file` 上传 markdown 假装看板就绪
3. **禁止**写入 `latest_urls.json` 中的 `mode: markdown_fallback` / `fallback_reason` 等降级字段
4. **禁止**以"复用历史 MR / BITS / AiTest 产物"的名义跳过编码 / push / 流水线 / 报告环节
5. **必须**立刻停止后续所有步骤(包括步骤 1~7),把以下信息直接抛给用户:
   - 具体失败节点(步骤 -1 装机 / 步骤 0 鉴权 / 步骤 2 同步 / 看板 init 哪一步)
   - 失败命令的真实日志关键行(npm 错误码 / `bytedcli` 退出码 / `coco` 报错原文)
   - 给出可执行修复建议(例如"sandbox 不可用,请重启 sandbox 重试" / "bnpm 网络不通,联系运维")
6. **必须**在主对话里输出一条独立的"路由阻断"回复消息给用户,内容含失败节点 + 真实日志摘要(无需走 lark-im / im-chat-manager,Mira 普通回复即等同私聊呈现)
7. **不允许**最终回复中出现"看板已就绪"之类的文案——因为它根本没就绪

> 设计意图:宁可让用户立刻看到失败,也不要让 Agent 在静默降级模式下持续推进、最后产出与用户预期不符的复用产物。

### 看板单节点失败处理(主流程已起跑后,某次 patch 失败)

主流程已起跑 + 步骤 -1/0/0.5 已通过 + 仅某次 `progress-facade.patch` 调用失败 → **仅 warn,不终止主链路**,但**最终回复仍必须给出 dashboard_url**(用最近一次成功 patch 后的 `html_url`)。
此条**仅适用于**已经 init 成功之后的增量 patch 失败,**不适用于** init 阶段或工具链阶段失败。

## 执行流程(严格按顺序)

### 步骤 -1:sandbox 工具链装机自检(**必须最先执行**,失败即硬阻断)

每次进入路由,**第一件事**是确认 sandbox 里 `bytedcli` 已就绪。这是**所有后续步骤的硬前提**——没有它,鉴权握手 / 仓库同步 / 看板原子全部不可用。

```bash
# 1) 探测
if ! command -v bytedcli >/dev/null 2>&1; then
  # 2) 安装到 $HOME/.npm-global(免 sudo,持久化目录)
  export NPM_CONFIG_PREFIX="$HOME/.npm-global"
  export NPM_CONFIG_REGISTRY="http://bnpm.byted.org"
  mkdir -p "$NPM_CONFIG_PREFIX"
  npm install -g @bytedance-dev/bytedcli@latest
  export PATH="$HOME/.npm-global/bin:$PATH"
fi
# 3) 把 PATH 注入到 ~/.bashrc(下一轮 shell 自动加载,避免每次重装)
grep -q 'npm-global/bin' ~/.bashrc || echo 'export PATH="$HOME/.npm-global/bin:$PATH"' >> ~/.bashrc
# 4) 验证
bytedcli --version
```

**纪律(失败即硬阻断,禁止降级)**:
- 装机命令最多 retry 1 次,仍失败 → **立刻停止后续所有步骤**,把 npm 真实日志(error code、registry、网络错误)直接抛给用户(以一条独立的"普通回复消息"形式输出,标题写「路由阻断:bytedcli 装机失败」+ 日志摘要;**禁止**调 lark-im / im-chat-manager)
- **禁止**在装机失败的情况下继续走 markdown 降级 / 复用历史产物 / 跳过原子
- 装机成功后**必须**继续走完整正向流程(步骤 0 → 0.5 → 1 → ... → 7)
- 同会话内已装机过 → 跳过装机,直接做版本探测即可

### 步骤 0:bytedcli 鉴权握手(必须每次执行)
1. `bytedcli auth status`,若 ByteCloud Auth 未就绪则触发 SSO 登录(最多 retry 1 次)
2. `bytedcli --json auth get-codebase-jwt-token` 取 Codebase JWT
3. `bytedcli codebase auth config-auth --jwt-token <jwt>` 注入
4. 任一步失败 → 暂停并向用户报错,**不降级到 PAT**
5. **额外要求**:本步执行前/后任何时机都不允许直接读取硬编码的 `business_lines` / `project_keys` / `skill_repo` 等团队相关字段;统一通过下一步「步骤 0.2」加载 `team_profile.yaml`,从内存中读取。

### 步骤 0.4:平台能力探测(必须每次执行,2026-06-01 起强制)

**设计动机**:auto-develop 既要在 Mira 平台跑(进度看板是核心体感),也要在 Codex / Claude Code / 纯 sandbox 容器 / CI worker 等非 Mira 平台跑(此时看板原子不可用)。统一在路由入口判定,后续所有"看板必须 emit / 看板硬阻断"约束都基于这个标志位。

**判定算法**(按顺序短路):

```python
def detect_dashboard_enabled(team_profile: dict) -> tuple[bool, str]:
    dash_cfg = team_profile.get("dashboard", {}) or {}
    # 1) 团队显式 force_disable → 关
    if dash_cfg.get("force_disable") is True:
        return False, "team_profile.dashboard.force_disable=true"
    # 2) 团队显式 force_enable → 开(适用于自建 Mira 兼容平台)
    if dash_cfg.get("force_enable") is True:
        return True, "team_profile.dashboard.force_enable=true"
    # 3) 环境变量信号(Mira 平台注入)
    import os
    if os.getenv("MIRA_PLATFORM") == "1" or os.getenv("MIRA_SESSION_ID"):
        return True, "MIRA_PLATFORM / MIRA_SESSION_ID env detected"
    # 4) ~/files/ 可写 + mcp__runtime__upload_file 工具可调(Mira sandbox 特征)
    home_files = os.path.expanduser("~/files")
    if os.path.isdir(home_files) and os.access(home_files, os.W_OK) and _mcp_runtime_upload_available():
        return True, "~/files writable and mcp__runtime__upload_file available"
    # 5) 默认 → 关
    return False, "no Mira platform signal detected"
```

> `_mcp_runtime_upload_available()` 的轻量探测:看 sandbox 里是否存在 `mcp__runtime__upload_file` 工具入口(通过路由层注入的能力清单判断,**不**真发空请求)。

**执行**:
1. 在步骤 0.2 加载 `team_profile` **之后**(因为要看 `team_profile.dashboard.*` 字段)、步骤 0.5 看板初始化**之前**调用
2. 把结果写入路由上下文:`ctx["dashboard_enabled"] = bool; ctx["dashboard_disabled_reason"] = str`
3. `dashboard_enabled = false` → **直接跳过**步骤 0.5(看板初始化)、跳过后续所有 `progress-facade.patch` / `progress-facade.publish` 调用、跳过"看板地址即时回复"独立消息,**主流程其余步骤(1 ~ 7)正常全跑**
4. `dashboard_enabled = false` 时,Agent 在第一次回复用户时**追加一行**:
   ```
   > 当前平台未启用进度看板(non-Mira platform: <reason>);主流程会完整执行,仅省略实时看板。
   ```
   之后所有阶段回复**省略** `看板:<dashboard_url>` 字段,但报告链接、MR 链接、commit_id 等其它产物**照常返回**
5. **fail-fast 例外**:本步的"探测"本身**绝不允许**因任何错误阻断主流程 — 探测异常时按 `dashboard_enabled = false` 兜底处理,记 `warnings: ["dashboard_detect_failed"]`,**不**抛错

**与原"看板硬阻断"规则的关系**:
- `dashboard_enabled = true` → 看板原子加载失败 = 硬阻断(原规则不变)
- `dashboard_enabled = false` → 整个看板节点 = skip,不存在"加载失败"概念,**严禁**因看板不可用就阻断主流程

### 步骤 0.2:加载 team profile(必须每次执行,失败即硬阻断)

**设计动机**:auto-develop SKILL 要支持**多个团队复用**(财经计费域是首批接入者,后续会有其它团队 fork)。每个接入团队的 `skill_repo` / `skill_branch` / Meego 业务线 / Meego project_keys / BITS 参数都不同,**不允许**在 SKILL.md 内硬编码任何团队相关值,统一通过各团队 skill 仓库根目录的 `config/team_profile.yaml` 配置化。

**加载顺序(优先级降序)**:
1. 环境变量 `AUTO_DEVELOP_PROFILE_PATH` 指向的文件(CI / sandbox 注入)
2. sandbox 本地默认 `~/charge-cache/<skill_repo_short>/config/team_profile.yaml`(步骤 2 同步后才有,所以本步如果找不到要继续步骤 2,再回头加载)

**实务编排**(顺序非常重要):
1. **步骤 0.2-prefetch**:仅当 `AUTO_DEVELOP_PROFILE_PATH` 已设置 → 直接加载;否则跳过,进入步骤 2 先拉仓库,再回头加载
2. **步骤 0.2-postsync**:步骤 2 完成 `bytedcli codebase repo file config/team_profile.yaml` 拉到本地 → 立即加载校验
3. 加载方式:`python3 atoms/auto-develop-runtime/profile_loader.py [path]` (或直接 import 该 module),它会做 schema 校验、字段必填检查、正则可编译性检查
4. **任一字段缺失 / 正则非法 / 文件不存在 → fail-fast**,把真实错误抛给用户(独立的"普通回复消息",同 0.5 第 5 条通道;**禁止**调 lark-im / im-chat-manager)
5. 加载成功 → 把 profile dict 写入路由上下文 `team_profile`,后续步骤 1 / 步骤 6 / 看板 patch 全部从中取值

**team_profile.yaml 必填字段** schema:

```yaml
display_name: <string>                   # 团队显示名,如"财经-计费/支付"
skill_repo:   <string>                   # 团队 skill 仓库,如"bytepay/settle_skill"
skill_branch: <string>                   # 该仓库的工作分支,如"feature/ai_native"
meego:
  business_lines:    [<string>, ...]     # 业务线候选,优先级降序;多 Meego 候选时按此挑业务线匹配的
  preferred_business_line: <string>      # (可选) 本域硬约束业务线;命中即选,优先级高于 business_lines
  project_keys:      [<string>, ...]     # Meego project key(如"zhifu")
  url_pattern:       <regex>             # Meego URL 正则(默认仅 larkoffice 域)
  prd_field_names:   [<string>, ...]     # Meego 工单中 PRD 文档字段的候选名,按顺序匹配
bits:
  meego_param_mode:  "url" | "id"        # 创建 BITS 任务时传给 --meego 的值是 URL 还是 ID
  space_id:          <string>            # SSO 默认空即可
  from_dev_id:       <string>            # SSO 默认空即可
dashboard:                               # 可选,默认按平台自动探测
  force_enable:      <bool>              # 强制启用看板(适用于自建 Mira 兼容平台)
  force_disable:     <bool>              # 强制关闭看板(便于在 Mira 平台调试主链路时跳过)
```

**加载失败示例**(fail-fast,严禁降级):
- `team_profile.yaml not found` → 提示用户「请在仓库根创建 config/team_profile.yaml 并设置 AUTO_DEVELOP_PROFILE_PATH」
- `meego.business_lines must be non-empty list` → 提示具体字段
- `bits.meego_param_mode must be in {url, id}` → 提示合法值

**绝不**内置默认 profile(如把 charge 域作为隐式默认),所有团队对等接入。

> ⛔ **禁止**在 SKILL.md / atom 代码中作为**运行时**值硬编码以下内容(出现在示例/历史用例处可以,但运行时取值必须 `profile.<field>`):
> - `bytepay/settle_skill` / `feature/ai_native` 作为路由层 repo / branch
> - `["支付","渠道对接层","合众-清结算"]` 作为业务线列表
> - `"zhifu"` 作为 project_key
> - `meego.larkoffice.com/zhifu/story/detail/\d+` 作为内置正则
> 一旦发现,视为路由实现错误。

### 步骤 0.3:Meego 鉴权前置(2026-06-03 新增,必须每次执行)

**设计动机**:旧版本把 Meego 登录推迟到 Step 1 内部 —— PRD 含 `<readonly-block type="meego">` 嵌入卡片时(飞书新版 PRD 模板默认形态)需要 MQL 反查 story_id,反查失败才回头拉登录,链路上出现"扫码 → 卡在中段 → 等用户 → 续跑"的体感断点。前置到本步统一鉴权,Step 1 永远在已登录态下跑,体感顺滑。

**调用顺序**:Step 0.2 完成后(已拿到 profile.meego.project_keys)、Step 0.4 之前。

**实现**:调用 `atoms/auto-develop-runtime/meego_auth_preflight.py` 的两段式接口:

```python
from meego_auth_preflight import begin_login, complete_login

handshake = begin_login(qr_dir="/tmp/auto-dev/meego-login")
status = handshake["status"]
```

**3 种结果分支**:

| status | 含义 | 路由层动作 |
|---|---|---|
| `already_authenticated` | 上次 token 仍有效 | **直接进入 Step 0.4**,不发任何独立消息 |
| `qr_pending` | 需要扫码 | 走「QR 扫码独立消息」流程(见下) |
| `error` | begin 阶段失败 | 把 message 透传给用户,**暂停**等用户排障(等同 Step 0 的 SSO 失败处理),**禁止降级到「跳过 Meego」** |

**QR 扫码独立消息流程**(`qr_pending` 分支,与 Step 0.5 看板独立消息走**同一通道纪律**):

1. `mcp__runtime__upload_file(handshake["qr_path"])` 拿到公网 PNG URL
2. 在主对话里 emit 一条**独立的"普通回复消息"**,**仅**包含以下 4 段(不塞别的内容,确保用户第一眼看清楚):
   ```
   【auto-develop】Meego 鉴权前置:请扫码登录

   ![扫码登录 Meego](<uploaded_png_url>)

   或点链接授权:<verification_uri_complete>
   user_code:<user_code>(若链接里 code 没自动填充,手动粘进去)

   扫码确认授权后回复「继续」,我立刻完成 token 交换并进入主流程。
   ```
3. **通道纪律同 Step 0.5 第 5.1 条**:
   - ⛔ 严禁 `lark-im` / `im-chat-manager` / `lark-cli bot 私聊` —— 沙箱 `open_id` 被 PII 脱敏,server 必拒
   - ✅ 唯一通道:主对话独立回复(Mira 与用户对话本身即为飞书私聊)
4. emit 后**等待用户回执**(关键词:「继续」/「ok」/「已扫」/「done」等),**不**预先 `--complete`(token 在 user_code 被 Meego 服务端确认前调用会报 `pending`)
5. 用户回执后调 `complete_login(handshake["complete_token"])`:
   - `status=success` → 继续 Step 0.4
   - `status=error` 且 message 含 `pending` → 提示用户「检测到尚未授权,请确认已在飞书完成扫码后再回复」,**最多 retry 1 次**
   - `status=error` 且 message 含 `expired` → 重新调 `begin_login` 拉新 QR,**最多重新拉 1 次**
   - 其它 error → 透传 message,**禁止降级**

**禁止事项**:
- ⛔ 禁止在沙箱里跑 `bytedcli meego login`(不带 `--begin`)— 该模式会在终端阻塞等扫码,沙箱无终端会卡死
- ⛔ 禁止尝试 `bytedcli auth login --session --feishu` 路径 —— 该路径依赖浏览器 cookie,沙箱不可达,会误导用户去本机操作
- ⛔ 禁止「跳过 Meego」降级方案 —— 即便 PRD 内无 Meego 卡片,后续 Step 6 创建 BITS 任务也需要 Meego 登录态

### 步骤 0.5:看板初始化(**仅 dashboard_enabled = true 时执行**;否则直接跳过本步)

> **短路条件**:若步骤 0.4 设置 `ctx["dashboard_enabled"] = false` → **整段跳过本步**(不调 init / 不写 latest_urls.json / 不 emit "看板已就绪"独立消息),直接进入步骤 1;`ctx["dashboard_url"] = None`,后续回复中**省略**看板字段即可,**不阻塞**主流程。

1. 生成 `session_id`(用 PRD 标题 slug + 时间戳)
2. 调用 `progress-facade.init(task_meta={title, prd_url, owner, started_at})`;原子未就绪 / init 失败 → **立即按「看板原子加载失败 = 硬阻断」终止整个路由**,严禁回退到 markdown + mira 上传
3. 校验 `~/files/charge-progress/<session_id>/latest_urls.json` 已写入 `html_url`
4. **绝不跳过本步**(否则后续无法保证最终 dashboard_url 非空)— ⚠️ 此条仅在 dashboard_enabled = true 时成立;false 时本步整体跳过
5. **【强制,仅 dashboard_enabled = true 时生效】拿到 `html_url` 后,立刻在主对话里输出一条独立的"普通回复消息"给用户**(Mira 在飞书侧会把回复呈现到当前用户与 Mira 机器人的私聊中,等同于即时推送),**不允许等到下一次用户回复 / 交互停顿 / 最终回复才出**。`dashboard_enabled = false` 时整条不发,主流程继续:

   **5.1 通道说明(关键纪律,违者即推送失败)**

   - ⛔ **禁用** `lark-im` skill 的 `send_direct_message`、`im-chat-manager` 的等价接口,以及任何"通过 lark-cli bot 直发私聊"的路径。原因:Mira 沙箱对 `open_id` 做 PII 脱敏(占位符形如 `[ph_USER_ACCOUNT_x_ph]`),server 端 `lark-cli im +messages-send --as bot` 强校验必须 `ou_` 前缀;且 union_id / user_id 均被 server 拒(`99992402 field validation failed`)。**这条坑已实测确认,不要再踩。**
   - ⛔ **禁用** `lark_contact me` / `get-user` / `search-users` 来"换取真实 open_id 再发 IM" —— 沙箱永远拿不到明文 `ou_xxx`,链路已闭死。
   - ✅ **唯一通道**:在主对话里以一条独立回复消息把看板地址给用户。Mira 与用户的对话本身就是私聊,该消息会立即在飞书的 Mira 机器人私聊里呈现,无需再二次推送。
   - **不需要做接收人解析**:Mira 路由层默认就发给当前会话的用户,且 LLM 输出的回复消息不会被 PII 脱敏拦截。

   **5.2 调用顺序**

   - 在 Coco / BITS 等长时阻塞开始**之前**,先 emit 一条独立的回复消息(只包含看板地址 + 标题 + 会话 ID,不要塞别的内容,确保用户第一眼就能看到)
   - 然后再继续走步骤 1+(进入正式编排)
   - 这条消息**不需要等用户回应**,emit 完立刻继续

   **5.3 消息体**

   - 看板就绪(步骤 0.5):
     ```
     【auto-develop】本次任务看板已就绪
     标题:<PRD 标题>
     看板:<html_url>
     会话:<session_id>
     ```
   - 报告就绪(步骤 7):
     ```
     【auto-develop】本次任务报告已就绪
     标题:<PRD 标题>
     看板:<html_url>
     报告:<report_url>
     MR:<mr_url 或 N/A>
     ```

   **5.4 状态记录**

   - 这条独立回复发出 → 上下文置 `dashboard_replied=true`
   - **绝不**再调 `lark-im` / `im-chat-manager` 试图"再补推一次",**绝不**因"消息没真发出去"为由阻塞主链路

6. 在每次后续的用户停顿点 / 最终回复中,仍需在置顶处展示 `dashboard_url`(已在步骤 5 emit 过独立回复 ≠ 后续回复中可省略)— ⚠️ `dashboard_enabled = false` 时本条不适用,后续回复中**省略**看板字段即可
7. **本步是「与用户体感对齐」的关键节点**(仅 dashboard_enabled = true 时):Coco 编码 / BITS 流水线启动前的「长时阻塞」开始之前,**用户必须已通过独立回复消息收到看板地址**;否则视为本次路由执行失败的体感故障(虽然代码侧成功)。`dashboard_enabled = false` 时本条不适用

### 步骤 1:接收并校验输入(PRD ↔ Meego 双向解析)

**输入识别**(必须支持两种入口):
1. **飞书文档 URL**(`https://*.larkoffice.com/wiki|docx|docs/...`)→ 走"飞书分支"
2. **Meego 工单 URL**(匹配 `team_profile.meego.url_pattern`)→ 走"Meego 分支"

**调用 `atoms/auto-develop-runtime/prd_meego_resolver.py:resolve(input_url, profile, lark_doc_fetcher, meego_workitem_getter)`** 完成解析。两个 callable 由路由层注入:

| 注入项 | 实现 |
|---|---|
| `lark_doc_fetcher(url) -> markdown_str` | 走 `lark-doc +fetch --doc-format markdown` |
| `meego_workitem_getter(url) -> dict` | 走 `bytedcli --json meego workitem get --url <url> --rich`,取 `.data` |

**Meego CLI 鉴权前提**:`bytedcli meego workitem get` / MQL 反查均要求 `bytedcli meego login` 已完成,**已由 Step 0.3「Meego 鉴权前置」统一兜底**(2026-06-03 起),本步执行时假定登录态可用。万一 Step 0.3 之后 token 过期(超时罕见场景),getter 抛 `MEEGO_AUTH_REQUIRED` 时路由层按下表分支:
- Meego 分支 → 立刻硬阻断,提示用户重启会话或回 Step 0.3 重新扫码
- 飞书分支(只是想顺便取 Meego)→ 仅 warn,把 `meego_url` 留空,主流程继续

**核心原则:Meego 链接缺失永不阻塞飞书分支主流程**(2026-05-30 起强制)。
飞书 PRD 是研发流程的**强依赖**(没有 PRD 没法继续),Meego 工单只是**弱关联**(便于回写状态/绑定 MR,缺失只影响"美观度"而非"可行性")。因此对"飞书 PRD 分支 + 0 Meego"这种最常见的轻量化场景,路由层必须**透明放行**:
- 不弹窗、不询问、不要求用户补 Meego
- `ctx.meego_url = ""`(空串,不是 None,便于下游 join 命令时拼空字符串自动消失)
- `warnings += ["no_meego_url_in_prd"]`(规范的 warning code,后续步骤可识别;**不**进 errors)
- 看板 s1 panel 在 `meego_url` 位置渲染 `⚠️ 未关联(继续推进)`
- 最终飞书报告"人工跟进"小节追加一条:"Meego 工单未关联;若需要追踪研发进度回写,可手工绑定 MR ↔ 工单"

**解析结果分支处理**:

| `resolve()` 返回字段 | 路由层动作 |
|---|---|
| `needs_user_input=True` | 把 `user_prompt` 作为独立回复消息发给用户,**阻塞等待**用户补 URL/重试触发词。**该路径仅在"PRD 完全缺失"时触发**,Meego 缺失不进此路径 |
| `prd_url` 非空 + `meego_url` 非空 + `needs_user_input=False` | 正常推进,把两者写入路由上下文 |
| `prd_url` 非空 + `meego_url` 空 + `warnings` 含 `no_meego_url_in_prd` | **不阻塞**(按上方"核心原则"处理);`ctx.meego_url = ""`;后续 BITS 创建任务时省略 `--meego` |
| `warnings` 非空(其它) | 在最终报告"人工跟进"段落列出 |

**降级行为(向后兼容旧行为)**:
- 用户给飞书 PRD,内含 0 Meego → BITS create 省略 `--meego`,完全等价 2026-05-19 前的旧行为,**且不向用户提问**
- 用户给 Meego,工单无 PRD 文档字段 → 硬阻塞,提示用户补 PRD URL(因为没 PRD 无法继续;PRD 与 Meego 的"强弱依赖"不对称)

**多 Meego 候选挑选规则**(2026-06-03 起强制):

飞书 PRD 内常出现多个 Meego 嵌入卡片(如「收单+结算」「直连+连锁+服务商」拆 N 个 story 同时落地),resolver 的 `_pick_meego_by_business_line` 按下面 3 级优先级选主链接:

| 优先级 | 匹配源 | 行为 |
|---|---|---|
| 1 | `profile.meego.preferred_business_line`(**本域硬约束**) | 候选业务线字段值含本字符串 → **立即选**,记 `meego_pick_by_preferred` warning(便于审计) |
| 2 | `profile.meego.business_lines`(通配候选池) | preferred 未命中时,逐候选与本列表做 `any(line in bl)` 匹配,首命中即选 |
| 3 | fallback | 全无匹配 → 取第一个成功 fetched 的候选 + `multi_meego_no_business_match` warning |

**配置约定**:
- 各团队接入时,`preferred_business_line` 填**本域的硬性业务线归属**(如计费域 = `合众-清结算`),`business_lines` 填**本域可能覆盖到但非首选**的业务线列表(如计费域兼接 `渠道对接层`/`支付`)
- 缺省 `preferred_business_line` → 行为退化到 2 级规则(向后兼容)
- 缺省 `business_lines` 是非法配置,profile_loader 会 fail-fast

**完成后**:`progress-facade.patch(s1=done, panels={prd_title, prd_url, meego_url, summary})`。看板 s1 panel 必须同时展示 `prd_url` 与 `meego_url`(后者缺失时显示 ⚠️ 待补充)。

### 步骤 2:同步子流程仓库(必须每次执行)
按本 SKILL 内置的 **`公共能力·settle_skill 仓库同步与提交(repo-sync)` 章节 → 模式 1(sync)** 执行,拉取/更新 `https://code.byted.org/bytepay/settle_skill`(**不**调外部 SKILL,直接执行该章节的 bash)。
- **试用期默认分支:`feature/ai_native`**
- 同步成功 → 加载本地 `manifest.yaml`
- 同步失败 → **直接终止整个路由**,把 `bytedcli codebase` 真实日志抛给用户(以一条独立的"普通回复消息"形式输出阻断信息;**禁止**调 lark-im / im-chat-manager);**不再**走 `charge-subflow-fallback` 之类的兜底

### 步骤 2.5:加载强基线知识源(必须每次执行,失败即硬阻断)

**设计原则**:计费域**强基线知识**清单沉淀在 settle_skill 仓库 `config/knowledge_sources.yaml` 中(本仓库只持有"文档地址 + 元信息",**内容每次运行实时拉取**)。本路由层不在 SKILL.md 中硬编码任何文档 URL。

**执行**:

```bash
SK=<sandbox>/settle_skill                           # 步骤 2 同步出来的本地路径
ATOM=$SK/atoms/charge-atom-knowledge-loader/load.py
CFG=$SK/config/knowledge_sources.yaml

# 默认 scope=charge;用户在对话中显式说"刷新知识库"时附加 --refresh force
python3 "$ATOM" --config "$CFG" --scope charge --refresh auto > /tmp/kn_loaded.json
echo "exit=$?"
```

输出 JSON 形如:

```json
{
  "loaded":[{"key":"payment_charge_whitepaper","cache_path":"/home/mira/files/charge-knowledge/payment_charge_whitepaper.md","bytes":26788,"hit_cache":true,"loader":"lark-docs-skill"}],
  "failed":[],
  "warnings":[]
}
```

**纪律**:
- exit=1(`failed` 非空,即任一 `required: true` 的源加载失败)→ **立刻硬阻断**,把 `failed[]` 中的 error 抛给用户(独立的"普通回复消息",同步骤 0.5 第 5 条通道;**禁止**调 lark-im / im-chat-manager)
- exit=0 但 `warnings` 非空 → 仅 warn,主流程继续,在最终报告"人工跟进项"列出
- 把 `loaded[].cache_path` 数组写入路由上下文 `loaded_knowledge_paths`,后续 PRD 解析 / 子流程编码任务可注入这些文件作为基线背景
- 用户显式说"刷新知识库 / 重新拉文档" → 本步使用 `--refresh force`,强制重拉并覆盖缓存
- **绝不**在本 SKILL.md 中硬编码具体文档 URL;新增/调整文档源走 settle_skill 仓库的 yaml PR

**完成后**:`progress-facade.patch(s4=run, panels={baseline_loaded_keys:[...], baseline_warnings:[...]})`(注意:s4 与步骤 3 共享,先标 run,步骤 3 完成时再置 done)

### 步骤 3:insearch 关键词增量召回(失败 warn,不阻塞)

调用 `bytedcli`,使用 `bytedance-insearch` 子命令域,按本次 PRD 关键词检索增量背景知识(白皮书等基线已在步骤 2.5 加载,本步只做**增量补充**)。
未授权或 0 命中 → 仅 warn 不阻塞,主流程继续(基线知识已齐备)。

- **完成后**:`progress-facade.patch(s4=done, panels={baseline_loaded_keys, used_keywords, hit_count, top3_links})`

### 步骤 4:读取并解析 PRD
调用 `bytedcli` `bytedance-feishu` 子命令域读取 PRD 全文,结构化产出:
- 需求标题 / 背景 / 目标
- 涉及系统列表
- **变更项清单**(item 列表):每个 item 包含 `description`、`type` 候选标签
- **关键词数组**(`prd_keywords`):用于下一步仓库路由

### 步骤 4.5:仓库与模块路由(多仓库并发的核心)

调用 **`charge-atom-repo-router`**,传入:

```yaml
prd_title: ${PRD标题}
prd_text: ${PRD全文}
prd_keywords: ${PRD解析产出的关键词数组}
config_path: ${sandbox路径}/settle_skill/config/repo_routing.yaml
force_repos: []                       # 仅当用户在对话中显式指定仓库时填写
```

**期待返回**:`matched_repos[]`(每条含 `repo_key`、`repo_meta`、`recommended_subflow`、`suggested_change_type`、`suggested_reference_files`、`suggested_strategy`、`matched_l1_keywords`、`matched_l2_keywords`)+ `execution_mode`(parallel | single | none)+ `needs_user_confirmation`。

**处理分支**:
- `execution_mode=none` 且 `needs_user_confirmation=true` → 把 `confirmation_prompt` 抛给用户,**停止**等待回复
- `execution_mode=single` → 进入步骤 6,只编排 1 条子流程链路
- `execution_mode=parallel` → 进入步骤 6,**并发派发** N 条独立的编排链路(每个 `matched_repos[i]` 一条),每条链路独立持有自己的 Coco task / 分支 / BITS dev_id / MR

**关键纪律**:
- 路由层**不在 SKILL.md 内硬编码任何关键词或仓库 ID**;所有业务知识来源于 `config/repo_routing.yaml`,通过 `charge-atom-repo-router` 读取
- 多仓库命中时**不做仓库择优**,全部派发,由各自子流程独立完成与上报
- `repo_meta.build_system` 字段决定走哪个子流程:`scm` → `channel_charge`,`bits` → `general_dev`

**完成后**:对每个 `matched_repos[i]` 调用 `progress-facade.patch(s2=done, panels={repo_path, build_system, l1_keywords, l2_keywords, reference_files})`。多仓时每个仓单独 patch。

### 步骤 4.7:复杂度判定 + 复杂需求技术方案前置(必须每次执行)

**设计动机**:历史路由层只用「肉眼判断」决定是否触发复杂需求技术方案(旧 `tech-solution-gen` 原子),直接进 `coco_full` 全量编码,导致复杂需求(架构/对账/链路改造类)在外部依赖未敲定时硬开跑,产出物易返工。本步把判定权交还给 `config/tech_solution_routing.yaml`,**禁止**路由层基于直觉跳过技术方案。complex 分支的技术方案实现已于 2026-07-10 切换为 `prd-tech-design` 编排 Agent。

#### 4.7.1 调用 `tech-solution-facade`

输入:
- `prd_text` / `prd_keywords`(步骤 4 产出)
- `change_type` / `reference_files`(步骤 4.5 产出)
- `repo_hit_count` = `len(matched_repos)`(步骤 4.5 产出)
- 配置文件路径:`<sandbox>/settle_skill/config/tech_solution_routing.yaml`

按 yaml 中 4 个维度逐一打分(`reference_files_count` / `change_type` / `repo_hit_count` / `prd_keywords`),聚合规则 = `any_complex_wins`。任一维度命中 complex → 整体 complex。

输出:
```json
{
  "complexity": "simple | complex",
  "complex_reasons": ["prd_keywords:[架构,对账]", ...],
  "matched_dimensions": [...],
  "atom_used": "__inline_template__ | prd-tech-design"
}
```

#### 4.7.2 路由分支

**Case A:complexity = simple**
- 走 facade 内置模板(`simple_template_sections`)生成"轻量技术方案"作为 `tech_solution_md`
- 直接进入步骤 5 / 6,**无需用户介入**
- 编码节点走 **Coco sandbox 自动编码**(老路径,保持现状)

**Case B:complexity = complex**(默认走本地交付,**不再自动跑 Coco 编码**)

> **2026-06-01 起强制更新**:复杂需求(架构 / 对账 / 跨链路 / 多仓改造类)Coco 原子产出物历史返工率高,**默认改为本地交付**模式。路由层只产出技术方案 + handoff 交付包,**暂停**等用户在本地 IDE 编码 + push 后回填 commit_id 即可恢复路由。如团队希望坚持自动编码,在 `team_profile.yaml` 中配置 `complex_coding_strategy: auto_coco` 显式覆盖(默认值 = `local_handoff`)。

**Step B-1:获取技术方案(prd-tech-design 编排 Agent)— 强制前置,缺失即停机**

> **硬纪律(违者视为路由实现错误,产物污染必须重做)**:
> 1. **必须执行,无条件**:进入 Step B-2 之前**必须**通过 Skill 工具调用 `prd-tech-design` 编排 Agent,完成 PRD → 飞书技术方案文档生成,拿到非空的 `tech_design_feishu_url`。**禁止**跳过本步直接 emit handoff。
> 2. **tech_design_feishu_url 缺失 ≠ 可以 emit handoff**:本步必须能拿到 `prd-tech-design` 产出的飞书技术方案文档 URL。拿不到 → 立即按本步底部「prd-tech-design 不可用 = 路由停机」处理,**禁止**用 PRD 自己拆解/脑补的内容塞进 handoff 包凑数。
> 3. **方案内容只来自 prd-tech-design**:`tech_solution_md` 的来源**只能**是 `prd-tech-design` 产出的飞书技术方案文档(通过 `lark-doc` 拉取);飞书创建失败时可退回其本地 `tech-design.md` 草稿。Agent **禁止**在 PRD / Meego / 复杂度判定原因 / 仓库路由结果 / 知识基线之上做任何"技术拆解 / 实现思路 / 改造步骤 / Step 1/2/3"的二次创作。

1. 通过 Skill 工具调用 `prd-tech-design` 编排 Agent(见 `prd-to-tech-design/prd-tech-design/SKILL.md` 与 `prd-to-tech-design/README.md`):
   - **输入**:PRD 飞书/Lark 文档 URL(`prd_url`,优先);无 URL 时用 `prd_requirements` 文本兜底。可附 Meego / Bits 链接、`repo_meta`(步骤 4.5 命中的目标仓库)。
   - **上下文透传**:把 `complex_reasons` + 仓库路由结果 + 知识基线摘要(`knowledge_context`)一并交给 `prd-tech-design`,由其内部的 `prd-feature-split`(含 `llm-wiki-git query` 现状调查)消费,确保方案有代码证据支撑。⚠️ 这些上下文是给 **prd-tech-design** 用的,**不是**给 Agent 自己用来写方案。
   - **产物**:`prd-tech-design` 按阶段串联 `prd-understand → prd-feature-split → (prd-wiki-query) → prd-sequence-diagram / prd-funds-flow-diagram → prd-design-generate`,最终由 `prd-design-generate` 创建飞书技术方案文档并返回 `tech_design_feishu_url`。
2. **资金流阻塞门禁**:`prd-tech-design` 在需求理解 / 功能点拆分阶段若命中资金流不确定点(借贷账户 / 退款结算路径 / 会计主体 / 金额币种 / 是否真实动账等),会阻塞式向用户提问。此时路由层如实把这些 `open_questions` 透传给用户等待确认,**不得**替 `prd-tech-design` 脑补答案。
3. **⛔ 禁止 Agent 自行推断决策点 / 拍板项 / 外部依赖问题清单 / 技术方案 / 实现拆解**:complex 路径下,所有方案内容由 `prd-tech-design` 独立产出,Agent 不得基于 PRD 自行推断任何"重点拍板"项,也不得以"技术拆解 / 实现思路 / Step 1/2/3 / 我先帮你梳理一下"等任何形式输出自创方案内容。
4. **落盘**:把 `tech_design_feishu_url` / `tech_design_local_md`(本地 `tech-design.md` 路径,若有)/ `tech_design_created_at` 写入路由上下文 `ctx["tech_design"] = {...}`,后续 4.7.3 路径 A 从这里读方案文档。

**⛔ prd-tech-design 不可用 = 路由停机,严禁脑补降级**

下列**任一**情况发生时,**必须**立即停止后续所有动作并在主对话 emit 一条独立的"路由阻断"回复(模板见下方),**禁止**绕过 `prd-tech-design` 用任何形式的"我先帮你拆解 / 草拟一版 / 预分析"凑出 handoff 包:

| 触发条件 | 含义 |
|---|---|
| `prd-tech-design` skill 未安装 / 无法调用 | 编排 Agent 不可用 |
| `prd-tech-design` 依赖的 `llm-wiki-git` 知识库检索失败(unauth / 空命中 / error)| 现状调查前置不满足 |
| `prd-tech-design` 执行中报错且无法恢复 | 方案生成中断 |
| `prd-design-generate` 创建飞书文档失败且无本地 `tech-design.md` 兜底 | 无任何权威方案产物 |

**路由阻断回复模板(必须严格按此结构,不可填充任何"我先帮你拆解"段落)**:

```
## 🚫 路由阻断:complex 路径无法获取 prd-tech-design 技术方案

**任务**:<PRD 标题>
**PRD**:<prd_url>
**阻断节点**:步骤 4.7.2 Step B-1(prd-tech-design 编排 Agent)
**真实失败日志**:
<skill 调用 / llm-wiki-git / lark-doc 真实输出,不要润色>

**根因可能**:
- prd-tech-design skill 未安装或不可调用
- llm-wiki-git 知识库检索失败(SSO 未登录 / 仓库不可达 / 空命中)
- prd-design-generate 创建飞书文档失败

**为什么不能继续 handoff**:
按 auto-develop 规范,complex 路径的「技术方案」字段**只能**来自 prd-tech-design 产出的飞书技术方案文档。本路由**严禁**用 Agent 基于 PRD 手工脑补的"技术拆解"代替 prd-tech-design 产出物,否则下游编码 / 评审会拿到一份**非权威**的预分析草稿,造成返工。

**请按以下任一方式恢复**:
1. **手动补链接**:自行用 prd-tech-design(或其它方式)产出飞书技术方案文档后,回复「方案换了链接 <飞书文档 URL>」 — 路由层会直接用该文档作为 tech_solution_md 并恢复
2. **修复依赖**:确认 prd-tech-design skill 已启用、SSO 已登录、llm-wiki-git 知识库可达后重试
3. **显式覆盖**:在 team_profile.yaml 设 `complex_coding_strategy: auto_coco` 跳过本地交付路径(不推荐,Coco 在复杂需求上返工率高)

> 在你做出上述任一选择之前,路由会保持暂停,不会输出任何形式的"我先帮你拆解" / "预分析" / "实现思路" — 那不是合法的 complex 方案产物。
```

> 这条阻断回复**必须**在主对话以独立的"普通回复消息"形式 emit(Mira 普通回复即等同私聊呈现,**禁止**调 lark-im / im-chat-manager)。emit 完成后,**整个路由停机**,等用户走上方三种恢复方式之一。

**Step B-2:【强制硬暂停】产出 handoff 交付包并提示用户在本地开发**

> **前置硬约束**:进入本步**必须**满足 Step B-1 完成 + `ctx["tech_design"]["tech_design_feishu_url"]` 非空(或有本地 `tech-design.md` 兜底)。否则**禁止 emit 任何 handoff**,直接走 Step B-1 底部的「prd-tech-design 不可用 = 路由停机」阻断回复。
>
> **handoff 包内容约束**:本步 emit 的 handoff 包中,「技术方案」/「变更要点」相关字段**只允许**填以下内容,**禁止**任何形式的 Agent 自创"实现思路 / 技术拆解 / 改造 step / 预分析":
> 1. 用 `lark-doc` 拉取 `tech_design_feishu_url` 飞书文档,把摘要 + 原文链接放进去
> 2. 飞书文档创建失败时 → 用本地 `tech-design.md` 草稿的摘要 + 本地路径**即可**,不要再附加任何"我先帮你拆解一版"

- 把 `s5/s6/s7` 看板状态置 `pend`,`s8` 置 `run`(panels 写明技术方案链接 + handoff 状态)
- 在主对话 emit **一条独立的"普通回复消息"**(与 0.5 第 5 条同一通道,**禁止** lark-im / im-chat-manager),按下方「公共能力·complex 路径本地交付协议」的 `handoff_prompt_template` 渲染。模板**必须包含**:
  - PRD 标题 + PRD 链接 + Meego 链接(若有)
  - 匹配仓库清单(每个仓库:`group/repo` + 建议分支名 `feature/auto_dev-<slug>` + base_branch + build_system)
  - 技术方案链接(`tech_design_feishu_url`,**必填非空**;飞书失败兜底时写本地 `tech-design.md` 路径)
  - 「请按以下方式提交编码结果」的回填示意(commit_id / branch,可选 MR URL)
  - 看板地址 + session_id(看板未启用时该字段省略)
- **必须停止整个路由**,等用户回复 commit_id / branch 才能恢复

⛔ complex 路径下,**禁止**路由层在本步触发 `bytedance-coco coco task send`、`git push`、`bytedcli codebase mr create` 或任何 Coco / 编码相关动作。本步唯一动作 = 拉 prd-tech-design 方案 + emit handoff 消息 + 等用户回复。

⛔ **再次重申**:本步 emit 的 handoff 包**禁止**包含 Agent 基于 PRD / 复杂度判定理由 / 知识基线 / 仓库路由结果 自行编写的任何「技术方案」「实现思路」「Step 1/2/3」「我先拆解一版」「需要做的事清单」段落。一经检测视为路由实现错误,产物污染必须重做。模板中已经存在的「变更要点(由复杂度判定与方案抽取)」段落,其内容**仅限**:复杂度判定原因(`complex_reasons` 字段值,**原样**)、L1/L2 关键词(`l1_keywords` / `l2_keywords` 字段值,**原样**)、参考文件路径(`reference_files` 字段值,**原样**)、推荐子流程名(`subflow_name` 字段值,**原样**)、技术方案链接。**禁止**在此段落下加入任何 Agent 二次创作的描述性文字。



> **覆盖路径(团队级)**:若 `team_profile.yaml` 中 `complex_coding_strategy: auto_coco`,则跳过 Step B-2 的"暂停 + handoff",改走与 Case A 一致的 Coco 自动编码;此时 Step B-1 拿到 `tech_design_feishu_url` 后,直接进入步骤 5 / 6。该覆盖仅推荐在团队 Coco 调优充分、复杂需求返工率验证 < 30% 时使用。

#### 4.7.3 complex 路径的恢复协议(用户回复后才能续跑)

恢复触发词(任一即视为"可以恢复"):
- 「方案已定」/「可以编程」/「开始编码」/「continue」/「resume」
- 「方案有调整,XX」(用户给出增量调整,合并到上下文)
- 「方案换了链接 <new_url>」(以新链接为最新方案文档)
- **(本地交付路径必读)**「编码已完成」/「代码已 push」/「commit_id=<sha>」/「branch=<name>」/「MR=<url>」
  - 用户回填编码结果时,**可以**只给 commit_id + branch(MR 由路由层在恢复后自动创建);也可以同时附 MR URL(此时路由层跳过 MR 创建,直接接到流水线节点)
  - 用户回填的 commit_id 必须是 40 位 sha;branch 必须以 `feature/auto_dev-` 开头或在 handoff 时建议过的名字

恢复流程(按 facade 路由结果 + `complex_coding_strategy` 决策分三条路径):

**路径 A:facade 路由到 prd-tech-design(complex)+ `complex_coding_strategy=local_handoff`(默认)**

本路径下编码节点**由用户在本地完成**,路由层恢复后**跳过**步骤 6 的 code-facade(Coco 编码),直接接到 MR 节点 + 流水线节点。

1. **方案文档读取**(prd-tech-design 编排 Agent 产出,**无需轮询**):
   - `prd-tech-design` 是同步编排 Agent,Step B-1 调用完成即返回 `tech_design_feishu_url`(已落盘到 dashboard 上下文);**不存在** ArchMate 那样的异步 conv_id 轮询通道
   - 从上下文读取 `tech_design_feishu_url`(complex 路径唯一权威方案产物)
   - 若用户在恢复指令中直接附带了飞书文档 URL(手动改过方案),**优先使用该 URL**,跳过 Agent 产出的链接
   - 用 `lark-doc` 读取该飞书文档全文,作为 `tech_solution_md`
   - 若 Step B-1 飞书创建失败、上下文只有本地 `prd2tech/<需求目录>/tech-design.md` 草稿,则读取本地草稿作为兜底 `tech_solution_md`
   - 若用户在恢复消息中附带了增量调整,合并到 `tech_solution_md` 末尾的 `Decisions` 段落

2. **编码结果解析**(本地交付路径专属,**必须**):
   - 从用户恢复消息中**正则抽取**编码结果:
     - `commit_id`:`\b[0-9a-f]{40}\b`,**必填**
     - `branch`:`branch=(\S+)` / `分支[::](\S+)` / 在 handoff 时建议过的 `feature/auto_dev-<slug>` 名字,**必填**
     - `repo`:多仓时必须明确(`repo=group/name` / `仓库[::]group/name`),单仓可省略沿用 handoff 时确定的仓库
     - `mr_url`(可选):`https://code\.byted\.org/[^/]+/[^/]+/merge_requests/\d+`,提供则路由层跳过 MR 创建步骤
   - 解析失败 / commit_id 不是 40 位 sha → 在主对话 emit 阻断回复,要求用户重新回填:
     ```
     【auto-develop】complex 本地交付路径无法解析编码结果
     看板:<dashboard_url>
     收到内容:<原文摘要>
     请按下面格式回复:
       commit_id=<40 位 sha>
       branch=<分支名,通常是 feature/auto_dev-xxx>
       repo=<group/name,多仓必填>
       MR=<可选,若已自建 MR>
     ```
   - 解析成功 → 用 `bytedcli --json codebase commit list -R <repo> --revision <branch> --limit 1` 校验 `commit_id` 真实存在于远端 `<branch>` head;不一致 → 阻断回复要求确认 push 是否完成

3. **看板更新与路由恢复**:
   - 看板 `s5/s6/s7` 解除 pend,`s8` 切到 `done`(panels 写明"方案已定 + 本地编码已完成",标注 `tech_design_feishu_url` 来源 + 编码来源 = `local_dev_handoff`,记录 commit_id / branch / 用户回填时间)
   - 进入步骤 5 / 6,但 **code-facade 自动判定为 `skip`**(implementation=`skip`,`static_outputs` 注入 `{branch, commit_id, repo, changed_files}`,`changed_files` 由 `bytedcli --json codebase commit get -R <repo> --commit-sha <commit_id>` 实时拉取)
   - **MR 节点**:若用户回填了 `mr_url` → 跳过创建,直接复用;否则路由层用 `bytedcli codebase mr create` 创建 MR(body 由路由层基于 PRD + `tech_solution_md` + `changed_files` 生成,与 Coco 路径一致)
   - **流水线节点**:与 Coco 路径完全一致,`bytedcli bits develop create --change "service=<psm>,branch=<branch>"` + `quick-run`

> ⛔ **常见踩坑(必读)**:complex 路径的方案来源已从 ArchMate 异步轮询切换为 `prd-tech-design` 编排 Agent 的同步产出。**禁止**再调用任何 `archmate-boe` / `get_trd_by_convId` / `create_tech_task` 接口或 `X-Jwt-Token` 轮询逻辑 —— 这些通道已于 2026-07-10 下线。方案文档唯一来源 = `tech_design_feishu_url`(或用户手动提供的飞书 URL / 本地 `tech-design.md` 兜底)。

> **本地交付路径的失败回环策略**:流水线 fail → 路由层**不**像 Coco 路径那样自动重发编码任务(没有 Coco task 可重发),而是在主对话 emit 独立回复,把流水线失败日志摘要 + 失败原因分类 + `fix_instructions` 推给用户,等用户在本地修完再 push 新 commit 并回复新 `commit_id` 触发流水线 quick-run 重跑(用同一 `dev_id`)。详见步骤 6.2「complex 本地交付路径的失败回环」。

**路径 B:facade 路由到 prd-tech-design(complex)+ `complex_coding_strategy=auto_coco`(显式覆盖)**

行为与历史一致 — 拿到 `tech_design_feishu_url` 后直接进 Coco 编码(原路径 A)。仅在团队 `team_profile.yaml` 显式配置 `complex_coding_strategy: auto_coco` 时启用。

1. 方案文档读取流程与上面「路径 A」步骤 1 完全一致
2. 看板 `s5/s6/s7` 解除 pend,`s8` 切到 `done`(panels 写明"方案已定 + 编码启动",标注 `tech_design_feishu_url` 来源 + 编码来源 = `coco_full`)
3. 进入步骤 5 / 6,code-facade 走 Coco 自动编码(老路径)

**路径 C:facade 路由到内置模板(simple)**

1. 解析用户答复中对决策点的回填(simple 路径在暂停时列出了 Agent 自判决策点)
2. 合并回填内容到轻量方案 → `tech_solution_md`
3. 进入步骤 5 / 6,code-facade 走 Coco 自动编码(老路径)

**关键纪律**:
- complex 路径下 Agent **禁止**自行推断决策点作为恢复输入,一切以 `prd-tech-design` 产出的 `tech_design_feishu_url` 飞书方案文档为准
- simple 路径下 Agent **必须**列出自判决策点供用户拍板,作为恢复输入的一部分
- 用户若直接给了飞书文档 URL(无论是否来自 prd-tech-design),优先使用该 URL 作为 `tech_solution_md` 来源
- **complex 路径默认走本地交付**,Agent 严禁在未读到 `complex_coding_strategy=auto_coco` 时擅自切回 Coco 自动编码

**完成后**(无论 simple / complex 都要):`progress-facade.patch(s3=done, panels={complexity, complex_reasons, atom_used, tech_design_feishu_url?, tech_solution_md_size?})`

### 步骤 5:变更分类(子流程标签)
对每个变更项打**子流程标签**,以 manifest.yaml 中声明的 `subflows[].triggers` 为准。
- **仓库已由步骤 4.5 决定**;本步只决定"用哪个子流程跑这个仓库"
- 多标签命中 → 多个子流程
- 一个标签都没命中 → 取 `repo_meta.recommended_subflow`(由 yaml 配置决定的默认子流程)
- 仍无 → 标记 `unknown`,让用户兜底
- **置信度 < 0.7** → 暂停并向用户确认
- **complex 路径已在步骤 4.7 暂停过一次**:本步不再就"是否需要技术方案"重复询问用户;但若 `tech_solution_md` 中明确指出某个变更项归属与默认 trigger 冲突,以方案为准
- **完成后**:`progress-facade.patch(s3=done 已在 4.7 完成)`;若分类有歧义停顿则 patch s3=warn 并补 `decisions`

### 步骤 6:子流程编排执行(多仓库并发)

**前置硬约束**:
- complex 路径(由 4.7 判定)进入本步的前提是「用户已在本会话回复恢复指令」+「`tech_solution_md` 已读取就绪」;否则立即抛错并指向 4.7.3
- simple 路径直接进入,无前置等待
- **complex 本地交付路径专属**(由 4.7.3 路径 A 进入):code-facade 在本步**自动判定为 `skip`** — `static_outputs = {branch, commit_id, repo, changed_files}` 已由 4.7.3 恢复阶段填充就绪,直接跳到 MR 节点 + 流水线节点。**严禁**路由层重新触发 Coco 编码;**严禁**对 `coding-facade.failure_mode` 做 `loop_back_to_coding` 自动回环(因为没有 Coco task 可回环 — 见 6.2)。

对 `matched_repos` 中的**每个仓库**:

1. 选定子流程:优先用步骤 5 的命中标签;无命中时用 `repo_meta.recommended_subflow`
2. 读取 `subflows/<name>/subflow.md`,解析 YAML frontmatter 中的 `facade_implementations`
3. **按通用 8-facade 顺序**逐个执行(见下方「声明式调度协议」)
4. 把路由上下文(PRD 产出、repo_meta、knowledge 等)作为 `${}` 变量池,供 `inputs` 模板解析
5. 单个 facade 失败 → 按该 facade 声明的 `failure_mode` 处理;`stop` 则中断**该仓库子流程**(不影响其它仓库)

#### 声明式调度协议(facade_implementations)

子流程 `subflow.md` 的 YAML frontmatter 中 `facade_implementations` 字段定义了每个 facade 的实现方式。路由层按以下**固定顺序**遍历 facade(顺序不可由子流程覆盖):

```
prd-facade → repo-route-facade → tech-solution-facade → knowledge-facade → coding-facade → delivery-facade → qa-facade → report-facade
```

对每个 facade,读取子流程声明:

| `implementation` 值 | 路由层行为 |
|---|---|
| `default` | 走 `manifest.yaml` 中该 facade 的 `default_atom` / `routing_rules`,inputs 用子流程声明的 `inputs` 覆盖默认值 |
| `skip` | 跳过该 facade,progress-facade patch 为 `skip`,记录 `reason`;若有 `static_outputs` 则注入上下文 |
| `replace_with_atom` | 绕过该 facade 的 default_atom,直接调 `atom` 字段指定的原子,inputs 由子流程声明显式给出 |

**`post_hook` 处理**:

若 facade 声明包含 `post_hook`,在该 facade 执行成功后**立即**执行 hook:

| `post_hook.type` | 行为 |
|---|---|
| `ask_user` | emit 独立回复消息(按 `prompt_template` 渲染),匹配用户回复关键词;<br>`blocking: true` = 永久阻塞(无超时);<br>`timeout_minutes: N` = N 分钟超时后按 `on_timeout` 处理 |

用户回复匹配规则:
- 命中 `accept_keywords` → 继续下一个 facade
- 命中 `reject_keywords` → 按 `on_reject` 处理(`stop_and_wait` = 停止等下次回复;`loop_back_to_coding` = 回到 coding-facade)
- `on_timeout: warn_and_proceed` = 超时后 warn 继续;`warn_and_skip_to` = 跳到指定 facade

**`failure_mode` 处理**:

| 值 | 行为 |
|---|---|
| `stop` / `stop_and_ask` | 停止子流程,向用户报告错误 |
| `warn_continue` | 仅 warn,继续下一个 facade |
| `fallback_to_simple` | 原子失败时 fallback 到内置 simple 模板,warn |
| `loop_back_to_coding` | 把失败日志转为 `fix_instructions`,回到 coding-facade(含 `auto_repair` 约束) |

**`auto_repair`**:当 `failure_mode=loop_back_to_coding` 且声明了 `auto_repair`,路由层自动回环修复(不问用户),复用同一 coco_task_id / branch / dev_id;达到 `max_rounds` 仍失败则按 `on_exhaust` 处理。

**变量解析规则**(`${}` 语法):
- `${facade-name.field}` → 上游 facade 的输出字段
- `${sandbox}` / `${session_id}` / `${slug}` → 路由上下文全局变量
- `${prev.field}` → 循环修复时上一轮同 facade 的输出字段(路由层自动注入)
- `${change_type_mapping[key]}` → 子流程正文中定义的映射表

**步骤 6 内的节点切分与看板 patch 时机(每仓独立刷)**:

> 步骤 6 拆为三个**独立节点**,**编码节点完成后,MR 节点与流水线节点并行触发**(2026-05-20 起强制):
> 1. **编码节点(code-facade / Coco round-1)**:基于 master 拉新分支 → 编码 → commit → push。**不创建 MR、不跑流水线**。
> 2. **MR 节点**:基于编码节点产出的 commit_id 与 changed_files,在路由层用本地 `bytedcli codebase mr create` 创建 MR。**仅依赖远端 `<branch>` 存在**,不依赖流水线。
> 3. **流水线节点**:**仅依赖远端 `<branch>` 存在**,不依赖 MR(`bytedcli bits develop create --change "service=<psm>,branch=<br>"` 不接受 / 不要求 `mr=<iid>`)。可与 MR 节点并行触发。
>
> ⚠️ **BITS 兜底自动创建 MR**:`bytedcli bits develop create` 在源分支无 open MR 时会**自动开一个 MR**(标题前缀 `feat: [development task] <title> (<dev_basic_id>)`,作者=路由层 SSO 身份)。这意味着即使 MR 节点失败 / 被跳过,流水线节点跑完也会留下一个 MR。路由层应:
> - 流水线节点先于 MR 节点完成时,**先用 `bytedcli bits develop get --dev-id <id>` 查 `branch.mr_url`**;若已存在,跳过 `mr create`,直接 `patch(s7=done)` 用 BITS 自动 MR
> - 仅当 BITS 没自动开(罕见,例如 service-type 未识别) 才走 `bytedcli codebase mr create`

- 编码节点完成(push 成功) → `progress-facade.patch(s5=done, hero="变更文件=N", panels={strategy_used, coco_task_id, changed_files, branch, commit_id})` ⛔ 此时**绝不**调 `bytedcli codebase mr create`,**也不阻止**流水线节点并发启动
- 编码节点输出就绪后,**并行**派发 MR 节点 + 流水线节点(两节点彼此独立,不要串行 await)
- MR 节点完成(MR 创建成功 或 BITS 已自动开) → `patch(s7=done, panels={mr_url, mr_number, mr_title, mr_creator(自动/手工)})`
- 流水线触发 → `progress-facade.patch(s6=run, hero="流水线状态=RUNNING:run:0.5", panels={dev_basic_id, dev_url, pipeline_run_url})`
- 流水线终态:
  - success → `patch(s6=done, hero="流水线状态=PASS:ok:1", panels={dev_id, pipeline_url, build_log_excerpt})`
  - failed → `patch(s6=err, ...)`,**自动回环**(2026-05-20 起强制,无需用户确认):路由层立刻拉日志、形成 `fix_instructions`,回到编码节点(同 task-id / 同 branch / 同 dev_id)重发 Coco 任务;Coco 修完 + 本地编译 PASS + push 后,自动 `bytedcli bits develop quick-run --dev-id <id>` 重跑流水线。MR 节点不需要重跑,沿用已有 MR。详见下方「失败自动回环」
  - timeout → `patch(s6=warn, ...)` 停止等用户决定

**BITS 流水线节点标准命令(2026-05-20 起强制)**:

```bash
# 创建(只需 service + branch,不需要 mr=)
bytedcli --json bits develop create \
  --space-id <SPACE_ID> \
  --team-flow-id <TEAM_FLOW_ID> \   # 取自 config 的 bits_team_flow_id(计费域=751258043394),⛔ 不可省略走默认模板
  --title "<MR 前缀> <PRD 简短标题>" \
  --services <psm> \
  --change "service=<psm>,branch=<feature/auto_dev-...>" \
  --service-type TCE \
  --lane <LANE_NAME> \              # 见下文「泳道命名规范」,⛔ 禁止用默认 test
  [--meego <meego_url_or_id>...]    # 见下文「Meego URL 发现」

# 触发自检流水线
bytedcli --json bits develop quick-run --dev-id <devBasicId> --space-id <SPACE_ID> [--wait]

# 查终态(返回 branch.mr_url / pipeline_status / pipeline_failures)
bytedcli --json bits develop get --dev-id <devBasicId>
```

> ⛔ **禁止**在 `--change` 里写 `mr=<iid>` 强行绑定 MR。这会让流水线依赖 MR 创建时序,与"BITS 仅需远端分支"的解耦设计冲突。仅当用户明确要求"复用已有 MR 跑流水线"时才用 `mr=` 复用。

**泳道命名规范(2026-06-11 起强制)**:

`bytedcli bits develop create` 的 `--lane` 默认值是 `test`,叠加 `--enable-lanes both`(默认)后,BITS 会自动生成 `boe_test` / `ppe_test` 两条泳道。⛔ **禁止使用该默认命名**:多任务并行时极易撞名、难以追溯归属。

**必须显式传 `--lane`**,命名规则:`<项目标识>_<随机后缀>`。
- `<项目标识>`:取自仓库名简写或 PRD 关键词(如 `fee_operation` → `feeop`、`charge_offline` → `chgoff`),小写、仅 `[a-z0-9]`,长度 ≤ 12。
- `<随机后缀>`:5~6 位随机小写字母数字串,保证并发唯一。
- 最终 BITS 加上 `boe_` / `ppe_` 前缀,形成形如 `boe_feeop_k7x2m` / `ppe_feeop_k7x2m` 的泳道名。

```bash
# 生成泳道名(项目标识 + 随机后缀)
PROJ_TAG="feeop"                                   # 由仓库名/PRD 关键词推导,[a-z0-9] ≤12
RAND=$(LC_ALL=C tr -dc 'a-z0-9' </dev/urandom | head -c 5)
LANE_NAME="${PROJ_TAG}_${RAND}"                    # 如 feeop_k7x2m
# bytedcli ... bits develop create ... --lane "$LANE_NAME" ...
```

⛔ **严禁**回退到 `--lane test` 或省略 `--lane`(省略=默认 test)。⛔ 严禁使用纯通用词(如 `test`、`dev`、`tmp`)作为项目标识。

**Meego URL 发现**(由步骤 1 双向解析的产出决定,本步不再独立抽取):

PRD ↔ Meego 的双向解析已在**步骤 1**完成,本步只**读取**路由上下文中的 `meego_url`。
**关键约定:`ctx.meego_url` 允许为空串("");空值=合法路径,不报错、不阻塞、不提示。**

| step 1 结果 | 本步动作 |
|---|---|
| `meego_url` 非空 | 按 `team_profile.bits.meego_param_mode` 决定传参形态:`url` → 整链接传给 `--meego`;`id` → 抽尾部数字传给 `--meego` |
| `meego_url` 为空("" 或 None) | 完全省略 `--meego` 整个参数(连开关一起省,不是传空字符串),`bytedcli bits develop create` 已验证可接受 `--meego` 缺省,流水线照常创建;最终报告"人工跟进"段落提示"Meego 工单未关联,建议补登并重跑或手动绑定" |

**示例(Python 伪代码,实际由路由层 shell 拼)**:
```python
profile = ctx["team_profile"]
meego_url = (ctx.get("meego_url") or "").strip()   # 空串/None 统一归一为空串
extra = []
if meego_url:
    if profile["bits"]["meego_param_mode"] == "id":
        m = re.search(r'/(\d+)$', meego_url)
        if m:
            extra = ["--meego", m.group(1)]
        # 抽不出数字也不报错,保持 extra=[](等价于"未关联")
    else:                                            # "url"
        extra = ["--meego", meego_url]
# 当 extra=[] 时,bytedcli 命令行不出现 --meego,BITS 正常接收
# bytedcli ... bits develop create ... *extra ...
```

⛔ **严禁**为了"凑齐"参数而塞入占位值(如 `--meego ""`、`--meego 0`、`--meego null`、`--meego TBD`),
这会让 BITS 后端校验失败甚至误关联到 id=0 的脏数据。**正确做法是整段省略 `--meego` 开关。**

⚠️ **本步严禁再次正则抽取 PRD 正文**(以前的实现这么做,容易和步骤 1 的业务线匹配结果不一致);所有 Meego URL 必须来自 `ctx.meego_url`。

**MR 节点幂等约定**:
- 触发前先 `bytedcli --json codebase mr list -R <repo> --source-branch <br> --state opened`,有命中即复用,**不重复开**
- BITS 自动开的 MR 由 SSO 身份 = 路由层执行身份,与本地 `mr create` 等价;不区分对待

**失败自动回环(2026-05-20 起强制,无需用户确认)**:

> 路由层在以下两类失败下,**自主**触发回环修复 — 不打断用户、不询问、不弹出确认,直到达到终止条件之一才停。

**触发场景**:
1. **流水线节点失败**(`pipeline_status=PipelineStatusFailed` / job_run 失败):路由层视失败原因决定是否回环
2. **编码节点 push 后远端 MR 流水线 / quick-run 自检失败**:同上

**回环执行步骤**(每仓独立,失败仅影响本仓):

```
1. 拉失败日志:
   - SCM 编译失败 → bytedcli bits joblog --pipeline-run-id <run_id> --job-run-id <job_run_id>
   - UT 失败 → 同上,搜 "FAIL " / "FAILED " / "panic:" / "Error:" 关键行
   - 取末尾 200 行作为 fix_instructions 上下文

2. 形成 fix_instructions(下发给 Coco round-N,N≥2):
   - 复用同一 task_id(避免 Coco 丢失上下文)
   - 复用同一 branch
   - prompt 模板:
       上一轮提交 commit=<sha> 的 BITS 流水线 SCM 编译/UT 失败,失败日志末尾如下:
       <log_tail_200_lines>
       请基于失败原因修复代码;修复完成后必须重新跑「Push 前置编译门禁」全部命令通过,再 commit + push。
       注意:仅修必要代码,不要改其它模块测试以"通过"。

3. 路由层等 Coco round-N 完成(同样以「本地编译 PASS && 远端 head 变更」为完成判据)

4. 编码节点完成后 → 路由层 bytedcli bits develop quick-run --dev-id <同上一轮的 dev_id> 重跑;MR 节点跳过(沿用)

5. 看板 patch:
   - 进入回环 → patch(s6=run, panels={loop_count: N, last_failure_summary})
   - 回环成功 → patch(s6=done)
   - 回环失败 → 累加 loop_count,继续回环;直到达到终止条件
```

**终止条件**(满足任一即停止回环并以最终状态收尾):
- ✅ 流水线 success → `patch(s6=done)`,正常进 步骤 7
- ✅ **SCM 编译 + 部署类阶段全部 PASS,但阻塞在 selector / 人工审批 / 数字人测试 等"人工网关"上** → `patch(s6=done, panels={blocked_at_manual_gate=true, gate_name, gate_job_run_id, reviewer_action_url})`,视为自动回环目标已达成,把决策交给 Reviewer(网关类型 `selector` 在 BITS CLI 中不支持自动应答,只能由人在 Web UI 中作答;不视为失败)
- ⛔ 累计回环次数 ≥ **3 轮**(可在 manifest 配置 `max_repair_rounds`,默认 3) → `patch(s6=err, panels={loop_count, last_failure_summary, terminal=true})`,立刻在主对话 emit 一条独立回复说明"自动修复 3 轮仍失败,请人工介入",并把全部失败日志摘要附上
- ⛔ 同一类失败连续出现 ≥ 2 轮(基于 stuck_atom + 失败 stderr 哈希比较) → 提前终止(避免无效消耗),走人工介入分支
- ⛔ 用户在主对话明确说"停"/"停止"/"先别动"/"我来" → 立刻停止回环,把当前态汇报后等用户

**禁止事项(回环过程中)**:
- ⛔ 禁止在每轮回环前向用户确认("是否继续修复"等)— 自主进行
- ⛔ 禁止改 dev_id / branch / task_id(同一组上下文复用)
- ⛔ 禁止跳过「本地编译门禁」直接 push 期望"流水线试试看"
- ⛔ 禁止把失败日志全文塞给 Coco(只截末尾 200 行 + 关键 stuck_atom 信息,避免 prompt 爆炸)

**多仓库之间默认并发**;manifest 声明 `depends_on` 时按依赖串行。每个仓库的子流程产出独立的报告片段,在步骤 7 合并。

#### 6.2 complex 本地交付路径的失败回环(与 Coco 路径不同)

> 仅对 complexity=complex 且 `complex_coding_strategy=local_handoff`(默认)路径生效。

**触发**:MR 已创建 + 流水线启动后,`bytedcli bits develop get` 返回 `pipeline_status=failed`。

**与 Coco 路径的关键区别**:**没有** Coco task 可重发,也**没有** sandbox workspace 可让 Agent 自动改代码。流水线失败 → 路由层**只能**把失败信息推回给用户,等用户在本地修复并 push 新 commit。

**回环动作(每轮)**:

1. 拉流水线失败日志:`bytedcli --json bits develop get --dev-id <id>` + `bytedcli --json bits develop logs --dev-id <id> --stage failed`
2. 失败原因分类(基于日志关键词):
   - `compile_error` / `build_failed` → 编译期错误
   - `unit_test_failed` → UT 失败
   - `integration_test_failed` / `aitest_failed` → 集成 / AiTest 失败
   - `lint_failed` / `static_check_failed` → 静态检查
   - `unknown` → 兜底
3. 看板 `s6` patch 为 `err`,panels 写明分类 + 失败步骤名 + 重试轮次
4. 在主对话 emit **一条独立的"普通回复消息"**,模板:
   ```
   【auto-develop】complex 本地交付路径 — 流水线第 <round> 轮失败
   看板:<dashboard_url>
   仓库:<group/repo>
   分支:<branch>
   当前 commit:<commit_id>
   MR:<mr_url>
   流水线:<pipeline_url>

   失败分类:<compile_error / unit_test_failed / ...>
   失败步骤:<stage_name>
   日志摘要(末尾 200 行):
     <log_tail>

   修复建议:<fix_instructions, 基于失败分类的话术>

   请在本地修复后重新 push 到同分支,回复:
     commit_id=<新的 40 位 sha>
   我会自动用 quick-run 重跑流水线(同 dev_id 复用)。
   ```
5. **暂停整个路由**,等用户回复新 `commit_id`
6. 用户回复后:
   - 校验新 commit_id 存在于远端 `<branch>` head(`bytedcli codebase commit list -R <repo> --revision <branch> --limit 1`)
   - 调 `bytedcli --json bits develop quick-run --dev-id <同一 dev_id>` 重跑流水线(**不**重建 dev、**不**重建 MR)
   - 重回流水线节点的 `polling` 状态,继续走原有完成判定

**回环上限**:`max_rounds = team_profile.local_handoff.max_rounds`(默认 5);达到上限后看板 `s6` 切 `err_exhausted`,在主对话 emit 终极阻断回复,要求用户人工介入(让 owner 接管或转给其他研发同学)。

#### 6.3 公共能力·complex 路径本地交付协议(local-dev-handoff)

> 本小节定义 4.7.2 Step B-2 中 handoff 消息的标准模板与解析契约,供路由层和子流程共用。

**handoff_prompt_template**(必须按此结构 emit,**禁止** lark-im / im-chat-manager,直接 Mira 主对话普通回复):

```markdown
## 📦 【auto-develop】complex 路径 — 请在本地完成编码

**任务**:<PRD 标题>
**PRD**:<prd_url>
**Meego**:<meego_url 或 "(未关联)">
**看板**:<dashboard_url>(会持续更新进度)
**会话**:<session_id>

### 一、技术方案

- **技术方案(prd-tech-design)**:<tech_design_feishu_url>(由 prd-tech-design 编排 Agent 同步产出的飞书技术方案文档)
- 等你评审完方案 / 回复"方案已定"任一发生时,本会话恢复

### 二、目标仓库与分支

<for each matched_repo:>
| 仓库 | 建议分支 | base | 构建系统 |
|---|---|---|---|
| <group/repo> | feature/auto_dev-<slug> | <base_branch, 通常 master> | <build_system> |
</for>

> 多仓时请分别 push,各自给出 commit_id;**不**要求一个分支跨多个仓库。

### 三、变更要点(由复杂度判定与方案抽取)

> ⛔ **本段只允许填以下 4 个原值,严禁 Agent 二次创作 / 拆解 / "Step 1/2/3" / "实现思路" / "我先帮你梳理一下"等任何描述性内容。技术方案完整内容请看 prd-tech-design 产出的飞书文档 `tech_design_feishu_url`(上方"技术方案"段)。**

- 复杂度判定原因:<complex_reasons join ', '>     # 原值,facade 输出
- L1/L2 关键词:<l1_keywords + l2_keywords>       # 原值,repo-route-facade 输出
- 参考文件:<reference_files>                      # 原值,repo-route-facade 输出
- 推荐子流程:<subflow_name>                       # 原值,manifest.yaml 配置

### 四、请按以下方式提交编码结果

在本地完成开发 + push 后,**回复一条消息**,格式:

```
commit_id=<40 位 sha>
branch=<分支名,如 feature/auto_dev-xxx>
repo=<group/name>            # 多仓必填,单仓可省
MR=<可选,若已自建 MR 链接>
```

或者用自然语言也可以,只要包含 40 位 sha + 分支名,我会自动解析。
解析成功后我会:
- 校验远端 `<branch>` head 与你给的 `commit_id` 一致
- 自动创建 MR(若你没给 MR 链接)
- 触发 BITS 自检流水线(`quick-run`)
- 流水线 fail 时把日志推回给你,等你修完再回填新 commit_id

### 五、若需要切回自动编码

在 team `config/team_profile.yaml` 设置:
```yaml
complex_coding_strategy: auto_coco
```
然后重新触发 auto-develop。默认值是 `local_handoff`。
```

**用户回复解析契约**(路由层必须实现):

```python
import re

def parse_handoff_reply(text: str) -> dict:
    """从用户回复中解析编码交付结果。允许结构化或自然语言。"""
    result = {}
    # commit_id: 40 位 sha,出现位置不限
    m = re.search(r'\b([0-9a-f]{40})\b', text)
    if m: result['commit_id'] = m.group(1)
    # branch: feature/auto_dev-xxx 优先,否则 branch=xxx / 分支:xxx
    m = re.search(r'\b(feature/auto_dev-[\w\-\.]+)\b', text)
    if m: result['branch'] = m.group(1)
    else:
        m = re.search(r'(?:branch|分支)\s*[=::]\s*(\S+)', text)
        if m: result['branch'] = m.group(1)
    # repo: group/name
    m = re.search(r'(?:repo|仓库)\s*[=::]\s*([\w\-]+/[\w\-\.]+)', text)
    if m: result['repo'] = m.group(1)
    # mr_url
    m = re.search(r'(https://code\.byted\.org/[^/]+/[^/]+/merge_requests/\d+)', text)
    if m: result['mr_url'] = m.group(1)
    return result

# 必填校验
def validate_handoff_reply(result: dict, handoff_ctx: dict) -> list[str]:
    errors = []
    if not result.get('commit_id'): errors.append('missing commit_id (40-char sha)')
    if not result.get('branch'): errors.append('missing branch')
    if not result.get('repo'):
        if len(handoff_ctx['matched_repos']) == 1:
            result['repo'] = handoff_ctx['matched_repos'][0]['repo_path']  # 单仓兜底
        else:
            errors.append('missing repo (multi-repo handoff)')
    return errors
```

**`team_profile.yaml` 新增字段**:

```yaml
# 复杂需求的编码策略(默认 local_handoff)
complex_coding_strategy: local_handoff   # 可选: local_handoff / auto_coco
local_handoff:
  max_rounds: 5                          # 流水线失败回环上限
  pipeline_log_tail_lines: 200           # handoff 失败回复中携带的日志尾行数
```

> 缺省时,行为等价于:
> ```yaml
> complex_coding_strategy: local_handoff
> local_handoff:
>   max_rounds: 5
>   pipeline_log_tail_lines: 200
> ```

### 步骤 7:产出变更报告
1. 组装 Markdown 报告(包含:PRD 链接、变更项清单、子流程执行链路、changed_files、自检结论、关键产物链接、人工跟进项)
2. 检查飞书 Bot 授权:执行一次 `bytedcli --json feishu docs create-doc --title <test>` 的 dry-run **是不必要的**,改为先看 `bytedcli feishu` 上次报错码:
   - 已授权 → `bytedcli --json feishu docs create-doc --title "[auto-develop 报告] <PRD 标题>" --markdown-file <report.md>` 取回文档链接
   - 未授权(错误码 `AUTH_REQUIRED`) → **不阻塞**,改为通过 mira 上传文件能力(`mcp__runtime__upload_file`)上传 `report.md` 取分享链接,在最终回复中提示"执行 `bytedcli feishu login` 完成 Bot 授权后说『重发报告到飞书』"
3. 报告链接通过最终回复返回给用户
4. **完成后**:`progress-facade.patch(s8=done, panels={report_url, mr_url})`,并把最新的 `latest_urls.json.html_url` 作为最终 `dashboard_url` 准备回复
5. **【强制】立刻在主对话里输出一条独立的"报告已就绪"普通回复消息给用户**(同步骤 0.5 第 5 条的通道,Mira 普通回复即等同私聊呈现):
   - 消息内容:`【auto-develop】本次任务报告已就绪\n标题:<PRD 标题>\n看板:<html_url>\n报告:<report_url>\nMR:<mr_url 或 N/A>`
   - ⛔ 禁止调 `lark-im` / `im-chat-manager` 二次推送(沙箱 PII 脱敏闭死了 bot 私聊路径)
   - emit 完即继续/结束,不需要等用户回应

## 自动执行 vs 用户确认
默认全自动。仅以下五种情况必须先和用户确认:
1. 变更分类置信度 < 0.7
2. 出现 `unknown` 标签的变更项
3. 同一步骤连续失败 ≥ 3 次(**注意**:步骤 6 流水线失败的"自动回环修复"独立计数,见「失败自动回环」;只有同一步骤的非回环类失败连续 3 次才走此分支)
4. `charge-atom-repo-router` 返回 `needs_user_confirmation=true`(L1 完全无命中,或配置为 ask_user 多选时)
5. **步骤 4.7 判定 `complexity=complex`**(必须暂停等技术方案定稿,见 4.7.2 / 4.7.3)

> ⛔ **流水线失败 ≠ 用户确认点**(2026-05-20 起强制):路由层**禁止**在流水线失败时停下来问用户"是否继续修复",必须按「失败自动回环」自主进入下一轮 Coco 修复 + quick-run,直到 success / 累计回环 ≥ 3 / 同类失败连续 2 轮 / 用户中断 这四种终止条件之一。

> 任一确认停顿点的回复也**必须**附带 `dashboard_url`(从 `latest_urls.json` 读),让用户随时能看到当前进度,而不是只在最终回复才给。
> complex 路径的暂停点还**必须**附带 `tech_design_feishu_url`(技术方案链接),不可缺。

## 关键约束
- **不操作生产环境**:env ∈ {boe, staging, ppe, test, dev}
- **不在源码或日志中暴露 token**
- **每次都必须同步仓库 + 读知识库**,不允许使用上一次会话的缓存
- **强基线知识必加载**:每次执行步骤 2.5 必须调 `charge-atom-knowledge-loader` 按 `settle_skill/config/knowledge_sources.yaml` 加载基线;`required: true` 的源加载失败即硬阻断;**SKILL.md 内禁止硬编码任何文档 URL**(新增/调整文档源走 settle_skill yaml PR)
- **不使用 Codebase PAT**,统一走 SSO → Codebase JWT 的转换路径
- **看板 dashboard_url 不可省略(仅 dashboard_enabled = true 时生效)**:任何最终回复 / 中途确认停顿点,都必须包含 `dashboard_url`;若看板渲染全程失败,需显式说明"看板渲染失败 + 失败原因",**严禁静默省略**。`dashboard_enabled = false`(非 Mira 平台)时本条不适用,回复中**省略**看板字段并在首次回复追加一行 `> 当前平台未启用进度看板`即可
- **看板地址即时回复不可省略(仅 dashboard_enabled = true 时生效)**:步骤 0.5 拿到 `html_url` 之后,**必须立刻**在主对话里 emit 一条独立的"普通回复消息"给用户(包含看板地址 + 标题 + 会话 ID),不允许累积到「下次用户停顿」才出;步骤 7 报告就绪同样**必须立刻**emit 一条带报告链接的独立回复。Mira 普通回复在飞书侧会呈现为机器人私聊消息,等同于即时推送。⛔ **禁止**调 `lark-im` / `im-chat-manager` 走 bot 私聊通道 —— 沙箱 PII 脱敏(占位符 `[ph_USER_ACCOUNT_x_ph]`)与 server `ou_` 强校验冲突,链路在当前环境闭死,不要再尝试。`dashboard_enabled = false` 时整条不发,主流程继续
- **工具链 / 看板原子加载失败 = 硬阻断,禁止任何降级**:步骤 -1 装机失败 / 步骤 0 鉴权失败 / 步骤 2 同步失败 → **立刻终止整个路由**,把真实失败日志抛给用户(以一条独立的"普通回复消息"形式输出阻断信息;**禁止**调 lark-im / im-chat-manager);**严禁**复用历史 MR / BITS / AiTest 产物冒充本次执行结果。**反模式**:看到 sandbox 没 bytedcli 就走 markdown + 复用历史 MR。⚠️ **步骤 0.5 看板原子 init 失败** 仅在 `dashboard_enabled = true` 时算硬阻断;`dashboard_enabled = false` 时本步整体跳过,**不阻断**主流程
- **平台兼容(2026-06-01 起强制)**:`ctx["dashboard_enabled"]` 由步骤 0.4 一次性判定,后续所有"看板必须 emit""看板硬阻断"约束**全部以该标志为前提**。非 Mira 平台(Codex / Claude Code / 纯 sandbox 容器 / CI worker 等)路由必须能跑完整主流程并产出 MR + 流水线 + 报告,**不允许**因看板不可用就降级或阻断
- **分支与 MR target 约定(2026-05-19 起强制,Agent 不询问用户直接套用)**:
  - **新 feature 分支命名**:统一使用 `feature/auto_dev-<slug>` 格式,`<slug>` 由 Agent 基于 PRD 标题/核心改造点用 kebab-case 自行命名(英文小写、单词以 `-` 分隔、≤50 字符、不含中文)。例:`feature/auto_dev-author-tiexi-fee-deduction`、`feature/auto_dev-monthly-installment-refund`
  - **业务仓库新建 feature 分支基线** = 业务仓库的 `master`(不再使用 `feature/ai_native` 作为基线 — 该分支在多数业务仓不存在)
  - **MR target branch 一律为 `master`**(不再使用 `feature/ai_native` 作为 target;试用期"不直发 master"的旧约束已废除)
  - **子流程仓库 `bytepay/settle_skill` 的读取分支** 仍 = `feature/ai_native`(此项不变,只用于 Agent 读 atom/config,不涉及业务仓 MR)
  - **用户在 PRD 或对话中明确指定其它分支/target 时以用户为准**;否则 Agent **不再就分支/target 询问用户**,直接按上述默认值执行
- **节点职责切分(2026-05-19 起强制,2026-05-20 解耦更新)**:步骤 6 编码节点(code-facade / Coco round-1)**仅负责**「基于 master 拉新分支 → 编码 → commit → push」;**严禁**在编码节点 prompt 中要求 Coco 创建 MR、跑编译/测试、触发流水线。MR 创建是**独立节点**,流水线触发是**第三个独立节点**。**MR 节点与流水线节点彼此不依赖,二者只依赖编码节点产出的远端分支**,触发顺序无强约束(默认并行;若仅串行串行也允许,但禁止把"必须先 MR 后流水线"写成强约束)。BITS 在源分支无 open MR 时会兜底自动创建 MR(等价于 MR 节点产物);路由层应优先复用 BITS 自动开的 MR,而非重复 `mr create`。这三件事**不允许合并**为单个 Coco 任务,也**不允许**在 Coco 任务消息中暗示 Coco 顺手做后两件。
- **Coco sandbox 行为提示**:Coco 在 sandbox 内若自行触发编译/UT(例如 `mvn test` / `go test ./...`),为通过自检而**放宽其它模块测试**(如把脆弱断言改为「不抛异常即可」)的情况已实测发生过。本路由层**禁止**在编码节点 prompt 中要求 Coco 做任何自检;若用户对话中显式要求 Coco 跑自检,需在最终报告的「人工跟进」段落显式列出此类附带改动让 Reviewer 复核
- **complex 路径必须等方案定稿**:步骤 4.7 判定 `complexity=complex` 时,**严禁**在用户回复恢复指令前发起任何 Coco 任务 / 业务仓库写入 / BITS 流水线触发;prd-tech-design 方案一产出就开跑编码 = 反模式。复杂需求的"思考时间"权利属于人,Agent 不能替人压缩
- **决策点产出策略按 facade 路由分流(强制)**:
  - **simple**(facade 路由到内置模板):Agent **必须**列出自判决策点交用户拍板,暂停回复中包含决策点列表
  - **complex**(facade 路由到 prd-tech-design):Agent **禁止**列出任何自行推断的决策点/拍板项/外部依赖问题;暂停回复仅含看板 URL + 技术方案飞书 URL(`tech_design_feishu_url`)+ 复杂度结论 + 路由状态 + 等待提示
  - 违反此规则 = 路由实现错误
- **complex 路径恢复以 prd-tech-design 飞书方案文档为准(强制)**:收到恢复信号后,方案文档来源 = Step B-1 中 prd-tech-design 编排 Agent 已产出的 `tech_design_feishu_url`(同步返回,**无需**任何轮询 API);若用户在恢复指令中附带了新的飞书文档 URL(手动改过方案),**优先使用该 URL**。用 `lark-doc` 拉飞书文档全文作为 tech_solution_md。**禁止**在未拿到方案文档的情况下进入编码步骤;**禁止**让 Agent 自己根据 PRD 编造方案。(2026-07-10 调整:complex 方案来源已由 ArchMate 异步轮询切换为 prd-tech-design 同步编排产出,旧的 `get_trd_by_convId` 轮询通道已下线)
- **complex 路径默认走本地交付(local_handoff,2026-06-01 起强制默认)**:`complex_coding_strategy` 缺省时**严禁**触发 Coco 编码,只产出 handoff 交付包等用户在本地 IDE / Cursor / Codex / Claude Code 完成编码 + push 后回填 `commit_id` 即可恢复;仅当 `team_profile.complex_coding_strategy: auto_coco` 显式覆盖时才走老的 Coco 自动编码路径。违反此规则 = 路由实现错误,产物会高概率返工。
- **complex 路径技术方案权威来源 = prd-tech-design,严禁 Agent 脑补降级(2026-07-10 起强制)**:complex 路径下 handoff 包里的「技术方案」字段**只能**来自 prd-tech-design 编排 Agent 产出的 `tech_design_feishu_url` 飞书文档(通过 `lark-doc` 拉取),飞书创建失败时可兜底为本地 `prd2tech/<需求目录>/tech-design.md` 草稿路径。**严禁** Agent 在 prd-tech-design 不可用(Skill 调用失败 / 阶段 skill 缺失 / 知识库鉴权失败 / 在纯外网容器跑)时,用 PRD / Meego / 复杂度判定理由 / 仓库路由结果 / 知识基线 自己拼一份"技术拆解 / 实现思路 / Step 1/2/3 / 我先帮你梳理一下 / 预分析草稿"塞进 handoff。一旦检测到 handoff 包中出现 Agent 自创的方案性描述段落 → 视为路由实现错误,产物污染必须重做。prd-tech-design 不可用时**正确做法 = 立即停机 + emit「prd-tech-design 不可用」阻断回复**(见 4.7.2 Step B-1 底部模板),等用户手动跑 prd-tech-design / 切平台 / 显式覆盖 `auto_coco` 三选一,**不**降级到脑补。
- **看板 s8 在 complex 路径的状态机**:`run`(prd-tech-design 已启动,方案生成中)→ 用户拍板后 `done`(方案就绪 + 编码启动);simple 路径直接 `done`(轻量方案就地落地)。不允许 complex 路径下 s8 长期 `run` 而 s5/s6 已经在动 —— 一旦看板出现这种"方案没定但代码已动"的态,视为路由实现错误

## 期望输出格式
最终回复**必须**按以下顺序输出(置顶字段缺失视为不合规):

1. **🔗 实时看板**:`<dashboard_url>`(从 `~/files/charge-progress/<session_id>/latest_urls.json.html_url` 读;若全程渲染失败,此处写"看板渲染失败:<原因>",**不可省略本节**)
2. 变更项清单表格(变更描述 / 子流程 / 状态 / 关键产物链接)
3. 飞书报告文档链接(或 Markdown 文件链接 + 飞书授权提示)
4. 后续人工跟进事项(若有)

### complex 路径暂停回复的特殊格式

步骤 4.7 进入暂停时,最终回复格式按 `complex_coding_strategy` 分两种:

**(a)`local_handoff`(默认)— 暂停 emit 完整 handoff 交付包**

格式以 6.3 节定义的 `handoff_prompt_template` 为准,**必须**包含:
1. **🔗 实时看板**:`<dashboard_url>`
2. **🔗 技术方案(prd-tech-design)**:`<tech_design_feishu_url>`
3. **目标仓库与建议分支表格**(每个 matched_repo 一行)
4. **变更要点摘要**(复杂度判定原因 + 关键词 + 参考文件)
5. **回填示意**(commit_id / branch / repo / 可选 MR)
6. **当前路由状态**(s5/s6/s7 = pend, s8 = run, 编码节点 = waiting_for_local_handoff)
7. **不**输出"变更报告"段落(因为代码还没动)

**(b)`auto_coco`(显式覆盖)— 暂停只 emit prd-tech-design 方案等待提示**

1. **🔗 实时看板**:`<dashboard_url>`
2. **🔗 技术方案(prd-tech-design)**:`<tech_design_feishu_url>`(必填,缺失视为不合规)
3. **复杂度判定结论**(`complexity / complex_reasons / matched_dimensions`)
4. **当前路由状态**(明确标注 s5/s6/s7 = pend, s8 = run)
5. **用户接下来要做的事**(评审 prd-tech-design 方案 → 方案定稿后回本会话回复恢复指令,如「方案已定」)
6. **不**输出"变更报告"段落(因为代码还没动)

## 配置化接入指南(给新团队)

auto-develop 已抽象为"公共 SKILL + 团队配置"双层架构。新团队接入步骤:

### 1. 在自家 skill 仓库根目录创建 `config/team_profile.yaml`

参考 `bytepay/settle_skill@feature/ai_native:config/team_profile.yaml` 模板,改值:

```yaml
display_name: "<团队显示名,如 推荐-广告流量>"
skill_repo:   "<group>/<your_skill_repo>"
skill_branch: "<your_branch,如 main>"
meego:
  business_lines:    ["<你们 Meego 的业务线>", ...]
  project_keys:      ["<你们的 project_key>"]
  url_pattern:       'https?://meego\.larkoffice\.com/[\w-]+/story/detail/\d+'
  prd_field_names:   ["需求文档", "PRD 文档", "PRD"]
bits:
  meego_param_mode:  "url"
  space_id:          ""        # 留空 = SSO 默认;需要覆盖时填
  from_dev_id:       ""

# 复杂需求的编码策略(2026-06-01 起默认 local_handoff)
# - local_handoff:complex 路径下不自动编码,等用户在本地 IDE / Cursor / Codex / Claude Code 完成 + push 后回填 commit_id 即可恢复路由
# - auto_coco:旧路径,complex 路径下继续走 Coco sandbox 自动编码(返工率较高,谨慎启用)
complex_coding_strategy: local_handoff
local_handoff:
  max_rounds: 5                       # 流水线失败回环上限
  pipeline_log_tail_lines: 200        # 失败回复中携带的日志尾行数

# 看板能力(2026-06-01 起强制)。缺省时由步骤 0.4 按平台自动探测:
# - Mira 平台(MIRA_PLATFORM=1 / MIRA_SESSION_ID 非空 / ~/files 可写且 mcp__runtime__upload_file 可调) → 启用
# - 非 Mira 平台(Codex / Claude Code / 纯 sandbox 容器 / CI worker 等) → 自动跳过整条看板链路,不阻塞主流程
# 仅在需要强制覆盖时设置以下字段;**不要**两个同时设 true(force_disable 优先)。
dashboard:
  force_enable:  false                # true = 强制启用看板(自建 Mira 兼容平台用)
  force_disable: false                # true = 强制关闭看板(Mira 平台调试主链路时用)
```

### 2. 在你的 skill 仓库根目录创建 `config/repo_routing.yaml` 与 `config/knowledge_sources.yaml`

格式参考 settle_skill 仓库同名文件;字段语义见 `charge-atom-repo-router` / `charge-atom-knowledge-loader` 两个原子的 SKILL.md。

### 3. 在你的 skill 仓库根目录创建 `manifest.yaml`

声明子流程列表(`subflows[].name`、`triggers`、`recommended_subflow`)与原子列表(`atomic_skills`)。

### 4. 把以下原子从 settle_skill 仓库拷贝到自家仓库的 `atoms/` 下,或直接 git submodule 引用

- `atoms/auto-develop-runtime/profile_loader.py`
- `atoms/auto-develop-runtime/prd_meego_resolver.py`
- `atoms/charge-atom-progress-render/`(看板原子)
- `atoms/charge-atom-knowledge-loader/`
- `atoms/charge-atom-repo-router/`

> 这些原子是**公共能力**,与团队解耦;留在 settle_skill 仓库也是历史原因(首批接入)。
> 后续考虑迁出到独立 `auto-develop-runtime` 仓库,各团队 git submodule 引用即可。

### 5. 调用 auto-develop 时注入环境变量

```bash
export AUTO_DEVELOP_BOOTSTRAP_REPO="<group>/<your_skill_repo>"
export AUTO_DEVELOP_BOOTSTRAP_BRANCH="<your_branch>"
# 推荐:把 team_profile.yaml 的绝对路径直接给出来,免得 sandbox 同步失败前找不到
export AUTO_DEVELOP_PROFILE_PATH="$HOME/charge-cache/<your_skill_repo>/config/team_profile.yaml"
```

### 6. 必备前置 CLI 登录态(主机一次性,token 长期复用)

```bash
bytedcli auth login --session --feishu     # 飞书 SSO,Web cookie 长期复用
bytedcli meego login                       # Meego device flow,授权一次即可
```

### 7. 验证

调用 auto-develop 入口,给一条 PRD 或 Meego 链接;观察:
- 步骤 0.2 是否报 profile 加载错误(报了就按提示补字段)
- 步骤 1 双向解析的 `prd_url` / `meego_url` 是否符合预期
- 看板 s1 panel 是否同时展示两个链接
- 步骤 6 BITS 创建命令是否带上了 `--meego <你们的 URL 或 ID>`
- 步骤 6 BITS 创建命令是否显式传了 `--lane <项目标识_随机后缀>`(⛔ 禁止默认 `test`,见「泳道命名规范」)

### 接入 checklist

- [ ] `config/team_profile.yaml` 字段全填
- [ ] `config/repo_routing.yaml` 至少 1 条仓库
- [ ] `config/knowledge_sources.yaml` 可空但要存在
- [ ] `manifest.yaml` 声明 subflows + atomic_skills
- [ ] auto-develop-runtime 原子已就位(profile_loader / prd_meego_resolver)
- [ ] 主机已 `bytedcli auth login --session --feishu` + `bytedcli meego login`
- [ ] 入口环境变量已注入
