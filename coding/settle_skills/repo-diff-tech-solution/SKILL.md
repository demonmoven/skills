---
name: repo-diff-tech-solution
description: 根据一个 Bits 开发任务链接自动提取关联需求(PRD)与代码仓库分支，并可选合并手动补充的额外仓库（按 repo_url+branch 去重），对每个仓库做指定分支 vs base 分支的 diff，结合 PRD 分析变更意图与影响，产出完整技术方案并写入用户个人目录下的飞书文档。能力包括：从 Bits 链接提取需求与仓库、合并去重补充仓库、批量拉取多仓库分支 diff、汇总代码改动、结合 PRD 分析核心改动与影响、识别 HTTP/RPC/API 接口变更生成接口文档（入参/出参/示例/字段变更标注）、基于 diff 规模与提交作者估算工作量及排期（改动点分析/项目成员排期/里程碑）、给出验证与灰度方案（三板斧/发布顺序/发布清单）、生成架构图/流程图/时序图、撰写章节带编号的结构化技术方案并创建飞书文档返回链接。适用于：根据 Bits 开发任务/代码变更和需求文档反向生成技术方案、多仓库联动改动的方案沉淀、PRD + 分支 diff 转技术设计文档等场景。
---

# Repo Diff 技术方案生成

根据多个代码仓库的分支变更（指定分支 vs base 分支的 diff）和 PRD 内容，自动产出完整技术方案并写入用户个人目录下的飞书文档。

## 输入

- `bits_url`（必填）：Bits 开发任务(develop flow)链接，如 `https://bits.bytedance.net/devops/{space_id}/develop/detail/{dev_id}/flow?...`。由 `bytedcli` 自动提取：关联需求(Meego/PRD) 与 关联的所有代码仓库+开发分支。
- `extra_repos`（可选）：额外手动补充的仓库列表，用于兜底 Bits 中未挂载/未绑定 MR 的仓库。每项包含：
  - `repo_url`：公司内网 Git 仓库地址，如 `https://code.<company-dev-domain>/bytepay/charge_prod_sdk`
  - `branch`：要对比的分支名（与该仓库 base 分支做 diff，base 默认 `master`）

规则：
- 从 Bits 提取的仓库与 `extra_repos` **合并后按 `repo_url + branch` 去重**，统一进入后续 diff → 分析 → 画图 → 写文档流程，避免遗漏未挂到 Bits 的仓库。
- PRD 来自 Bits 关联的 Meego 需求；若无法自动获取（见下），可让用户直接提供 PRD 文本/飞书链接，用 `lark-doc` skill 读取正文作为 `prd_content`。

## 产出

一个飞书技术方案文档链接（存放在用户个人目录下），所有一级/二级章节带编号，文档包含：背景/需求说明、变更范围概述、核心改动分析、接口文档（涉及接口变更时）、影响分析、工作量及排期（改动点分析/项目成员及排期/里程碑）、验证与灰度（三板斧/发布顺序/发布清单），以及架构图、流程图、时序图。

## 工作流

整体是一条流水线：**Bits 提取(+合并补充仓库) → 拉 diff → 结合 PRD 分析 → 画图 → 写文档**。前一步产物是后一步的输入，不要跳步或凭空编造代码改动。

### 1. 从 Bits 链接提取需求与仓库（并合并补充仓库）

用自带脚本从 Bits 开发任务链接提取需求与关联仓库分支（底层调用已安装的 `bytedcli`，走 SSO 鉴权）。若有 `extra_repos`，先写成 JSON 文件用 `--extra-repos` 传入，脚本会自动合并去重：

```bash
cd user_skills/repo-diff-tech-solution
# 可选：把 extra_repos 写入文件（无补充仓库则省略 --extra-repos）
cat > output/extra_repos.json <<'EOF'
[{"repo_url": "https://code.<company-dev-domain>/xxx/yyy", "branch": "feature/z"}]
EOF
python3 scripts/extract_from_bits.py --bits-url "<bits_url>" --extra-repos output/extra_repos.json --output-dir output
```

产物：
- `output/bits_context.json`：`title`、`requirement`（Meego 需求 id/url/name）、`repos`（Bits 提取的绑定 MR 仓库）、`merged_repos`（合并去重后的最终仓库，含 `origin=bits/extra` 来源标记）、`extra_repos_added`、`prd`（PRD 抽取情况）。
- `output/repos.json`：**已合并 Bits + extra_repos 并按 `repo_url+branch` 去重**的仓库配置，第 3 步可直接用。

提取原理（供理解，不必额外调用）：仓库+分支来自 `bytedcli bits develop inspect-changes`（实际绑定 MR 的 code-change 卡片，最可靠）；diff base 取每个绑定的 `target_branch`（通常 master）；需求来自 `bytedcli bits develop get` 的 `related.workItems`。

