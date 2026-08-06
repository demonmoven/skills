# auto-develop 知识库 facade 接入 caijing-knowledge (v3.1) — PR 草稿

> **提交目标**:`bytepay/settle_skill@feature/ai_native`
> **PR 标题建议**:`feat(knowledge-facade): 接入 caijing-knowledge 作为 default_atom + tree 裁剪 + ArchMate 上下文注入 + 跨 bizcode 复用`

## 一、背景

auto-develop 路由层「步骤 3 — insearch 关键词增量召回」当前调用 `bytedance-insearch`,存在两类问题:
1. **召回精度低**:insearch 全字节内网检索,跨业务噪声大,关键词匹配粒度粗
2. **缺少结构化知识**:没有 A1/A2/A3 架构层级、PSM 服务清单、Interface/Table 关系等信息,下游 Coco/ArchMate 拿不到精确上下文

财经业务架构知识图谱 [caijing-knowledge](https://bytedance.larkoffice.com/wiki/HEpcwMRAzibD8ekGkZ2cmE5lnvg) 正是为计费域设计的检索能力,提供:
- 4 个 bizcode(支付/消金/风控/保险)的全量 A1/A2/A3 + PSM + Interface + DB Table 结构化图谱
- 基于 ByteRAG 的 wiki 文档切片召回(按 A2 收敛,精度远高于全局 insearch)
- 既支持 Agentic Search(tree + search 组合)又支持 RAG 召回(retrieve-doc)

## 二、改动文件清单

```
+ atoms/charge-atom-caijing-knowledge/SKILL.md         # 原子契约文档
+ atoms/charge-atom-caijing-knowledge/kg_query.py      # 原子实现 (457 行 Python, 零运行时依赖)
~ manifest.yaml                                         # 见 patches/manifest.patch.yaml
~ tech-solution-gen/SKILL.md                            # 见 patches/tech-solution-gen.patch.md
~ config/team_profile.yaml                              # 见 patches/team_profile.patch.yaml
~ auto-develop/SKILL.md (cjadk 自检 + 步骤顺序微调)     # 见本文档 § 三-D
```

## 三、改动详情

### 改动 A:新增原子 `charge-atom-caijing-knowledge`

封装 `cjadk knowledge` 的三个 CLI 为单次 JSON-in/JSON-out 调用,内部自动:
1. **bizcode 推断**:explicit → `scope_to_bizcode[scope]` → 关键词推断 → 默认 `bizcode_cj_pay`,四级兜底
2. **A2 推断**:跑 tree → 解析二级目录 → 与 PRD keywords 求子串交集 → 自动填 `retrieve-doc` 的 `--a2-list`
3. **tree 裁剪**:tree 行数 > 200 时按 keywords 命中度收敛子树到 ≤120 行,直接喂给下游 prompt
4. **退出码**:`tree` 成功 → exit 0;`tree` 失败或 cjadk 未装 → **exit 1 硬停**,不允许 fallback 到 insearch
5. **retrieve-doc 容错**:HTTP 500(`resConfigMap empty` = A2 未导入 wiki)记为 warn 不影响退出码

输入示例:
```json
{"mode":"auto","scope":"charge","keywords":["计费","对账"],"prd_text":"..."}
```

输出包含完整 tree 文本 + 裁剪 tree + summary 统计 + errors 审计。详细契约见 `atoms/charge-atom-caijing-knowledge/SKILL.md`。

### 改动 B:`knowledge-facade` 切换 default_atom + 失败硬停 + 暴露新字段

```yaml
# Before
default_atom: knowledge-search   # = bytedcli/bytedance-insearch
fallback_strategy: warn_continue
outputs: [hits, top_links, used_keywords]

# After
default_atom: charge-atom-caijing-knowledge
fallback_strategy: stop_and_ask  # ⇐ 失败即停,不 fallback 到 insearch
outputs:
  - hits                                      # 老契约保留 (= summary.psm_hits + a2_hits)
  - top_links                                 # 老契约保留 (= [], cjadk 不返回 URL)
  - used_keywords                             # 老契约保留 (透传 input.keywords)
  - biz_code_used                             # 新增
  - a2_hits                                   # 新增
  - tree                                      # 新增 (审计/降级用,~600 行)
  - tree_pruned                               # 新增 (~100 行, 优先用于 prompt 注入)
  - psm_hits                                  # 新增
  - retrieve_doc                              # 新增
routing_rules:
  # 删除了原 caijing_exit == 1 的 fallback 规则,不再允许 fallback
  - when: keywords contains "跨域"  # 仅显式跨域调研走 insearch
    atom: knowledge-search
```

**fallback_strategy 从 `warn_continue` 改为 `stop_and_ask`**:`charge-atom-caijing-knowledge` 效果已验证稳定,失败必然是环境问题(cjadk 未装/鉴权过期/网络不通),必须停下来让用户排查,不能静默降级到精度更低的 insearch。

### 改动 C:`tech-solution-facade` 简单/复杂分支都消费 `knowledge_context`(必填)

`knowledge_context` 从"可选"升级为**必填**。两条分支都消费:

| 分支 | 消费方式 |
|---|---|
| 简单(`__inline_template__`) | 内置模板把 `summary.psm_hits` / `tree_pruned` / `retrieve_doc.raw` 拼为 "## 业务架构上下文" 区块,注入到生成的 `tech_solution_md` 开头 |
| 复杂(`tech-solution-gen`) | Step 2.5 `enrich_user_prompt()` 把同样的区块前置到 ArchMate API 的 `user_prompt` |

**为什么简单分支也要**:即使是简单需求,`tech_solution_md` 中带上 PSM 列表和架构子树能让下游 `coding-facade` 准确定位到具体服务和模块,减少 Coco 编码阶段的猜测和误改。

详见 `patches/tech-solution-gen.patch.md`。

### 改动 D:跨 bizcode 复用 — `team_profile.yaml` 新增 `scope_to_bizcode` 节

```yaml
# config/team_profile.yaml 追加
scope_to_bizcode:
  charge: bizcode_cj_pay
  settle: bizcode_cj_pay
  billing: bizcode_cj_pay
  pay: bizcode_cj_pay
  xj: bizcode_cj_xj
  credit: bizcode_cj_xj
  risk: bizcode_cj_risk
  insurance: bizcode_cj_insurance
  test: bizcode_cj_test
```

`kg_query.py` 启动时通过环境变量 `AUTO_DEVELOP_PROFILE_PATH` 读取本节,覆盖内置 `DEFAULT_SCOPE_TO_BIZCODE`。**消金 / 风控 / 保险团队接入只需在自己 skill 仓库的 `team_profile.yaml` 写自己的映射即可,不动 kg_query.py**。

退化路径:无 PyYAML → 简单行解析;再失败 → 用内置默认表(charge 团队零中断)。详见 `patches/team_profile.patch.yaml`。

### 改动 E:auto-develop SKILL.md 路由层调整(子流程级)

#### E.1 step 0.5 自检 cjadk (fail-fast, 不允许跳过)

```bash
# cjadk 自检 + 自动安装 — 必须成功才继续,安装失败整条 auto-develop 停止
command -v cjadk >/dev/null 2>&1 || {
  echo "[step 0.5] cjadk 缺失,自动安装..."
  npm install -g @byted/cjadk@latest --registry=https://bnpm.byted.org \
    || { echo "❌ cjadk 安装失败,无法继续。请手动执行: npm install -g @byted/cjadk@latest --registry=https://bnpm.byted.org"; exit 1; }
  echo "[step 0.5] cjadk 安装完成"
}
# 导出 team_profile 路径供 kg_query.py 跨 bizcode 复用
export AUTO_DEVELOP_PROFILE_PATH="$(realpath config/team_profile.yaml)"
```

#### E.2 子流程步骤顺序调整:knowledge 提前到 tech-solution 之前

| 老顺序 | 新顺序 |
|---|---|
| 2 prd → 3 repo → 4 tech-solution → 5 knowledge → 6 coding | 2 prd → 3 repo → 4 **knowledge** → 5 **tech-solution(消费 knowledge_context)** → 6 coding |

理由:knowledge-facade 输出的 `biz_code_used / summary / tree_pruned / retrieve_doc` 要透传给 tech-solution-facade complex 路径,作为 `knowledge_context` 注入 ArchMate user_prompt。

## 四、自测记录(沙箱真实环境)

| 测试 | 输入 | 结果 |
|---|---|---|
| cjadk install | `npm i -g @byted/cjadk` | ✅ 7s 完成 |
| tree 全量 | `--path "" --biz-code bizcode_cj_pay --entity-types arch,psm` | ✅ 8s 返回 24 个 A2 + 750+ PSM |
| bizcode 推断 (scope) | `scope=charge` | ✅ 命中 `bizcode_cj_pay`, source=scope |
| bizcode 推断 (team_profile 覆盖) | `AUTO_DEVELOP_PROFILE_PATH=/tmp/fake_profile.yaml` + `scope=custom_scope` | ✅ 命中 `bizcode_cj_xj`, source=team_profile:/tmp/... |
| A2 推断 | `keywords=["聚合计费结算","对账"]` | ✅ 从 tree 推出 `a2_list=["聚合计费结算"]` |
| tree 裁剪 | tree 原 612 行, keywords=["聚合"], max=120 | ✅ 裁剪到 98 行, kept_a2=["聚合计费结算"] |
| retrieve-doc | A2 = 聚合计费结算 | ⚠️ 500 `resConfigMap is empty` → 转为 warn,主流程 exit 0 |
| 端到端 auto | scope+keywords+prd_text 完整输入 | ✅ 10.8s 完成,tree 成功 + 裁剪 + retrieve-doc warn |
| 语法 lint | `python3 -c "import ast; ast.parse(...)"` | ✅ 457 行无错误 |

## 五、影响面 / 回滚方案

- **影响子流程**:`general_dev` / `channel_charge` / `backend` / `frontend` / `config` 调用 `knowledge-facade` 的步骤。outputs 字段向后兼容,无需修改。
- **影响 knowledge-facade**:失败即停(`stop_and_ask`),不再有 insearch fallback。这是有意的:caijing-knowledge 效果远好于 insearch,失败时必须排查环境而非降级。
- **影响 tech-solution-facade**:`knowledge_context` 为必填,简单/复杂分支都消费。由于 knowledge-facade 先于 tech-solution-facade 执行且 `stop_and_ask`,走到 tech-solution 时 `knowledge_context` 一定非空。
- **回滚**:把 manifest 中 `knowledge-facade.default_atom` 改回 `knowledge-search`、`fallback_strategy` 改回 `warn_continue`、`tech-solution-facade.inputs` 去掉 `knowledge_context`、删除新原子目录即可。无数据迁移。

## 六、未做的事(后续 PR)

1. caijing-knowledge 的 search 模式接入(本 PR 在 auto 模式下,只有 path 含 `/` 时才跑 search;后续可在 facade 层加 `mode=search` 显式触发)
2. 路由层 step 0 把 `AUTO_DEVELOP_PROFILE_PATH` 写进 systemd-style 全局 env(目前依赖 SKILL.md 的 export)
3. tech-solution-gen Step 2.5 的 `enrich_user_prompt` 实现从 patch 落到 SKILL.md 的代码示例区,后续考虑提取为独立 helper 脚本
4. revision-id 弱缓存 + tags 相关性裁剪(原 v1 patch 的 knowledge-loader 优化,与本 PR 解耦,可独立提交)