**PRD 获取**：脚本会尝试从关联的 Meego 需求里抽取 PRD 飞书文档链接：
- 若 `bits_context.json` 里 `prd.prd_urls` 非空 → 用 `lark-doc` skill 下载读取该飞书文档正文作为 `prd_content`。
- 若 `prd.need_login=true`（Meego 未登录，抽取 PRD 需要）→ 提示用户在终端执行 `bytedcli meego login` 后重试；或请用户直接提供 PRD 文本/飞书链接。**不要**在拿不到 PRD 时编造需求背景。
- 若脚本以非零码退出（合并后仍无任何仓库）→ 检查链接是否为有效的 develop flow 链接、`bytedcli` 是否鉴权正常（`bytedcli bits auth`），或让用户用 `extra_repos` 补充，再重试。

> 脚本通过 `bash` 工具直接执行；`bytedcli` 依赖当前环境凭证，若报鉴权错误用 include_secrets=true 重试。

### 2. 准备输入

- 最终要处理的 `repos` 直接用 `output/repos.json`（已是 Bits + extra_repos 合并去重结果）。校验非空、每项都有 `repo_url` 和 `branch`。
- 确认拿到 `prd_content` 文本（从提取到的 PRD 飞书文档读取；或用户提供的 PRD 文本/链接）。

### 3. 拉取各仓库分支 diff

把 repos 写成配置文件，用脚本批量拉取（脚本会 blobless 部分克隆、找 merge-base、计算三点 diff `base...branch`，并处理 base 分支为 master/main 的差异）。入口 A 已生成 `output/repos.json`，可直接复用；入口 B 手动写：

```bash
cat > output/repos.json <<'EOF'
{
  "repos": [
    {"repo_url": "https://code.<company-dev-domain>/bytepay/charge_prod_sdk", "branch": "feature/x"}
  ],
  "output_dir": "output/repo_diffs",
  "base": "master"
}
EOF
python3 scripts/fetch_repo_diffs.py --config output/repos.json
```

产物在 `output/repo_diffs/`：
- `diff_summary.json`：结构化摘要（每仓库变更文件数、增删行、changed_files 列表、diff 文件路径、是否截断）。
- `diff_summary.md`：可读汇总表。
- `<repo>__<branch>.diff`：每个仓库的完整 diff（超限自动截断，用 changed_files 补全全貌）。

若某仓库克隆/找分支失败，脚本会在摘要里记录 `error` 并继续处理其余仓库。全部失败时脚本以非零码退出——此时先排查仓库地址/分支名/网络与 git 凭证，再重试，不要在没有 diff 的情况下硬编写方案。

> 该脚本通过 `bash` 工具直接执行；访问公司内网 Git 依赖当前环境的 git 凭证，若报鉴权错误，用 include_secrets=true 重试。

### 4. 结合 PRD 分析变更

阅读 `diff_summary.md`、各 `.diff` 文件全文和 `prd_content`，形成结构化理解——这是方案质量的核心，不能省。重点回答：

- **变更意图**：PRD 想解决什么问题？每处代码改动对应 PRD 的哪条需求？
- **核心改动**：按「模块 / 接口 / 数据结构」归类。新增或修改了哪些接口（签名、入参、返回）？哪些数据结构/表结构/配置发生变化？跨仓库改动如何联动。
- **接口文档素材**：识别 diff 中涉及 HTTP handler / RPC handler / API 接口的变更（路由注册、gin/hertz/kite handler、thrift/protobuf service 方法、IDL、请求/响应 struct/DTO），逐个记录接口名、路由/方法、入参/出参字段（名/类型/必填/说明）及本次新增(✨)/修改(⚠️)/删除(❌)的字段。字段依据来自 diff 中的 IDL/struct/注解，不可编造。
- **影响分析**：受影响的上下游依赖、调用方；兼容性/数据迁移/回滚风险；需要关注的测试与灰度点。
- **工作量与排期素材**：按仓库归纳核心改动点及规模（文件数/增删行/接口复杂度，来自 `diff_summary.md`）用于粗估研发人日；负责人从 `diff_summary.md` 的「提交作者」（即 MR/commit author）或 Bits 需求创建人提取，未知记「待指派」，不要编造。
- **验证与灰度素材**：可监控的指标/告警（结合改动的核心链路与指标）、可灰度的开关/配置位（若 diff 中有特性开关/TCC/配置项则记下其名）、可应急的回滚方式；以及各仓库/DML/配置的依赖关系，用于推导发布顺序。

分析结论落到本地 `output/analysis_notes.md`，作为画图的事实来源和写文档的依据。文档章节结构与写作要点见 [references/solution-outline.md](references/solution-outline.md)。

### 5. 生成图表

**使用 `lark-whiteboard` skill** 生成三类图（分别产出 `.svg` / `.puml` / `.mermaid` 源文件，放在 `output/` 下），把第 4 步的 `analysis_notes.md` 作为事实依据传给它：

- **架构图**：改动涉及的系统/服务/模块及其关系（体现本次变更触及的部分）。
- **流程图**：新增或变化的业务处理流程 / 核心链路。
- **时序图**：关键交互的调用时序（跨仓库/跨服务的接口调用顺序）。

只描述每张图要表达的内容（需求卡片），把布局、配色、图型细节交给 `lark-whiteboard` skill 决定。拿到源文件路径后用于第 6 步嵌入。

### 6. 撰写技术方案并创建飞书文档

**使用 `lark-doc` skill** 完成方案文档撰写与飞书文档创建：

- 按 [references/solution-outline.md](references/solution-outline.md) 的章节撰写 `.lark.md`。**所有一级/二级章节标题统一带编号**（一级 `1.`/`2.`…，二级 `1.1`/`3.2`…，TL;DR callout 除外）。章节包含：背景/需求说明、变更范围概述、核心改动分析（模块/接口/数据结构）、**接口文档**（涉及 HTTP/RPC/API 接口变更时，含接口名与路由/方法、入参/出参表、请求/响应 JSON 示例，字段变更用 ✨新增/⚠️修改/❌删除 标注）、影响分析（依赖、风险）、**工作量及排期**（6.1 改动点分析表「所属系统/改动点/负责人/研发估时」、6.2 项目成员及排期「分系统估时表 + 里程碑节点表」）、**验证与灰度**（三板斧「可监控/可灰度/可应急」、结合仓库依赖的发布顺序、含执行人/预期结果/回滚条件的发布清单 CheckList）。
- **排期日期计算**：「工作量及排期」的日期以执行当天真实日期为起点、按工作日顺推（跳过周末），先用 `truetime` skill 确认今天日期再推算，禁止照抄示例里的旧日期；工时与负责人须有依据（见第 4 步素材），不可编造人名。
- 用 `![preview](output/xxx.svg)`（或 `.puml` / `.mermaid`）把第 4 步的架构图、流程图、时序图嵌入到对应章节，让它们在飞书里渲染为可交互画板。
- 这是一份技术方案文档，撰写时遵循 `lark-doc` skill 对 design-doc 类文档的实例与写作规范。
- 创建飞书文档时**要求存放到用户个人空间目录下**（个人目录），标题不含用户名。

### 7. 返回结果

把 `lark-doc` 返回的飞书文档链接交付给用户，并简要说明覆盖了哪些仓库、变更规模（可引用 `diff_summary.md` 的统计），一并附上 Bits 需求链接与各仓库 MR 链接。

## 依赖说明

本 skill 只做编排，具体能力交给对应 skill/工具，不直接调用其内部脚本或 MCP 工具：

- 从 Bits 链接提取需求与仓库：本 skill 自带 `scripts/extract_from_bits.py`（底层调用系统已安装的 `bytedcli`）。
- 拉取 diff：本 skill 自带 `scripts/fetch_repo_diffs.py`。
- 画图（架构图/流程图/时序图）：使用 `lark-whiteboard` skill。
- 读取 PRD 飞书文档、撰写并创建飞书文档：使用 `lark-doc` skill。

## 边界与注意

- diff 用三点 `base...branch`（merge-base 到分支），即分支相对 base 引入的改动，等价于常见的 MR/PR 变更范围。
- 仓库+分支来自实际绑定 MR 的 code-change 卡片；若开发任务尚未绑定 MR 或有仓库未挂到 Bits，用 `extra_repos` 手动补充，脚本会与 Bits 结果合并去重。
- PRD 依赖 Meego：Meego 需单独 `bytedcli meego login`（交互式 OAuth）。未登录时脚本会标记 `need_login`，此时提示用户登录后重试，或让用户直接提供 PRD 文本/飞书链接，**不要**在缺 PRD 时编造需求背景。
- 大 diff 会被截断，分析时结合 `changed_files` 列表把握全貌，不要因为截断遗漏关键文件。
- 接口文档的入参/出参/示例字段必须来自 diff 中的 IDL/handler/struct/注解，无法推断的字段如实标注「diff 中未体现」，不要编造；本次无对外接口变更时省略「接口文档」章节并顺延后续编号。
- 发布顺序需结合各仓库/DML/配置的真实依赖关系推导并说明理由，执行人默认取需求创建人/改动作者，无法确定时写「待指派」，不要臆造具体人名。
- 工作量及排期：研发估时按 diff 规模粗估并与统计自洽，负责人取 `diff_summary.md` 的提交作者或 Bits 创建人（未知写「待指派」）；排期日期以执行当天真实日期为起点按工作日顺推，用 `truetime` skill 校准，不套用旧示例日期。
- 若 PRD 与代码改动明显不一致，在「影响分析」中如实指出差异，不要强行圆场。
